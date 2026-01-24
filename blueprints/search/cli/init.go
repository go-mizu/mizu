package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/go-mizu/mizu/blueprints/search/store/postgres"
	"github.com/spf13/cobra"
)

// NewInit creates the init command
func NewInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the database and create required schemas",
		Long: `Initialize the Search database with all required schemas and extensions.

This command will:
  - Connect to PostgreSQL
  - Create required extensions (pg_trgm, pgcrypto, uuid-ossp)
  - Create search schema with all tables
  - Set up full-text search indexes`,
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
	fmt.Println(infoStyle.Render("Connecting to PostgreSQL..."))

	store, err := postgres.New(ctx, GetDatabaseURL())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer store.Close()

	fmt.Println(successStyle.Render("  Connected"))

	fmt.Println(infoStyle.Render("Creating extensions..."))
	if err := store.CreateExtensions(ctx); err != nil {
		return fmt.Errorf("failed to create extensions: %w", err)
	}
	fmt.Println(successStyle.Render("  Extensions created"))

	fmt.Println(infoStyle.Render("Creating schemas and tables..."))
	if err := store.Ensure(ctx); err != nil {
		return fmt.Errorf("failed to create schemas: %w", err)
	}
	fmt.Println(successStyle.Render("  Schemas created"))

	return nil
}
