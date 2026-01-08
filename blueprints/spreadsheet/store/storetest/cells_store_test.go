package storetest

import (
	"testing"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
)

func TestCellsStore_Set(t *testing.T) {
	RunForAllDrivers(t, "Set", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

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

		err := store.Set(t.Context(), cell)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		got, err := store.Get(t.Context(), f.Sheet.ID, 5, 3)
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
	})
}

func TestCellsStore_Set_Upsert(t *testing.T) {
	RunForAllDrivers(t, "Set_Upsert", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

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
		if err := store.Set(t.Context(), cell); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Update same cell (upsert)
		cell.Value = "Updated"
		cell.Display = "Updated"
		cell.UpdatedAt = now.Add(time.Hour)
		if err := store.Set(t.Context(), cell); err != nil {
			t.Fatalf("Set() upsert error = %v", err)
		}

		got, err := store.Get(t.Context(), f.Sheet.ID, 0, 0)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if got.Value != "Updated" {
			t.Errorf("Value = %v, want Updated", got.Value)
		}

		// Verify only one cell exists (not duplicated)
		cellList, err := store.GetRange(t.Context(), f.Sheet.ID, 0, 0, 0, 0)
		if err != nil {
			t.Fatalf("GetRange() error = %v", err)
		}
		if len(cellList) != 1 {
			t.Errorf("GetRange() returned %d cells, want 1", len(cellList))
		}
	})
}

func TestCellsStore_Get_NotFound(t *testing.T) {
	RunForAllDrivers(t, "Get_NotFound", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

		_, err := store.Get(t.Context(), f.Sheet.ID, 999, 999)
		if err != cells.ErrNotFound {
			t.Errorf("Get() error = %v, want %v", err, cells.ErrNotFound)
		}
	})
}

func TestCellsStore_Delete(t *testing.T) {
	RunForAllDrivers(t, "Delete", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

		// Create a cell
		f.CreateTestCell(t, f.Sheet.ID, 5, 5)

		// Delete it
		if err := store.Delete(t.Context(), f.Sheet.ID, 5, 5); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify it's gone
		_, err := store.Get(t.Context(), f.Sheet.ID, 5, 5)
		if err != cells.ErrNotFound {
			t.Errorf("Get() after Delete() error = %v, want %v", err, cells.ErrNotFound)
		}
	})
}

func TestCellsStore_GetRange(t *testing.T) {
	RunForAllDrivers(t, "GetRange", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

		// Create a 3x3 grid of cells
		for row := 0; row < 3; row++ {
			for col := 0; col < 3; col++ {
				f.CreateTestCell(t, f.Sheet.ID, row, col)
			}
		}

		cellList, err := store.GetRange(t.Context(), f.Sheet.ID, 0, 0, 2, 2)
		if err != nil {
			t.Fatalf("GetRange() error = %v", err)
		}

		if len(cellList) != 9 {
			t.Errorf("GetRange() returned %d cells, want 9", len(cellList))
		}
	})
}

func TestCellsStore_GetRange_Empty(t *testing.T) {
	RunForAllDrivers(t, "GetRange_Empty", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

		cellList, err := store.GetRange(t.Context(), f.Sheet.ID, 0, 0, 10, 10)
		if err != nil {
			t.Fatalf("GetRange() error = %v", err)
		}

		if cellList == nil {
			t.Error("GetRange() returned nil, want empty slice")
		}
		if len(cellList) != 0 {
			t.Errorf("GetRange() returned %d cells, want 0", len(cellList))
		}
	})
}

func TestCellsStore_BatchSet(t *testing.T) {
	RunForAllDrivers(t, "BatchSet", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

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

		if err := store.BatchSet(t.Context(), cellList); err != nil {
			t.Fatalf("BatchSet() error = %v", err)
		}

		// Verify all cells created
		result, err := store.GetRange(t.Context(), f.Sheet.ID, 0, 0, 9, 0)
		if err != nil {
			t.Fatalf("GetRange() error = %v", err)
		}

		if len(result) != 10 {
			t.Errorf("GetRange() returned %d cells, want 10", len(result))
		}
	})
}

func TestCellsStore_CreateMerge(t *testing.T) {
	RunForAllDrivers(t, "CreateMerge", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

		region := &cells.MergedRegion{
			ID:       NewTestID(),
			SheetID:  f.Sheet.ID,
			StartRow: 0,
			StartCol: 0,
			EndRow:   2,
			EndCol:   3,
		}

		if err := store.CreateMerge(t.Context(), region); err != nil {
			t.Fatalf("CreateMerge() error = %v", err)
		}

		regions, err := store.GetMergedRegions(t.Context(), f.Sheet.ID)
		if err != nil {
			t.Fatalf("GetMergedRegions() error = %v", err)
		}

		if len(regions) != 1 {
			t.Fatalf("GetMergedRegions() returned %d regions, want 1", len(regions))
		}

		if regions[0].StartRow != 0 || regions[0].EndRow != 2 {
			t.Errorf("Region rows = (%d,%d), want (0,2)", regions[0].StartRow, regions[0].EndRow)
		}
	})
}

