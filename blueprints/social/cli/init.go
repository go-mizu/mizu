package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/social/store/duckdb"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the database",
	Long:  `Initialize the Social database with the required schema.`,
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	ui := NewUI()
	ui.Header("Social", Version)

	ui.Info("Initializing database...")

	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	ui.Item("Data", dataDir)

	// Open database
	dbPath := filepath.Join(dataDir, "social.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Create store and ensure schema
	store, err := duckdb.New(db)
	if err != nil {
		return fmt.Errorf("create store: %w", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}

	ui.Success("Database initialized")
	ui.Item("Path", dbPath)

	return nil
}
