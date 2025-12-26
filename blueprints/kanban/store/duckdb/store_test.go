package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}

	store, err := New(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		t.Fatalf("failed to ensure schema: %v", err)
	}

	return store, func() {
		db.Close()
	}
}

func TestStore_New(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if store.DB() != db {
		t.Error("DB() should return the database connection")
	}
}

func TestStore_New_NilDB(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("expected error for nil db")
	}
}

func TestStore_Ensure(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Ensure should be idempotent
	if err := store.Ensure(context.Background()); err != nil {
		t.Errorf("second Ensure failed: %v", err)
	}
}

func TestStore_Stats(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	stats, err := store.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	expectedTables := []string{"users", "workspaces", "teams", "projects", "columns", "cycles", "issues", "comments", "fields", "field_values"}
	for _, table := range expectedTables {
		if _, ok := stats[table]; !ok {
			t.Errorf("expected stats for table %s", table)
		}
	}
}
