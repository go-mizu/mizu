package cli

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/email/store/sqlite"
	"github.com/spf13/cobra"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long: `Seed the database with sample data for development and testing.

This creates:
  - System labels (Inbox, Starred, Important, Sent, Drafts, All Mail, Spam, Trash)
  - Custom labels (Work, Personal, Finance, Travel)
  - Sample contacts with realistic names and emails
  - Realistic email threads with conversations`,
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

	fmt.Println(infoStyle.Render("Creating labels..."))
	if err := store.SeedLabels(ctx); err != nil {
		return fmt.Errorf("failed to seed labels: %w", err)
	}
	fmt.Println(successStyle.Render("  Labels created"))

	fmt.Println(infoStyle.Render("Creating contacts..."))
	if err := store.SeedContacts(ctx); err != nil {
		return fmt.Errorf("failed to seed contacts: %w", err)
	}
	fmt.Println(successStyle.Render("  Contacts created"))

	fmt.Println(infoStyle.Render("Creating emails..."))
	if err := store.SeedEmails(ctx); err != nil {
		return fmt.Errorf("failed to seed emails: %w", err)
	}
	fmt.Println(successStyle.Render("  Emails created"))

	return nil
}
