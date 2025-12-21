// Package cli provides the command-line interface for microblog.
package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	dataDir string
)

// NewRoot creates the root command.
func NewRoot() *cobra.Command {
	homeDir, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(homeDir, "data", "blueprint", "microblog")

	cmd := &cobra.Command{
		Use:   "microblog",
		Short: "A modern microblogging platform",
		Long: `Microblog is a self-hosted microblogging platform combining the best
features from X/Twitter, Threads, and Mastodon.

Features include:
  - Short-form posts with mentions and hashtags
  - Reply threads and conversations
  - Likes, reposts, and bookmarks
  - Following/followers social graph
  - Content warnings and visibility controls
  - Full-text search and trending topics`,
	}

	cmd.PersistentFlags().StringVar(&dataDir, "data", defaultDataDir, "Data directory")

	cmd.AddCommand(
		NewServe(),
		NewInit(),
		NewUser(),
	)

	return cmd
}

// Execute runs the CLI.
func Execute() error {
	return NewRoot().Execute()
}
