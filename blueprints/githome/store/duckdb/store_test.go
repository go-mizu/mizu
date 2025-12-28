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

func TestStore_Ensure(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Ensure should be idempotent
	if err := store.Ensure(context.Background()); err != nil {
		t.Errorf("second Ensure failed: %v", err)
	}
}

func TestStore_Ensure_CreatesAllTables(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Verify key tables exist by querying them
	tables := []string{
		"users", "sessions", "repositories", "collaborators", "stars",
		"issues", "issue_labels", "issue_assignees", "labels", "milestones",
	}

	for _, table := range tables {
		var count int
		err := store.DB().QueryRowContext(context.Background(),
			"SELECT COUNT(*) FROM "+table).Scan(&count)
		if err != nil {
			t.Errorf("table %s does not exist or query failed: %v", table, err)
		}
	}
}

