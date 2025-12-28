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

	expectedTables := []string{
		"users", "organizations", "repositories", "teams",
		"issues", "labels", "milestones", "issue_comments",
		"pull_requests", "pr_reviews", "stars", "watches",
		"collaborators", "reactions", "releases", "events",
		"notifications", "webhooks",
	}
	for _, table := range expectedTables {
		if _, ok := stats[table]; !ok {
			t.Errorf("expected stats for table %s", table)
		}
	}
}

func TestGenerateNodeID(t *testing.T) {
	tests := []struct {
		prefix string
		id     int64
	}{
		{"U", 1},
		{"R", 123},
		{"I", 456789},
	}

	for _, tt := range tests {
		nodeID := generateNodeID(tt.prefix, tt.id)
		if nodeID == "" {
			t.Errorf("generateNodeID(%q, %d) returned empty string", tt.prefix, tt.id)
		}
	}
}

func TestPaginationParams(t *testing.T) {
	tests := []struct {
		page, perPage        int
		wantOffset, wantLimit int
	}{
		{1, 30, 0, 30},
		{2, 30, 30, 30},
		{1, 100, 0, 100},
		{1, 150, 0, 100}, // capped at 100
		{0, 0, 0, 30},    // defaults
		{-1, -1, 0, 30},  // negative defaults
	}

	for _, tt := range tests {
		offset, limit := paginationParams(tt.page, tt.perPage)
		if offset != tt.wantOffset || limit != tt.wantLimit {
			t.Errorf("paginationParams(%d, %d) = (%d, %d), want (%d, %d)",
				tt.page, tt.perPage, offset, limit, tt.wantOffset, tt.wantLimit)
		}
	}
}
