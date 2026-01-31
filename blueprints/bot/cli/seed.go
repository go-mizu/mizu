package cli

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/bot/store/sqlite"
	"github.com/spf13/cobra"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long: `Seed the database with sample data for development and testing.

This creates:
  - Sample AI agents (main, coder, writer)
  - Sample channels (Telegram, Discord, Mattermost, Webhook)
  - Routing bindings (channel-to-agent rules)
  - Sample sessions with conversation history`,
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
	fmt.Println(infoStyle.Render("Opening SQLite database..."))

	store, err := sqlite.New(GetDatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	fmt.Println(successStyle.Render("  Database opened"))

	// Ensure all tables exist before seeding
	fmt.Println(infoStyle.Render("Ensuring tables exist..."))
	if err := store.Ensure(ctx); err != nil {
		return fmt.Errorf("failed to ensure tables: %w", err)
	}
	fmt.Println(successStyle.Render("  Tables ready"))

	fmt.Println(infoStyle.Render("Seeding data..."))
	if err := store.SeedData(ctx); err != nil {
		return fmt.Errorf("failed to seed data: %w", err)
	}
	fmt.Println(successStyle.Render("  Data seeded"))

	return nil
}
