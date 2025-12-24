// Package cli provides the command-line interface.
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

// Global flags
var (
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

// Execute runs the CLI with the given context.
func Execute(ctx context.Context) error {
	root := &cobra.Command{
		Use:   "chat",
		Short: "Realtime messaging platform",
		Long: `Chat is a realtime messaging platform with servers, channels, and direct messages.

Features include:
  - Server/guild management
  - Text and voice channels
  - Direct messages and group DMs
  - Role-based permissions
  - Real-time messaging via WebSocket
  - User presence and status`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetVersionTemplate("chat {{.Version}}\n")
	root.Version = versionString()
	root.PersistentFlags().StringVar(&dataDir, "data", defaultDataDir(), "Data directory")
	root.PersistentFlags().StringVar(&addr, "addr", ":8080", "Server address")
	root.PersistentFlags().BoolVar(&dev, "dev", false, "Development mode")

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
