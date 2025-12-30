// Package cli provides command-line interface.
package cli

import (
	"context"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// Execute runs the CLI.
func Execute(ctx context.Context) error {
	root := &cobra.Command{
		Use:     "cms",
		Short:   "Payload CMS compatible headless CMS",
		Long:    "A Payload CMS compatible headless content management system built on the Mizu framework.",
		Version: Version + " (" + Commit + ")",
	}

	root.AddCommand(NewServe())
	root.AddCommand(NewInit())
	root.AddCommand(NewSeed())

	return fang.Execute(ctx, root)
}
