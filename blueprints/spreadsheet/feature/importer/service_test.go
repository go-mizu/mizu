package importer

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// Mock implementations for testing
type mockWorkbooksAPI struct {
	workbook *workbooks.Workbook
}

func (m *mockWorkbooksAPI) Create(ctx context.Context, in *workbooks.CreateIn) (*workbooks.Workbook, error) {
	return m.workbook, nil
}

func (m *mockWorkbooksAPI) GetByID(ctx context.Context, id string) (*workbooks.Workbook, error) {
	return m.workbook, nil
}

func (m *mockWorkbooksAPI) Update(ctx context.Context, id string, in *workbooks.UpdateIn) (*workbooks.Workbook, error) {
	return m.workbook, nil
}

func (m *mockWorkbooksAPI) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockWorkbooksAPI) List(ctx context.Context, ownerID string) ([]*workbooks.Workbook, error) {
	return []*workbooks.Workbook{m.workbook}, nil
}

func (m *mockWorkbooksAPI) Copy(ctx context.Context, id string, newName string, userID string) (*workbooks.Workbook, error) {
	return m.workbook, nil
}

type mockSheetsAPI struct {
	sheets       []*sheets.Sheet
	createdSheet *sheets.Sheet
}

func (m *mockSheetsAPI) Create(ctx context.Context, in *sheets.CreateIn) (*sheets.Sheet, error) {
	sheet := &sheets.Sheet{
		ID:         "new-sheet-id",
		WorkbookID: in.WorkbookID,
		Name:       in.Name,
		Index:      in.Index,
	}
	m.createdSheet = sheet
	return sheet, nil
}

func (m *mockSheetsAPI) GetByID(ctx context.Context, id string) (*sheets.Sheet, error) {
	if len(m.sheets) > 0 {
		return m.sheets[0], nil
	}
	return nil, nil
}

func (m *mockSheetsAPI) Update(ctx context.Context, id string, in *sheets.UpdateIn) (*sheets.Sheet, error) {
	if len(m.sheets) > 0 {
		return m.sheets[0], nil
	}
	return nil, nil
}

func (m *mockSheetsAPI) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockSheetsAPI) List(ctx context.Context, workbookID string) ([]*sheets.Sheet, error) {
	return m.sheets, nil
}

func (m *mockSheetsAPI) Copy(ctx context.Context, id string, newName string) (*sheets.Sheet, error) {
	if len(m.sheets) > 0 {
		return m.sheets[0], nil
	}
	return nil, nil
}

func (m *mockSheetsAPI) SetRowHeight(ctx context.Context, sheetID string, row int, height int) error {
	return nil
}

func (m *mockSheetsAPI) SetColWidth(ctx context.Context, sheetID string, col int, width int) error {
	return nil
}

func (m *mockSheetsAPI) HideRow(ctx context.Context, sheetID string, row int) error {
	return nil
}

func (m *mockSheetsAPI) HideCol(ctx context.Context, sheetID string, col int) error {
	return nil
}

func (m *mockSheetsAPI) ShowRow(ctx context.Context, sheetID string, row int) error {
	return nil
}

func (m *mockSheetsAPI) ShowCol(ctx context.Context, sheetID string, col int) error {
	return nil
}

type mockCellsAPI struct {
	cells         []*cells.Cell
	batchUpdates  []cells.CellUpdate
	mergedRegions []*cells.MergedRegion
}

func (m *mockCellsAPI) GetRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([]*cells.Cell, error) {
	return m.cells, nil
}

