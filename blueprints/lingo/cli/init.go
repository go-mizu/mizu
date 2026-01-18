package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-mizu/mizu/blueprints/lingo/store/postgres"
	"github.com/spf13/cobra"
)

// NewInit creates the init command
func NewInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the Lingo database",
		Long: `Initialize the database with required schemas and tables:
  - Users and authentication
  - Languages and courses
  - Units, skills, lessons, exercises
  - Progress tracking (XP, streaks, hearts)
  - Achievements and leagues
  - Social features (friends, quests)
  - Stories`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd.Context())
		},
	}
	return cmd
}

func runInit(ctx context.Context) error {
	fmt.Println(Banner())
	fmt.Println(infoStyle.Render("Initializing Lingo..."))

	// Create data directories
	storageDir := filepath.Join(GetDataDir(), "audio")
	imagesDir := filepath.Join(GetDataDir(), "images")

	fmt.Println(infoStyle.Render("Creating directories..."))
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create audio directory: %w", err)
	}
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return fmt.Errorf("failed to create images directory: %w", err)
	}
	fmt.Println(successStyle.Render("  Created data directories"))

	// Connect to database
	fmt.Println(infoStyle.Render("Connecting to PostgreSQL..."))
	store, err := postgres.New(ctx, GetDatabaseURL())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer store.Close()
	fmt.Println(successStyle.Render("  Connected"))

	// Create extensions
	fmt.Println(infoStyle.Render("Creating extensions..."))
	if err := store.CreateExtensions(ctx); err != nil {
		return fmt.Errorf("failed to create extensions: %w", err)
	}
	fmt.Println(successStyle.Render("  Extensions created"))

	// Create schemas and tables
	fmt.Println(infoStyle.Render("Creating schemas and tables..."))
	if err := store.Ensure(ctx); err != nil {
		return fmt.Errorf("failed to create schemas: %w", err)
	}
	fmt.Println(successStyle.Render("  Schemas created"))

	fmt.Println()
	fmt.Println(successStyle.Render("Lingo initialized successfully!"))
	fmt.Println()
	fmt.Println(subtitleStyle.Render("Next steps:"))
	fmt.Println(subtitleStyle.Render("  lingo seed    - Seed with sample data"))
	fmt.Println(subtitleStyle.Render("  lingo serve   - Start the server"))
	fmt.Println()

	return nil
}
