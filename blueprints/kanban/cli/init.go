package cli

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/kanban/app/web"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the database",
	Long: `Initialize the Kanban database with the required schema.

This command creates all necessary tables and indexes for:
  - Users and sessions
  - Workspaces and members
  - Projects and labels
  - Issues, comments, and activities
  - Sprints and notifications

The database will be created if it doesn't exist.

Examples:
  kanban init                     # Initialize default database
  kanban init --db /path/to/db    # Initialize at specific path`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	dbPath, _ := cmd.Root().PersistentFlags().GetString("db")

	log.Printf("Initializing database: %s", dbPath)

	dataDir := filepath.Dir(dbPath)
	if dataDir == "." {
		dataDir = "."
	}

	srv, err := web.New(web.Config{
		Addr:    ":8080",
		DataDir: dataDir,
		Dev:     false,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer srv.Close()

	log.Println("✅ Database initialized successfully")
	return nil
}
