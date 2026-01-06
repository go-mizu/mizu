package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
)

// =============================================================================
// Basic CRUD Operations
// =============================================================================

func TestCellsStore_Set(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()
	cell := &cells.Cell{
		ID:      NewTestID(),
		SheetID: f.Sheet.ID,
		Row:     5,
		Col:     3,
		Value:   "Hello World",
		Formula: "",
		Display: "Hello World",
		Type:    cells.CellTypeText,
		Format: cells.Format{
			Bold:     true,
			FontSize: 14,
		},
		Note:      "Test note",
		UpdatedAt: now,
	}

	err := store.Set(ctx, cell)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := store.Get(ctx, f.Sheet.ID, 5, 3)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.ID != cell.ID {
		t.Errorf("ID = %v, want %v", got.ID, cell.ID)
	}
	if got.Row != 5 {
		t.Errorf("Row = %v, want 5", got.Row)
	}
	if got.Col != 3 {
		t.Errorf("Col = %v, want 3", got.Col)
	}
	if got.Value != "Hello World" {
		t.Errorf("Value = %v, want Hello World", got.Value)
	}
	if got.Note != "Test note" {
		t.Errorf("Note = %v, want Test note", got.Note)
	}
}

func TestCellsStore_Set_Upsert(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()

	// Create initial cell
	cell := &cells.Cell{
		ID:        NewTestID(),
		SheetID:   f.Sheet.ID,
		Row:       0,
		Col:       0,
		Value:     "Original",
		Display:   "Original",
		Type:      cells.CellTypeText,
		UpdatedAt: now,
	}
	if err := store.Set(ctx, cell); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Update same cell (upsert)
	cell.Value = "Updated"
	cell.Display = "Updated"
	cell.UpdatedAt = now.Add(time.Hour)
	if err := store.Set(ctx, cell); err != nil {
		t.Fatalf("Set() upsert error = %v", err)
	}

	got, err := store.Get(ctx, f.Sheet.ID, 0, 0)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Value != "Updated" {
		t.Errorf("Value = %v, want Updated", got.Value)
	}

	// Verify only one cell exists (not duplicated)
	cellList, err := store.GetRange(ctx, f.Sheet.ID, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("GetRange() error = %v", err)
	}
	if len(cellList) != 1 {
		t.Errorf("GetRange() returned %d cells, want 1", len(cellList))
	}
}

func TestCellsStore_Set_WithFormula(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	cell := &cells.Cell{
		ID:        NewTestID(),
		SheetID:   f.Sheet.ID,
		Row:       0,
		Col:       0,
		Value:     float64(15),
		Formula:   "=SUM(A1:A10)",
		Display:   "15",
		Type:      cells.CellTypeFormula,
		UpdatedAt: FixedTime(),
	}

	if err := store.Set(ctx, cell); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := store.Get(ctx, f.Sheet.ID, 0, 0)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Formula != "=SUM(A1:A10)" {
		t.Errorf("Formula = %v, want =SUM(A1:A10)", got.Formula)
	}
	if got.Type != cells.CellTypeFormula {
		t.Errorf("Type = %v, want formula", got.Type)
	}
}

