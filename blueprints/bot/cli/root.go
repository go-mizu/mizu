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

var (
	// Version information (set via ldflags)
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// dataDir is the default data directory
var dataDir string

// databasePath is the SQLite database path
var databasePath string

// Execute runs the CLI
func Execute(ctx context.Context) error {
	root := &cobra.Command{
		Use:   "bot",
		Short: "Bot - Multi-Channel Chat Gateway",
		Long: `Bot is a multi-channel chat orchestration gateway inspired by OpenClaw.

It connects AI agents to messaging platforms like Telegram, Discord,
Mattermost, and generic webhooks with unified session management,
message routing, and security controls.

Features:
  - Multi-channel messaging (Telegram, Discord, Mattermost, Webhook)
  - AI agent management with configurable models
  - Session management with automatic reset policies
  - Message routing via binding rules (most-specific wins)
  - In-chat slash commands (/new, /status, /model, /help)
  - Webhook support for custom integrations
  - Web dashboard for monitoring and management
  - Pairing-based DM security

Get started:
  bot init      Initialize the database
  bot seed      Seed with sample data
  bot serve     Start the gateway server`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Set default data directory and database path
	home, _ := os.UserHomeDir()
	dataDir = filepath.Join(home, "data", "blueprints", "bot")
	databasePath = filepath.Join(dataDir, "bot.db")

	// Global flags
	root.SetVersionTemplate("bot {{.Version}}\n")
	root.Version = versionString()
	root.PersistentFlags().StringVar(&dataDir, "data", dataDir, "Data directory")
	root.PersistentFlags().StringVar(&databasePath, "database", databasePath, "SQLite database path")

	// Add subcommands
	root.AddCommand(NewInit())
	root.AddCommand(NewSeed())
	root.AddCommand(NewServe())

	if err := fang.Execute(ctx, root,
		fang.WithVersion(Version),
		fang.WithCommit(Commit),
	); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render("[ERROR] "+err.Error()))
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

// GetDataDir returns the data directory
func GetDataDir() string {
	return dataDir
}

// GetDatabasePath returns the SQLite database path
func GetDatabasePath() string {
	return databasePath
}
