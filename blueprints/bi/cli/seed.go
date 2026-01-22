package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/bi/app/web"
	"github.com/go-mizu/blueprints/bi/pkg/seed"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long: `Seed the BI database with sample data for testing.

This creates realistic data including:

  Data Sources:
    - Sample SQLite database with demo tables
    - orders, products, customers, analytics tables

  Questions:
    - Revenue by month (line chart)
    - Top products by sales (bar chart)
    - Customer distribution (pie chart)
    - Orders table
    - Key metrics

  Dashboards:
    - Sales Overview dashboard
    - Marketing Analytics dashboard

  Collections:
    - Sales
    - Marketing
    - Operations

  Users:
    - admin@example.com (password: admin)

Examples:
  bi seed                     # Seed with sample data
  bi seed --data /path/to/dir # Seed specific database`,
		RunE: runSeed,
	}

	return cmd
}

func runSeed(cmd *cobra.Command, args []string) error {
	Blank()
	Header("", "Seed Database")
	Blank()

	Summary("Data", dataDir)
	Blank()

	start := time.Now()
	stop := StartSpinner("Seeding database...")

	srv, err := web.New(web.Config{
		Addr:    ":0",
		DataDir: dataDir,
		Dev:     true,
	})
	if err != nil {
		stop()
		Error(fmt.Sprintf("Failed to create server: %v", err))
		return err
	}
	defer srv.Close()

	// Use the seeder
	seeder := seed.New(srv.Store())
	if err := seeder.Run(context.Background()); err != nil {
		stop()
		Error(fmt.Sprintf("Failed to seed: %v", err))
		return err
	}

	stop()
	Step("", "Database seeded", time.Since(start))
	Blank()
	Success("Sample data created")
	Hint("Run 'bi serve --dev' to start the server")
	Hint("Login with: admin@example.com / admin")
	Blank()

	return nil
}
