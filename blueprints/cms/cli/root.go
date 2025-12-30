// Package cli provides the command-line interface for cms.
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
)

// Execute runs the CLI with the given context.
func Execute(ctx context.Context) error {
	homeDir, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(homeDir, "data", "blueprint", "cms")

	root := &cobra.Command{
		Use:   "cms",
		Short: "A WordPress-compatible content management system",
		Long: `CMS is a WordPress-compatible content management system built on Mizu.

Features include:
  - Posts, pages, and custom post types
  - Categories, tags, and custom taxonomies
  - Media library with image processing
  - User management with roles and capabilities
  - Comments with moderation
  - WordPress REST API v2 compatibility
  - XML-RPC API for legacy tools
  - Themes and customization
  - Full WordPress plugin/theme compatibility`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetVersionTemplate("cms {{.Version}}\n")
	root.Version = versionString()
	root.PersistentFlags().StringVar(&dataDir, "data", defaultDataDir, "Data directory")
	root.PersistentFlags().StringVar(&addr, "addr", ":8080", "Server address")
	root.PersistentFlags().BoolVar(&dev, "dev", false, "Development mode")

	root.AddCommand(
		NewServe(),
		NewInit(),
		NewSeed(),
		NewInstall(),
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
