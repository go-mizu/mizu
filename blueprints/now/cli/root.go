package cli

import (
	"context"

	chatcmd "now/cli/chat"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

// New returns the root command.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "now",
		Short: "Instant CLI for humans and agents",
		Long: `now is an instant CLI for humans and agents.

It provides simple commands for chat today, and can grow into
other areas such as database, storage, and more.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		chatcmd.New(),
	)

	return cmd
}

// Execute runs the CLI.
func Execute(ctx context.Context) error {
	return fang.Execute(ctx, New())
}