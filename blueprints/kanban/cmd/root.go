package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kanban",
	Short: "Kanban - A modern project management system",
	Long: `Kanban is a full-featured project management system inspired by Linear, Jira, and Trello.

Features:
  - Workspaces and projects organization
  - Kanban boards with drag-and-drop
  - Issue tracking with statuses, priorities, and labels
  - Sprint planning and backlog management
  - Team collaboration with comments and mentions
  - Real-time notifications

Get started:
  kanban init      Initialize the database
  kanban serve     Start the web server`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("db", "d", "kanban.db", "Database file path")
}
