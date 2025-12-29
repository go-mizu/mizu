package cli

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-mizu/blueprints/githome/store/duckdb"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the database",
		Long: `Initialize the GitHome database and create required directories.

This command creates:
  - Data directory for the database
  - Repos directory for git repositories
  - DuckDB database with the required schema

Safe to run multiple times - existing data is preserved.`,
		Example: `  githome init
  githome init --data-dir /var/lib/githome`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create data directory
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return fmt.Errorf("create data dir: %w", err)
			}

			// Create repos directory
			if err := os.MkdirAll(reposDir, 0755); err != nil {
				return fmt.Errorf("create repos dir: %w", err)
			}

			dbPath := dataDir + "/githome.db"
			db, err := sql.Open("duckdb", dbPath)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			store, err := duckdb.New(db)
			if err != nil {
				return fmt.Errorf("create store: %w", err)
			}

			ctx := context.Background()
			if err := store.Ensure(ctx); err != nil {
				return fmt.Errorf("ensure schema: %w", err)
			}

			slog.Info("database initialized", "path", dbPath)
			return nil
		},
	}
}
