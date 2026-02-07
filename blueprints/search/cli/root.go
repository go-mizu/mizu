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
		Use:   "search",
		Short: "Search - A Full-Featured Search Engine",
		Long: `Search is a comprehensive, full-featured search engine inspired by Google and Kagi.

Features:
  - Full-text search with BM25 ranking
  - Autocomplete suggestions
  - Instant answers (calculator, converter, weather)
  - Knowledge panels
  - Image, video, and news search
  - Custom search lenses
  - Domain preferences (upvote, downvote, block)
  - Search history
  - Clean, Google-like UI

Get started:
  search init      Initialize the database
  search serve     Start the search server
  search seed      Seed with sample data`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Set default data directory and database path
	home, _ := os.UserHomeDir()
	dataDir = filepath.Join(home, "data", "blueprints", "search")
	databasePath = filepath.Join(dataDir, "search.db")

	// Global flags
	root.SetVersionTemplate("search {{.Version}}\n")
	root.Version = versionString()
	root.PersistentFlags().StringVar(&dataDir, "data", dataDir, "Data directory")
	root.PersistentFlags().StringVar(&databasePath, "database", databasePath, "SQLite database path")
	root.PersistentFlags().Bool("dev", false, "Enable development mode")

	// Add subcommands
	root.AddCommand(NewServe())
	root.AddCommand(NewInit())
	root.AddCommand(NewSeed())
	root.AddCommand(NewCrawl())
	root.AddCommand(NewRecrawl())
	root.AddCommand(NewAnalytics())
	root.AddCommand(NewFW1())
	root.AddCommand(NewFW2())
	root.AddCommand(NewCC())

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
