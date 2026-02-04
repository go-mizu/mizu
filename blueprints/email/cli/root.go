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
		Use:   "email",
		Short: "Email - A Gmail-like Email Client",
		Long: `Email is a Gmail-like email client built on the Mizu web framework.

Features:
  - Full email inbox with threading
  - Labels and label management
  - Starred and important emails
  - Drafts and sent mail
  - Full-text search
  - Contact management
  - Batch operations (archive, trash, delete)
  - Conversation view

Get started:
  email init      Initialize the database
  email serve     Start the email server
  email seed      Seed with sample data`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Set default data directory and database path
	home, _ := os.UserHomeDir()
	dataDir = filepath.Join(home, "data", "blueprints", "email")
	databasePath = filepath.Join(dataDir, "email.db")

	// Global flags
	root.SetVersionTemplate("email {{.Version}}\n")
	root.Version = versionString()
	root.PersistentFlags().StringVar(&dataDir, "data", dataDir, "Data directory")
	root.PersistentFlags().StringVar(&databasePath, "database", databasePath, "SQLite database path")
	root.PersistentFlags().Bool("dev", false, "Enable development mode")

	// Add subcommands
	root.AddCommand(NewServe())
	root.AddCommand(NewInit())
	root.AddCommand(NewSeed())

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
