// Package cli provides command-line interface for the forum.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "forum",
	Short: "Forum - A production-ready forum platform",
	Long: `Forum is a full-featured discussion platform built with Mizu.

Features include hierarchical forums, threaded discussions, voting,
moderation tools, and more.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(initCmd)
}
