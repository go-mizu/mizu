// Package cli provides the command-line interface for qa.
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
	defaultDataDir := filepath.Join(homeDir, "data", "blueprint", "qa")

	root := &cobra.Command{
		Use:   "qa",
		Short: "A Stack Overflow-style Q&A platform",
		Long: `QA is a full-featured Q&A platform inspired by Stack Overflow.

Features include:
  - Questions, answers, and comments
  - Voting, favorites, and accepted answers
  - Tags, search, and discovery
  - Reputation, badges, and moderation tools
  - Stack Overflow-style UI`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetVersionTemplate("qa {{.Version}}\n")
	root.Version = versionString()
	root.PersistentFlags().StringVar(&dataDir, "data", defaultDataDir, "Data directory")
	root.PersistentFlags().StringVar(&addr, "addr", ":8080", "Server address")
	root.PersistentFlags().BoolVar(&dev, "dev", false, "Development mode")
	root.PersistentFlags().StringVar(&theme, "theme", "default", "UI theme (default)")

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
