package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/go-mizu/fang"
	"github.com/spf13/cobra"
)

var (
	// Version information (set via ldflags)
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// dataDir is the default data directory
var dataDir string

// Execute runs the CLI
func Execute(ctx context.Context) error {
	root := &cobra.Command{
		Use:   "kanban",
		Short: "Kanban - Linear-style project management",
		Long: `Kanban is a full-featured project management system inspired by Linear.

Features:
  - Workspaces and teams organization
  - Projects with kanban boards
  - Issues with custom fields
  - Cycles for sprint planning
  - Drag-and-drop board interface

Get started:
  kanban init      Initialize the database
  kanban serve     Start the web server
  kanban seed      Seed with sample data`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Set default data directory
	home, _ := os.UserHomeDir()
	dataDir = filepath.Join(home, "data", "blueprint", "kanban")

	// Global flags
	root.PersistentFlags().StringVar(&dataDir, "data", dataDir, "Data directory")
	root.PersistentFlags().Bool("dev", false, "Enable development mode")

	// Add subcommands
	root.AddCommand(NewServe())
	root.AddCommand(NewInit())
	root.AddCommand(NewSeed())

	// Configure fang styling
	theme := fang.Theme{
		Primary: lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")),
		Muted:   lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")),
		Warning: lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")),
	}

	return fang.Execute(ctx, root, fang.WithTheme(theme))
}
