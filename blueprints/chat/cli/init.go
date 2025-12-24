package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/chat/store/duckdb"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the database",
	Long:  `Initialize the Chat database schema.`,
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	ui := NewUI()
	ui.Header("Chat", Version)
	ui.Info("Initializing database...")

	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "chat.duckdb")
	ui.Item("Database", dbPath)

	// Open database
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Create store
	store, err := duckdb.New(db)
	if err != nil {
		return fmt.Errorf("create store: %w", err)
	}

	// Ensure schema
	if err := store.Ensure(context.Background()); err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}

	ui.Success("Database initialized")
	return nil
}
