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

// databaseURL is the PostgreSQL connection string
var databaseURL string

// Execute runs the CLI
func Execute(ctx context.Context) error {
	root := &cobra.Command{
		Use:   "lingo",
		Short: "Lingo - Language Learning Platform",
		Long: `Lingo is a comprehensive language learning platform with gamification.

Features:
  - Duolingo-style learning path
  - 13+ exercise types (translation, listening, speaking, etc.)
  - XP, streaks, hearts, and gems
  - Leagues and leaderboards
  - Friends and social features
  - Achievements and badges
  - Stories for immersive learning
  - Spaced repetition system

Get started:
  lingo init      Initialize the database
  lingo seed      Seed with sample courses
  lingo serve     Start the server`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Set default data directory
	home, _ := os.UserHomeDir()
	dataDir = filepath.Join(home, "data", "blueprint", "lingo")
	databaseURL = "postgres://lingo:lingo@localhost:5432/lingo?sslmode=disable"

	// Global flags
	root.SetVersionTemplate("lingo {{.Version}}\n")
	root.Version = versionString()
	root.PersistentFlags().StringVar(&dataDir, "data", dataDir, "Data directory")
	root.PersistentFlags().StringVar(&databaseURL, "database-url", databaseURL, "PostgreSQL connection URL")
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

// GetDatabaseURL returns the database connection URL
func GetDatabaseURL() string {
	return databaseURL
}
