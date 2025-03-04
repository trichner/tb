package kraki

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/trichner/oauthflows"
	"github.com/trichner/tb/pkg/directory"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/storage/v1"
	gvault "google.golang.org/api/vault/v1"
)

const clientSecretFilepath = "client_secret.json"

var scopes = []string{
	admin.AdminDirectoryUserScope,
	gvault.EdiscoveryScope,
	storage.DevstorageReadOnlyScope,
}

type ExportResource int

const (
	ResourceNone ExportResource = iota
	ResourceEmail
	ResourceDrive
)

var cli struct {
	SuspendUser struct {
		Email     string `help:"Email of the user to suspend" required:""`
		Suspended bool   `help:"Suspended of the user, 'true' for suspended" required:"" default:"true"`
	} `cmd:"" help:"Create a new user."`
	DeleteUser struct {
		Email string `help:"Email of the user to delete" required:""`
	} `cmd:"" help:"Create a new user."`
	ExportUser struct {
		Email     string `help:"Email of the user to export" required:""`
		Resources string `help:"Resources to export, comma separated" default:"email,drive"`
	} `cmd:"" help:"export users resources"`
	BatchExport struct {
		File      string `help:"List of emails to export" required:""`
		Resources string `help:"Resources to export, comma separated" default:"email,drive"`
	} `cmd:"" help:"export users resources"`
	BatchDelete struct {
		File string `help:"List of emails to delete" required:""`
	} `cmd:"" help:"export users resources"`
	DescribeMatter struct {
		MatterId string `help:"ID of the matter" required:""`
	} `cmd:"" help:"Add groups to an existing user."`
	DownloadExport struct {
		MatterId string `help:"ID of the matter" required:""`
	} `cmd:"" help:"Download all exports of a matter"`
}

func Exec(c context.Context, args []string) {
	k, err := kong.New(&cli)
	if err != nil {
		log.Fatal(err)
	}
	ctx, err := k.Parse(args[1:])
	if err != nil {
		log.Fatal(err)
	}

	switch ctx.Command() {
	case "suspend-user":

		email := cli.SuspendUser.Email
		suspended := cli.SuspendUser.Suspended
		err := suspendUser(email, suspended)
		if err != nil {
			log.Fatal(err)
		}
	case "export-user":
		resources := parseExportResources(cli.ExportUser.Resources)
		email := cli.ExportUser.Email
		err := exportUser(email, resources)
		if err != nil {
			log.Fatal(err)
		}
	case "describe-matter":
		matterId := cli.DescribeMatter.MatterId
		err := describeMatter(matterId)
		if err != nil {
			log.Fatal(err)
		}
	case "delete-user":
		err := deleteUser(cli.DeleteUser.Email)
		if err != nil {
			log.Fatal(err)
		}
	case "download-export":
		matterId := cli.DownloadExport.MatterId
		err := downloadExport(matterId)
		if err != nil {
			log.Fatal(err)
		}
	case "batch-export":
		file := cli.BatchExport.File
		resources := parseExportResources(cli.BatchExport.Resources)
		err := batchExport(file, resources)
		if err != nil {
			log.Fatal(err)
		}
	case "batch-delete":
		file := cli.BatchDelete.File
		err := batchDelete(file)
		if err != nil {
			log.Fatal(err)
		}
	default:
		panic(ctx.Command())
	}
}

func suspendUser(email string, suspended bool) error {
	ctx := context.Background()

	config, err := getOAuth2Config()
	if err != nil {
		return err
	}
	config.Scopes = scopes

	src, err := oauthflows.NewBrowserFlowTokenSource(ctx, config)
	if err != nil {
		return err
	}

	svc, err := directory.NewService(ctx, src)
	if err != nil {
		return err
	}

	user, err := svc.SuspendUserByPrimaryEmail(ctx, email, suspended)
	if err != nil {
		return err
	}

	return printJson(user)
}

func getOAuth2Config() (*oauth2.Config, error) {
	slurp, err := ioutil.ReadFile(clientSecretFilepath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", clientSecretFilepath, err)
	}

	config, err := google.ConfigFromJSON(slurp)
	if err != nil {
		return nil, fmt.Errorf("cannot parse config %s: %w", clientSecretFilepath, err)
	}
	return config, nil
}

func parseExportResources(s string) []ExportResource {
	resources := []ExportResource{}
	splits := strings.Split(s, ",")
	for _, r := range splits {
		resources = append(resources, mapResource(r))
	}
	return resources
}

func mapResource(s string) ExportResource {
	switch s {
	case "email":
		return ResourceEmail
	case "drive":
		return ResourceDrive
	}
	return ResourceNone
}

func printJson(i interface{}) error {
	bytes, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Printf("%s", bytes)
	return err
}
