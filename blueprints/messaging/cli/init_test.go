package cli

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/messaging/feature/accounts"
	"github.com/go-mizu/blueprints/messaging/store/duckdb"
)

func TestRunInit(t *testing.T) {
	t.Run("creates database and schema", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Set global flag
		oldDataDir := dataDir
		dataDir = tmpDir
		defer func() { dataDir = oldDataDir }()

		// Run init
		err := runInit(nil, nil)
		if err != nil {
			t.Fatalf("runInit() returned error: %v", err)
		}

		// Verify database file exists
		dbPath := filepath.Join(tmpDir, "messaging.duckdb")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("database file was not created")
		}

		// Verify schema was applied by opening and checking tables
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer db.Close()

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
		tmpDir := t.TempDir()

		oldDataDir := dataDir
		dataDir = tmpDir
		defer func() { dataDir = oldDataDir }()

		// Run init twice
		if err := runInit(nil, nil); err != nil {
			t.Fatalf("first runInit() returned error: %v", err)
		}

		if err := runInit(nil, nil); err != nil {
			t.Fatalf("second runInit() returned error: %v", err)
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		nestedDir := filepath.Join(tmpDir, "nested", "deep", "data")

		oldDataDir := dataDir
		dataDir = nestedDir
		defer func() { dataDir = oldDataDir }()

		if err := runInit(nil, nil); err != nil {
			t.Fatalf("runInit() returned error: %v", err)
		}

		if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
			t.Error("nested directory was not created")
		}
	})
}

func TestInitCommand(t *testing.T) {
	cmd := NewInit()

	if cmd.Use != "init" {
		t.Errorf("expected Use to be 'init', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestInitIntegration(t *testing.T) {
	// This test verifies the full integration with the store package
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.duckdb")

	// Create database
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create store and ensure schema
	store, err := duckdb.New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	// Verify we can use accounts service to insert and query data
	usersStore := duckdb.NewUsersStore(db)
	accountsSvc := accounts.NewService(usersStore)

	// Create a user using the service
	user, err := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username:    "testuser",
		Email:       "test@example.com",
		Password:    "password123",
		DisplayName: "Test User",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Query the user
	foundUser, err := accountsSvc.GetByUsername(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}

	if foundUser.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", foundUser.Username)
	}

	if foundUser.ID != user.ID {
		t.Errorf("expected ID '%s', got '%s'", user.ID, foundUser.ID)
	}
}
