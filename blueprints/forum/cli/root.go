package cli

import (
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
func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "forum",
	Short: "Forum - A modern discussion platform",
	Long: `Forum is a full-featured discussion platform inspired by Reddit and Discourse.
Built with Mizu framework.`,
}

func init() {
	home, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(home, ".forum")

	rootCmd.PersistentFlags().StringVar(&dataDir, "data", defaultDataDir, "Data directory")
	rootCmd.PersistentFlags().StringVar(&addr, "addr", ":8080", "Server address")
	rootCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Development mode")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(seedCmd)
}