func (m *mockCellsAPI) GetCell(ctx context.Context, sheetID string, row, col int) (*cells.Cell, error) {
	for _, c := range m.cells {
		if c.Row == row && c.Col == col {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockCellsAPI) SetCell(ctx context.Context, sheetID string, row, col int, in *cells.SetCellIn) (*cells.Cell, error) {
	return &cells.Cell{SheetID: sheetID, Row: row, Col: col}, nil
}

func (m *mockCellsAPI) DeleteCell(ctx context.Context, sheetID string, row, col int) error {
	return nil
}

func (m *mockCellsAPI) BatchUpdate(ctx context.Context, sheetID string, in *cells.BatchUpdateIn) ([]*cells.Cell, error) {
	m.batchUpdates = append(m.batchUpdates, in.Cells...)
	var result []*cells.Cell
	for _, update := range in.Cells {
		result = append(result, &cells.Cell{
			SheetID: sheetID,
			Row:     update.Row,
			Col:     update.Col,
			Value:   update.Value,
			Formula: update.Formula,
		})
	}
	return result, nil
}

func (m *mockCellsAPI) InsertRows(ctx context.Context, sheetID string, rowIndex, count int) error {
	return nil
}

func (m *mockCellsAPI) DeleteRows(ctx context.Context, sheetID string, startRow, count int) error {
	return nil
}

func (m *mockCellsAPI) InsertCols(ctx context.Context, sheetID string, colIndex, count int) error {
	return nil
}

func (m *mockCellsAPI) DeleteCols(ctx context.Context, sheetID string, startCol, count int) error {
	return nil
}

func (m *mockCellsAPI) GetMergedRegions(ctx context.Context, sheetID string) ([]*cells.MergedRegion, error) {
	return m.mergedRegions, nil
}

func (m *mockCellsAPI) Merge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) (*cells.MergedRegion, error) {
	region := &cells.MergedRegion{
		SheetID:  sheetID,
		StartRow: startRow,
		StartCol: startCol,
		EndRow:   endRow,
		EndCol:   endCol,
	}
	m.mergedRegions = append(m.mergedRegions, region)
	return region, nil
}

func (m *mockCellsAPI) Unmerge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error {
	return nil
}

func (m *mockCellsAPI) Get(ctx context.Context, sheetID string, row, col int) (*cells.Cell, error) {
	for _, c := range m.cells {
		if c.Row == row && c.Col == col {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockCellsAPI) Set(ctx context.Context, sheetID string, row, col int, in *cells.SetCellIn) (*cells.Cell, error) {
	return &cells.Cell{SheetID: sheetID, Row: row, Col: col}, nil
}

func (m *mockCellsAPI) Delete(ctx context.Context, sheetID string, row, col int) error {
	return nil
}

func (m *mockCellsAPI) SetFormat(ctx context.Context, in *cells.SetFormatIn) error {
	return nil
}

func (m *mockCellsAPI) SetRangeFormat(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int, format cells.Format) error {
	return nil
}

func (m *mockCellsAPI) Clear(ctx context.Context, sheetID string, row, col int) error {
	return nil
}

func (m *mockCellsAPI) ClearRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error {
	return nil
}

func (m *mockCellsAPI) SetNote(ctx context.Context, sheetID string, row, col int, note string) error {
	return nil
}

func (m *mockCellsAPI) SetHyperlink(ctx context.Context, sheetID string, row, col int, hyperlink *cells.Hyperlink) error {
	return nil
}

func (m *mockCellsAPI) CopyRange(ctx context.Context, sourceSheetID string, sourceRange cells.Range, destSheetID string, destRow, destCol int) error {
	return nil
}

func (m *mockCellsAPI) EvaluateFormula(ctx context.Context, sheetID, formula string) (interface{}, error) {
	return nil, nil
}

func TestImportCSVToWorkbook(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{
		sheets: []*sheets.Sheet{},
	}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	csvData := "Name,Value\nItem1,100\nItem2,200"
	reader := strings.NewReader(csvData)

	result, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.csv", FormatCSV, nil)
	if err != nil {
		t.Fatalf("ImportToWorkbook failed: %v", err)
	}

	if result.RowsImported != 3 {
		t.Errorf("Expected 3 rows imported, got %d", result.RowsImported)
	}

	if result.ColsImported != 2 {
		t.Errorf("Expected 2 cols imported, got %d", result.ColsImported)
	}

	if result.CellsImported != 6 {
		t.Errorf("Expected 6 cells imported, got %d", result.CellsImported)
	}

	// Verify batch updates were made
	if len(mockCe.batchUpdates) != 6 {
		t.Errorf("Expected 6 batch updates, got %d", len(mockCe.batchUpdates))
	}
}

func TestImportCSVWithHeaders(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{sheets: []*sheets.Sheet{}}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	csvData := "Name,Value\nItem1,100\nItem2,200"
	reader := strings.NewReader(csvData)

	result, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.csv", FormatCSV, &Options{
		HasHeaders: true,
	})
	if err != nil {
		t.Fatalf("ImportToWorkbook failed: %v", err)
	}

	// With HasHeaders=true, first row is skipped
	if result.CellsImported != 4 {
		t.Errorf("Expected 4 cells imported (headers skipped), got %d", result.CellsImported)
	}
}

func TestImportCSVSkipEmptyRows(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{sheets: []*sheets.Sheet{}}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	csvData := "A,B\n\nC,D\n  ,  \nE,F"
	reader := strings.NewReader(csvData)

	result, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.csv", FormatCSV, &Options{
		SkipEmptyRows:  true,
		TrimWhitespace: true,
	})
	if err != nil {
		t.Fatalf("ImportToWorkbook failed: %v", err)
	}

	// Should import A,B and C,D and E,F (3 rows * 2 cols = 6 cells)
	if result.CellsImported != 6 {
		t.Errorf("Expected 6 cells imported, got %d", result.CellsImported)
	}
}

