package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/chat/store/duckdb"
)

// NewInit creates the init command.
func NewInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the database",
		Long: `Initialize the Chat database schema.

Creates the data directory and database file if they don't exist,
then runs all schema migrations.`,
		RunE: runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconInfo, "Initializing Chat Database")
	ui.Blank()

	// Create data directory
	ui.StartSpinner("Creating data directory...")
	start := time.Now()

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		ui.StopSpinnerError("Failed to create data directory")
		return fmt.Errorf("create data dir: %w", err)
	}

	ui.StopSpinner("Data directory ready", time.Since(start))

	dbPath := filepath.Join(dataDir, "chat.duckdb")

	// Open database
	ui.StartSpinner("Opening database...")
	start = time.Now()

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		ui.StopSpinnerError("Failed to open database")
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	ui.StopSpinner("Database opened", time.Since(start))

	// Create store and ensure schema
	ui.StartSpinner("Running migrations...")
	start = time.Now()

	store, err := duckdb.New(db)
	if err != nil {
		ui.StopSpinnerError("Failed to create store")
		return fmt.Errorf("create store: %w", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		ui.StopSpinnerError("Failed to run migrations")
		return fmt.Errorf("ensure schema: %w", err)
	}

	ui.StopSpinner("Migrations complete", time.Since(start))

	// Summary
	ui.Summary([][2]string{
		{"Data Dir", dataDir},
		{"Database", dbPath},
	})

	ui.Success("Database initialized successfully")
	return nil
}
