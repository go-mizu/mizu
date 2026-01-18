package cli

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/lingo/store/postgres"
	"github.com/spf13/cobra"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long: `Seed the database with sample languages, courses, and exercises:
  - Languages: English, Spanish, French, German, Japanese, etc.
  - Sample courses with units, skills, and lessons
  - Exercise content for learning
  - Sample users and progress data
  - Achievements and league data`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeed(cmd.Context())
		},
	}
	return cmd
}

func runSeed(ctx context.Context) error {
	fmt.Println(Banner())
	fmt.Println(infoStyle.Render("Seeding database..."))

	// Connect to database
	fmt.Println(infoStyle.Render("Connecting to PostgreSQL..."))
	store, err := postgres.New(ctx, GetDatabaseURL())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer store.Close()
	fmt.Println(successStyle.Render("  Connected"))

	// Ensure schemas exist
	fmt.Println(infoStyle.Render("Ensuring schemas..."))
	if err := store.Ensure(ctx); err != nil {
		return fmt.Errorf("failed to ensure schemas: %w", err)
	}
	fmt.Println(successStyle.Render("  Schemas ready"))

	// Seed languages
	fmt.Println(infoStyle.Render("Seeding languages..."))
	if err := store.SeedLanguages(ctx); err != nil {
		return fmt.Errorf("failed to seed languages: %w", err)
	}
	fmt.Println(successStyle.Render("  8 languages created"))

	// Seed courses
	fmt.Println(infoStyle.Render("Seeding courses..."))
	if err := store.SeedCourses(ctx); err != nil {
		return fmt.Errorf("failed to seed courses: %w", err)
	}
	fmt.Println(successStyle.Render("  4 courses created"))

	// Seed achievements
	fmt.Println(infoStyle.Render("Seeding achievements..."))
	if err := store.SeedAchievements(ctx); err != nil {
		return fmt.Errorf("failed to seed achievements: %w", err)
	}
	fmt.Println(successStyle.Render("  20+ achievements created"))

	// Seed leagues
	fmt.Println(infoStyle.Render("Seeding leagues..."))
	if err := store.SeedLeagues(ctx); err != nil {
		return fmt.Errorf("failed to seed leagues: %w", err)
	}
	fmt.Println(successStyle.Render("  10 leagues created"))

	// Seed sample users
	fmt.Println(infoStyle.Render("Seeding sample users..."))
	if err := store.SeedUsers(ctx); err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}
	fmt.Println(successStyle.Render("  Sample users created"))

	fmt.Println()
	fmt.Println(successStyle.Render("Database seeded successfully!"))
	fmt.Println()
	fmt.Println(subtitleStyle.Render("Sample accounts:"))
	fmt.Println(subtitleStyle.Render("  Email: demo@lingo.dev"))
	fmt.Println(subtitleStyle.Render("  Password: password123"))
	fmt.Println()
	fmt.Println(subtitleStyle.Render("Next step:"))
	fmt.Println(subtitleStyle.Render("  lingo serve   - Start the server"))
	fmt.Println()

	return nil
}
