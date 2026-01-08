package storetest

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/charts"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
	"github.com/go-mizu/blueprints/spreadsheet/store/duckdb"
)

// DuckDBFactory implements StoreFactory for the duckdb driver.
type DuckDBFactory struct{}

func (f *DuckDBFactory) Name() string {
	return "duckdb"
}

func (f *DuckDBFactory) SetupDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory duckdb: %v", err)
	}

	store, err := duckdb.New(db)
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

func (f *DuckDBFactory) NewUsersStore(db *sql.DB) users.Store {
	return duckdb.NewUsersStore(db)
}

func (f *DuckDBFactory) NewWorkbooksStore(db *sql.DB) workbooks.Store {
	return duckdb.NewWorkbooksStore(db)
}

func (f *DuckDBFactory) NewSheetsStore(db *sql.DB) sheets.Store {
	return duckdb.NewSheetsStore(db)
}

func (f *DuckDBFactory) NewCellsStore(db *sql.DB) cells.Store {
	return duckdb.NewCellsStore(db)
}

func (f *DuckDBFactory) NewChartsStore(db *sql.DB) charts.Store {
	return duckdb.NewChartsStore(db)
}

func init() {
	RegisterDriver(&DuckDBFactory{})
}
