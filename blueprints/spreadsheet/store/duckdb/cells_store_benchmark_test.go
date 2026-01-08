package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// =============================================================================
// BatchSet Benchmarks - Various sizes
// =============================================================================

func BenchmarkBatchSet_100(b *testing.B) {
	benchmarkBatchSet(b, 100)
}

func BenchmarkBatchSet_500(b *testing.B) {
	benchmarkBatchSet(b, 500)
}

func BenchmarkBatchSet_1000(b *testing.B) {
	benchmarkBatchSet(b, 1000)
}

func BenchmarkBatchSet_5000(b *testing.B) {
	benchmarkBatchSet(b, 5000)
}

func BenchmarkBatchSet_10000(b *testing.B) {
	benchmarkBatchSet(b, 10000)
}

func benchmarkBatchSet(b *testing.B, count int) {
	f := setupBenchFixture(b)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create new sheet for each iteration to avoid conflicts
		sheet := createBenchSheet(b, f.DB, f.Workbook.ID, i)

		// Generate fresh cells with new IDs for each iteration
		cellList := generateCells(sheet.ID, count)

		if err := store.BatchSet(ctx, cellList); err != nil {
			b.Fatalf("BatchSet() error = %v", err)
		}
	}

	b.ReportMetric(float64(count), "cells/op")
	b.ReportMetric(float64(count)/b.Elapsed().Seconds()*float64(b.N), "cells/sec")
}

// =============================================================================
// GetByPositions Benchmarks - Sparse vs Dense
// =============================================================================

func BenchmarkGetByPositions_Sparse_10(b *testing.B) {
	benchmarkGetByPositionsSparse(b, 10)
}

func BenchmarkGetByPositions_Sparse_50(b *testing.B) {
	benchmarkGetByPositionsSparse(b, 50)
}

func BenchmarkGetByPositions_Sparse_100(b *testing.B) {
	benchmarkGetByPositionsSparse(b, 100)
}

func benchmarkGetByPositionsSparse(b *testing.B, count int) {
	f := setupBenchFixture(b)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create sparse cells (far apart)
	positions := make([]cells.CellPosition, count)
	for i := 0; i < count; i++ {
		row := i * 100
		col := i * 100
		createBenchCell(b, f.DB, f.Sheet.ID, row, col)
		positions[i] = cells.CellPosition{Row: row, Col: col}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := store.GetByPositions(ctx, f.Sheet.ID, positions)
		if err != nil {
			b.Fatalf("GetByPositions() error = %v", err)
		}
	}
}

func BenchmarkGetByPositions_Dense_100(b *testing.B) {
	benchmarkGetByPositionsDense(b, 10, 10) // 10x10 grid = 100 cells
}

func BenchmarkGetByPositions_Dense_400(b *testing.B) {
	benchmarkGetByPositionsDense(b, 20, 20) // 20x20 grid = 400 cells
}

func BenchmarkGetByPositions_Dense_1000(b *testing.B) {
	benchmarkGetByPositionsDense(b, 50, 20) // 50x20 grid = 1000 cells
}

func benchmarkGetByPositionsDense(b *testing.B, rows, cols int) {
	f := setupBenchFixture(b)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create dense grid
	positions := make([]cells.CellPosition, 0, rows*cols)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			createBenchCell(b, f.DB, f.Sheet.ID, row, col)
			positions = append(positions, cells.CellPosition{Row: row, Col: col})
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := store.GetByPositions(ctx, f.Sheet.ID, positions)
		if err != nil {
			b.Fatalf("GetByPositions() error = %v", err)
		}
	}
}

// =============================================================================
// CreateMerge Benchmarks - Individual vs Batch (to compare)
// =============================================================================

func BenchmarkCreateMerge_Individual_10(b *testing.B) {
	benchmarkCreateMergeIndividual(b, 10)
}

func BenchmarkCreateMerge_Individual_50(b *testing.B) {
	benchmarkCreateMergeIndividual(b, 50)
}

func BenchmarkCreateMerge_Individual_100(b *testing.B) {
	benchmarkCreateMergeIndividual(b, 100)
}

