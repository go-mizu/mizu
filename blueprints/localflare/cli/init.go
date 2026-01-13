package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/localflare/app/web"
)

// NewInit creates the init command
func NewInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the database",
		Long: `Initialize the Localflare database with the required schema.

This command creates all necessary tables for:
  - Zones and DNS records
  - SSL certificates
  - Firewall rules
  - Cache configuration
  - Workers and routes
  - KV namespaces
  - R2 buckets
  - D1 databases
  - Load balancers
  - Analytics

The database will be created if it doesn't exist.

Examples:
  localflare init                     # Initialize default database
  localflare init --data /path/to/dir # Initialize at specific directory`,
		RunE: runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	Blank()
	Header("", "Initialize Database")
	Blank()

	Summary("Data", dataDir)
	Blank()

	start := time.Now()
	stop := StartSpinner("Initializing database...")

	srv, err := web.New(web.Config{
		Addr:    ":0",
		DataDir: dataDir,
		Dev:     false,
	})
	if err != nil {
		stop()
		Error(fmt.Sprintf("Failed to initialize: %v", err))
		return err
	}
	srv.Close()

	stop()
	Step("", "Database initialized", time.Since(start))
	Blank()
	Success("Database ready")
	Hint(fmt.Sprintf("Data directory: %s", dataDir))
	Blank()

	return nil
}
