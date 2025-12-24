// Package cli provides the command-line interface.
package cli

import (
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time.
	Version = "dev"

	// Global flags
	dataDir string
	addr    string
	dev     bool
)

// Execute runs the CLI.
func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:     "social",
	Short:   "Social - A social network platform",
	Long:    `Social is a general-purpose social network with profiles, feeds, and relationships.`,
	Version: Version,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dataDir, "data", "./data", "Data directory")
	rootCmd.PersistentFlags().StringVar(&addr, "addr", ":8080", "Server address")
	rootCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Development mode")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(seedCmd)
}
