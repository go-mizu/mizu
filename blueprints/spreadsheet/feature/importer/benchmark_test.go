package importer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// =============================================================================
// CSV Import Benchmarks - Various Sizes
// =============================================================================

func BenchmarkImportCSV_Small(b *testing.B) {
	// 100 rows x 10 cols = 1,000 cells
	benchmarkImportCSV(b, 100, 10)
}

func BenchmarkImportCSV_Medium(b *testing.B) {
	// 1,000 rows x 50 cols = 50,000 cells
	benchmarkImportCSV(b, 1000, 50)
}

func BenchmarkImportCSV_Large(b *testing.B) {
	// 10,000 rows x 20 cols = 200,000 cells
	benchmarkImportCSV(b, 10000, 20)
}

func BenchmarkImportCSV_XLarge(b *testing.B) {
	// 50,000 rows x 10 cols = 500,000 cells
	benchmarkImportCSV(b, 50000, 10)
}

func benchmarkImportCSV(b *testing.B, rows, cols int) {
	ctx := context.Background()

	mockWB := &benchWorkbooksAPI{}
	mockSh := &benchSheetsAPI{}
	mockCe := &benchCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	// Generate CSV data
	csvData := generateCSVData(rows, cols)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(csvData)
		result, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.csv", FormatCSV, nil)
		if err != nil {
			b.Fatalf("ImportToWorkbook failed: %v", err)
		}

		if result.CellsImported != rows*cols {
			b.Fatalf("Expected %d cells, got %d", rows*cols, result.CellsImported)
		}

		// Reset mock state
		mockSh.createdSheet = nil
		mockCe.batchUpdates = nil
	}

	totalCells := rows * cols
	b.ReportMetric(float64(totalCells), "cells/op")
	b.ReportMetric(float64(totalCells)/b.Elapsed().Seconds()*float64(b.N), "cells/sec")
}

// =============================================================================
// CSV Import with Options Benchmarks
// =============================================================================

func BenchmarkImportCSV_WithTypeDetection(b *testing.B) {
	// 1,000 rows x 10 cols with type detection
	ctx := context.Background()

	mockWB := &benchWorkbooksAPI{}
	mockSh := &benchSheetsAPI{}
	mockCe := &benchCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	// Generate CSV with mixed types
	csvData := generateMixedTypeCSV(1000, 10)

	opts := &Options{
		AutoDetectTypes: true,
		TrimWhitespace:  true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(csvData)
		_, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.csv", FormatCSV, opts)
		if err != nil {
			b.Fatalf("ImportToWorkbook failed: %v", err)
		}

		mockSh.createdSheet = nil
		mockCe.batchUpdates = nil
	}
}

func BenchmarkImportCSV_WithSkipEmpty(b *testing.B) {
	// 1,000 rows x 10 cols with 30% empty rows
	ctx := context.Background()

	mockWB := &benchWorkbooksAPI{}
	mockSh := &benchSheetsAPI{}
	mockCe := &benchCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	// Generate CSV with empty rows
	csvData := generateCSVWithEmptyRows(1000, 10, 30) // 30% empty

	opts := &Options{
		SkipEmptyRows:  true,
		TrimWhitespace: true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(csvData)
		_, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.csv", FormatCSV, opts)
		if err != nil {
			b.Fatalf("ImportToWorkbook failed: %v", err)
		}

		mockSh.createdSheet = nil
		mockCe.batchUpdates = nil
	}
}

// =============================================================================
// JSON Import Benchmarks
// =============================================================================

func BenchmarkImportJSON_Small(b *testing.B) {
	benchmarkImportJSON(b, 100, 10, 0) // 1000 cells, no merges
}

func BenchmarkImportJSON_Medium(b *testing.B) {
	benchmarkImportJSON(b, 1000, 20, 50) // 20000 cells, 50 merges
}

func BenchmarkImportJSON_Large(b *testing.B) {
	benchmarkImportJSON(b, 5000, 20, 100) // 100000 cells, 100 merges
}

func benchmarkImportJSON(b *testing.B, rows, cols, merges int) {
	ctx := context.Background()

	mockWB := &benchWorkbooksAPI{}
	mockSh := &benchSheetsAPI{}
	mockCe := &benchCellsAPI{}

	svc := NewService(mockWB, mockSh, mockCe)

	// Generate JSON data
	jsonData := generateJSONData(rows, cols, merges)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(jsonData)
		result, err := svc.ImportToWorkbook(ctx, "wb1", reader, "test.json", FormatJSON, nil)
		if err != nil {
			b.Fatalf("ImportToWorkbook failed: %v", err)
		}

		if result.CellsImported != rows*cols {
			b.Fatalf("Expected %d cells, got %d", rows*cols, result.CellsImported)
		}

		mockSh.createdSheet = nil
		mockCe.batchUpdates = nil
		mockCe.mergedRegions = nil
	}

	totalCells := rows * cols
	b.ReportMetric(float64(totalCells), "cells/op")
	b.ReportMetric(float64(merges), "merges/op")
}