func benchmarkCreateMergeIndividual(b *testing.B, count int) {
	f := setupBenchFixture(b)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create new sheet for each iteration
		sheet := createBenchSheet(b, f.DB, f.Workbook.ID, i)

		for j := 0; j < count; j++ {
			region := &cells.MergedRegion{
				ID:       NewTestID(),
				SheetID:  sheet.ID,
				StartRow: j * 3,
				StartCol: 0,
				EndRow:   j*3 + 1,
				EndCol:   1,
			}
			if err := store.CreateMerge(ctx, region); err != nil {
				b.Fatalf("CreateMerge() error = %v", err)
			}
		}
	}

	b.ReportMetric(float64(count), "merges/op")
}

// =============================================================================
// BatchCreateMerge Benchmarks - Batch vs Individual comparison
// =============================================================================

func BenchmarkBatchCreateMerge_10(b *testing.B) {
	benchmarkBatchCreateMerge(b, 10)
}

func BenchmarkBatchCreateMerge_50(b *testing.B) {
	benchmarkBatchCreateMerge(b, 50)
}

func BenchmarkBatchCreateMerge_100(b *testing.B) {
	benchmarkBatchCreateMerge(b, 100)
}

func benchmarkBatchCreateMerge(b *testing.B, count int) {
	f := setupBenchFixture(b)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create new sheet for each iteration
		sheet := createBenchSheet(b, f.DB, f.Workbook.ID, i)

		// Create all regions in a batch
		regions := make([]*cells.MergedRegion, count)
		for j := 0; j < count; j++ {
			regions[j] = &cells.MergedRegion{
				ID:       NewTestID(),
				SheetID:  sheet.ID,
				StartRow: j * 3,
				StartCol: 0,
				EndRow:   j*3 + 1,
				EndCol:   1,
			}
		}

		if err := store.BatchCreateMerge(ctx, regions); err != nil {
			b.Fatalf("BatchCreateMerge() error = %v", err)
		}
	}

	b.ReportMetric(float64(count), "merges/op")
}

// =============================================================================
// GetRange Benchmarks
// =============================================================================

func BenchmarkGetRange_Small(b *testing.B) {
	benchmarkGetRange(b, 10, 10) // 100 cells
}

func BenchmarkGetRange_Medium(b *testing.B) {
	benchmarkGetRange(b, 100, 50) // 5000 cells
}

func BenchmarkGetRange_Large(b *testing.B) {
	benchmarkGetRange(b, 500, 100) // 50000 cells
}

func benchmarkGetRange(b *testing.B, rows, cols int) {
	f := setupBenchFixture(b)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create cells
	cellList := make([]*cells.Cell, 0, rows*cols)
	now := FixedTime()
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cellList = append(cellList, &cells.Cell{
				ID:        NewTestID(),
				SheetID:   f.Sheet.ID,
				Row:       row,
				Col:       col,
				Value:     fmt.Sprintf("cell-%d-%d", row, col),
				Display:   fmt.Sprintf("cell-%d-%d", row, col),
				Type:      cells.CellTypeText,
				UpdatedAt: now,
			})
		}
	}

	if err := store.BatchSet(ctx, cellList); err != nil {
		b.Fatalf("BatchSet() error = %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := store.GetRange(ctx, f.Sheet.ID, 0, 0, rows-1, cols-1)
		if err != nil {
			b.Fatalf("GetRange() error = %v", err)
		}
	}

	b.ReportMetric(float64(rows*cols), "cells/op")
}

// =============================================================================
// JSON Marshaling Benchmark (overhead measurement)
// =============================================================================

func BenchmarkBatchSet_WithFormat(b *testing.B) {
	f := setupBenchFixture(b)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	const count = 1000

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sheet := createBenchSheet(b, f.DB, f.Workbook.ID, i)
		now := FixedTime()
		cellList := make([]*cells.Cell, count)

		for j := 0; j < count; j++ {
			cellList[j] = &cells.Cell{
				ID:      NewTestID(),
				SheetID: sheet.ID,
				Row:     j / 100,
				Col:     j % 100,
				Value:   fmt.Sprintf("value-%d", j),
				Display: fmt.Sprintf("value-%d", j),
				Type:    cells.CellTypeText,
				Format: cells.Format{
					FontFamily:      "Arial",
					FontSize:        12,
					FontColor:       "#000000",
					Bold:            true,
					BackgroundColor: "#FFFFFF",
					HAlign:          "left",
					NumberFormat:    "#,##0.00",
				},
				Hyperlink: &cells.Hyperlink{
					URL:   "https://example.com",
					Label: "Example",
				},
				Note:      "Test note",
				UpdatedAt: now,
			}
		}

		if err := store.BatchSet(ctx, cellList); err != nil {
			b.Fatalf("BatchSet() error = %v", err)
		}
	}
}