func TestCellsStore_Set_WithFormat(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	cell := &cells.Cell{
		ID:      NewTestID(),
		SheetID: f.Sheet.ID,
		Row:     0,
		Col:     0,
		Value:   "Formatted",
		Display: "Formatted",
		Type:    cells.CellTypeText,
		Format: cells.Format{
			FontFamily:      "Arial",
			FontSize:        16,
			FontColor:       "#FF0000",
			Bold:            true,
			Italic:          true,
			Underline:       true,
			Strikethrough:   true,
			BackgroundColor: "#FFFF00",
			HAlign:          "center",
			VAlign:          "middle",
			WrapText:        true,
			TextRotation:    45,
			NumberFormat:    "#,##0.00",
		},
		UpdatedAt: FixedTime(),
	}

	if err := store.Set(ctx, cell); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := store.Get(ctx, f.Sheet.ID, 0, 0)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Format.FontFamily != "Arial" {
		t.Errorf("Format.FontFamily = %v, want Arial", got.Format.FontFamily)
	}
	if got.Format.FontSize != 16 {
		t.Errorf("Format.FontSize = %v, want 16", got.Format.FontSize)
	}
	if !got.Format.Bold {
		t.Error("Format.Bold = false, want true")
	}
	if got.Format.HAlign != "center" {
		t.Errorf("Format.HAlign = %v, want center", got.Format.HAlign)
	}
	if got.Format.TextRotation != 45 {
		t.Errorf("Format.TextRotation = %v, want 45", got.Format.TextRotation)
	}
}

func TestCellsStore_Set_WithHyperlink(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	cell := &cells.Cell{
		ID:      NewTestID(),
		SheetID: f.Sheet.ID,
		Row:     0,
		Col:     0,
		Value:   "Click here",
		Display: "Click here",
		Type:    cells.CellTypeText,
		Hyperlink: &cells.Hyperlink{
			URL:   "https://example.com",
			Label: "Example Link",
		},
		UpdatedAt: FixedTime(),
	}

	if err := store.Set(ctx, cell); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := store.Get(ctx, f.Sheet.ID, 0, 0)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Hyperlink == nil {
		t.Fatal("Hyperlink is nil")
	}
	if got.Hyperlink.URL != "https://example.com" {
		t.Errorf("Hyperlink.URL = %v, want https://example.com", got.Hyperlink.URL)
	}
	if got.Hyperlink.Label != "Example Link" {
		t.Errorf("Hyperlink.Label = %v, want Example Link", got.Hyperlink.Label)
	}
}

func TestCellsStore_Get_NotFound(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	_, err := store.Get(ctx, f.Sheet.ID, 999, 999)
	if err != cells.ErrNotFound {
		t.Errorf("Get() error = %v, want %v", err, cells.ErrNotFound)
	}
}

func TestCellsStore_Delete(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create a cell
	cell := CreateTestCell(t, f.DB, f.Sheet.ID, 5, 5)

	// Delete it
	if err := store.Delete(ctx, f.Sheet.ID, 5, 5); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err := store.Get(ctx, f.Sheet.ID, 5, 5)
	if err != cells.ErrNotFound {
		t.Errorf("Get() after Delete() error = %v, want %v", err, cells.ErrNotFound)
	}
	_ = cell // suppress unused warning
}

func TestCellsStore_Delete_NonExistent(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Deleting non-existent cell should not error (idempotent)
	err := store.Delete(ctx, f.Sheet.ID, 999, 999)
	if err != nil {
		t.Errorf("Delete() non-existent error = %v, want nil", err)
	}
}

// =============================================================================
// Range Operations
// =============================================================================

func TestCellsStore_GetRange(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create a 3x3 grid of cells
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			CreateTestCell(t, f.DB, f.Sheet.ID, row, col)
		}
	}

	cellList, err := store.GetRange(ctx, f.Sheet.ID, 0, 0, 2, 2)
	if err != nil {
		t.Fatalf("GetRange() error = %v", err)
	}

	if len(cellList) != 9 {
		t.Errorf("GetRange() returned %d cells, want 9", len(cellList))
	}

	// Verify ordering (row-major)
	if cellList[0].Row != 0 || cellList[0].Col != 0 {
		t.Errorf("First cell at (%d,%d), want (0,0)", cellList[0].Row, cellList[0].Col)
	}
	if cellList[8].Row != 2 || cellList[8].Col != 2 {
		t.Errorf("Last cell at (%d,%d), want (2,2)", cellList[8].Row, cellList[8].Col)
	}
}

