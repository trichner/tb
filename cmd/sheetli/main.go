package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/trichner/tb/pkg/sheets"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <spreadsheetId>\n", os.Args[0])
		return
	}
	spreadsheetId := os.Args[1]

	ctx := context.Background()

	service, err := sheets.NewSheetService(ctx)
	if err != nil {
		log.Fatalf("cannot create service: %v", err)
	}

	sheet, err := service.GetSpreadSheet(spreadsheetId)

	spreadsheet, err := sheet.Get()

	allSheets := spreadsheet.Sheets
	sort.Slice(allSheets, func(i, j int) bool {
		return allSheets[i].Index < allSheets[j].Index
	})

	fmt.Printf("ID: %s\n", spreadsheet.Id)
	fmt.Printf("Index             Id     Title\n")
	for _, s := range allSheets {
		fmt.Printf("%3d     %12d \t %s\n", s.Index, s.Id, s.Title)
	}
}
