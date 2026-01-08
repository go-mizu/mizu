package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
	"github.com/go-mizu/blueprints/spreadsheet/store/duckdb"
	"github.com/go-mizu/blueprints/spreadsheet/store/postgres"
	"github.com/go-mizu/blueprints/spreadsheet/store/sqlite"
	"github.com/go-mizu/blueprints/spreadsheet/store/swandb"
	"github.com/oklog/ulid/v2"
)

// DriverRegistry manages driver setup for benchmarks.
type DriverRegistry struct {
	setupFuncs map[string]func() (*DriverContext, error)
}

// NewDriverRegistry creates a new driver registry.
func NewDriverRegistry() *DriverRegistry {
	r := &DriverRegistry{
		setupFuncs: make(map[string]func() (*DriverContext, error)),
	}

	// Register all drivers
	r.setupFuncs["duckdb"] = setupDuckDB
	r.setupFuncs["sqlite"] = setupSQLite
	r.setupFuncs["swandb"] = setupSwanDB
	r.setupFuncs["postgres"] = setupPostgres

	return r
}

// SetupDriver initializes a driver by name.
func (r *DriverRegistry) SetupDriver(name string) (*DriverContext, error) {
	setupFn, ok := r.setupFuncs[name]
	if !ok {
		return nil, fmt.Errorf("unknown driver: %s", name)
	}
	return setupFn()
}

// setupDuckDB initializes DuckDB driver.
func setupDuckDB() (*DriverContext, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb: %w", err)
	}

	store, err := duckdb.New(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create duckdb store: %w", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ensure duckdb schema: %w", err)
	}

	ctx, err := createFixture(db, "duckdb", duckdb.NewUsersStore(db), duckdb.NewWorkbooksStore(db), duckdb.NewSheetsStore(db))
	if err != nil {
		db.Close()
		return nil, err
	}

	ctx.CellsStore = duckdb.NewCellsStore(db)
	return ctx, nil
}

// setupSQLite initializes SQLite driver.
func setupSQLite() (*DriverContext, error) {
	tmpFile, err := os.CreateTemp("", "storebench-*.db")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFile.Close()

	db, err := sql.Open("sqlite3", tmpFile.Name()+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	store, err := sqlite.New(db)
	if err != nil {
		db.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to create sqlite store: %w", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to ensure sqlite schema: %w", err)
	}

	ctx, err := createFixture(db, "sqlite", sqlite.NewUsersStore(db), sqlite.NewWorkbooksStore(db), sqlite.NewSheetsStore(db))
	if err != nil {
		db.Close()
		os.Remove(tmpFile.Name())
		return nil, err
	}

	ctx.CellsStore = sqlite.NewCellsStore(db)
	return ctx, nil
}

// setupSwanDB initializes SwanDB driver.
func setupSwanDB() (*DriverContext, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb for swandb: %w", err)
	}

	store, err := swandb.New(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create swandb store: %w", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ensure swandb schema: %w", err)
	}

	ctx, err := createFixture(db, "swandb", swandb.NewUsersStore(db), swandb.NewWorkbooksStore(db), swandb.NewSheetsStore(db))
	if err != nil {
		db.Close()
		return nil, err
	}

	ctx.CellsStore = swandb.NewCellsStore(db)
	return ctx, nil
}

// setupPostgres initializes PostgreSQL driver.
func setupPostgres() (*DriverContext, error) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		dsn = "postgres://spreadsheet:spreadsheet@localhost:5432/spreadsheet_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgresql not available: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgresql not reachable: %w", err)
	}

	// Create isolated schema
	schemaName := fmt.Sprintf("bench_%s", ulid.Make().String())
	_, err = db.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaName))
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	_, err = db.Exec(fmt.Sprintf("SET search_path TO %s", schemaName))
	if err != nil {
		db.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName))
		db.Close()
		return nil, fmt.Errorf("failed to set search_path: %w", err)
	}

	store, err := postgres.New(db)
	if err != nil {
		db.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName))
		db.Close()
		return nil, fmt.Errorf("failed to create postgres store: %w", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName))
		db.Close()
		return nil, fmt.Errorf("failed to ensure postgres schema: %w", err)
	}

	ctx, err := createFixture(db, "postgres", postgres.NewUsersStore(db), postgres.NewWorkbooksStore(db), postgres.NewSheetsStore(db))
	if err != nil {
		db.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName))
		db.Close()
		return nil, err
	}

	ctx.CellsStore = postgres.NewCellsStore(db)
	return ctx, nil
}

// createFixture creates the test user, workbook, and sheet.
func createFixture(
	db *sql.DB,
	name string,
	userStore users.Store,
	wbStore workbooks.Store,
	sheetStore sheets.Store,
) (*DriverContext, error) {
	now := time.Now()

	// Create user
	userID := ulid.Make().String()
	user := &users.User{
		ID:        userID,
		Email:     "bench-" + userID + "@example.com",
		Name:      "Bench User",
		Password:  "hashedpassword",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := userStore.Create(context.Background(), user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create workbook
	wb := &workbooks.Workbook{
		ID:      ulid.Make().String(),
		Name:    "Bench Workbook",
		OwnerID: user.ID,
		Settings: workbooks.Settings{
			Locale:          "en-US",
			TimeZone:        "UTC",
			CalculationMode: "auto",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := wbStore.Create(context.Background(), wb); err != nil {
		return nil, fmt.Errorf("failed to create workbook: %w", err)
	}

	// Create initial sheet
	sheet := &sheets.Sheet{
		ID:               ulid.Make().String(),
		WorkbookID:       wb.ID,
		Name:             "Sheet1",
		Index:            0,
		Hidden:           false,
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
	if err := sheetStore.Create(context.Background(), sheet); err != nil {
		return nil, fmt.Errorf("failed to create sheet: %w", err)
	}

	return &DriverContext{
		Name:     name,
		DB:       db,
		User:     user,
		Workbook: wb,
		Sheet:    sheet,
	}, nil
}