func TestCellsStore_GetRange_Sparse(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create sparse cells (not a complete grid)
	CreateTestCell(t, f.DB, f.Sheet.ID, 0, 0)
	CreateTestCell(t, f.DB, f.Sheet.ID, 2, 2)
	CreateTestCell(t, f.DB, f.Sheet.ID, 4, 4)

	// Get range that includes some empty positions
	cellList, err := store.GetRange(ctx, f.Sheet.ID, 0, 0, 4, 4)
	if err != nil {
		t.Fatalf("GetRange() error = %v", err)
	}

	// Should only return existing cells
	if len(cellList) != 3 {
		t.Errorf("GetRange() returned %d cells, want 3 (sparse)", len(cellList))
	}
}

func TestCellsStore_GetRange_Empty(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	cellList, err := store.GetRange(ctx, f.Sheet.ID, 0, 0, 10, 10)
	if err != nil {
		t.Fatalf("GetRange() error = %v", err)
	}

	if cellList == nil {
		t.Error("GetRange() returned nil, want empty slice")
	}
	if len(cellList) != 0 {
		t.Errorf("GetRange() returned %d cells, want 0", len(cellList))
	}
}

func TestCellsStore_GetRange_SingleCell(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	CreateTestCell(t, f.DB, f.Sheet.ID, 5, 5)

	cellList, err := store.GetRange(ctx, f.Sheet.ID, 5, 5, 5, 5)
	if err != nil {
		t.Fatalf("GetRange() error = %v", err)
	}

	if len(cellList) != 1 {
		t.Errorf("GetRange() returned %d cells, want 1", len(cellList))
	}
	if cellList[0].Row != 5 || cellList[0].Col != 5 {
		t.Errorf("Cell at (%d,%d), want (5,5)", cellList[0].Row, cellList[0].Col)
	}
}

func TestCellsStore_DeleteRange(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create cells
	for row := 0; row < 5; row++ {
		for col := 0; col < 5; col++ {
			CreateTestCell(t, f.DB, f.Sheet.ID, row, col)
		}
	}

	// Delete a 3x3 range
	if err := store.DeleteRange(ctx, f.Sheet.ID, 1, 1, 3, 3); err != nil {
		t.Fatalf("DeleteRange() error = %v", err)
	}

	// Get all remaining cells
	remaining, err := store.GetRange(ctx, f.Sheet.ID, 0, 0, 4, 4)
	if err != nil {
		t.Fatalf("GetRange() error = %v", err)
	}

	// Should have 25 - 9 = 16 cells remaining
	if len(remaining) != 16 {
		t.Errorf("Remaining cells = %d, want 16", len(remaining))
	}

	// Verify the deleted range is empty
	deleted, err := store.GetRange(ctx, f.Sheet.ID, 1, 1, 3, 3)
	if err != nil {
		t.Fatalf("GetRange() error = %v", err)
	}
	if len(deleted) != 0 {
		t.Errorf("Deleted range has %d cells, want 0", len(deleted))
	}
}

// =============================================================================
// Batch Operations
// =============================================================================

func TestCellsStore_BatchSet(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()
	cellList := make([]*cells.Cell, 10)
	for i := 0; i < 10; i++ {
		cellList[i] = &cells.Cell{
			ID:        NewTestID(),
			SheetID:   f.Sheet.ID,
			Row:       i,
			Col:       0,
			Value:     i,
			Display:   string(rune('0' + i)),
			Type:      cells.CellTypeNumber,
			UpdatedAt: now,
		}
	}

	if err := store.BatchSet(ctx, cellList); err != nil {
		t.Fatalf("BatchSet() error = %v", err)
	}

	// Verify all cells created
	result, err := store.GetRange(ctx, f.Sheet.ID, 0, 0, 9, 0)
	if err != nil {
		t.Fatalf("GetRange() error = %v", err)
	}

	if len(result) != 10 {
		t.Errorf("GetRange() returned %d cells, want 10", len(result))
	}
}