// =============================================================================
// Batch Processing Benchmarks (measure batch size impact)
// =============================================================================

func BenchmarkBatchProcessing_BatchSize100(b *testing.B) {
	benchmarkBatchProcessing(b, 10000, 100)
}

func BenchmarkBatchProcessing_BatchSize500(b *testing.B) {
	benchmarkBatchProcessing(b, 10000, 500)
}

func BenchmarkBatchProcessing_BatchSize1000(b *testing.B) {
	benchmarkBatchProcessing(b, 10000, 1000)
}

func BenchmarkBatchProcessing_BatchSize2000(b *testing.B) {
	benchmarkBatchProcessing(b, 10000, 2000)
}

func benchmarkBatchProcessing(b *testing.B, totalCells, batchSize int) {
	ctx := context.Background()

	mockCe := &benchCellsAPI{}

	// Generate cells
	cellsToImport := make([]*cells.Cell, totalCells)
	for i := 0; i < totalCells; i++ {
		cellsToImport[i] = &cells.Cell{
			SheetID: "sheet1",
			Row:     i / 100,
			Col:     i % 100,
			Value:   fmt.Sprintf("value-%d", i),
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate batch processing with different batch sizes
		for j := 0; j < len(cellsToImport); j += batchSize {
			end := j + batchSize
			if end > len(cellsToImport) {
				end = len(cellsToImport)
			}

			batch := cellsToImport[j:end]
			updates := make([]cells.CellUpdate, len(batch))
			for k, cell := range batch {
				updates[k] = cells.CellUpdate{
					Row:   cell.Row,
					Col:   cell.Col,
					Value: cell.Value,
				}
			}

			_, err := mockCe.BatchUpdate(ctx, "sheet1", &cells.BatchUpdateIn{Cells: updates})
			if err != nil {
				b.Fatalf("BatchUpdate failed: %v", err)
			}
		}

		mockCe.batchUpdates = nil
	}

	b.ReportMetric(float64(totalCells)/float64(batchSize), "batches/op")
}

// =============================================================================
// Merge Region Benchmarks
// =============================================================================

func BenchmarkMergeRegions_10(b *testing.B) {
	benchmarkMergeRegions(b, 10)
}

func BenchmarkMergeRegions_50(b *testing.B) {
	benchmarkMergeRegions(b, 50)
}

func BenchmarkMergeRegions_100(b *testing.B) {
	benchmarkMergeRegions(b, 100)
}

func BenchmarkMergeRegions_500(b *testing.B) {
	benchmarkMergeRegions(b, 500)
}

func benchmarkMergeRegions(b *testing.B, count int) {
	ctx := context.Background()

	mockCe := &benchCellsAPI{}

	// Generate merge regions
	mergeRegions := make([]MergedRegionImport, count)
	for i := 0; i < count; i++ {
		mergeRegions[i] = MergedRegionImport{
			StartRow: i * 3,
			StartCol: 0,
			EndRow:   i*3 + 1,
			EndCol:   1,
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, mr := range mergeRegions {
			_, err := mockCe.Merge(ctx, "sheet1", mr.StartRow, mr.StartCol, mr.EndRow, mr.EndCol)
			if err != nil {
				b.Fatalf("Merge failed: %v", err)
			}
		}

		mockCe.mergedRegions = nil
	}

	b.ReportMetric(float64(count), "merges/op")
}

// =============================================================================
// Memory Allocation Benchmarks
// =============================================================================

func BenchmarkMemory_ParseCSV_1000(b *testing.B) {
	svc := NewService(nil, nil, nil)

	csvData := generateCSVData(1000, 10)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(csvData)
		_, err := svc.parseCSV(reader, ',', &Options{})
		if err != nil {
			b.Fatalf("parseCSV failed: %v", err)
		}
	}
}

func BenchmarkMemory_ParseJSON_1000(b *testing.B) {
	svc := NewService(nil, nil, nil)

	jsonData := generateJSONData(1000, 10, 10)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(jsonData)
		_, err := svc.parseJSON(reader, &Options{})
		if err != nil {
			b.Fatalf("parseJSON failed: %v", err)
		}
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func generateCSVData(rows, cols int) string {
	var b strings.Builder
	b.Grow(rows * cols * 10) // Preallocate

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			if col > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(&b, "cell-%d-%d", row, col)
		}
		b.WriteByte('\n')
	}

	return b.String()
}