func TestImportTSV(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{sheets: []*sheets.Sheet{}}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	tsvData := "A\tB\tC\n1\t2\t3"
	reader := strings.NewReader(tsvData)

	result, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.tsv", FormatTSV, nil)
	if err != nil {
		t.Fatalf("ImportToWorkbook failed: %v", err)
	}

	if result.CellsImported != 6 {
		t.Errorf("Expected 6 cells imported, got %d", result.CellsImported)
	}
}

func TestImportJSON(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{sheets: []*sheets.Sheet{}}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	jsonData := map[string]interface{}{
		"version": "1.0",
		"sheets": []map[string]interface{}{
			{
				"id":   "sh1",
				"name": "Sheet1",
				"cells": []map[string]interface{}{
					{"row": 0, "col": 0, "value": "Hello"},
					{"row": 0, "col": 1, "value": 123},
					{"row": 1, "col": 0, "value": "World"},
				},
			},
		},
	}
	data, _ := json.Marshal(jsonData)
	reader := bytes.NewReader(data)

	result, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.json", FormatJSON, nil)
	if err != nil {
		t.Fatalf("ImportToWorkbook failed: %v", err)
	}

	if result.CellsImported != 3 {
		t.Errorf("Expected 3 cells imported, got %d", result.CellsImported)
	}
}

func TestImportJSONWithFormulas(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{sheets: []*sheets.Sheet{}}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	jsonData := map[string]interface{}{
		"version": "1.0",
		"sheets": []map[string]interface{}{
			{
				"name": "Sheet1",
				"cells": []map[string]interface{}{
					{"row": 0, "col": 0, "value": 10},
					{"row": 0, "col": 1, "value": 20},
					{"row": 0, "col": 2, "formula": "=A1+B1"},
				},
			},
		},
	}
	data, _ := json.Marshal(jsonData)
	reader := bytes.NewReader(data)

	result, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.json", FormatJSON, &Options{
		ImportFormulas: true,
	})
	if err != nil {
		t.Fatalf("ImportToWorkbook failed: %v", err)
	}

	if result.CellsImported != 3 {
		t.Errorf("Expected 3 cells imported, got %d", result.CellsImported)
	}

	// Check that formula was imported
	var foundFormula bool
	for _, update := range mockCe.batchUpdates {
		if update.Formula == "=A1+B1" {
			foundFormula = true
			break
		}
	}
	if !foundFormula {
		t.Error("Formula was not imported")
	}
}

