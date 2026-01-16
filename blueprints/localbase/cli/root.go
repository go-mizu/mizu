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
		Use:   "localbase",
		Short: "Localbase - Offline Supabase Clone",
		Long: `Localbase is a comprehensive, offline-first implementation of Supabase's core features.

Features:
  - PostgreSQL database with pgvector, pg_graphql
  - GoTrue-compatible authentication
  - S3-compatible file storage
  - Realtime subscriptions (WebSocket)
  - Edge Functions (Deno runtime)
  - PostgREST auto-generated API
  - GraphQL API
  - Full dashboard UI

Get started:
  localbase init      Initialize the database
  localbase serve     Start all services
  localbase seed      Seed with sample data`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Set default data directory
	home, _ := os.UserHomeDir()
	dataDir = filepath.Join(home, "data", "blueprint", "localbase")
	databaseURL = "postgres://localbase:localbase@localhost:5432/localbase?sslmode=disable"

	// Global flags
	root.SetVersionTemplate("localbase {{.Version}}\n")
	root.Version = versionString()
	root.PersistentFlags().StringVar(&dataDir, "data", dataDir, "Data directory")
	root.PersistentFlags().StringVar(&databaseURL, "database-url", databaseURL, "PostgreSQL connection URL")
	root.PersistentFlags().Bool("dev", false, "Enable development mode")

	// Add subcommands
	root.AddCommand(NewServe())
	root.AddCommand(NewInit())
	root.AddCommand(NewSeed())
	root.AddCommand(NewMigrate())

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