func TestCellsStore_BatchSet_Empty(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	err := store.BatchSet(ctx, []*cells.Cell{})
	if err != nil {
		t.Errorf("BatchSet() empty list error = %v, want nil", err)
	}
}

func TestCellsStore_BatchSet_Upsert(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()

	// Create initial cells
	initial := []*cells.Cell{
		{ID: NewTestID(), SheetID: f.Sheet.ID, Row: 0, Col: 0, Value: "A", Display: "A", Type: cells.CellTypeText, UpdatedAt: now},
		{ID: NewTestID(), SheetID: f.Sheet.ID, Row: 0, Col: 1, Value: "B", Display: "B", Type: cells.CellTypeText, UpdatedAt: now},
	}
	if err := store.BatchSet(ctx, initial); err != nil {
		t.Fatalf("BatchSet() initial error = %v", err)
	}

	// Batch update (mix of updates and new)
	updated := []*cells.Cell{
		{ID: NewTestID(), SheetID: f.Sheet.ID, Row: 0, Col: 0, Value: "A-Updated", Display: "A-Updated", Type: cells.CellTypeText, UpdatedAt: now},
		{ID: NewTestID(), SheetID: f.Sheet.ID, Row: 0, Col: 2, Value: "C-New", Display: "C-New", Type: cells.CellTypeText, UpdatedAt: now},
	}
	if err := store.BatchSet(ctx, updated); err != nil {
		t.Fatalf("BatchSet() update error = %v", err)
	}

	// Verify
	cellA, _ := store.Get(ctx, f.Sheet.ID, 0, 0)
	if cellA.Value != "A-Updated" {
		t.Errorf("Cell A value = %v, want A-Updated", cellA.Value)
	}

	cellB, _ := store.Get(ctx, f.Sheet.ID, 0, 1)
	if cellB.Value != "B" {
		t.Errorf("Cell B value = %v, want B (unchanged)", cellB.Value)
	}

	cellC, _ := store.Get(ctx, f.Sheet.ID, 0, 2)
	if cellC.Value != "C-New" {
		t.Errorf("Cell C value = %v, want C-New", cellC.Value)
	}
}

