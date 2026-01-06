package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
	"github.com/oklog/ulid/v2"
)

// SetupTestDB creates an in-memory DuckDB instance with schema initialized.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory duckdb: %v", err)
	}

	store, err := New(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		t.Fatalf("failed to initialize schema: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// NewTestID generates a deterministic ULID for testing.
func NewTestID() string {
	return ulid.Make().String()
}

// FixedTime returns a fixed time for deterministic testing.
func FixedTime() time.Time {
	return time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
}

// CreateTestUser creates a user for foreign key requirements.
func CreateTestUser(t *testing.T, db *sql.DB) *users.User {
	t.Helper()

	store := NewUsersStore(db)
	now := FixedTime()
	userID := NewTestID()
	user := &users.User{
		ID:        userID,
		Email:     "test-" + userID + "@example.com", // Use full ULID for uniqueness
		Name:      "Test User",
		Password:  "hashedpassword",
		Avatar:    "",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

// CreateTestWorkbook creates a workbook for foreign key requirements.
func CreateTestWorkbook(t *testing.T, db *sql.DB, ownerID string) *workbooks.Workbook {
	t.Helper()

	store := NewWorkbooksStore(db)
	now := FixedTime()
	wb := &workbooks.Workbook{
		ID:      NewTestID(),
		Name:    "Test Workbook",
		OwnerID: ownerID,
		Settings: workbooks.Settings{
			Locale:          "en-US",
			TimeZone:        "UTC",
			CalculationMode: "auto",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.Create(context.Background(), wb); err != nil {
		t.Fatalf("failed to create test workbook: %v", err)
	}

	return wb
}

// CreateTestSheet creates a sheet for foreign key requirements.
func CreateTestSheet(t *testing.T, db *sql.DB, workbookID string) *sheets.Sheet {
	t.Helper()

	store := NewSheetsStore(db)
	now := FixedTime()
	sheet := &sheets.Sheet{
		ID:               NewTestID(),
		WorkbookID:       workbookID,
		Name:             "Sheet1",
		Index:            0,
		Hidden:           false,
		Color:            "",
		GridColor:        "#E2E8F0",
		FrozenRows:       0,
		FrozenCols:       0,
		DefaultRowHeight: 21,
		DefaultColWidth:  100,
		RowHeights:       make(map[int]int),
		ColWidths:        make(map[int]int),
		HiddenRows:       []int{},
		HiddenCols:       []int{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := store.Create(context.Background(), sheet); err != nil {
		t.Fatalf("failed to create test sheet: %v", err)
	}

	return sheet
}

// CreateTestCell creates a cell for testing.
func CreateTestCell(t *testing.T, db *sql.DB, sheetID string, row, col int) *cells.Cell {
	t.Helper()

	store := NewCellsStore(db)
	now := FixedTime()
	cell := &cells.Cell{
		ID:        NewTestID(),
		SheetID:   sheetID,
		Row:       row,
		Col:       col,
		Value:     "test value",
		Formula:   "",
		Display:   "test value",
		Type:      cells.CellTypeText,
		Format:    cells.Format{},
		Hyperlink: nil,
		Note:      "",
		UpdatedAt: now,
	}

	if err := store.Set(context.Background(), cell); err != nil {
		t.Fatalf("failed to create test cell: %v", err)
	}

	return cell
}

// TestFixture provides all necessary test data with proper FK relationships.
type TestFixture struct {
	DB       *sql.DB
	User     *users.User
	Workbook *workbooks.Workbook
	Sheet    *sheets.Sheet
}

// SetupTestFixture creates a complete test environment.
func SetupTestFixture(t *testing.T) *TestFixture {
	t.Helper()

	db := SetupTestDB(t)
	user := CreateTestUser(t, db)
	workbook := CreateTestWorkbook(t, db, user.ID)
	sheet := CreateTestSheet(t, db, workbook.ID)

	return &TestFixture{
		DB:       db,
		User:     user,
		Workbook: workbook,
		Sheet:    sheet,
	}
}
