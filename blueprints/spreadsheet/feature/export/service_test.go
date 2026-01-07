package export

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
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
	sheet *sheets.Sheet
}

func (m *mockSheetsAPI) Create(ctx context.Context, in *sheets.CreateIn) (*sheets.Sheet, error) {
	return m.sheet, nil
}

func (m *mockSheetsAPI) GetByID(ctx context.Context, id string) (*sheets.Sheet, error) {
	return m.sheet, nil
}

func (m *mockSheetsAPI) Update(ctx context.Context, id string, in *sheets.UpdateIn) (*sheets.Sheet, error) {
	return m.sheet, nil
}

func (m *mockSheetsAPI) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockSheetsAPI) List(ctx context.Context, workbookID string) ([]*sheets.Sheet, error) {
	return []*sheets.Sheet{m.sheet}, nil
}

func (m *mockSheetsAPI) Copy(ctx context.Context, id string, newName string) (*sheets.Sheet, error) {
	return m.sheet, nil
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
	cells []*cells.Cell
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
	return m.cells, nil
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
	return nil, nil
}

func (m *mockCellsAPI) Merge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) (*cells.MergedRegion, error) {
	return nil, nil
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

func TestExportCSV(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{
			ID:   "wb1",
			Name: "Test Workbook",
		},
	}

	mockSh := &mockSheetsAPI{
		sheet: &sheets.Sheet{
			ID:         "sh1",
			WorkbookID: "wb1",
			Name:       "Sheet1",
		},
	}

	mockCe := &mockCellsAPI{
		cells: []*cells.Cell{
			{Row: 0, Col: 0, Value: "Name", Display: "Name", Type: cells.CellTypeText},
			{Row: 0, Col: 1, Value: "Value", Display: "Value", Type: cells.CellTypeText},
			{Row: 1, Col: 0, Value: "Item1", Display: "Item1", Type: cells.CellTypeText},
			{Row: 1, Col: 1, Value: 100, Display: "100", Type: cells.CellTypeNumber},
			{Row: 2, Col: 0, Value: "Item2", Display: "Item2", Type: cells.CellTypeText},
			{Row: 2, Col: 1, Value: 200, Display: "200", Type: cells.CellTypeNumber},
		},
	}

	svc := NewService(mockWB, mockSh, mockCe)

	result, err := svc.ExportWorkbook(ctx, "wb1", FormatCSV, nil)
	if err != nil {
		t.Fatalf("ExportWorkbook failed: %v", err)
	}

	if result.ContentType != "text/csv; charset=utf-8" {
		t.Errorf("Expected content type 'text/csv; charset=utf-8', got '%s'", result.ContentType)
	}

	if !strings.HasSuffix(result.Filename, ".csv") {
		t.Errorf("Expected filename to end with .csv, got '%s'", result.Filename)
	}

	// Read CSV content
	data, _ := io.ReadAll(result.Data)
	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(records))
	}

	if records[0][0] != "Name" || records[0][1] != "Value" {
		t.Errorf("Unexpected header row: %v", records[0])
	}

	if records[1][0] != "Item1" || records[1][1] != "100" {
		t.Errorf("Unexpected data row 1: %v", records[1])
	}
}

func TestExportTSV(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{
		sheet: &sheets.Sheet{ID: "sh1", WorkbookID: "wb1", Name: "Sheet1"},
	}
	mockCe := &mockCellsAPI{
		cells: []*cells.Cell{
			{Row: 0, Col: 0, Value: "A", Display: "A"},
			{Row: 0, Col: 1, Value: "B", Display: "B"},
		},
	}

	svc := NewService(mockWB, mockSh, mockCe)

	result, err := svc.ExportWorkbook(ctx, "wb1", FormatTSV, nil)
	if err != nil {
		t.Fatalf("ExportWorkbook failed: %v", err)
	}

	if result.ContentType != "text/tab-separated-values; charset=utf-8" {
		t.Errorf("Expected TSV content type, got '%s'", result.ContentType)
	}

	if !strings.HasSuffix(result.Filename, ".tsv") {
		t.Errorf("Expected filename to end with .tsv, got '%s'", result.Filename)
	}

	data, _ := io.ReadAll(result.Data)
	content := string(data)
	if !strings.Contains(content, "\t") {
		t.Error("TSV content should contain tabs")
	}
}

func TestExportJSON(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{
		sheet: &sheets.Sheet{ID: "sh1", WorkbookID: "wb1", Name: "Sheet1"},
	}
	mockCe := &mockCellsAPI{
		cells: []*cells.Cell{
			{Row: 0, Col: 0, Value: "Test", Display: "Test"},
		},
	}

	svc := NewService(mockWB, mockSh, mockCe)

	result, err := svc.ExportWorkbook(ctx, "wb1", FormatJSON, &Options{IncludeMetadata: true})
	if err != nil {
		t.Fatalf("ExportWorkbook failed: %v", err)
	}

	if result.ContentType != "application/json; charset=utf-8" {
		t.Errorf("Expected JSON content type, got '%s'", result.ContentType)
	}

	// Verify valid JSON
	data, _ := io.ReadAll(result.Data)
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		t.Errorf("Invalid JSON output: %v", err)
	}

	if jsonData["version"] != "1.0" {
		t.Errorf("Expected version 1.0, got %v", jsonData["version"])
	}
}

