package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupTestDB creates an in-memory DuckDB database with schema initialized.
func setupTestDB(t *testing.T) *sql.DB {
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

	t.Cleanup(func() { db.Close() })
	return db
}

// Test time used for reproducible tests
var testTime = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

// assertEqual fails the test if the values are not equal.
func assertEqual[T comparable](t *testing.T, field string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", field, got, want)
	}
}

// assertNil fails the test if the value is not nil.
func assertNil(t *testing.T, field string, got any) {
	t.Helper()
	if got != nil {
		t.Errorf("%s: expected nil, got %v", field, got)
	}
}

// assertNotNil fails the test if the value is nil.
func assertNotNil(t *testing.T, field string, got any) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: expected non-nil value", field)
	}
}

// assertNoError fails the test if err is not nil.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// assertError fails the test if err is nil.
func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// assertLen checks the length of a slice.
func assertLen[T any](t *testing.T, slice []T, expected int) {
	t.Helper()
	if len(slice) != expected {
		t.Errorf("expected length %d, got %d", expected, len(slice))
	}
}
