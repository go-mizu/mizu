package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
	"github.com/spf13/cobra"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long: `Seed the database with sample data for development and testing.

This creates:
  - Sample users
  - Sample storage buckets and files
  - Sample tables with data
  - Sample logs across all sources (edge, auth, storage, postgres, etc.)`,
		RunE: runSeed,
	}

	return cmd
}

func runSeed(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Seeding database..."))
	fmt.Println()

	if err := seedDatabase(ctx); err != nil {
		return fmt.Errorf("failed to seed database: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("Database seeded successfully!"))
	fmt.Println()

	return nil
}

func seedDatabase(ctx context.Context) error {
	fmt.Println(infoStyle.Render("Connecting to PostgreSQL..."))

	store, err := postgres.New(ctx, GetDatabaseURL())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer store.Close()

	fmt.Println(successStyle.Render("  Connected"))

	// Ensure all schemas and tables exist before seeding
	fmt.Println(infoStyle.Render("Ensuring schemas exist..."))
	if err := store.Ensure(ctx); err != nil {
		return fmt.Errorf("failed to ensure schemas: %w", err)
	}
	fmt.Println(successStyle.Render("  Schemas ready"))

	fmt.Println(infoStyle.Render("Creating sample users..."))
	if err := store.SeedUsers(ctx); err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}
	fmt.Println(successStyle.Render("  Users created"))

	fmt.Println(infoStyle.Render("Creating sample storage buckets..."))
	if err := store.SeedStorage(ctx); err != nil {
		return fmt.Errorf("failed to seed storage: %w", err)
	}
	fmt.Println(successStyle.Render("  Buckets created"))

	// Seed storage files to filesystem
	fmt.Println(infoStyle.Render("Creating sample storage files..."))
	dataDir := os.Getenv("LOCALBASE_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data/storage"
	}
	if err := SeedStorageFiles(ctx, store, dataDir); err != nil {
		return fmt.Errorf("failed to seed storage files: %w", err)
	}
	fmt.Println(successStyle.Render("  Files created"))

	fmt.Println(infoStyle.Render("Creating sample tables..."))
	if err := store.SeedTables(ctx); err != nil {
		return fmt.Errorf("failed to seed tables: %w", err)
	}
	fmt.Println(successStyle.Render("  Tables created"))

	fmt.Println(infoStyle.Render("Generating sample logs..."))
	if err := store.SeedLogs(ctx); err != nil {
		return fmt.Errorf("failed to seed logs: %w", err)
	}
	fmt.Println(successStyle.Render("  Logs generated"))

	return nil
}
