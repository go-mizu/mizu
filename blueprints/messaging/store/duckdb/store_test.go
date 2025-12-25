package duckdb

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// testDB creates a temporary in-memory DuckDB database for testing.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// testStore creates a test store with the schema applied.
func testStore(t *testing.T) *Store {
	t.Helper()
	db := testDB(t)
	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}
	return store
}

func TestOpen(t *testing.T) {
	t.Run("in-memory", func(t *testing.T) {
		db, err := Open(":memory:")
		if err != nil {
			t.Fatalf("failed to open in-memory database: %v", err)
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			t.Fatalf("failed to ping database: %v", err)
		}
	})

	t.Run("file-based", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("failed to open file-based database: %v", err)
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			t.Fatalf("failed to ping database: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Fatal("database file was not created")
		}
	})
}

func TestNew(t *testing.T) {
	db := testDB(t)

	store, err := New(db)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if store == nil {
		t.Fatal("New() returned nil store")
	}

	if store.db != db {
		t.Fatal("store.db does not match provided db")
	}
}

func TestStore_Ensure(t *testing.T) {
	db := testDB(t)
	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	t.Run("creates schema", func(t *testing.T) {
		if err := store.Ensure(context.Background()); err != nil {
			t.Fatalf("Ensure() returned error: %v", err)
		}

		// Verify tables were created
		tables := []string{"users", "sessions", "chats", "chat_participants", "messages"}
		for _, table := range tables {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", table).Scan(&count)
			if err != nil {
				t.Fatalf("failed to check table %s: %v", table, err)
			}
			if count == 0 {
				t.Errorf("table %s was not created", table)
			}
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		// Running Ensure multiple times should not error
		if err := store.Ensure(context.Background()); err != nil {
			t.Fatalf("second Ensure() returned error: %v", err)
		}
	})
}

func TestStore_DB(t *testing.T) {
	db := testDB(t)
	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if store.DB() != db {
		t.Fatal("DB() did not return the underlying database connection")
	}
}

func TestStore_Close(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	// Verify the database is closed
	if err := db.Ping(); err == nil {
		t.Fatal("database should be closed after Close()")
	}
}

func TestSchema(t *testing.T) {
	if schema == "" {
		t.Fatal("embedded schema is empty")
	}

	// Schema should contain key table definitions
	keywords := []string{
		"CREATE TABLE",
		"users",
		"sessions",
		"chats",
		"chat_participants",
		"messages",
		"message_media",
		"message_reactions",
	}

	for _, kw := range keywords {
		if !containsString(schema, kw) {
			t.Errorf("schema does not contain expected keyword: %s", kw)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsStringInner(s, substr)))
}

func containsStringInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
