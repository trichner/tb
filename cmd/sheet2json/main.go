package sheet2json

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"

	"github.com/alecthomas/kong"
	"github.com/posener/complete/v2"
	"github.com/posener/complete/v2/predict"
	"github.com/trichner/tb/pkg/sheet2json"
)

var cli struct {
	SpreadsheetID  string `help:"spreadsheet ID"`
	SheetID        int64  `help:"ID of the sheet within the spreadsheet"`
	SpreadsheetUrl string `help:"complete URL to the spreadsheet"`
}

func Completions() complete.Completer {
	parser := kong.Must(&cli)

	flags := map[string]complete.Predictor{}
	for _, f := range parser.Model.Flags {
		flags[f.Name] = predict.Something
		if f.Short != 0 {
			flags[string(f.Short)] = predict.Something
		}
	}
	return &complete.Command{Flags: flags}
}

func Exec(ctx context.Context, args []string) {
	parser := kong.Must(&cli, kong.Name(args[0]))
	_, err := parser.Parse(args[1:])
	parser.FatalIfErrorf(err)

	var spreadsheetId string
	var sheetId int64 = -1
	if cli.SpreadsheetUrl != "" {
		spreadsheetId, sheetId, err = urlToSpreadsheetID(cli.SpreadsheetUrl)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		spreadsheetId = cli.SpreadsheetID
		sheetId = cli.SheetID
	}

	if spreadsheetId == "" || sheetId < 0 {
		log.Fatal(fmt.Errorf("spreadsheetId and sheetId are not set"))
	}

	err = sheet2json.ReadFromSheet(ctx, spreadsheetId, sheetId, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

// urlToSpreadsheetID parses a URL to a spreadsheet such as: https://docs.google.com/spreadsheets/d/1dAN8MO9NDVPqVIoOxC9H_j4Ir5c1viQ97igGdXOyXsU/edit#gid=886605725
func urlToSpreadsheetID(u string) (string, int64, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", -1, err
	}

	const googleDocsHost = "docs.google.com"
	if parsed.Host != googleDocsHost {
		return "", -1, fmt.Errorf("unexpected host '%s', expected '%s'", parsed.Host, googleDocsHost)
	}

	const httpsScheme = "https"
	if parsed.Scheme != "https" {
		return "", -1, fmt.Errorf("unexpected scheme '%s', expected '%s'", parsed.Scheme, httpsScheme)
	}

	pathPattern := regexp.MustCompile("^/spreadsheets/d/([^/]+)/edit$")
	matches := pathPattern.FindStringSubmatch(parsed.Path)
	if matches == nil {
		return "", -1, fmt.Errorf("can't find spreadsheetId in path: '%s'", parsed.Path)
	}
	spreadsheetId := matches[1]

	q, err := url.ParseQuery(parsed.Fragment)
	if err != nil {
		return "", -1, fmt.Errorf("can't parse fragment '%s': %w", parsed.Fragment, err)
	}

	const queryParamGid = "gid"
	rawSheetId := q.Get(queryParamGid)
	if rawSheetId == "" {
		return "", -1, fmt.Errorf("can't find '%s' in '%s'", queryParamGid, parsed.Fragment)
	}

	sheetId, err := strconv.ParseInt(rawSheetId, 10, 64)

	return spreadsheetId, sheetId, err
}
