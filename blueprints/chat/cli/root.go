// Package cli provides the command-line interface.
package cli

import (
	"os"
	"path/filepath"

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

// defaultDataDir returns the default data directory ($HOME/data/blueprint/chat).
func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./data"
	}
	return filepath.Join(home, "data", "blueprint", "chat")
}

// Execute runs the CLI.
func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:     "chat",
	Short:   "Chat - Realtime messaging platform",
	Long:    `Chat is a realtime messaging platform with servers, channels, and direct messages.`,
	Version: Version,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dataDir, "data", defaultDataDir(), "Data directory")
	rootCmd.PersistentFlags().StringVar(&addr, "addr", ":8080", "Server address")
	rootCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Development mode")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(seedCmd)
}
