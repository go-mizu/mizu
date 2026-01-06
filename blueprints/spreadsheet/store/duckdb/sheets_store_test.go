package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
)

func TestSheetsStore_Create(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewSheetsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()
	sheet := &sheets.Sheet{
		ID:               NewTestID(),
		WorkbookID:       f.Workbook.ID,
		Name:             "New Sheet",
		Index:            1,
		Hidden:           false,
		Color:            "#FF0000",
		GridColor:        "#E2E8F0",
		FrozenRows:       2,
		FrozenCols:       1,
		DefaultRowHeight: 25,
		DefaultColWidth:  120,
		RowHeights:       map[int]int{1: 30, 5: 40},
		ColWidths:        map[int]int{0: 150, 2: 200},
		HiddenRows:       []int{10, 20},
		HiddenCols:       []int{5},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	err := store.Create(ctx, sheet)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByID(ctx, sheet.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != sheet.ID {
		t.Errorf("ID = %v, want %v", got.ID, sheet.ID)
	}
	if got.Name != "New Sheet" {
		t.Errorf("Name = %v, want New Sheet", got.Name)
	}
	if got.Index != 1 {
		t.Errorf("Index = %v, want 1", got.Index)
	}
	if got.Color != "#FF0000" {
		t.Errorf("Color = %v, want #FF0000", got.Color)
	}
	if got.FrozenRows != 2 {
		t.Errorf("FrozenRows = %v, want 2", got.FrozenRows)
	}
	if got.FrozenCols != 1 {
		t.Errorf("FrozenCols = %v, want 1", got.FrozenCols)
	}
}

func TestSheetsStore_Create_WithRowHeightsColWidths(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewSheetsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()
	sheet := &sheets.Sheet{
		ID:               NewTestID(),
		WorkbookID:       f.Workbook.ID,
		Name:             "Custom Dimensions Sheet",
		Index:            1,
		GridColor:        "#E2E8F0",
		DefaultRowHeight: 21,
		DefaultColWidth:  100,
		RowHeights:       map[int]int{0: 50, 5: 100, 10: 75},
		ColWidths:        map[int]int{0: 200, 3: 300},
		HiddenRows:       []int{},
		HiddenCols:       []int{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := store.Create(ctx, sheet); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByID(ctx, sheet.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if len(got.RowHeights) != 3 {
		t.Errorf("RowHeights length = %d, want 3", len(got.RowHeights))
	}
	if got.RowHeights[0] != 50 {
		t.Errorf("RowHeights[0] = %v, want 50", got.RowHeights[0])
	}
	if got.RowHeights[5] != 100 {
		t.Errorf("RowHeights[5] = %v, want 100", got.RowHeights[5])
	}

	if len(got.ColWidths) != 2 {
		t.Errorf("ColWidths length = %d, want 2", len(got.ColWidths))
	}
	if got.ColWidths[0] != 200 {
		t.Errorf("ColWidths[0] = %v, want 200", got.ColWidths[0])
	}
}

func TestSheetsStore_Create_WithHiddenRowsCols(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewSheetsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()
	sheet := &sheets.Sheet{
		ID:               NewTestID(),
		WorkbookID:       f.Workbook.ID,
		Name:             "Hidden Rows/Cols Sheet",
		Index:            1,
		GridColor:        "#E2E8F0",
		DefaultRowHeight: 21,
		DefaultColWidth:  100,
		RowHeights:       make(map[int]int),
		ColWidths:        make(map[int]int),
		HiddenRows:       []int{5, 10, 15, 20},
		HiddenCols:       []int{2, 4},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := store.Create(ctx, sheet); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByID(ctx, sheet.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if len(got.HiddenRows) != 4 {
		t.Errorf("HiddenRows length = %d, want 4", len(got.HiddenRows))
	}
	if len(got.HiddenCols) != 2 {
		t.Errorf("HiddenCols length = %d, want 2", len(got.HiddenCols))
	}
}

func TestSheetsStore_GetByID(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewSheetsStore(f.DB)
	ctx := context.Background()

	got, err := store.GetByID(ctx, f.Sheet.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != f.Sheet.ID {
		t.Errorf("ID = %v, want %v", got.ID, f.Sheet.ID)
	}
	if got.WorkbookID != f.Workbook.ID {
		t.Errorf("WorkbookID = %v, want %v", got.WorkbookID, f.Workbook.ID)
	}
}

func TestSheetsStore_GetByID_NotFound(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewSheetsStore(f.DB)
	ctx := context.Background()

	_, err := store.GetByID(ctx, "nonexistent-id")
	if err != sheets.ErrNotFound {
		t.Errorf("GetByID() error = %v, want %v", err, sheets.ErrNotFound)
	}
}

func TestSheetsStore_GetByID_NullColor(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewSheetsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()
	sheet := &sheets.Sheet{
		ID:               NewTestID(),
		WorkbookID:       f.Workbook.ID,
		Name:             "No Color Sheet",
		Index:            1,
		Color:            "", // Empty color
		GridColor:        "#E2E8F0",
		DefaultRowHeight: 21,
		DefaultColWidth:  100,
		RowHeights:       make(map[int]int),
		ColWidths:        make(map[int]int),
		HiddenRows:       []int{},
		HiddenCols:       []int{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := store.Create(ctx, sheet); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByID(ctx, sheet.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Color != "" {
		t.Errorf("Color = %v, want empty string", got.Color)
	}
}

func TestSheetsStore_GetByID_EmptyMaps(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewSheetsStore(f.DB)
	ctx := context.Background()

	// Sheet created by fixture has empty maps
	got, err := store.GetByID(ctx, f.Sheet.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// Should return initialized empty maps, not nil
	if got.RowHeights == nil {
		t.Error("RowHeights is nil, want initialized empty map")
	}
	if got.ColWidths == nil {
		t.Error("ColWidths is nil, want initialized empty map")
	}
}

func TestSheetsStore_ListByWorkbook(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewSheetsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()

	// Create additional sheets with different indices
	sheet2 := &sheets.Sheet{
		ID:               NewTestID(),
		WorkbookID:       f.Workbook.ID,
		Name:             "Sheet2",
		Index:            2,
		GridColor:        "#E2E8F0",
		DefaultRowHeight: 21,
		DefaultColWidth:  100,
		RowHeights:       make(map[int]int),
		ColWidths:        make(map[int]int),
		HiddenRows:       []int{},
		HiddenCols:       []int{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	sheet3 := &sheets.Sheet{
		ID:               NewTestID(),
		WorkbookID:       f.Workbook.ID,
		Name:             "Sheet3",
		Index:            1, // Index 1, should come after fixture's sheet (index 0)
		GridColor:        "#E2E8F0",
		DefaultRowHeight: 21,
		DefaultColWidth:  100,
		RowHeights:       make(map[int]int),
		ColWidths:        make(map[int]int),
		HiddenRows:       []int{},
		HiddenCols:       []int{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	for _, s := range []*sheets.Sheet{sheet2, sheet3} {
		if err := store.Create(ctx, s); err != nil {
			t.Fatalf("Create(%s) error = %v", s.Name, err)
		}
	}

	list, err := store.ListByWorkbook(ctx, f.Workbook.ID)
	if err != nil {
		t.Fatalf("ListByWorkbook() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("ListByWorkbook() returned %d sheets, want 3", len(list))
	}

	// Verify ordered by index_num
	if list[0].Index != 0 {
		t.Errorf("First sheet index = %d, want 0", list[0].Index)
	}
	if list[1].Index != 1 {
		t.Errorf("Second sheet index = %d, want 1", list[1].Index)
	}
	if list[2].Index != 2 {
		t.Errorf("Third sheet index = %d, want 2", list[2].Index)
	}
}

func TestSheetsStore_ListByWorkbook_Empty(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	wb := CreateTestWorkbook(t, db, user.ID)
	// Don't create any sheets

	store := NewSheetsStore(db)
	ctx := context.Background()

	// Delete the default sheet if one was created
	db.ExecContext(ctx, `DELETE FROM sheets WHERE workbook_id = ?`, wb.ID)

	list, err := store.ListByWorkbook(ctx, wb.ID)
	if err != nil {
		t.Fatalf("ListByWorkbook() error = %v", err)
	}

	if list == nil {
		t.Error("ListByWorkbook() returned nil, want empty slice")
	}
	if len(list) != 0 {
		t.Errorf("ListByWorkbook() returned %d sheets, want 0", len(list))
	}
}

func TestSheetsStore_ListByWorkbook_MultipleWorkbooks(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	wb1 := CreateTestWorkbook(t, db, user.ID)
	wb2 := CreateTestWorkbook(t, db, user.ID)

	store := NewSheetsStore(db)
	ctx := context.Background()

	now := FixedTime()

	// Create sheet for wb1
	sheet1 := &sheets.Sheet{
		ID:               NewTestID(),
		WorkbookID:       wb1.ID,
		Name:             "WB1 Sheet",
		Index:            0,
		GridColor:        "#E2E8F0",
		DefaultRowHeight: 21,
		DefaultColWidth:  100,
		RowHeights:       make(map[int]int),
		ColWidths:        make(map[int]int),
		HiddenRows:       []int{},
		HiddenCols:       []int{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Create sheet for wb2
	sheet2 := &sheets.Sheet{
		ID:               NewTestID(),
		WorkbookID:       wb2.ID,
		Name:             "WB2 Sheet",
		Index:            0,
		GridColor:        "#E2E8F0",
		DefaultRowHeight: 21,
		DefaultColWidth:  100,
		RowHeights:       make(map[int]int),
		ColWidths:        make(map[int]int),
		HiddenRows:       []int{},
		HiddenCols:       []int{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	for _, s := range []*sheets.Sheet{sheet1, sheet2} {
		if err := store.Create(ctx, s); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	list1, err := store.ListByWorkbook(ctx, wb1.ID)
	if err != nil {
		t.Fatalf("ListByWorkbook(wb1) error = %v", err)
	}
	if len(list1) != 1 {
		t.Errorf("ListByWorkbook(wb1) returned %d sheets, want 1", len(list1))
	}
	if list1[0].Name != "WB1 Sheet" {
		t.Errorf("list1[0].Name = %v, want WB1 Sheet", list1[0].Name)
	}
}

func TestSheetsStore_Update(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewSheetsStore(f.DB)
	ctx := context.Background()

	// Get current sheet
	sheet, err := store.GetByID(ctx, f.Sheet.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// Update fields
	sheet.Name = "Updated Sheet Name"
	sheet.Index = 5
	sheet.Color = "#00FF00"
	sheet.FrozenRows = 3
	sheet.FrozenCols = 2
	sheet.RowHeights = map[int]int{1: 50, 2: 60}
	sheet.ColWidths = map[int]int{0: 250}
	sheet.HiddenRows = []int{10}
	sheet.UpdatedAt = FixedTime().Add(time.Hour)

	if err := store.Update(ctx, sheet); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := store.GetByID(ctx, sheet.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Name != "Updated Sheet Name" {
		t.Errorf("Name = %v, want Updated Sheet Name", got.Name)
	}
	if got.Index != 5 {
		t.Errorf("Index = %v, want 5", got.Index)
	}
	if got.Color != "#00FF00" {
		t.Errorf("Color = %v, want #00FF00", got.Color)
	}
	if got.FrozenRows != 3 {
		t.Errorf("FrozenRows = %v, want 3", got.FrozenRows)
	}
	if got.RowHeights[1] != 50 {
		t.Errorf("RowHeights[1] = %v, want 50", got.RowHeights[1])
	}
}

func TestSheetsStore_Update_ToggleHidden(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewSheetsStore(f.DB)
	ctx := context.Background()

	sheet, err := store.GetByID(ctx, f.Sheet.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// Initially not hidden
	if sheet.Hidden {
		t.Error("Sheet should not be hidden initially")
	}

	// Hide it
	sheet.Hidden = true
	sheet.UpdatedAt = FixedTime().Add(time.Hour)
	if err := store.Update(ctx, sheet); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := store.GetByID(ctx, sheet.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if !got.Hidden {
		t.Error("Sheet should be hidden after update")
	}

	// Unhide it
	got.Hidden = false
	got.UpdatedAt = FixedTime().Add(2 * time.Hour)
	if err := store.Update(ctx, got); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	final, err := store.GetByID(ctx, sheet.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if final.Hidden {
		t.Error("Sheet should not be hidden after second update")
	}
}

func TestSheetsStore_Delete(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewSheetsStore(f.DB)
	ctx := context.Background()

	// Create a new sheet to delete (keep fixture sheet intact)
	now := FixedTime()
	sheet := &sheets.Sheet{
		ID:               NewTestID(),
		WorkbookID:       f.Workbook.ID,
		Name:             "To Be Deleted",
		Index:            99,
		GridColor:        "#E2E8F0",
		DefaultRowHeight: 21,
		DefaultColWidth:  100,
		RowHeights:       make(map[int]int),
		ColWidths:        make(map[int]int),
		HiddenRows:       []int{},
		HiddenCols:       []int{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Create(ctx, sheet); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := store.Delete(ctx, sheet.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := store.GetByID(ctx, sheet.ID)
	if err != sheets.ErrNotFound {
		t.Errorf("GetByID() after Delete() error = %v, want %v", err, sheets.ErrNotFound)
	}
}

func TestSheetsStore_Delete_CascadesCells(t *testing.T) {
	f := SetupTestFixture(t)
	sheetsStore := NewSheetsStore(f.DB)
	cellsStore := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create a cell in the sheet
	CreateTestCell(t, f.DB, f.Sheet.ID, 0, 0)

	// Verify cell exists
	_, err := cellsStore.Get(ctx, f.Sheet.ID, 0, 0)
	if err != nil {
		t.Fatalf("Cell should exist before delete: %v", err)
	}

	// Delete sheet
	if err := sheetsStore.Delete(ctx, f.Sheet.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify cells deleted
	var count int
	err = f.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM cells WHERE sheet_id = ?`, f.Sheet.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Count cells error = %v", err)
	}
	if count != 0 {
		t.Errorf("cells count = %d, want 0", count)
	}
}

func TestSheetsStore_Delete_CascadesMergedRegions(t *testing.T) {
	f := SetupTestFixture(t)
	sheetsStore := NewSheetsStore(f.DB)
	ctx := context.Background()

	// Create a merged region
	_, err := f.DB.ExecContext(ctx, `
		INSERT INTO merged_regions (id, sheet_id, start_row, start_col, end_row, end_col)
		VALUES (?, ?, 0, 0, 2, 2)
	`, NewTestID(), f.Sheet.ID)
	if err != nil {
		t.Fatalf("Insert merged_region error = %v", err)
	}

	// Delete sheet
	if err := sheetsStore.Delete(ctx, f.Sheet.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify merged regions deleted
	var count int
	err = f.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM merged_regions WHERE sheet_id = ?`, f.Sheet.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Count merged_regions error = %v", err)
	}
	if count != 0 {
		t.Errorf("merged_regions count = %d, want 0", count)
	}
}

func TestSheetsStore_Delete_CascadesConditionalFormats(t *testing.T) {
	f := SetupTestFixture(t)
	sheetsStore := NewSheetsStore(f.DB)
	ctx := context.Background()

	// Create a conditional format
	_, err := f.DB.ExecContext(ctx, `
		INSERT INTO conditional_formats (id, sheet_id, ranges, priority, format_type, rule, format)
		VALUES (?, ?, '[]', 1, 'cellIs', '{}', '{}')
	`, NewTestID(), f.Sheet.ID)
	if err != nil {
		t.Fatalf("Insert conditional_formats error = %v", err)
	}

	// Delete sheet
	if err := sheetsStore.Delete(ctx, f.Sheet.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify conditional formats deleted
	var count int
	err = f.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM conditional_formats WHERE sheet_id = ?`, f.Sheet.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Count conditional_formats error = %v", err)
	}
	if count != 0 {
		t.Errorf("conditional_formats count = %d, want 0", count)
	}
}

func TestSheetsStore_Delete_CascadesCharts(t *testing.T) {
	f := SetupTestFixture(t)
	sheetsStore := NewSheetsStore(f.DB)
	ctx := context.Background()

	// Create a chart
	_, err := f.DB.ExecContext(ctx, `
		INSERT INTO charts (id, sheet_id, chart_type, position, size, data_ranges)
		VALUES (?, ?, 'bar', '{}', '{}', '[]')
	`, NewTestID(), f.Sheet.ID)
	if err != nil {
		t.Fatalf("Insert charts error = %v", err)
	}

	// Delete sheet
	if err := sheetsStore.Delete(ctx, f.Sheet.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify charts deleted
	var count int
	err = f.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM charts WHERE sheet_id = ?`, f.Sheet.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Count charts error = %v", err)
	}
	if count != 0 {
		t.Errorf("charts count = %d, want 0", count)
	}
}
