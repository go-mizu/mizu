package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/qa/store/duckdb"
)

// NewInit creates the init command.
func NewInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the QA database",
		Long: `Creates the database and runs all migrations to set up the schema.

This command is safe to run multiple times - it will not overwrite existing data.`,
		RunE: runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconDatabase, "Initializing QA Database")
	ui.Blank()

	start := time.Now()
	ui.StartSpinner("Creating database...")

	store, err := duckdb.Open(dataDir)
	if err != nil {
		ui.StopSpinnerError("Failed to create database")
		return err
	}
	defer store.Close()

	ui.StopSpinner("Database created", time.Since(start))

	ui.Summary([][2]string{
		{"Location", dataDir},
		{"Status", "Ready"},
	})

	ui.Success("Database initialized successfully!")
	ui.Blank()
	ui.Hint("Next: run 'qa seed' to add sample data, or 'qa serve' to start the server")

	return nil
}
