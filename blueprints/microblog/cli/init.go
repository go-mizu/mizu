package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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
			// Ensure data directory exists
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return fmt.Errorf("create data dir: %w", err)
			}

			// Open database
			dbPath := filepath.Join(dataDir, "microblog.duckdb")
			db, err := sql.Open("duckdb", dbPath)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			// Create store and initialize schema
			store, err := duckdb.New(db)
			if err != nil {
				return fmt.Errorf("create store: %w", err)
			}

			if err := store.Ensure(context.Background()); err != nil {
				return fmt.Errorf("initialize schema: %w", err)
			}

			fmt.Printf("Database initialized at: %s\n", dbPath)
			return nil
		},
	}

	return cmd
}