func generateMixedTypeCSV(rows, cols int) string {
	var b strings.Builder
	b.Grow(rows * cols * 15)

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			if col > 0 {
				b.WriteByte(',')
			}
			switch col % 5 {
			case 0:
				fmt.Fprintf(&b, "text-%d", row)
			case 1:
				fmt.Fprintf(&b, "%d", row*100+col)
			case 2:
				fmt.Fprintf(&b, "%.2f", float64(row)+float64(col)/100)
			case 3:
				if row%2 == 0 {
					b.WriteString("true")
				} else {
					b.WriteString("false")
				}
			case 4:
				fmt.Fprintf(&b, "2025-01-%02d", (row%28)+1)
			}
		}
		b.WriteByte('\n')
	}

	return b.String()
}

func generateCSVWithEmptyRows(rows, cols, emptyPercent int) string {
	var b strings.Builder
	b.Grow(rows * cols * 10)

	for row := 0; row < rows; row++ {
		isEmpty := (row*100/rows < emptyPercent)
		for col := 0; col < cols; col++ {
			if col > 0 {
				b.WriteByte(',')
			}
			if !isEmpty {
				fmt.Fprintf(&b, "cell-%d-%d", row, col)
			}
		}
		b.WriteByte('\n')
	}

	return b.String()
}

func generateJSONData(rows, cols, merges int) []byte {
	cellsData := make([]map[string]interface{}, 0, rows*cols)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cellsData = append(cellsData, map[string]interface{}{
				"row":   row,
				"col":   col,
				"value": fmt.Sprintf("cell-%d-%d", row, col),
			})
		}
	}

	mergesData := make([]map[string]interface{}, merges)
	for i := 0; i < merges; i++ {
		mergesData[i] = map[string]interface{}{
			"startRow": i * 3,
			"startCol": 0,
			"endRow":   i*3 + 1,
			"endCol":   1,
		}
	}

	jsonData := map[string]interface{}{
		"version": "1.0",
		"sheets": []map[string]interface{}{
			{
				"name":          "Sheet1",
				"cells":         cellsData,
				"mergedRegions": mergesData,
			},
		},
	}

	data, _ := json.Marshal(jsonData)
	return data
}

// =============================================================================
// Mock Implementations for Benchmarks
// =============================================================================

type benchWorkbooksAPI struct{}

func (m *benchWorkbooksAPI) Create(ctx context.Context, in *workbooks.CreateIn) (*workbooks.Workbook, error) {
	return &workbooks.Workbook{ID: "wb1"}, nil
}

func (m *benchWorkbooksAPI) GetByID(ctx context.Context, id string) (*workbooks.Workbook, error) {
	return &workbooks.Workbook{ID: id}, nil
}

func (m *benchWorkbooksAPI) Update(ctx context.Context, id string, in *workbooks.UpdateIn) (*workbooks.Workbook, error) {
	return &workbooks.Workbook{ID: id}, nil
}

func (m *benchWorkbooksAPI) Delete(ctx context.Context, id string) error { return nil }

func (m *benchWorkbooksAPI) List(ctx context.Context, ownerID string) ([]*workbooks.Workbook, error) {
	return []*workbooks.Workbook{}, nil
}

func (m *benchWorkbooksAPI) Copy(ctx context.Context, id string, newName string, userID string) (*workbooks.Workbook, error) {
	return &workbooks.Workbook{ID: "wb-copy"}, nil
}

type benchSheetsAPI struct {
	createdSheet *sheets.Sheet
}

func (m *benchSheetsAPI) Create(ctx context.Context, in *sheets.CreateIn) (*sheets.Sheet, error) {
	m.createdSheet = &sheets.Sheet{ID: "sheet1", WorkbookID: in.WorkbookID, Name: in.Name}
	return m.createdSheet, nil
}

func (m *benchSheetsAPI) GetByID(ctx context.Context, id string) (*sheets.Sheet, error) {
	return &sheets.Sheet{ID: id}, nil
}

func (m *benchSheetsAPI) Update(ctx context.Context, id string, in *sheets.UpdateIn) (*sheets.Sheet, error) {
	return &sheets.Sheet{ID: id}, nil
}

func (m *benchSheetsAPI) Delete(ctx context.Context, id string) error { return nil }

func (m *benchSheetsAPI) List(ctx context.Context, workbookID string) ([]*sheets.Sheet, error) {
	return []*sheets.Sheet{}, nil
}

