package sheets

import (
	"fmt"

	"golang.org/x/exp/constraints"

	googlesheets "google.golang.org/api/sheets/v4"
)

type SheetOps interface {
	UpdateValues(data [][]string) error
	AppendValues(data [][]string) error
	Values() ([][]any, error)
	Get() (*Sheet, error)
}

type sheetOps struct {
	*spreadsheetOps
	sheetId int64
}

func (s *sheetOps) Get() (*Sheet, error) {
	sheet, err := s.filteredSheets(func(p *googlesheets.SheetProperties) bool {
		return p.SheetId == s.sheetId
	})
	if err != nil {
		return nil, err
	}
	return toSheet(sheet), nil
}

func (s *sheetOps) UpdateValues(data [][]string) error {
	values := toValues(data)

	if err := s.grow(values); err != nil {
		return err
	}

	filterRange := &googlesheets.DataFilterValueRange{
		DataFilter: &googlesheets.DataFilter{
			GridRange: &googlesheets.GridRange{
				EndColumnIndex:   int64(len(values[0])),
				EndRowIndex:      int64(len(values)),
				SheetId:          s.sheetId,
				StartColumnIndex: 0,
				StartRowIndex:    0,
				ForceSendFields:  nil,
				NullFields:       nil,
			},
		},
		MajorDimension:  "ROWS",
		Values:          values,
		ForceSendFields: nil,
		NullFields:      nil,
	}

	req := &googlesheets.BatchUpdateValuesByDataFilterRequest{
		Data:                         []*googlesheets.DataFilterValueRange{filterRange},
		IncludeValuesInResponse:      false,
		ResponseDateTimeRenderOption: "",
		ResponseValueRenderOption:    "",
		ValueInputOption:             "RAW",
		ForceSendFields:              nil,
		NullFields:                   nil,
	}

	_, err := s.service.Spreadsheets.Values.BatchUpdateByDataFilter(s.spreadsheetId(), req).Do()
	if err != nil {
		return fmt.Errorf("unable to update data from sheet: %w", err)
	}

	return nil
}

func (s *sheetOps) grow(values [][]any) error {
	sheet, err := s.filteredSheets(func(p *googlesheets.SheetProperties) bool {
		return p.SheetId == s.sheetId
	})
	if err != nil {
		return err
	}
	curColumns := sheet.GridProperties.ColumnCount
	curRows := sheet.GridProperties.RowCount

	var appendDimensions []*googlesheets.AppendDimensionRequest
	missingColumns := max(len(values[0])-int(curColumns), 0)
	if missingColumns > 0 {
		appendDimensions = append(appendDimensions, &googlesheets.AppendDimensionRequest{
			Dimension: "COLUMNS",
			Length:    int64(missingColumns),
			SheetId:   s.sheetId,
		})
	}
	missingRows := max(len(values)-int(curRows), 0)
	if missingRows > 0 {
		appendDimensions = append(appendDimensions, &googlesheets.AppendDimensionRequest{
			Dimension: "ROWS",
			Length:    int64(missingRows),
			SheetId:   s.sheetId,
		})
	}

	if len(appendDimensions) == 0 {
		return nil
	}

	var requests []*googlesheets.Request
	for _, r := range appendDimensions {
		requests = append(requests, &googlesheets.Request{AppendDimension: r})
	}
	req := &googlesheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}
	_, err = s.service.Spreadsheets.BatchUpdate(s.spreadsheetId(), req).Do()
	if err != nil {
		return fmt.Errorf("failed to expand data range to fit data: %w", err)
	}
	return nil
}

func (s *sheetOps) AppendValues(data [][]string) error {
	sheet, err := s.Get()
	if err != nil {
		return fmt.Errorf("unable to append data, spreadsheet='%s' sheetId='%d': %w", s.spreadsheetId(), s.sheetId, err)
	}

	insertRange := fmt.Sprintf("'%s'!A:A", sheet.Title)
	values := toValues(data)
	valueRange := &googlesheets.ValueRange{
		MajorDimension: "ROWS",
		Values:         values,
	}

	_, err = s.service.Spreadsheets.Values.Append(s.spreadsheetId(), insertRange, valueRange).
		ValueInputOption("RAW").
		InsertDataOption("INSERT_ROWS").
		Do()
	if err != nil {
		return fmt.Errorf("unable to append data, spreadsheet='%s' sheetId='%d': %w", s.spreadsheetId(), s.sheetId, err)
	}

	return nil
}

func (s *sheetOps) Values() ([][]any, error) {
	resp, err := s.service.Spreadsheets.Values.BatchGetByDataFilter(s.spreadsheetId(), &googlesheets.BatchGetValuesByDataFilterRequest{
		DataFilters: []*googlesheets.DataFilter{{GridRange: &googlesheets.GridRange{
			EndColumnIndex:   0,
			EndRowIndex:      0,
			SheetId:          s.sheetId,
			StartColumnIndex: 0,
			StartRowIndex:    0,
			ForceSendFields:  nil,
			NullFields:       nil,
		}}},
	}).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %w", err)
	}

	if len(resp.ValueRanges) == 0 {
		return nil, fmt.Errorf("empty spreadsheet")
	}

	values := resp.ValueRanges[0].ValueRange.Values
	if len(values) == 0 {
		return nil, fmt.Errorf("empty spreadsheet, no values found")
	}
	return values, nil
}

func toValues(data [][]string) [][]interface{} {
	values := make([][]interface{}, len(data))
	for i, row := range data {
		values[i] = make([]interface{}, len(row))
		for j, cell := range row {
			values[i][j] = cell
		}
	}
	return values
}

func max[T constraints.Ordered](a1, a2 T) T {
	if a1 > a2 {
		return a1
	}
	return a2
}
