package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
	"github.com/spf13/cobra"
)

// NewInit creates the init command
func NewInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the database and create required schemas",
		Long: `Initialize the Localbase database with all required schemas and extensions.

This command will:
  - Connect to PostgreSQL
  - Create required extensions (pgvector, pgcrypto, uuid-ossp)
  - Create auth, storage, functions, and realtime schemas
  - Set up initial database structure`,
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

	// Create storage directory
	storageDir := filepath.Join(GetDataDir(), "storage")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create functions directory
	functionsDir := filepath.Join(GetDataDir(), "functions")
	if err := os.MkdirAll(functionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create functions directory: %w", err)
	}

	// Initialize database
	if err := initDatabase(ctx); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("Database initialized successfully!"))
	fmt.Println()
	fmt.Println(infoStyle.Render("Next steps:"))
	fmt.Println("  1. Run 'localbase seed' to add sample data")
	fmt.Println("  2. Run 'localbase serve' to start the server")
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

	fmt.Println(infoStyle.Render("Creating schemas..."))
	if err := store.Ensure(ctx); err != nil {
		return fmt.Errorf("failed to create schemas: %w", err)
	}
	fmt.Println(successStyle.Render("  Schemas created"))

	return nil
}
