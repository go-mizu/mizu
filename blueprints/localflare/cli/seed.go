package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/localflare/app/web"
	"github.com/go-mizu/blueprints/localflare/pkg/seed"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with comprehensive sample data",
		Long: `Seed the Localflare database with comprehensive sample data for testing.

This creates realistic data across all Localflare products:

  Zones & DNS:
    - 4 zones (example.com, api.myapp.io, store.acme.co, internal.corp)
    - DNS records (A, AAAA, CNAME, MX, TXT) for each zone

  Security:
    - SSL certificates and settings
    - Firewall rules, IP access rules, rate limits

  Performance:
    - Cache settings and rules
    - Load balancers with origin pools and health checks

  Compute:
    - Workers with bindings (KV, R2, D1)
    - Worker routes per zone
    - Durable Objects with storage
    - Message queues with consumers

  Storage:
    - KV namespaces with sample data
    - R2 buckets with objects
    - D1 databases with tables and records

  AI & Search:
    - Vector indexes with embeddings
    - AI Gateway configurations

  Analytics:
    - 7 days of traffic analytics
    - Analytics Engine datasets

  Scheduling:
    - Cron triggers with execution history
    - Hyperdrive database configs

Examples:
  localflare seed                     # Seed with comprehensive data
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
	defer srv.Close()

	// Use the new comprehensive seeder
	seeder := seed.New(srv.Store())
	if err := seeder.Run(context.Background()); err != nil {
		stop()
		Error(fmt.Sprintf("Failed to seed: %v", err))
		return err
	}

	stop()
	Step("", "Database seeded", time.Since(start))
	Blank()
	Success("Comprehensive sample data created")
	Hint("Run 'localflare serve --dev' to start the server")
	Blank()

	return nil
}
