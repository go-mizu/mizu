package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupTestDB creates an in-memory DuckDB database for testing.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	store, err := New(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		t.Fatalf("failed to initialize schema: %v", err)
	}

	return db, func() {
		db.Close()
	}
}

func TestNew(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if store == nil {
		t.Error("New() returned nil store")
	}

	if store.DB() != db {
		t.Error("Store.DB() returned wrong database connection")
	}
}

func TestNew_NilDB(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("New(nil) should return error")
	}
}

func TestEnsure(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// First ensure should succeed
	if err := store.Ensure(context.Background()); err != nil {
		t.Errorf("Ensure() first call error = %v", err)
	}

	// Second ensure should also succeed (idempotent)
	if err := store.Ensure(context.Background()); err != nil {
		t.Errorf("Ensure() second call error = %v", err)
	}

	// Verify tables exist
	tables := []string{"accounts", "posts", "follows", "likes", "reposts", "notifications"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			t.Errorf("Table %s should exist, got error: %v", table, err)
		}
	}
}

func TestStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, _ := New(db)
	stats, err := store.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	// All tables should exist with 0 rows initially
	expectedTables := []string{"accounts", "posts", "follows", "likes", "reposts", "notifications"}
	for _, table := range expectedTables {
		count, ok := stats[table]
		if !ok {
			t.Errorf("Stats missing table: %s", table)
			continue
		}
		if count.(int64) != 0 {
			t.Errorf("Stats[%s] = %v, want 0", table, count)
		}
	}
}

func TestExec(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, _ := New(db)

	// Insert a test account
	_, err := store.Exec(context.Background(),
		"INSERT INTO accounts (id, username, created_at, updated_at) VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		"test-id", "testuser")
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}

	// Verify insert
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&count)
	if err != nil {
		t.Fatalf("query error = %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 account, got %d", count)
	}
}

func TestQuery(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, _ := New(db)

	// Insert test accounts
	for i, username := range []string{"alice", "bob", "charlie"} {
		_, err := store.Exec(context.Background(),
			"INSERT INTO accounts (id, username, created_at, updated_at) VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
			"id-"+string(rune('a'+i)), username)
		if err != nil {
			t.Fatalf("failed to insert account: %v", err)
		}
	}

	// Query accounts
	rows, err := store.Query(context.Background(), "SELECT id, username FROM accounts ORDER BY username")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		count++
	}
	if count != 3 {
		t.Errorf("Query returned %d rows, want 3", count)
	}
}

func TestQueryRow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, _ := New(db)

	// Insert test account
	_, err := store.Exec(context.Background(),
		"INSERT INTO accounts (id, username, created_at, updated_at) VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		"test-id", "testuser")
	if err != nil {
		t.Fatalf("failed to insert account: %v", err)
	}

	// Query single row
	var id, username string
	err = store.QueryRow(context.Background(), "SELECT id, username FROM accounts WHERE id = $1", "test-id").Scan(&id, &username)
	if err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}

	if id != "test-id" {
		t.Errorf("got id = %s, want test-id", id)
	}
	if username != "testuser" {
		t.Errorf("got username = %s, want testuser", username)
	}
}