func TestCellsStore_BatchSet_Large(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()
	const count = 1000
	cellList := make([]*cells.Cell, count)
	for i := 0; i < count; i++ {
		cellList[i] = &cells.Cell{
			ID:        NewTestID(),
			SheetID:   f.Sheet.ID,
			Row:       i / 100,
			Col:       i % 100,
			Value:     i,
			Display:   "value",
			Type:      cells.CellTypeNumber,
			UpdatedAt: now,
		}
	}

	if err := store.BatchSet(ctx, cellList); err != nil {
		t.Fatalf("BatchSet() large error = %v", err)
	}

	// Verify count
	var dbCount int
	err := f.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM cells WHERE sheet_id = ?`, f.Sheet.ID).Scan(&dbCount)
	if err != nil {
		t.Fatalf("Count error = %v", err)
	}
	if dbCount != count {
		t.Errorf("DB count = %d, want %d", dbCount, count)
	}
}

// =============================================================================
// Merged Regions
// =============================================================================

func TestCellsStore_CreateMerge(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	region := &cells.MergedRegion{
		ID:       NewTestID(),
		SheetID:  f.Sheet.ID,
		StartRow: 0,
		StartCol: 0,
		EndRow:   2,
		EndCol:   3,
	}

	if err := store.CreateMerge(ctx, region); err != nil {
		t.Fatalf("CreateMerge() error = %v", err)
	}

	regions, err := store.GetMergedRegions(ctx, f.Sheet.ID)
	if err != nil {
		t.Fatalf("GetMergedRegions() error = %v", err)
	}

	if len(regions) != 1 {
		t.Fatalf("GetMergedRegions() returned %d regions, want 1", len(regions))
	}

	if regions[0].StartRow != 0 || regions[0].EndRow != 2 {
		t.Errorf("Region rows = (%d,%d), want (0,2)", regions[0].StartRow, regions[0].EndRow)
	}
	if regions[0].StartCol != 0 || regions[0].EndCol != 3 {
		t.Errorf("Region cols = (%d,%d), want (0,3)", regions[0].StartCol, regions[0].EndCol)
	}
}

func TestCellsStore_DeleteMerge(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	region := &cells.MergedRegion{
		ID:       NewTestID(),
		SheetID:  f.Sheet.ID,
		StartRow: 0,
		StartCol: 0,
		EndRow:   2,
		EndCol:   2,
	}
	if err := store.CreateMerge(ctx, region); err != nil {
		t.Fatalf("CreateMerge() error = %v", err)
	}

	if err := store.DeleteMerge(ctx, f.Sheet.ID, 0, 0, 2, 2); err != nil {
		t.Fatalf("DeleteMerge() error = %v", err)
	}

	regions, err := store.GetMergedRegions(ctx, f.Sheet.ID)
	if err != nil {
		t.Fatalf("GetMergedRegions() error = %v", err)
	}

	if len(regions) != 0 {
		t.Errorf("GetMergedRegions() returned %d regions, want 0", len(regions))
	}
}

func TestCellsStore_GetMergedRegions_Empty(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	regions, err := store.GetMergedRegions(ctx, f.Sheet.ID)
	if err != nil {
		t.Fatalf("GetMergedRegions() error = %v", err)
	}

	if regions == nil {
		t.Error("GetMergedRegions() returned nil, want empty slice")
	}
	if len(regions) != 0 {
		t.Errorf("GetMergedRegions() returned %d regions, want 0", len(regions))
	}
}

func TestCellsStore_GetMergedRegions_MultipleSheets(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create another sheet
	sheet2 := CreateTestSheet(t, f.DB, f.Workbook.ID)

	// Create merge in sheet1
	region1 := &cells.MergedRegion{
		ID:       NewTestID(),
		SheetID:  f.Sheet.ID,
		StartRow: 0,
		StartCol: 0,
		EndRow:   1,
		EndCol:   1,
	}
	if err := store.CreateMerge(ctx, region1); err != nil {
		t.Fatalf("CreateMerge(sheet1) error = %v", err)
	}

	// Create merge in sheet2
	region2 := &cells.MergedRegion{
		ID:       NewTestID(),
		SheetID:  sheet2.ID,
		StartRow: 5,
		StartCol: 5,
		EndRow:   6,
		EndCol:   6,
	}
	if err := store.CreateMerge(ctx, region2); err != nil {
		t.Fatalf("CreateMerge(sheet2) error = %v", err)
	}

	// Get regions for sheet1 only
	regions, err := store.GetMergedRegions(ctx, f.Sheet.ID)
	if err != nil {
		t.Fatalf("GetMergedRegions() error = %v", err)
	}

	if len(regions) != 1 {
		t.Errorf("GetMergedRegions() returned %d regions, want 1", len(regions))
	}
	if regions[0].StartRow != 0 {
		t.Errorf("Region from wrong sheet returned")
	}
}

// =============================================================================
// Row/Column Shifting
// =============================================================================

func TestCellsStore_ShiftRows_Insert(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create cells at rows 0, 1, 2
	CreateTestCell(t, f.DB, f.Sheet.ID, 0, 0)
	CreateTestCell(t, f.DB, f.Sheet.ID, 1, 0)
	CreateTestCell(t, f.DB, f.Sheet.ID, 2, 0)

	// Insert 2 rows at row 1
	if err := store.ShiftRows(ctx, f.Sheet.ID, 1, 2); err != nil {
		t.Fatalf("ShiftRows() error = %v", err)
	}

	// Row 0 should be unchanged
	cell0, err := store.Get(ctx, f.Sheet.ID, 0, 0)
	if err != nil {
		t.Fatalf("Get(0,0) error = %v", err)
	}
	if cell0.Row != 0 {
		t.Errorf("Row 0 cell row = %d, want 0", cell0.Row)
	}

	// Row 1 should now be at row 3
	cell1, err := store.Get(ctx, f.Sheet.ID, 3, 0)
	if err != nil {
		t.Fatalf("Get(3,0) error = %v", err)
	}
	if cell1.Row != 3 {
		t.Errorf("Former row 1 cell now at row = %d, want 3", cell1.Row)
	}

	// Row 2 should now be at row 4
	cell2, err := store.Get(ctx, f.Sheet.ID, 4, 0)
	if err != nil {
		t.Fatalf("Get(4,0) error = %v", err)
	}
	if cell2.Row != 4 {
		t.Errorf("Former row 2 cell now at row = %d, want 4", cell2.Row)
	}

	// Rows 1 and 2 should be empty now
	_, err = store.Get(ctx, f.Sheet.ID, 1, 0)
	if err != cells.ErrNotFound {
		t.Error("Row 1 should be empty after shift")
	}
}

func TestCellsStore_ShiftRows_Delete(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create cells at rows 0, 1, 2, 3
	CreateTestCell(t, f.DB, f.Sheet.ID, 0, 0)
	CreateTestCell(t, f.DB, f.Sheet.ID, 1, 0)
	CreateTestCell(t, f.DB, f.Sheet.ID, 2, 0)
	CreateTestCell(t, f.DB, f.Sheet.ID, 3, 0)

	// Delete row 1 (count = -1)
	if err := store.ShiftRows(ctx, f.Sheet.ID, 1, -1); err != nil {
		t.Fatalf("ShiftRows() error = %v", err)
	}

	// Row 0 should be unchanged
	cell0, err := store.Get(ctx, f.Sheet.ID, 0, 0)
	if err != nil {
		t.Fatalf("Get(0,0) error = %v", err)
	}
	if cell0.Row != 0 {
		t.Errorf("Row 0 cell row = %d, want 0", cell0.Row)
	}

	// Former row 2 should now be at row 1
	cell1, err := store.Get(ctx, f.Sheet.ID, 1, 0)
	if err != nil {
		t.Fatalf("Get(1,0) error = %v", err)
	}
	if cell1.Row != 1 {
		t.Errorf("Former row 2 now at row = %d, want 1", cell1.Row)
	}

	// Former row 3 should now be at row 2
	cell2, err := store.Get(ctx, f.Sheet.ID, 2, 0)
	if err != nil {
		t.Fatalf("Get(2,0) error = %v", err)
	}
	if cell2.Row != 2 {
		t.Errorf("Former row 3 now at row = %d, want 2", cell2.Row)
	}

	// Total should be 3 cells
	all, err := store.GetRange(ctx, f.Sheet.ID, 0, 0, 10, 0)
	if err != nil {
		t.Fatalf("GetRange() error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Total cells = %d, want 3", len(all))
	}
}

func TestCellsStore_ShiftCols_Insert(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create cells at cols 0, 1, 2
	CreateTestCell(t, f.DB, f.Sheet.ID, 0, 0)
	CreateTestCell(t, f.DB, f.Sheet.ID, 0, 1)
	CreateTestCell(t, f.DB, f.Sheet.ID, 0, 2)

	// Insert 1 column at col 1
	if err := store.ShiftCols(ctx, f.Sheet.ID, 1, 1); err != nil {
		t.Fatalf("ShiftCols() error = %v", err)
	}

	// Col 0 should be unchanged
	cell0, err := store.Get(ctx, f.Sheet.ID, 0, 0)
	if err != nil {
		t.Fatalf("Get(0,0) error = %v", err)
	}
	if cell0.Col != 0 {
		t.Errorf("Col 0 cell col = %d, want 0", cell0.Col)
	}

	// Col 1 should now be at col 2
	cell1, err := store.Get(ctx, f.Sheet.ID, 0, 2)
	if err != nil {
		t.Fatalf("Get(0,2) error = %v", err)
	}
	if cell1.Col != 2 {
		t.Errorf("Former col 1 cell now at col = %d, want 2", cell1.Col)
	}

	// Col 1 should be empty
	_, err = store.Get(ctx, f.Sheet.ID, 0, 1)
	if err != cells.ErrNotFound {
		t.Error("Col 1 should be empty after shift")
	}
}

func TestCellsStore_ShiftCols_Delete(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create cells at cols 0, 1, 2
	CreateTestCell(t, f.DB, f.Sheet.ID, 0, 0)
	CreateTestCell(t, f.DB, f.Sheet.ID, 0, 1)
	CreateTestCell(t, f.DB, f.Sheet.ID, 0, 2)

	// Delete col 1
	if err := store.ShiftCols(ctx, f.Sheet.ID, 1, -1); err != nil {
		t.Fatalf("ShiftCols() error = %v", err)
	}

	// Col 0 should be unchanged
	cell0, err := store.Get(ctx, f.Sheet.ID, 0, 0)
	if err != nil {
		t.Fatalf("Get(0,0) error = %v", err)
	}
	if cell0.Col != 0 {
		t.Errorf("Col 0 cell col = %d, want 0", cell0.Col)
	}

	// Former col 2 should now be at col 1
	cell1, err := store.Get(ctx, f.Sheet.ID, 0, 1)
	if err != nil {
		t.Fatalf("Get(0,1) error = %v", err)
	}
	if cell1.Col != 1 {
		t.Errorf("Former col 2 now at col = %d, want 1", cell1.Col)
	}

	// Total should be 2 cells
	all, err := store.GetRange(ctx, f.Sheet.ID, 0, 0, 0, 10)
	if err != nil {
		t.Fatalf("GetRange() error = %v", err)
	}
	if len(all) != 2 {
		t.Errorf("Total cells = %d, want 2", len(all))
	}
}

func TestCellsStore_ShiftRows_PreservesColumns(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create cells at different columns
	CreateTestCell(t, f.DB, f.Sheet.ID, 1, 0)
	CreateTestCell(t, f.DB, f.Sheet.ID, 1, 5)
	CreateTestCell(t, f.DB, f.Sheet.ID, 1, 10)

	// Insert row at 1
	if err := store.ShiftRows(ctx, f.Sheet.ID, 1, 1); err != nil {
		t.Fatalf("ShiftRows() error = %v", err)
	}

	// All cells should now be at row 2, preserving columns
	cellList, err := store.GetRange(ctx, f.Sheet.ID, 2, 0, 2, 10)
	if err != nil {
		t.Fatalf("GetRange() error = %v", err)
	}

	if len(cellList) != 3 {
		t.Fatalf("GetRange() returned %d cells, want 3", len(cellList))
	}

	cols := make(map[int]bool)
	for _, c := range cellList {
		cols[c.Col] = true
		if c.Row != 2 {
			t.Errorf("Cell at row %d, want 2", c.Row)
		}
	}

	if !cols[0] || !cols[5] || !cols[10] {
		t.Error("Column positions not preserved")
	}
}

// =============================================================================
// Data Types
// =============================================================================

func TestCellsStore_CellTypes(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	tests := []struct {
		name     string
		cellType cells.CellType
		value    interface{}
	}{
		{"text", cells.CellTypeText, "Hello"},
		{"number", cells.CellTypeNumber, 42.5},
		{"bool", cells.CellTypeBool, true},
		{"date", cells.CellTypeDate, "2025-01-01"},
		{"error", cells.CellTypeError, "#DIV/0!"},
		{"formula", cells.CellTypeFormula, 100},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cell := &cells.Cell{
				ID:        NewTestID(),
				SheetID:   f.Sheet.ID,
				Row:       i,
				Col:       0,
				Value:     tt.value,
				Display:   "display",
				Type:      tt.cellType,
				UpdatedAt: FixedTime(),
			}

			if err := store.Set(ctx, cell); err != nil {
				t.Fatalf("Set() error = %v", err)
			}

			got, err := store.Get(ctx, f.Sheet.ID, i, 0)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if got.Type != tt.cellType {
				t.Errorf("Type = %v, want %v", got.Type, tt.cellType)
			}
		})
	}
}

func TestCellsStore_SpecialCharacters(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	specialValues := []string{
		"Hello \"World\"",
		"Line1\nLine2",
		"Tab\there",
		"Unicode: \u4e2d\u6587",
		"Emoji: \U0001F600",
		`Backslash: \\ and quote: "`,
		"Single 'quotes'",
		"<html>&amp;</html>",
	}

	for i, val := range specialValues {
		cell := &cells.Cell{
			ID:        NewTestID(),
			SheetID:   f.Sheet.ID,
			Row:       i,
			Col:       0,
			Value:     val,
			Display:   val,
			Type:      cells.CellTypeText,
			UpdatedAt: FixedTime(),
		}

		if err := store.Set(ctx, cell); err != nil {
			t.Fatalf("Set() error for %q: %v", val, err)
		}

		got, err := store.Get(ctx, f.Sheet.ID, i, 0)
		if err != nil {
			t.Fatalf("Get() error for %q: %v", val, err)
		}

		if got.Value != val {
			t.Errorf("Value = %v, want %v", got.Value, val)
		}
	}
}

