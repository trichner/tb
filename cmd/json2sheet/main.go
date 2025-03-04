package json2sheet

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/trichner/tb/pkg/json2sheet"
)

var cli struct {
	SpreadsheetUrl string `help:"complete URL to the spreadsheet"`
}

func Exec(ctx context.Context, args []string) {
	parser := kong.Must(&cli, kong.Name(args[0]))
	_, err := parser.Parse(args[1:])
	parser.FatalIfErrorf(err)

	spreadsheetUrl := strings.TrimSpace(cli.SpreadsheetUrl)
	if spreadsheetUrl != "" {
		url, err := json2sheet.UpdateSheet(ctx, spreadsheetUrl, os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(url)
	} else {
		url, err := json2sheet.WriteToNewSheet(ctx, os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(url)
	}
}
