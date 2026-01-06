package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

func TestWorkbooksStore_Create(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	store := NewWorkbooksStore(db)
	ctx := context.Background()

	now := FixedTime()
	wb := &workbooks.Workbook{
		ID:      NewTestID(),
		Name:    "My Workbook",
		OwnerID: user.ID,
		Settings: workbooks.Settings{
			Locale:          "en-US",
			TimeZone:        "America/New_York",
			CalculationMode: "auto",
			IterativeCalc:   true,
			MaxIterations:   100,
			MaxChange:       0.001,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := store.Create(ctx, wb)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify workbook was created
	got, err := store.GetByID(ctx, wb.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != wb.ID {
		t.Errorf("ID = %v, want %v", got.ID, wb.ID)
	}
	if got.Name != wb.Name {
		t.Errorf("Name = %v, want %v", got.Name, wb.Name)
	}
	if got.OwnerID != wb.OwnerID {
		t.Errorf("OwnerID = %v, want %v", got.OwnerID, wb.OwnerID)
	}
	if got.Settings.Locale != "en-US" {
		t.Errorf("Settings.Locale = %v, want en-US", got.Settings.Locale)
	}
	if got.Settings.IterativeCalc != true {
		t.Errorf("Settings.IterativeCalc = %v, want true", got.Settings.IterativeCalc)
	}
	if got.Settings.MaxIterations != 100 {
		t.Errorf("Settings.MaxIterations = %v, want 100", got.Settings.MaxIterations)
	}
}

func TestWorkbooksStore_Create_NullSettings(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	store := NewWorkbooksStore(db)
	ctx := context.Background()

	now := FixedTime()
	wb := &workbooks.Workbook{
		ID:        NewTestID(),
		Name:      "No Settings Workbook",
		OwnerID:   user.ID,
		Settings:  workbooks.Settings{}, // Empty settings
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := store.Create(ctx, wb)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByID(ctx, wb.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != wb.ID {
		t.Errorf("ID = %v, want %v", got.ID, wb.ID)
	}
}

func TestWorkbooksStore_GetByID(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	store := NewWorkbooksStore(db)
	ctx := context.Background()

	now := FixedTime()
	wb := &workbooks.Workbook{
		ID:      NewTestID(),
		Name:    "Test Workbook",
		OwnerID: user.ID,
		Settings: workbooks.Settings{
			Locale:   "de-DE",
			TimeZone: "Europe/Berlin",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Create(ctx, wb); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByID(ctx, wb.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != wb.ID {
		t.Errorf("ID = %v, want %v", got.ID, wb.ID)
	}
	if got.Settings.Locale != "de-DE" {
		t.Errorf("Settings.Locale = %v, want de-DE", got.Settings.Locale)
	}
}

func TestWorkbooksStore_GetByID_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	store := NewWorkbooksStore(db)
	ctx := context.Background()

	_, err := store.GetByID(ctx, "nonexistent-id")
	if err != workbooks.ErrNotFound {
		t.Errorf("GetByID() error = %v, want %v", err, workbooks.ErrNotFound)
	}
}

func TestWorkbooksStore_ListByOwner(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	store := NewWorkbooksStore(db)
	ctx := context.Background()

	now := FixedTime()

	// Create multiple workbooks with different timestamps
	wb1 := &workbooks.Workbook{
		ID:        NewTestID(),
		Name:      "Workbook 1",
		OwnerID:   user.ID,
		Settings:  workbooks.Settings{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	wb2 := &workbooks.Workbook{
		ID:        NewTestID(),
		Name:      "Workbook 2",
		OwnerID:   user.ID,
		Settings:  workbooks.Settings{},
		CreatedAt: now.Add(time.Hour),
		UpdatedAt: now.Add(time.Hour),
	}
	wb3 := &workbooks.Workbook{
		ID:        NewTestID(),
		Name:      "Workbook 3",
		OwnerID:   user.ID,
		Settings:  workbooks.Settings{},
		CreatedAt: now.Add(2 * time.Hour),
		UpdatedAt: now.Add(2 * time.Hour),
	}

	for _, wb := range []*workbooks.Workbook{wb1, wb2, wb3} {
		if err := store.Create(ctx, wb); err != nil {
			t.Fatalf("Create(%s) error = %v", wb.Name, err)
		}
	}

	list, err := store.ListByOwner(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListByOwner() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("ListByOwner() returned %d workbooks, want 3", len(list))
	}

	// Verify ordered by updated_at DESC (wb3 first)
	if list[0].Name != "Workbook 3" {
		t.Errorf("First workbook = %v, want Workbook 3 (most recent)", list[0].Name)
	}
	if list[2].Name != "Workbook 1" {
		t.Errorf("Last workbook = %v, want Workbook 1 (oldest)", list[2].Name)
	}
}

func TestWorkbooksStore_ListByOwner_Empty(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	store := NewWorkbooksStore(db)
	ctx := context.Background()

	list, err := store.ListByOwner(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListByOwner() error = %v", err)
	}

	if list == nil {
		t.Error("ListByOwner() returned nil, want empty slice")
	}
	if len(list) != 0 {
		t.Errorf("ListByOwner() returned %d workbooks, want 0", len(list))
	}
}

func TestWorkbooksStore_ListByOwner_MultipleOwners(t *testing.T) {
	db := SetupTestDB(t)
	user1 := CreateTestUser(t, db)
	user2 := CreateTestUser(t, db)
	store := NewWorkbooksStore(db)
	ctx := context.Background()

	now := FixedTime()

	// Create workbooks for user1
	wb1 := &workbooks.Workbook{
		ID:        NewTestID(),
		Name:      "User1 Workbook",
		OwnerID:   user1.ID,
		Settings:  workbooks.Settings{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create workbooks for user2
	wb2 := &workbooks.Workbook{
		ID:        NewTestID(),
		Name:      "User2 Workbook",
		OwnerID:   user2.ID,
		Settings:  workbooks.Settings{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	for _, wb := range []*workbooks.Workbook{wb1, wb2} {
		if err := store.Create(ctx, wb); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// List user1's workbooks
	list1, err := store.ListByOwner(ctx, user1.ID)
	if err != nil {
		t.Fatalf("ListByOwner(user1) error = %v", err)
	}
	if len(list1) != 1 {
		t.Errorf("ListByOwner(user1) returned %d workbooks, want 1", len(list1))
	}
	if list1[0].Name != "User1 Workbook" {
		t.Errorf("list1[0].Name = %v, want User1 Workbook", list1[0].Name)
	}

	// List user2's workbooks
	list2, err := store.ListByOwner(ctx, user2.ID)
	if err != nil {
		t.Fatalf("ListByOwner(user2) error = %v", err)
	}
	if len(list2) != 1 {
		t.Errorf("ListByOwner(user2) returned %d workbooks, want 1", len(list2))
	}
	if list2[0].Name != "User2 Workbook" {
		t.Errorf("list2[0].Name = %v, want User2 Workbook", list2[0].Name)
	}
}

func TestWorkbooksStore_Update(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	store := NewWorkbooksStore(db)
	ctx := context.Background()

	now := FixedTime()
	wb := &workbooks.Workbook{
		ID:      NewTestID(),
		Name:    "Original Name",
		OwnerID: user.ID,
		Settings: workbooks.Settings{
			Locale:          "en-US",
			CalculationMode: "auto",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Create(ctx, wb); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update workbook
	wb.Name = "Updated Name"
	wb.Settings.Locale = "fr-FR"
	wb.Settings.CalculationMode = "manual"
	wb.UpdatedAt = now.Add(time.Hour)

	if err := store.Update(ctx, wb); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := store.GetByID(ctx, wb.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", got.Name)
	}
	if got.Settings.Locale != "fr-FR" {
		t.Errorf("Settings.Locale = %v, want fr-FR", got.Settings.Locale)
	}
	if got.Settings.CalculationMode != "manual" {
		t.Errorf("Settings.CalculationMode = %v, want manual", got.Settings.CalculationMode)
	}
}

func TestWorkbooksStore_Delete(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	store := NewWorkbooksStore(db)
	ctx := context.Background()

	now := FixedTime()
	wb := &workbooks.Workbook{
		ID:        NewTestID(),
		Name:      "To Be Deleted",
		OwnerID:   user.ID,
		Settings:  workbooks.Settings{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Create(ctx, wb); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete workbook
	if err := store.Delete(ctx, wb.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify workbook is gone
	_, err := store.GetByID(ctx, wb.ID)
	if err != workbooks.ErrNotFound {
		t.Errorf("GetByID() after Delete() error = %v, want %v", err, workbooks.ErrNotFound)
	}
}

func TestWorkbooksStore_Delete_CascadesNamedRanges(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	wbStore := NewWorkbooksStore(db)
	ctx := context.Background()

	now := FixedTime()
	wb := &workbooks.Workbook{
		ID:        NewTestID(),
		Name:      "Workbook With Named Ranges",
		OwnerID:   user.ID,
		Settings:  workbooks.Settings{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := wbStore.Create(ctx, wb); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Insert a named range directly
	_, err := db.ExecContext(ctx, `
		INSERT INTO named_ranges (id, workbook_id, name, range_ref, created_at)
		VALUES (?, ?, 'TestRange', 'A1:B10', ?)
	`, NewTestID(), wb.ID, now)
	if err != nil {
		t.Fatalf("Insert named_range error = %v", err)
	}

	// Delete workbook
	if err := wbStore.Delete(ctx, wb.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify named ranges deleted
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM named_ranges WHERE workbook_id = ?`, wb.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Count named_ranges error = %v", err)
	}
	if count != 0 {
		t.Errorf("named_ranges count = %d, want 0 (should be cascade deleted)", count)
	}
}

func TestWorkbooksStore_Delete_CascadesShares(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	wbStore := NewWorkbooksStore(db)
	ctx := context.Background()

	now := FixedTime()
	wb := &workbooks.Workbook{
		ID:        NewTestID(),
		Name:      "Workbook With Shares",
		OwnerID:   user.ID,
		Settings:  workbooks.Settings{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := wbStore.Create(ctx, wb); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Insert a share directly
	_, err := db.ExecContext(ctx, `
		INSERT INTO shares (id, workbook_id, permission, created_by, created_at)
		VALUES (?, ?, 'view', ?, ?)
	`, NewTestID(), wb.ID, user.ID, now)
	if err != nil {
		t.Fatalf("Insert share error = %v", err)
	}

	// Delete workbook
	if err := wbStore.Delete(ctx, wb.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify shares deleted
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM shares WHERE workbook_id = ?`, wb.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Count shares error = %v", err)
	}
	if count != 0 {
		t.Errorf("shares count = %d, want 0 (should be cascade deleted)", count)
	}
}

func TestWorkbooksStore_Delete_CascadesVersions(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	wbStore := NewWorkbooksStore(db)
	ctx := context.Background()

	now := FixedTime()
	wb := &workbooks.Workbook{
		ID:        NewTestID(),
		Name:      "Workbook With Versions",
		OwnerID:   user.ID,
		Settings:  workbooks.Settings{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := wbStore.Create(ctx, wb); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Insert a version directly
	_, err := db.ExecContext(ctx, `
		INSERT INTO versions (id, workbook_id, name, created_by, created_at)
		VALUES (?, ?, 'Version 1', ?, ?)
	`, NewTestID(), wb.ID, user.ID, now)
	if err != nil {
		t.Fatalf("Insert version error = %v", err)
	}

	// Delete workbook
	if err := wbStore.Delete(ctx, wb.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify versions deleted
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM versions WHERE workbook_id = ?`, wb.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Count versions error = %v", err)
	}
	if count != 0 {
		t.Errorf("versions count = %d, want 0 (should be cascade deleted)", count)
	}
}