func TestCellsStore_DeleteMerge(t *testing.T) {
	RunForAllDrivers(t, "DeleteMerge", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

		region := &cells.MergedRegion{
			ID:       NewTestID(),
			SheetID:  f.Sheet.ID,
			StartRow: 0,
			StartCol: 0,
			EndRow:   2,
			EndCol:   2,
		}
		if err := store.CreateMerge(t.Context(), region); err != nil {
			t.Fatalf("CreateMerge() error = %v", err)
		}

		if err := store.DeleteMerge(t.Context(), f.Sheet.ID, 0, 0, 2, 2); err != nil {
			t.Fatalf("DeleteMerge() error = %v", err)
		}

		regions, err := store.GetMergedRegions(t.Context(), f.Sheet.ID)
		if err != nil {
			t.Fatalf("GetMergedRegions() error = %v", err)
		}

		if len(regions) != 0 {
			t.Errorf("GetMergedRegions() returned %d regions, want 0", len(regions))
		}
	})
}

func TestCellsStore_BatchCreateMerge(t *testing.T) {
	RunForAllDrivers(t, "BatchCreateMerge", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

		regions := []*cells.MergedRegion{
			{
				ID:       NewTestID(),
				SheetID:  f.Sheet.ID,
				StartRow: 0,
				StartCol: 0,
				EndRow:   1,
				EndCol:   1,
			},
			{
				ID:       NewTestID(),
				SheetID:  f.Sheet.ID,
				StartRow: 3,
				StartCol: 0,
				EndRow:   4,
				EndCol:   2,
			},
		}

		if err := store.BatchCreateMerge(t.Context(), regions); err != nil {
			t.Fatalf("BatchCreateMerge() error = %v", err)
		}

		created, err := store.GetMergedRegions(t.Context(), f.Sheet.ID)
		if err != nil {
			t.Fatalf("GetMergedRegions() error = %v", err)
		}

		if len(created) != 2 {
			t.Errorf("Expected 2 merged regions, got %d", len(created))
		}
	})
}

func TestCellsStore_GetByPositions(t *testing.T) {
	RunForAllDrivers(t, "GetByPositions", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

		// Create cells at specific positions
		f.CreateTestCell(t, f.Sheet.ID, 0, 0)
		f.CreateTestCell(t, f.Sheet.ID, 5, 5)
		f.CreateTestCell(t, f.Sheet.ID, 10, 10)

		positions := []cells.CellPosition{
			{Row: 0, Col: 0},
			{Row: 5, Col: 5},
			{Row: 10, Col: 10},
			{Row: 99, Col: 99}, // doesn't exist
		}

		result, err := store.GetByPositions(t.Context(), f.Sheet.ID, positions)
		if err != nil {
			t.Fatalf("GetByPositions() error = %v", err)
		}

		if len(result) != 3 {
			t.Errorf("GetByPositions() returned %d cells, want 3", len(result))
		}

		if _, ok := result[cells.CellPosition{Row: 0, Col: 0}]; !ok {
			t.Error("Missing cell at (0,0)")
		}
		if _, ok := result[cells.CellPosition{Row: 5, Col: 5}]; !ok {
			t.Error("Missing cell at (5,5)")
		}
	})
}

func TestCellsStore_DeleteRowsRange(t *testing.T) {
	RunForAllDrivers(t, "DeleteRowsRange", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

		// Create 10 rows of cells
		for row := 0; row < 10; row++ {
			f.CreateTestCell(t, f.Sheet.ID, row, 0)
		}

		// Delete rows 3-5 (3 rows starting at row 3)
		if err := store.DeleteRowsRange(t.Context(), f.Sheet.ID, 3, 3); err != nil {
			t.Fatalf("DeleteRowsRange() error = %v", err)
		}

		// Total should be 7 cells
		all, err := store.GetRange(t.Context(), f.Sheet.ID, 0, 0, 20, 0)
		if err != nil {
			t.Fatalf("GetRange() error = %v", err)
		}
		if len(all) != 7 {
			t.Errorf("Total cells = %d, want 7", len(all))
		}
	})
}

func TestCellsStore_DeleteColsRange(t *testing.T) {
	RunForAllDrivers(t, "DeleteColsRange", func(t *testing.T, factory StoreFactory) {
		f := SetupFixture(t, factory)
		store := factory.NewCellsStore(f.DB)

		// Create 10 columns of cells
		for col := 0; col < 10; col++ {
			f.CreateTestCell(t, f.Sheet.ID, 0, col)
		}

		// Delete columns 2-4 (3 columns starting at col 2)
		if err := store.DeleteColsRange(t.Context(), f.Sheet.ID, 2, 3); err != nil {
			t.Fatalf("DeleteColsRange() error = %v", err)
		}

		// Total should be 7 cells
		all, err := store.GetRange(t.Context(), f.Sheet.ID, 0, 0, 0, 20)
		if err != nil {
			t.Fatalf("GetRange() error = %v", err)
		}
		if len(all) != 7 {
			t.Errorf("Total cells = %d, want 7", len(all))
		}
	})
}