func TestImportJSONWithMergedRegions(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{sheets: []*sheets.Sheet{}}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	jsonData := map[string]interface{}{
		"version": "1.0",
		"sheets": []map[string]interface{}{
			{
				"name": "Sheet1",
				"cells": []map[string]interface{}{
					{"row": 0, "col": 0, "value": "Merged"},
				},
				"mergedRegions": []map[string]interface{}{
					{"startRow": 0, "startCol": 0, "endRow": 1, "endCol": 1},
				},
			},
		},
	}
	data, _ := json.Marshal(jsonData)
	reader := bytes.NewReader(data)

	_, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.json", FormatJSON, nil)
	if err != nil {
		t.Fatalf("ImportToWorkbook failed: %v", err)
	}

	// Check that merged region was created
	if len(mockCe.mergedRegions) != 1 {
		t.Errorf("Expected 1 merged region, got %d", len(mockCe.mergedRegions))
	}
}

func TestImportToExistingSheet(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{
		sheets: []*sheets.Sheet{
			{ID: "sh1", WorkbookID: "wb1", Name: "Sheet1"},
		},
	}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	csvData := "A,B\nC,D"
	reader := strings.NewReader(csvData)

	result, err := svc.ImportToSheet(ctx, "sh1", reader, "test.csv", FormatCSV, nil)
	if err != nil {
		t.Fatalf("ImportToSheet failed: %v", err)
	}

	if result.SheetID != "sh1" {
		t.Errorf("Expected sheetID 'sh1', got '%s'", result.SheetID)
	}

	if result.CellsImported != 4 {
		t.Errorf("Expected 4 cells imported, got %d", result.CellsImported)
	}
}

func TestImportWithStartPosition(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{
		sheets: []*sheets.Sheet{
			{ID: "sh1", WorkbookID: "wb1", Name: "Sheet1"},
		},
	}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	csvData := "A,B"
	reader := strings.NewReader(csvData)

	_, err := svc.ImportToSheet(ctx, "sh1", reader, "test.csv", FormatCSV, &Options{
		StartRow: 5,
		StartCol: 3,
	})
	if err != nil {
		t.Fatalf("ImportToSheet failed: %v", err)
	}

	// Check that cells were placed at correct positions
	for _, update := range mockCe.batchUpdates {
		if update.Row < 5 {
			t.Errorf("Cell row %d should be >= 5", update.Row)
		}
		if update.Col < 3 {
			t.Errorf("Cell col %d should be >= 3", update.Col)
		}
	}
}

func TestDetectFormat(t *testing.T) {
	svc := NewService(nil, nil, nil)

	tests := []struct {
		filename string
		want     Format
	}{
		{"test.csv", FormatCSV},
		{"test.CSV", FormatCSV},
		{"test.tsv", FormatTSV},
		{"test.tab", FormatTSV},
		{"test.xlsx", FormatXLSX},
		{"test.xlsm", FormatXLSX},
		{"test.json", FormatJSON},
		{"test.txt", FormatCSV}, // Unknown defaults to CSV
		{"test", FormatCSV},    // No extension defaults to CSV
	}

	for _, tt := range tests {
		got := svc.DetectFormat(tt.filename)
		if got != tt.want {
			t.Errorf("DetectFormat(%s) = %s, want %s", tt.filename, got, tt.want)
		}
	}
}

