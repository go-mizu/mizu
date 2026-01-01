package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/drive/store/duckdb"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the database",
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	store, err := duckdb.Open(dataDir)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	if err := store.Ensure(context.Background()); err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}

	fmt.Println("Database initialized successfully")
	return nil
}
