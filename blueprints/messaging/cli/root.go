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

// defaultDataDir returns the default data directory.
func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./data"
	}
	return filepath.Join(home, "data", "blueprint", "messaging")
}

// Execute runs the CLI with the given context.
func Execute(ctx context.Context) error {
	root := &cobra.Command{
		Use:   "messaging",
		Short: "WhatsApp/Telegram-style messaging platform",
		Long: `Messaging is a comprehensive messaging platform with personal chats, groups, and stories.

Features include:
  - One-to-one and group messaging
  - Media sharing and voice messages
  - Stories/status updates
  - Message reactions and replies
  - Real-time messaging via WebSocket
  - Contact management`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetVersionTemplate("messaging {{.Version}}\n")
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
