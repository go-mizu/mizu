// Package storetest provides driver-agnostic tests for store implementations.
package storetest

import (
	"database/sql"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/charts"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
	"github.com/oklog/ulid/v2"
)

// StoreFactory creates store instances for a specific driver.
type StoreFactory interface {
	// Name returns the driver name for test output.
	Name() string

	// SetupDB creates and initializes a test database.
	SetupDB(t *testing.T) *sql.DB

	// NewUsersStore creates a users store.
	NewUsersStore(db *sql.DB) users.Store

	// NewWorkbooksStore creates a workbooks store.
	NewWorkbooksStore(db *sql.DB) workbooks.Store

	// NewSheetsStore creates a sheets store.
	NewSheetsStore(db *sql.DB) sheets.Store

	// NewCellsStore creates a cells store.
	NewCellsStore(db *sql.DB) cells.Store

	// NewChartsStore creates a charts store.
	NewChartsStore(db *sql.DB) charts.Store
}

// TestFixture provides all necessary test data with proper FK relationships.
type TestFixture struct {
	Factory  StoreFactory
	DB       *sql.DB
	User     *users.User
	Workbook *workbooks.Workbook
	Sheet    *sheets.Sheet
}

// NewTestID generates a ULID for testing.
func NewTestID() string {
	return ulid.Make().String()
}

// FixedTime returns a fixed time for deterministic testing.
func FixedTime() time.Time {
	return time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
}

// SetupFixture creates a complete test environment using the given factory.
func SetupFixture(t *testing.T, factory StoreFactory) *TestFixture {
	t.Helper()

	db := factory.SetupDB(t)

	// Create user
	userStore := factory.NewUsersStore(db)
	now := FixedTime()
	userID := NewTestID()
	user := &users.User{
		ID:        userID,
		Email:     "test-" + userID + "@example.com",
		Name:      "Test User",
		Password:  "hashedpassword",
		Avatar:    "",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := userStore.Create(t.Context(), user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create workbook
	wbStore := factory.NewWorkbooksStore(db)
	wb := &workbooks.Workbook{
		ID:      NewTestID(),
		Name:    "Test Workbook",
		OwnerID: user.ID,
		Settings: workbooks.Settings{
			Locale:          "en-US",
			TimeZone:        "UTC",
			CalculationMode: "auto",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := wbStore.Create(t.Context(), wb); err != nil {
		t.Fatalf("failed to create test workbook: %v", err)
	}

	// Create sheet
	sheetStore := factory.NewSheetsStore(db)
	sheet := &sheets.Sheet{
		ID:               NewTestID(),
		WorkbookID:       wb.ID,
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
	if err := sheetStore.Create(t.Context(), sheet); err != nil {
		t.Fatalf("failed to create test sheet: %v", err)
	}

	return &TestFixture{
		Factory:  factory,
		DB:       db,
		User:     user,
		Workbook: wb,
		Sheet:    sheet,
	}
}

// CreateTestCell creates a cell for testing.
func (f *TestFixture) CreateTestCell(t *testing.T, sheetID string, row, col int) *cells.Cell {
	t.Helper()

	store := f.Factory.NewCellsStore(f.DB)
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

	if err := store.Set(t.Context(), cell); err != nil {
		t.Fatalf("failed to create test cell: %v", err)
	}

	return cell
}

// CreateTestSheet creates an additional sheet for testing.
func (f *TestFixture) CreateTestSheet(t *testing.T, workbookID string) *sheets.Sheet {
	t.Helper()

	store := f.Factory.NewSheetsStore(f.DB)
	now := FixedTime()
	sheet := &sheets.Sheet{
		ID:               NewTestID(),
		WorkbookID:       workbookID,
		Name:             "Sheet" + NewTestID()[:8],
		Index:            1,
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

	if err := store.Create(t.Context(), sheet); err != nil {
		t.Fatalf("failed to create test sheet: %v", err)
	}

	return sheet
}

// Drivers holds the registered store factories for testing.
var Drivers []StoreFactory

// RegisterDriver adds a store factory for testing.
func RegisterDriver(factory StoreFactory) {
	Drivers = append(Drivers, factory)
}

// RunForAllDrivers runs a test function for all registered drivers.
func RunForAllDrivers(t *testing.T, name string, testFn func(t *testing.T, factory StoreFactory)) {
	for _, factory := range Drivers {
		t.Run(factory.Name()+"/"+name, func(t *testing.T) {
			testFn(t, factory)
		})
	}
}
