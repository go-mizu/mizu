package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/go-mizu/mizu/blueprints/search/store/sqlite"
	"github.com/spf13/cobra"
)

// NewInit creates the init command
func NewInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the SQLite database",
		Long: `Initialize the Search SQLite database with all required tables.

This command will:
  - Create the SQLite database file
  - Create all required tables
  - Set up FTS5 full-text search indexes`,
		RunE: runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Initializing database..."))
	fmt.Println()

	// Create data directory
	if err := os.MkdirAll(GetDataDir(), 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize database
	if err := initDatabase(ctx); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("Database initialized successfully!"))
	fmt.Println()
	fmt.Println(infoStyle.Render("Next steps:"))
	fmt.Println("  1. Run 'search seed' to add sample data")
	fmt.Println("  2. Run 'search serve' to start the server")
	fmt.Println()

	return nil
}

func initDatabase(ctx context.Context) error {
	fmt.Println(infoStyle.Render("Opening SQLite database..."))

	store, err := sqlite.New(GetDatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	fmt.Println(successStyle.Render("  Database opened"))

	fmt.Println(infoStyle.Render("Creating tables and indexes..."))
	if err := store.Ensure(ctx); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	fmt.Println(successStyle.Render("  Tables created"))

	return nil
}
