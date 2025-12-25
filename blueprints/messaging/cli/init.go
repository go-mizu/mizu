package cli

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/messaging/store/duckdb"
)

// NewInit creates the init command.
func NewInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the database",
		Long:  `Creates the database and schema for the messaging application.`,
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconDatabase, "Initializing Database")
	ui.Blank()

	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		ui.Error("Failed to create data directory")
		return err
	}

	ui.StartSpinner("Creating database...")
	start := time.Now()

	dbPath := filepath.Join(dataDir, "messaging.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		ui.StopSpinnerError("Failed to open database")
		return err
	}
	defer db.Close()

	store, err := duckdb.New(db)
	if err != nil {
		ui.StopSpinnerError("Failed to create store")
		return err
	}

	if err := store.Ensure(context.Background()); err != nil {
		ui.StopSpinnerError("Failed to create schema")
		return err
	}

	ui.StopSpinner("Database initialized", time.Since(start))

	ui.Summary([][2]string{
		{"Database", dbPath},
		{"Status", "Ready"},
	})

	ui.Blank()
	ui.Success("Database initialized successfully!")

	return nil
}
