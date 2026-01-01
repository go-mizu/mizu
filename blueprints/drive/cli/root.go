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
		Use:   "drive",
		Short: "Drive - Cloud file storage like Google Drive, Box, Dropbox",
		Long: `Drive is a full-featured cloud file storage system.

Features:
  - File and folder management
  - Drag-and-drop uploads
  - File sharing with permissions
  - Multiple access protocols (REST, S3, WebDAV, SFTP)
  - File versioning and trash
  - Activity logging

Get started:
  drive init      Initialize the database
  drive serve     Start the web server
  drive seed      Seed with sample data`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Set default data directory
	home, _ := os.UserHomeDir()
	dataDir = filepath.Join(home, "data", "blueprint", "drive")

	// Global flags
	root.SetVersionTemplate("drive {{.Version}}\n")
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
