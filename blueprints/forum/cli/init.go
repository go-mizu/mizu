package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/forum/store/duckdb"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the forum database",
	Long:  `Creates the database and runs all migrations to set up the schema.`,
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Printf("Initializing forum database at %s...\n", dataDir)

	store, err := duckdb.Open(dataDir)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	fmt.Println("Database initialized successfully!")
	return nil
}
