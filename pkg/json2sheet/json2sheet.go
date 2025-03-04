package json2sheet

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/trichner/tb/pkg/sheets"
)

const (
	streamTypeUnknown = iota
	streamTypeObjects
	streamTypeArrays
)

type SheetUpdater interface {
	UpdateValues(data [][]string) error
}

type SheetAppender interface {
	AppendValues(data [][]string) error
}

func UpdateSheet(ctx context.Context, spreadsheetUrl string, r io.Reader) (*url.URL, error) {
	svc, err := sheets.NewSheetService(ctx)
	if err != nil {
		return nil, err
	}

	spreadsheetID, sheetID, err := sheets.ParseSpreadsheetUrl(spreadsheetUrl)
	if err != nil {
		return nil, err
	}

	ss, err := svc.GetSpreadSheet(spreadsheetID)
	if err != nil {
		return nil, err
	}

	sheet, err := ss.SheetById(sheetID)
	if err != nil {
		return nil, err
	}

	err = WriteObjectsTo(sheet, r)
	if err != nil {
		return nil, err
	}

	return url.Parse(spreadsheetUrl)
}

func WriteToNewSheet(ctx context.Context, r io.Reader) (*url.URL, error) {
	svc, err := sheets.NewSheetService(ctx)
	if err != nil {
		return nil, err
	}

	ss, err := svc.CreateSpreadSheet("json2sheet")
	if err != nil {
		return nil, err
	}

	sheet, err := ss.FirstSheet()
	if err != nil {
		return nil, err
	}

	br := bufio.NewReader(r)

	streamType := streamTypeUnknown
	peek, err := br.Peek(2)
	if err == nil {
		streamType = guessJsonStreamType(peek)
	}

	if streamType == streamTypeArrays {
		// using append makes chunking easier and auto-extends the range
		err = AppendArraysTo(sheet, br)
		if err != nil {
			return nil, err
		}
	} else {
		err = AppendObjectsTo(sheet, br)
		if err != nil {
			return nil, err
		}
	}

	info, err := ss.Get()
	if err != nil {
		return nil, err
	}

	raw := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/edit#gid=0", info.Id)
	return url.Parse(raw)
}

func guessJsonStreamType(peeked []byte) int {
	if peeked[0] == '[' {
		return streamTypeArrays
	}
	if peeked[0] == '{' {
		return streamTypeObjects
	}
	return streamTypeUnknown
}
