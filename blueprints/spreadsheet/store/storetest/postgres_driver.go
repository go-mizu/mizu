package storetest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/charts"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
	"github.com/go-mizu/blueprints/spreadsheet/store/postgres"
)

// PostgresFactory implements StoreFactory for the PostgreSQL driver.
type PostgresFactory struct{}

func (f *PostgresFactory) Name() string {
	return "postgres"
}

func (f *PostgresFactory) SetupDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		dsn = "postgres://spreadsheet:spreadsheet@localhost:5432/spreadsheet_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("PostgreSQL not reachable: %v", err)
	}

	// Create isolated test schema for this test
	schemaName := fmt.Sprintf("test_%s", ulid.Make().String())
	_, err = db.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaName))
	if err != nil {
		db.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}

	// Set search_path to the test schema
	_, err = db.Exec(fmt.Sprintf("SET search_path TO %s", schemaName))
	if err != nil {
		db.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName))
		db.Close()
		t.Fatalf("failed to set search_path: %v", err)
	}

	// Initialize the store with schema
	store, err := postgres.New(db)
	if err != nil {
		db.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName))
		db.Close()
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName))
		db.Close()
		t.Fatalf("failed to initialize schema: %v", err)
	}

	t.Cleanup(func() {
		// Drop the test schema
		db.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName))
		db.Close()
	})

	return db
}

func (f *PostgresFactory) NewUsersStore(db *sql.DB) users.Store {
	return postgres.NewUsersStore(db)
}

func (f *PostgresFactory) NewWorkbooksStore(db *sql.DB) workbooks.Store {
	return postgres.NewWorkbooksStore(db)
}

func (f *PostgresFactory) NewSheetsStore(db *sql.DB) sheets.Store {
	return postgres.NewSheetsStore(db)
}

func (f *PostgresFactory) NewCellsStore(db *sql.DB) cells.Store {
	return postgres.NewCellsStore(db)
}

func (f *PostgresFactory) NewChartsStore(db *sql.DB) charts.Store {
	return postgres.NewChartsStore(db)
}

func init() {
	RegisterDriver(&PostgresFactory{})
}
