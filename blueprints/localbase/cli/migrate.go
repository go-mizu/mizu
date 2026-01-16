package cli

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
	"github.com/spf13/cobra"
)

// NewMigrate creates the migrate command
func NewMigrate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		Long: `Run database migrations to update the schema.

This command applies any pending migrations to the database.`,
		RunE: runMigrate,
	}

	return cmd
}

func runMigrate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Running migrations..."))
	fmt.Println()

	if err := runMigrations(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("Migrations completed successfully!"))
	fmt.Println()

	return nil
}

func runMigrations(ctx context.Context) error {
	fmt.Println(infoStyle.Render("Connecting to PostgreSQL..."))

	store, err := postgres.New(ctx, GetDatabaseURL())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer store.Close()

	fmt.Println(successStyle.Render("  Connected"))

	fmt.Println(infoStyle.Render("Applying migrations..."))
	if err := store.Ensure(ctx); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	fmt.Println(successStyle.Render("  Migrations applied"))

	return nil
}