func BenchmarkBatchSet_NoFormat(b *testing.B) {
	f := setupBenchFixture(b)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	const count = 1000

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sheet := createBenchSheet(b, f.DB, f.Workbook.ID, i)
		now := FixedTime()
		cellList := make([]*cells.Cell, count)

		for j := 0; j < count; j++ {
			cellList[j] = &cells.Cell{
				ID:        NewTestID(),
				SheetID:   sheet.ID,
				Row:       j / 100,
				Col:       j % 100,
				Value:     fmt.Sprintf("value-%d", j),
				Display:   fmt.Sprintf("value-%d", j),
				Type:      cells.CellTypeText,
				UpdatedAt: now,
			}
		}

		if err := store.BatchSet(ctx, cellList); err != nil {
			b.Fatalf("BatchSet() error = %v", err)
		}
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func setupBenchFixture(b *testing.B) *TestFixture {
	b.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		b.Fatalf("failed to open in-memory duckdb: %v", err)
	}

	store, err := New(db)
	if err != nil {
		db.Close()
		b.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		b.Fatalf("failed to initialize schema: %v", err)
	}

	b.Cleanup(func() {
		db.Close()
	})

	user := createBenchUser(b, db)
	workbook := createBenchWorkbook(b, db, user.ID)
	sheet := createBenchSheet(b, db, workbook.ID, 0)

	return &TestFixture{
		DB:       db,
		User:     user,
		Workbook: workbook,
		Sheet:    sheet,
	}
}

func createBenchUser(b *testing.B, db *sql.DB) *users.User {
	b.Helper()

	userID := NewTestID()
	now := FixedTime()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO users (id, email, name, password, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, "bench-"+userID+"@example.com", "Bench User", "hash", now, now)
	if err != nil {
		b.Fatalf("failed to create bench user: %v", err)
	}

	return &users.User{ID: userID}
}

func createBenchWorkbook(b *testing.B, db *sql.DB, ownerID string) *workbooks.Workbook {
	b.Helper()

	wbID := NewTestID()
	now := FixedTime()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO workbooks (id, name, owner_id, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, wbID, "Bench Workbook", ownerID, "{}", now, now)
	if err != nil {
		b.Fatalf("failed to create bench workbook: %v", err)
	}

	return &workbooks.Workbook{ID: wbID, OwnerID: ownerID}
}

func createBenchSheet(b *testing.B, db *sql.DB, workbookID string, index int) *sheets.Sheet {
	b.Helper()

	sheetID := NewTestID()
	now := FixedTime()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO sheets (id, workbook_id, name, index_num, hidden, grid_color,
			frozen_rows, frozen_cols, default_row_height, default_col_width,
			row_heights, col_widths, hidden_rows, hidden_cols, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sheetID, workbookID, fmt.Sprintf("Sheet%d", index), index, false, "#E2E8F0",
		0, 0, 21, 100, "{}", "{}", "[]", "[]", now, now)
	if err != nil {
		b.Fatalf("failed to create bench sheet: %v", err)
	}

	return &sheets.Sheet{ID: sheetID, WorkbookID: workbookID}
}

func createBenchCell(b *testing.B, db *sql.DB, sheetID string, row, col int) {
	b.Helper()

	cellID := NewTestID()
	now := FixedTime()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO cells (id, sheet_id, row_num, col_num, value, formula, display, cell_type,
			format, hyperlink, note, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, cellID, sheetID, row, col, `"test"`, "", "test", "text", "{}", "null", "", now)
	if err != nil {
		b.Fatalf("failed to create bench cell: %v", err)
	}
}

func generateCells(sheetID string, count int) []*cells.Cell {
	cellList := make([]*cells.Cell, count)
	now := time.Now()
	cols := 100

	for i := 0; i < count; i++ {
		cellList[i] = &cells.Cell{
			ID:        NewTestID(),
			SheetID:   sheetID,
			Row:       i / cols,
			Col:       i % cols,
			Value:     fmt.Sprintf("value-%d", i),
			Display:   fmt.Sprintf("value-%d", i),
			Type:      cells.CellTypeText,
			UpdatedAt: now,
		}
	}

	return cellList
}
