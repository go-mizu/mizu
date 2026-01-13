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

// Execute runs the CLI
func Execute(ctx context.Context) error {
	root := &cobra.Command{
		Use:   "localflare",
		Short: "Localflare - Offline Cloudflare Clone",
		Long: `Localflare is a comprehensive, offline-first implementation of Cloudflare's core features.

Features:
  - DNS server with zone management
  - Reverse proxy with caching
  - SSL/TLS certificate generation
  - Web Application Firewall (WAF)
  - Workers runtime (JavaScript)
  - KV storage
  - R2-compatible object storage
  - D1 SQLite databases
  - Load balancing with health checks
  - Analytics and logging

Get started:
  localflare init      Initialize the database
  localflare serve     Start all services
  localflare seed      Seed with sample data`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Set default data directory
	home, _ := os.UserHomeDir()
	dataDir = filepath.Join(home, "data", "blueprint", "localflare")

	// Global flags
	root.SetVersionTemplate("localflare {{.Version}}\n")
	root.Version = versionString()
	root.PersistentFlags().StringVar(&dataDir, "data", dataDir, "Data directory")
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
