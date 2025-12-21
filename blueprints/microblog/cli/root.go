// Package cli provides the command-line interface for microblog.
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

// Version information (set at build time via ldflags)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

var dataDir string

// Execute runs the CLI with the given context.
func Execute(ctx context.Context) error {
	homeDir, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(homeDir, "data", "blueprint", "microblog")

	root := &cobra.Command{
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
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetVersionTemplate("microblog {{.Version}}\n")
	root.Version = versionString()
	root.PersistentFlags().StringVar(&dataDir, "data", defaultDataDir, "Data directory")

	root.AddCommand(
		NewServe(),
		NewInit(),
		NewUser(),
	)

	if err := fang.Execute(ctx, root,
		fang.WithVersion(Version),
		fang.WithCommit(Commit),
	); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(iconCross+" "+err.Error()))
		return err
	}
	return nil
}

func versionString() string {
	if strings.TrimSpace(Version) != "" && Version != "dev" {
		return Version
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			return bi.Main.Version
		}
	}
	return "dev"
}
