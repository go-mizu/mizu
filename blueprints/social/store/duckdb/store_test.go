package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/oklog/ulid/v2"
)

// setupTestStore creates an in-memory DuckDB store for testing.
func setupTestStore(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open in-memory duckdb: %v", err)
	}

	// Initialize schema
	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		db.Close()
		t.Fatalf("failed to initialize schema: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// newTestID generates a new ULID for testing.
func newTestID() string {
	return ulid.Make().String()
}

// testTime returns a fixed time for testing.
func testTime() time.Time {
	return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

func TestStore_New(t *testing.T) {
	db := setupTestStore(t)

	store, err := New(db)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if store.DB() != db {
		t.Error("DB() should return the underlying database")
	}
}

func TestStore_Ensure(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Ensure should create schema
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure failed: %v", err)
	}

	// Calling Ensure again should be idempotent
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure (second call) failed: %v", err)
	}
}

func TestStore_Tx(t *testing.T) {
	db := setupTestStore(t)
	store, _ := New(db)
	ctx := context.Background()

	// Successful transaction
	err := store.Tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO meta (k, v) VALUES ('test', 'value')")
		return err
	})
	if err != nil {
		t.Fatalf("Tx failed: %v", err)
	}

	// Verify value was inserted
	var v string
	err = db.QueryRow("SELECT v FROM meta WHERE k = 'test'").Scan(&v)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if v != "value" {
		t.Errorf("value: got %q, want %q", v, "value")
	}
}

func TestStore_Tx_Rollback(t *testing.T) {
	db := setupTestStore(t)
	store, _ := New(db)
	ctx := context.Background()

	// Transaction that returns error should rollback
	err := store.Tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO meta (k, v) VALUES ('rollback', 'test')")
		if err != nil {
			return err
		}
		return sql.ErrNoRows // Simulate an error
	})
	if err != sql.ErrNoRows {
		t.Fatalf("expected ErrNoRows, got: %v", err)
	}

	// Verify value was NOT inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM meta WHERE k = 'rollback'").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows after rollback, got %d", count)
	}
}
