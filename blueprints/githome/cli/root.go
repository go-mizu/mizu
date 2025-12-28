package cli

import (
	"context"
	"fmt"
	"os"

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
		Short: "GitHome - Self-hosted GitHub clone",
		Long: `GitHome is a full-featured self-hosted Git platform.
It provides repository hosting, issues, pull requests, and more.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildTime),
	}

	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", dataDir, "Data directory path")
	rootCmd.PersistentFlags().StringVar(&reposDir, "repos-dir", dataDir+"/repos", "Git repositories directory path")

	rootCmd.AddCommand(
		newServeCmd(),
		newInitCmd(),
		newSeedCmd(),
	)

	return rootCmd.ExecuteContext(ctx)
}