func TestValidateFile(t *testing.T) {
	ctx := context.Background()
	svc := NewService(nil, nil, nil)

	tests := []struct {
		name    string
		content []byte
		format  Format
		wantErr bool
	}{
		{
			name:    "valid CSV",
			content: []byte("a,b,c\n1,2,3"),
			format:  FormatCSV,
			wantErr: false,
		},
		{
			name:    "valid TSV",
			content: []byte("a\tb\tc\n1\t2\t3"),
			format:  FormatTSV,
			wantErr: false,
		},
		{
			name:    "valid JSON object",
			content: []byte(`{"version":"1.0"}`),
			format:  FormatJSON,
			wantErr: false,
		},
		{
			name:    "valid JSON array",
			content: []byte(`[{"row":0}]`),
			format:  FormatJSON,
			wantErr: false,
		},
		{
			name:    "empty file",
			content: []byte{},
			format:  FormatCSV,
			wantErr: true,
		},
		{
			name:    "binary for CSV",
			content: []byte{0x00, 0x01, 0x02},
			format:  FormatCSV,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			content: []byte("not json"),
			format:  FormatJSON,
			wantErr: true,
		},
		{
			name:    "invalid XLSX",
			content: []byte("not a zip file"),
			format:  FormatXLSX,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.content)
			err := svc.ValidateFile(ctx, reader, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDetectType(t *testing.T) {
	svc := NewService(nil, nil, nil)

	tests := []struct {
		input    string
		wantType string
	}{
		{"true", "bool"},
		{"false", "bool"},
		{"yes", "bool"},
		{"no", "bool"},
		{"TRUE", "bool"},
		{"123", "int"},
		{"-456", "int"},
		{"12.34", "float"},
		{"-56.78", "float"},
		{"hello", "string"},
		{"2023-01-15", "time"},
		{"01/15/2023", "time"},
	}

	for _, tt := range tests {
		result := svc.detectType(tt.input, "")
		var gotType string
		switch result.(type) {
		case bool:
			gotType = "bool"
		case int64:
			gotType = "int"
		case float64:
			gotType = "float"
		case string:
			gotType = "string"
		default:
			gotType = "time"
		}
		if gotType != tt.wantType {
			t.Errorf("detectType(%s) = %s, want %s", tt.input, gotType, tt.wantType)
		}
	}
}

func TestDetectTypeWithCustomDateFormat(t *testing.T) {
	svc := NewService(nil, nil, nil)

	// Custom date format: DD-MM-YYYY
	result := svc.detectType("15-01-2023", "02-01-2006")
	_, isTime := result.(interface{ Year() int })
	if !isTime {
		t.Errorf("detectType with custom date format should return time.Time")
	}
}

func TestImportAutoDetectTypes(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{sheets: []*sheets.Sheet{}}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	csvData := "Name,Count,Active\nItem,100,true"
	reader := strings.NewReader(csvData)

	_, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.csv", FormatCSV, &Options{
		AutoDetectTypes: true,
		HasHeaders:      true,
	})
	if err != nil {
		t.Fatalf("ImportToWorkbook failed: %v", err)
	}

	// Find the cells and check types
	var foundInt, foundBool bool
	for _, update := range mockCe.batchUpdates {
		switch v := update.Value.(type) {
		case int64:
			if v == 100 {
				foundInt = true
			}
		case bool:
			if v == true {
				foundBool = true
			}
		}
	}

	if !foundInt {
		t.Error("Integer type was not detected")
	}
	if !foundBool {
		t.Error("Boolean type was not detected")
	}
}

func TestImportSpecialCharacters(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{sheets: []*sheets.Sheet{}}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	// CSV with quotes and commas inside values
	csvData := `"Hello, World","Line1
Line2","Quote ""test"""`
	reader := strings.NewReader(csvData)

	result, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.csv", FormatCSV, nil)
	if err != nil {
		t.Fatalf("ImportToWorkbook failed: %v", err)
	}

	if result.CellsImported != 3 {
		t.Errorf("Expected 3 cells imported, got %d", result.CellsImported)
	}

	// Verify values were imported correctly
	values := make(map[string]bool)
	for _, update := range mockCe.batchUpdates {
		if v, ok := update.Value.(string); ok {
			values[v] = true
		}
	}

	if !values["Hello, World"] {
		t.Error("Comma in value not handled correctly")
	}
	if !values["Line1\nLine2"] {
		t.Error("Newline in value not handled correctly")
	}
	if !values[`Quote "test"`] {
		t.Error("Quotes in value not handled correctly")
	}
}

func TestImportEmptyFile(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{sheets: []*sheets.Sheet{}}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	csvData := ""
	reader := strings.NewReader(csvData)

	_, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.csv", FormatCSV, nil)
	if err == nil {
		t.Error("Expected error for empty file")
	}
}

func TestSupportedFormats(t *testing.T) {
	svc := NewService(nil, nil, nil)

	formats := svc.SupportedFormats()

	expectedFormats := []Format{FormatCSV, FormatTSV, FormatXLSX, FormatJSON}
	if len(formats) != len(expectedFormats) {
		t.Errorf("Expected %d formats, got %d", len(expectedFormats), len(formats))
	}

	formatMap := make(map[Format]bool)
	for _, f := range formats {
		formatMap[f] = true
	}

	for _, expected := range expectedFormats {
		if !formatMap[expected] {
			t.Errorf("Expected format %s not found", expected)
		}
	}
}

func TestImportJSONWithFormatting(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{sheets: []*sheets.Sheet{}}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	jsonData := map[string]interface{}{
		"version": "1.0",
		"sheets": []map[string]interface{}{
			{
				"name": "Sheet1",
				"cells": []map[string]interface{}{
					{
						"row":   0,
						"col":   0,
						"value": "Formatted",
						"format": map[string]interface{}{
							"bold":            true,
							"fontColor":       "#FF0000",
							"backgroundColor": "#FFFF00",
						},
					},
				},
			},
		},
	}
	data, _ := json.Marshal(jsonData)
	reader := bytes.NewReader(data)

	_, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.json", FormatJSON, &Options{
		ImportFormatting: true,
	})
	if err != nil {
		t.Fatalf("ImportToWorkbook failed: %v", err)
	}

	// Check that formatting was imported
	if len(mockCe.batchUpdates) != 1 {
		t.Fatalf("Expected 1 cell update, got %d", len(mockCe.batchUpdates))
	}

	update := mockCe.batchUpdates[0]
	if update.Format == nil {
		t.Error("Format should not be nil")
	} else {
		if !update.Format.Bold {
			t.Error("Bold formatting not imported")
		}
		if update.Format.FontColor != "#FF0000" {
			t.Errorf("FontColor not imported correctly, got %s", update.Format.FontColor)
		}
		if update.Format.BackgroundColor != "#FFFF00" {
			t.Errorf("BackgroundColor not imported correctly, got %s", update.Format.BackgroundColor)
		}
	}
}

