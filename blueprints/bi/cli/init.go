package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/bi/app/web"
)

// NewInit creates the init command
func NewInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the database",
		Long: `Initialize the BI database with the required schema.

This command creates all necessary tables for:
  - Data sources (database connections)
  - Tables and columns metadata
  - Questions (saved queries)
  - Dashboards and cards
  - Collections (folders)
  - Models (curated datasets)
  - Metrics (canonical calculations)
  - Alerts and subscriptions
  - Users and permissions

The database will be created if it doesn't exist.

Examples:
  bi init                     # Initialize default database
  bi init --data /path/to/dir # Initialize at specific directory`,
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
