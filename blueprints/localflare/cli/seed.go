package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/localflare/app/web"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long: `Seed the Localflare database with sample data for testing.

This creates:
  - Sample zones (example.com, test.local)
  - DNS records (A, AAAA, CNAME, MX, TXT)
  - Firewall rules
  - Sample worker
  - KV namespace with sample data
  - R2 bucket
  - D1 database

Examples:
  localflare seed                     # Seed with sample data
  localflare seed --data /path/to/dir # Seed specific database`,
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

	// Seed data
	if err := srv.SeedData(context.Background()); err != nil {
		stop()
		srv.Close()
		Error(fmt.Sprintf("Failed to seed: %v", err))
		return err
	}
	srv.Close()

	stop()
	Step("", "Database seeded", time.Since(start))
	Blank()
	Success("Sample data created")
	Hint("Run 'localflare serve --dev' to start the server")
	Blank()

	return nil
}
