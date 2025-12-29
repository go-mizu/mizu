package cli

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"

	dataDir  = os.Getenv("HOME") + "/data/blueprint/githome"
	reposDir = dataDir + "/repos"
)

func Execute(ctx context.Context) error {
	rootCmd := &cobra.Command{
		Use:   "githome",
		Short: "Self-hosted GitHub clone",
		Long: `GitHome is a full-featured self-hosted Git platform.
It provides repository hosting, issues, pull requests, and more.

Get started:
  githome init       Initialize the database
  githome seed demo  Add sample data
  githome serve      Start the server`,
	}

	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", dataDir, "Data directory path")
	rootCmd.PersistentFlags().StringVar(&reposDir, "repos-dir", dataDir+"/repos", "Git repositories directory path")

	rootCmd.AddCommand(
		newServeCmd(),
		newInitCmd(),
		newSeedCmd(),
	)

	// Use Fang for modern CLI styling with automatic version flag
	return fang.Execute(ctx, rootCmd, fang.WithVersion(Version))
}
