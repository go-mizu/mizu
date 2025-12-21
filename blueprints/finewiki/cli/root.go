package cli

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

// Execute runs the CLI.
func Execute(ctx context.Context) error {
	root := &cobra.Command{
		Use:   "finewiki",
		Short: "FineWiki: fast read-only wiki viewer (DuckDB + Parquet)",
		Long: `FineWiki serves FineWiki Parquet shards with fast title search and SSR pages.

Usage:
  finewiki serve <lang>     Start the web server for a language
  finewiki import <lang>    Download parquet data for a language
  finewiki list             List available languages

Examples:
  finewiki import vi        # Download Vietnamese data
  finewiki serve vi         # Start server for Vietnamese wiki
  finewiki list --installed # Show installed languages`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetVersionTemplate("finewiki {{.Version}}\n")
	root.Version = versionString()

	root.AddCommand(serveCmd())
	root.AddCommand(importCmd())
	root.AddCommand(listCmd())

	if err := fang.Execute(ctx, root); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return err
	}
	return nil
}

func versionString() string {
	if v := os.Getenv("FINEWIKI_VERSION"); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			return bi.Main.Version
		}
	}
	return "dev"
}