func TestImportMultipleSheets(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{sheets: []*sheets.Sheet{}}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	jsonData := map[string]interface{}{
		"version": "1.0",
		"sheets": []map[string]interface{}{
			{
				"name": "Sheet1",
				"cells": []map[string]interface{}{
					{"row": 0, "col": 0, "value": "A1"},
					{"row": 0, "col": 1, "value": "B1"},
				},
			},
			{
				"name": "Sheet2",
				"cells": []map[string]interface{}{
					{"row": 0, "col": 0, "value": "C1"},
					{"row": 0, "col": 1, "value": "D1"},
				},
			},
		},
	}
	data, _ := json.Marshal(jsonData)
	reader := bytes.NewReader(data)

	result, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.json", FormatJSON, nil)
	if err != nil {
		t.Fatalf("ImportToWorkbook failed: %v", err)
	}

	// Should aggregate cells from both sheets
	if result.CellsImported != 4 {
		t.Errorf("Expected 4 cells imported from 2 sheets, got %d", result.CellsImported)
	}
}

func TestDetectCellType(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected cells.CellType
	}{
		{true, cells.CellTypeBool},
		{false, cells.CellTypeBool},
		{42, cells.CellTypeNumber},
		{int64(42), cells.CellTypeNumber},
		{3.14, cells.CellTypeNumber},
		{"hello", cells.CellTypeText},
	}

	for _, tt := range tests {
		result := detectCellType(tt.value)
		if result != tt.expected {
			t.Errorf("detectCellType(%v) = %v, want %v", tt.value, result, tt.expected)
		}
	}
}

func TestImportWithSheetName(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{sheets: []*sheets.Sheet{}}
	mockCe := &mockCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	csvData := "A,B"
	reader := strings.NewReader(csvData)

	_, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.csv", FormatCSV, &Options{
		SheetName: "Custom Sheet Name",
	})
	if err != nil {
		t.Fatalf("ImportToWorkbook failed: %v", err)
	}

	if mockSh.createdSheet == nil {
		t.Fatal("No sheet was created")
	}

	if mockSh.createdSheet.Name != "Custom Sheet Name" {
		t.Errorf("Expected sheet name 'Custom Sheet Name', got '%s'", mockSh.createdSheet.Name)
	}
}
