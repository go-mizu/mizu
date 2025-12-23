// Package cli provides the command-line interface for forum.
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

var (
	dataDir string
	addr    string
	dev     bool
	theme   string
)

// Execute runs the CLI with the given context.
func Execute(ctx context.Context) error {
	homeDir, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(homeDir, "data", "blueprint", "forum")

	root := &cobra.Command{
		Use:   "forum",
		Short: "A modern discussion platform",
		Long: `Forum is a full-featured discussion platform inspired by Reddit and Discourse.

Features include:
  - Subreddit-like boards with moderators
  - Threaded discussions with nested comments
  - Upvotes, downvotes, and scoring
  - Bookmarks and notifications
  - User profiles and karma system
  - Rich text posts with links and media`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetVersionTemplate("forum {{.Version}}\n")
	root.Version = versionString()
	root.PersistentFlags().StringVar(&dataDir, "data", defaultDataDir, "Data directory")
	root.PersistentFlags().StringVar(&addr, "addr", ":8080", "Server address")
	root.PersistentFlags().BoolVar(&dev, "dev", false, "Development mode")
	root.PersistentFlags().StringVar(&theme, "theme", "default", "UI theme (default, old, hn)")

	root.AddCommand(
		NewServe(),
		NewInit(),
		NewSeed(),
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
