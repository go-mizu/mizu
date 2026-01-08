package storetest

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/charts"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
	"github.com/go-mizu/blueprints/spreadsheet/store/sqlite"
)

// SQLiteFactory implements StoreFactory for the SQLite driver.
type SQLiteFactory struct{}

func (f *SQLiteFactory) Name() string {
	return "sqlite"
}

func (f *SQLiteFactory) SetupDB(t *testing.T) *sql.DB {
	t.Helper()

	// Create a temporary file for test isolation
	tmpFile, err := os.CreateTemp("", "spreadsheet-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	// Open SQLite database with foreign keys enabled
	db, err := sql.Open("sqlite3", tmpFile.Name()+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to open sqlite: %v", err)
	}

	store, err := sqlite.New(db)
	if err != nil {
		db.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to initialize schema: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
		os.Remove(tmpFile.Name())
	})

	return db
}

func (f *SQLiteFactory) NewUsersStore(db *sql.DB) users.Store {
	return sqlite.NewUsersStore(db)
}

func (f *SQLiteFactory) NewWorkbooksStore(db *sql.DB) workbooks.Store {
	return sqlite.NewWorkbooksStore(db)
}

func (f *SQLiteFactory) NewSheetsStore(db *sql.DB) sheets.Store {
	return sqlite.NewSheetsStore(db)
}

func (f *SQLiteFactory) NewCellsStore(db *sql.DB) cells.Store {
	return sqlite.NewCellsStore(db)
}

func (f *SQLiteFactory) NewChartsStore(db *sql.DB) charts.Store {
	return sqlite.NewChartsStore(db)
}

func init() {
	RegisterDriver(&SQLiteFactory{})
}
