package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

// NewInit creates the init command.
func NewInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the database",
		Long:  `Initialize the database and create all required tables.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ui := NewUI()
			start := time.Now()

			ui.Header(iconDatabase, "Initializing Database")
			ui.Info("Data directory", dataDir)
			ui.Blank()

			// Create directory
			ui.StartSpinner("Creating data directory...")
			dirStart := time.Now()
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				ui.StopSpinnerError("Failed to create directory")
				return fmt.Errorf("create data dir: %w", err)
			}
			ui.StopSpinner("Directory ready", time.Since(dirStart))

			// Open database
			dbPath := filepath.Join(dataDir, "microblog.duckdb")
			ui.StartSpinner("Opening database...")
			dbStart := time.Now()
			db, err := sql.Open("duckdb", dbPath)
			if err != nil {
				ui.StopSpinnerError("Failed to open database")
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()
			ui.StopSpinner("Database opened", time.Since(dbStart))

			// Create store
			store, err := duckdb.New(db)
			if err != nil {
				ui.Error("Failed to create store")
				return fmt.Errorf("create store: %w", err)
			}

			// Initialize schema
			ui.StartSpinner("Creating tables and indexes...")
			schemaStart := time.Now()
			if err := store.Ensure(context.Background()); err != nil {
				ui.StopSpinnerError("Failed to initialize schema")
				return fmt.Errorf("initialize schema: %w", err)
			}
			ui.StopSpinner("Schema initialized", time.Since(schemaStart))

			// Get stats
			stats, err := store.Stats(context.Background())
			if err == nil {
				ui.Blank()
				ui.Info("Tables created", fmt.Sprintf("%d", len(stats)))
			}

			ui.Success("Database initialized successfully")
			ui.Hint(fmt.Sprintf("Location: %s", dbPath))
			ui.Hint(fmt.Sprintf("Total time: %s", time.Since(start).Round(time.Millisecond)))

			return nil
		},
	}

	return cmd
}
