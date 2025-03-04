package sheets

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/trichner/tb/pkg/oauth2tokenstore"

	"golang.org/x/oauth2"

	"github.com/trichner/oauthflows"
	"golang.org/x/oauth2/google"
	. "google.golang.org/api/option"
	googlesheets "google.golang.org/api/sheets/v4"
)

const (
	keyringItemServiceName = "toolbox googleapis.com"
)

var scopes = []string{
	"https://www.googleapis.com/auth/drive",
	"https://www.googleapis.com/auth/drive.file",
	"https://www.googleapis.com/auth/drive.readonly",
	"https://www.googleapis.com/auth/spreadsheets",
	"https://www.googleapis.com/auth/spreadsheets.readonly",
}

type SheetsService interface {
	CreateSpreadSheet(title string) (SpreadsheetOps, error)
	GetSpreadSheet(id string) (SpreadsheetOps, error)
}

type sheetsService struct {
	service *googlesheets.Service
}

type SpreadSheet struct {
	Id     string
	Sheets []*Sheet
}

type Sheet struct {
	Id    int64
	Title string
	Index int64
}

func NewSheetService(ctx context.Context) (SheetsService, error) {
	var err error
	var client *http.Client

	oauthConfig := &oauth2.Config{
		ClientID:     deobfuscate("11f54bda98e094e03def692dffdfa93fb0773efcf9258ea5f28c4a3e9825a102d2c0628a3a7205c43f719c757ae2dfcbae35eddd2e5c3f276c83ed10259203240be3afa1ce519e"),
		ClientSecret: deobfuscate("74a04fb1f8fbc9b443ea2345e2fbad3198607ed6f22cdad9"),
		Endpoint:     google.Endpoint,
		Scopes:       scopes,
	}

	if client, err = google.DefaultClient(ctx, scopes...); err == nil {
		// client already set
		slog.Warn("using default client, use 'gcloud auth application-default revoke' to setup new credentials or make sure to use the correct scopes", "scopes", strings.Join(scopes, ", "))
	} else if client, err = oauthflows.NewClient(oauthflows.WithConfig(oauthConfig), oauthflows.WithTokenStore(oauth2tokenstore.NewKeyringTokenStore(keyringItemServiceName))); err == nil {
		// client already set
	} else {
		return nil, fmt.Errorf("cannot initialize oauth client: %w", err)
	}

	service, err := googlesheets.NewService(ctx, WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("cannot create service: %w", err)
	}

	return &sheetsService{service: service}, nil
}

func (s *sheetsService) GetSpreadSheet(id string) (SpreadsheetOps, error) {
	res, err := s.service.Spreadsheets.Get(id).Do()
	if err != nil {
		return nil, err
	}
	return &spreadsheetOps{
		service:     s.service,
		spreadsheet: res,
	}, nil
}

func (s *sheetsService) CreateSpreadSheet(title string) (SpreadsheetOps, error) {
	ss := &googlesheets.Spreadsheet{
		Properties: &googlesheets.SpreadsheetProperties{Title: title},
	}
	res, err := s.service.Spreadsheets.Create(ss).Do()
	if err != nil {
		return nil, fmt.Errorf("cannot create spreadsheet: %w", err)
	}
	return &spreadsheetOps{
		service:     s.service,
		spreadsheet: res,
	}, nil
}
