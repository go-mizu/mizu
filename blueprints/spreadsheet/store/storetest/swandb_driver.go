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
	"github.com/go-mizu/blueprints/spreadsheet/store/swandb"
)

// SwanDBFactory implements StoreFactory for the swandb driver.
type SwanDBFactory struct{}

func (f *SwanDBFactory) Name() string {
	return "swandb"
}

func (f *SwanDBFactory) SetupDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory duckdb: %v", err)
	}

	store, err := swandb.New(db)
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

func (f *SwanDBFactory) NewUsersStore(db *sql.DB) users.Store {
	return swandb.NewUsersStore(db)
}

func (f *SwanDBFactory) NewWorkbooksStore(db *sql.DB) workbooks.Store {
	return swandb.NewWorkbooksStore(db)
}

func (f *SwanDBFactory) NewSheetsStore(db *sql.DB) sheets.Store {
	return swandb.NewSheetsStore(db)
}

func (f *SwanDBFactory) NewCellsStore(db *sql.DB) cells.Store {
	return swandb.NewCellsStore(db)
}

func (f *SwanDBFactory) NewChartsStore(db *sql.DB) charts.Store {
	return swandb.NewChartsStore(db)
}

func init() {
	RegisterDriver(&SwanDBFactory{})
}
