package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"

	dataDir = "./data"
	reposDir = "./data/repos"
)

func Execute(ctx context.Context) error {
	rootCmd := &cobra.Command{
		Use:   "githome",
		Short: "GitHome - Self-hosted GitHub clone",
		Long: `GitHome is a full-featured self-hosted Git platform.
It provides repository hosting, issues, pull requests, and more.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildTime),
	}

	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "./data", "Data directory path")
	rootCmd.PersistentFlags().StringVar(&reposDir, "repos-dir", "./data/repos", "Git repositories directory path")

	rootCmd.AddCommand(
		newServeCmd(),
		newInitCmd(),
		newSeedCmd(),
	)

	return rootCmd.ExecuteContext(ctx)
}