func TestCellsStore_Format_Borders(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	cell := &cells.Cell{
		ID:      NewTestID(),
		SheetID: f.Sheet.ID,
		Row:     0,
		Col:     0,
		Value:   "Bordered",
		Display: "Bordered",
		Type:    cells.CellTypeText,
		Format: cells.Format{
			BorderTop:    cells.Border{Style: "thick", Color: "#000000"},
			BorderRight:  cells.Border{Style: "thin", Color: "#FF0000"},
			BorderBottom: cells.Border{Style: "dashed", Color: "#00FF00"},
			BorderLeft:   cells.Border{Style: "double", Color: "#0000FF"},
		},
		UpdatedAt: FixedTime(),
	}

	if err := store.Set(ctx, cell); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := store.Get(ctx, f.Sheet.ID, 0, 0)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Format.BorderTop.Style != "thick" {
		t.Errorf("BorderTop.Style = %v, want thick", got.Format.BorderTop.Style)
	}
	if got.Format.BorderRight.Color != "#FF0000" {
		t.Errorf("BorderRight.Color = %v, want #FF0000", got.Format.BorderRight.Color)
	}
	if got.Format.BorderBottom.Style != "dashed" {
		t.Errorf("BorderBottom.Style = %v, want dashed", got.Format.BorderBottom.Style)
	}
	if got.Format.BorderLeft.Style != "double" {
		t.Errorf("BorderLeft.Style = %v, want double", got.Format.BorderLeft.Style)
	}
}

func TestCellsStore_LargeNote(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewCellsStore(f.DB)
	ctx := context.Background()

	// Create a large note (10KB)
	largeNote := make([]byte, 10*1024)
	for i := range largeNote {
		largeNote[i] = byte('a' + (i % 26))
	}

	cell := &cells.Cell{
		ID:        NewTestID(),
		SheetID:   f.Sheet.ID,
		Row:       0,
		Col:       0,
		Value:     "Has large note",
		Display:   "Has large note",
		Type:      cells.CellTypeText,
		Note:      string(largeNote),
		UpdatedAt: FixedTime(),
	}

	if err := store.Set(ctx, cell); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := store.Get(ctx, f.Sheet.ID, 0, 0)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(got.Note) != len(largeNote) {
		t.Errorf("Note length = %d, want %d", len(got.Note), len(largeNote))
	}
}
