package sheet2json

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/trichner/tb/pkg/sheets"
)

func ReadFromSheet(ctx context.Context, spreadsheetId string, sheetId int64, w io.Writer) error {
	svc, err := sheets.NewSheetService(ctx)
	if err != nil {
		return err
	}

	ss, err := svc.GetSpreadSheet(spreadsheetId)
	if err != nil {
		return err
	}

	sheet, err := ss.SheetById(sheetId)
	if err != nil {
		return err
	}

	err = writeSheetToJsonObjects(sheet, newJsonWriter(w))
	if err != nil {
		return err
	}

	return nil
}

type JsonWriter func(n any) error

func newJsonWriter(w io.Writer) JsonWriter {
	e := json.NewEncoder(w)
	return func(n any) error {
		return e.Encode(n)
	}
}

func writeSheetToJsonObjects(sheet sheets.SheetOps, w JsonWriter) error {
	values, err := sheet.Values()
	if err != nil {
		return fmt.Errorf("failed to fetch sheet values: %w", err)
	}

	if len(values) <= 1 {
		// no values or only headers
		return nil
	}

	headers := parseHeaders(values[0])
	values = values[1:]

	for i, row := range values {
		m := map[string]any{}
		for j, cell := range row {
			if cell == "" {
			}
			m[headers[j]] = cell
		}
		if err := w(m); err != nil {
			return fmt.Errorf("failed do write line %d (%+v): %w", i, row, err)
		}
	}
	return nil
}

func parseHeaders(row []any) []string {
	headers := make([]string, len(row))
	for i, v := range row {
		headers[i] = cellToString(v)
	}

	return headers
}

func cellToString(c any) string {
	switch v := c.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case time.Time:
		return v.Format(time.RFC3339)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprintf("%s", v)
	}
}