func TestExportHTML(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{
		sheet: &sheets.Sheet{ID: "sh1", WorkbookID: "wb1", Name: "Sheet1"},
	}
	mockCe := &mockCellsAPI{
		cells: []*cells.Cell{
			{Row: 0, Col: 0, Value: "Hello", Display: "Hello"},
		},
	}

	svc := NewService(mockWB, mockSh, mockCe)

	result, err := svc.ExportWorkbook(ctx, "wb1", FormatHTML, nil)
	if err != nil {
		t.Fatalf("ExportWorkbook failed: %v", err)
	}

	if result.ContentType != "text/html; charset=utf-8" {
		t.Errorf("Expected HTML content type, got '%s'", result.ContentType)
	}

	data, _ := io.ReadAll(result.Data)
	content := string(data)

	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Error("HTML should contain DOCTYPE")
	}

	if !strings.Contains(content, "<table>") {
		t.Error("HTML should contain table")
	}

	if !strings.Contains(content, "Hello") {
		t.Error("HTML should contain cell value")
	}
}

func TestExportWithFormulas(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{
		sheet: &sheets.Sheet{ID: "sh1", WorkbookID: "wb1", Name: "Sheet1"},
	}
	mockCe := &mockCellsAPI{
		cells: []*cells.Cell{
			{Row: 0, Col: 0, Value: 10, Display: "10"},
			{Row: 0, Col: 1, Value: 20, Display: "20"},
			{Row: 0, Col: 2, Formula: "=A1+B1", Display: "30"},
		},
	}

	svc := NewService(mockWB, mockSh, mockCe)

	// Export with formulas
	result, err := svc.ExportWorkbook(ctx, "wb1", FormatCSV, &Options{ExportFormulas: true})
	if err != nil {
		t.Fatalf("ExportWorkbook failed: %v", err)
	}

	data, _ := io.ReadAll(result.Data)
	content := string(data)

	if !strings.Contains(content, "=A1+B1") {
		t.Error("CSV with formulas should contain formula text")
	}
}

func TestExportEmptySheet(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{
		sheet: &sheets.Sheet{ID: "sh1", WorkbookID: "wb1", Name: "Sheet1"},
	}
	mockCe := &mockCellsAPI{
		cells: []*cells.Cell{},
	}

	svc := NewService(mockWB, mockSh, mockCe)

	result, err := svc.ExportWorkbook(ctx, "wb1", FormatCSV, nil)
	if err != nil {
		t.Fatalf("ExportWorkbook failed: %v", err)
	}

	// Should succeed with empty content
	if result == nil {
		t.Error("Result should not be nil for empty sheet")
	}
}

func TestExportSpecialCharacters(t *testing.T) {
	ctx := context.Background()

	mockWB := &mockWorkbooksAPI{
		workbook: &workbooks.Workbook{ID: "wb1", Name: "Test"},
	}
	mockSh := &mockSheetsAPI{
		sheet: &sheets.Sheet{ID: "sh1", WorkbookID: "wb1", Name: "Sheet1"},
	}
	mockCe := &mockCellsAPI{
		cells: []*cells.Cell{
			{Row: 0, Col: 0, Value: "Hello, World", Display: "Hello, World"},
			{Row: 0, Col: 1, Value: "Line1\nLine2", Display: "Line1\nLine2"},
			{Row: 0, Col: 2, Value: `Quote "test"`, Display: `Quote "test"`},
		},
	}

	svc := NewService(mockWB, mockSh, mockCe)

	result, err := svc.ExportWorkbook(ctx, "wb1", FormatCSV, nil)
	if err != nil {
		t.Fatalf("ExportWorkbook failed: %v", err)
	}

	data, _ := io.ReadAll(result.Data)
	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV with special characters: %v", err)
	}

	// Verify that special characters were escaped properly
	if len(records) != 1 || len(records[0]) != 3 {
		t.Errorf("Expected 1 row with 3 columns, got %d rows", len(records))
	}

	if records[0][0] != "Hello, World" {
		t.Errorf("Comma not handled properly: got %s", records[0][0])
	}
}

func TestSupportedFormats(t *testing.T) {
	svc := NewService(nil, nil, nil)

	formats := svc.SupportedFormats()

	expectedFormats := []Format{FormatCSV, FormatTSV, FormatXLSX, FormatJSON, FormatPDF, FormatHTML}
	if len(formats) != len(expectedFormats) {
		t.Errorf("Expected %d formats, got %d", len(expectedFormats), len(formats))
	}
}

func TestGetColumnLabel(t *testing.T) {
	tests := []struct {
		index int
		want  string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{51, "AZ"},
		{52, "BA"},
		{701, "ZZ"},
		{702, "AAA"},
	}

	for _, tt := range tests {
		got := getColumnLabel(tt.index)
		if got != tt.want {
			t.Errorf("getColumnLabel(%d) = %s, want %s", tt.index, got, tt.want)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"test", "test"},
		{"my/file", "my_file"},
		{"file:name", "file_name"},
		{"test*file?", "test_file_"},
		{`"quotes"`, "_quotes_"},
	}

	for _, tt := range tests {
		got := sanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeFilename(%s) = %s, want %s", tt.input, got, tt.want)
		}
	}
}
