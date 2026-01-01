// Package cli provides the command-line interface.
package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	dataDir string
	addr    string
	dev     bool
)

// Execute runs the CLI.
func Execute(ctx context.Context) error {
	root := &cobra.Command{
		Use:     "drive",
		Short:   "Drive - Cloud file storage",
		Version: "1.0.0",
	}

	// Default data directory
	home, _ := os.UserHomeDir()
	defaultData := filepath.Join(home, ".drive")

	root.PersistentFlags().StringVar(&dataDir, "data", defaultData, "Data directory path")
	root.PersistentFlags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	root.PersistentFlags().BoolVar(&dev, "dev", false, "Development mode")

	root.AddCommand(
		newServeCmd(),
		newInitCmd(),
		newSeedCmd(),
	)

	return root.ExecuteContext(ctx)
}