func (m *benchSheetsAPI) Copy(ctx context.Context, id string, newName string) (*sheets.Sheet, error) {
	return &sheets.Sheet{ID: "sheet-copy"}, nil
}

func (m *benchSheetsAPI) SetRowHeight(ctx context.Context, sheetID string, row int, height int) error {
	return nil
}

func (m *benchSheetsAPI) SetColWidth(ctx context.Context, sheetID string, col int, width int) error {
	return nil
}

func (m *benchSheetsAPI) HideRow(ctx context.Context, sheetID string, row int) error { return nil }
func (m *benchSheetsAPI) HideCol(ctx context.Context, sheetID string, col int) error { return nil }
func (m *benchSheetsAPI) ShowRow(ctx context.Context, sheetID string, row int) error { return nil }
func (m *benchSheetsAPI) ShowCol(ctx context.Context, sheetID string, col int) error { return nil }

type benchCellsAPI struct {
	batchUpdates  []cells.CellUpdate
	mergedRegions []*cells.MergedRegion
}

func (m *benchCellsAPI) Get(ctx context.Context, sheetID string, row, col int) (*cells.Cell, error) {
	return nil, nil
}

func (m *benchCellsAPI) GetRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([]*cells.Cell, error) {
	return nil, nil
}

func (m *benchCellsAPI) Set(ctx context.Context, sheetID string, row, col int, in *cells.SetCellIn) (*cells.Cell, error) {
	return &cells.Cell{}, nil
}

func (m *benchCellsAPI) BatchUpdate(ctx context.Context, sheetID string, in *cells.BatchUpdateIn) ([]*cells.Cell, error) {
	m.batchUpdates = append(m.batchUpdates, in.Cells...)
	result := make([]*cells.Cell, len(in.Cells))
	for i, u := range in.Cells {
		result[i] = &cells.Cell{SheetID: sheetID, Row: u.Row, Col: u.Col, Value: u.Value}
	}
	return result, nil
}

func (m *benchCellsAPI) Delete(ctx context.Context, sheetID string, row, col int) error { return nil }

func (m *benchCellsAPI) SetFormat(ctx context.Context, in *cells.SetFormatIn) error { return nil }

func (m *benchCellsAPI) SetRangeFormat(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int, format cells.Format) error {
	return nil
}

func (m *benchCellsAPI) Clear(ctx context.Context, sheetID string, row, col int) error { return nil }

func (m *benchCellsAPI) ClearRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error {
	return nil
}

func (m *benchCellsAPI) SetNote(ctx context.Context, sheetID string, row, col int, note string) error {
	return nil
}

func (m *benchCellsAPI) SetHyperlink(ctx context.Context, sheetID string, row, col int, hyperlink *cells.Hyperlink) error {
	return nil
}

func (m *benchCellsAPI) Merge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) (*cells.MergedRegion, error) {
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

func (m *benchCellsAPI) BatchMerge(ctx context.Context, sheetID string, regions []cells.MergedRegion) ([]*cells.MergedRegion, error) {
	result := make([]*cells.MergedRegion, len(regions))
	for i, r := range regions {
		result[i] = &cells.MergedRegion{
			SheetID:  sheetID,
			StartRow: r.StartRow,
			StartCol: r.StartCol,
			EndRow:   r.EndRow,
			EndCol:   r.EndCol,
		}
		m.mergedRegions = append(m.mergedRegions, result[i])
	}
	return result, nil
}

func (m *benchCellsAPI) Unmerge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error {
	return nil
}

func (m *benchCellsAPI) GetMergedRegions(ctx context.Context, sheetID string) ([]*cells.MergedRegion, error) {
	return m.mergedRegions, nil
}

func (m *benchCellsAPI) CopyRange(ctx context.Context, sourceSheetID string, sourceRange cells.Range, destSheetID string, destRow, destCol int) error {
	return nil
}

func (m *benchCellsAPI) InsertRows(ctx context.Context, sheetID string, rowIndex, count int) error {
	return nil
}

func (m *benchCellsAPI) DeleteRows(ctx context.Context, sheetID string, startRow, count int) error {
	return nil
}

func (m *benchCellsAPI) InsertCols(ctx context.Context, sheetID string, colIndex, count int) error {
	return nil
}

func (m *benchCellsAPI) DeleteCols(ctx context.Context, sheetID string, startCol, count int) error {
	return nil
}

func (m *benchCellsAPI) EvaluateFormula(ctx context.Context, sheetID, formula string) (interface{}, error) {
	return nil, nil
}
