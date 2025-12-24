package duckdb

import (
	"context"
	"database/sql"
	"testing"
)

func TestNew(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if store == nil {
		t.Fatal("New() returned nil store")
	}

	if store.DB() != db {
		t.Error("DB() returned different database")
	}
}

func TestEnsure(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// First ensure should create schema
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	// Verify tables exist by querying
	tables := []string{"users", "servers", "channels", "messages", "members", "roles", "presence"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			t.Errorf("table %s not created: %v", table, err)
		}
	}

	// Second ensure should be idempotent
	if err := store.Ensure(context.Background()); err != nil {
		t.Errorf("second Ensure() error = %v", err)
	}
}

func TestDB(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := store.DB(); got != db {
		t.Errorf("DB() = %v, want %v", got, db)
	}
}

func TestClose(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	store, err := New(db)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := store.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify db is closed by trying a query
	_, err = db.Query("SELECT 1")
	if err == nil {
		t.Error("expected error after Close()")
	}
}

func TestOpen(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	// Verify connection works
	var result int
	if err := db.QueryRow("SELECT 1").Scan(&result); err != nil {
		t.Errorf("query error: %v", err)
	}
	if result != 1 {
		t.Errorf("expected 1, got %d", result)
	}
}
