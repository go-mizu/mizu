// Package main is the entry point for the microblog CLI.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-mizu/blueprints/microblog/cli"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	// Set version info for CLI
	cli.Version = Version
	cli.Commit = Commit
	cli.BuildTime = BuildTime

	// Create context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := cli.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
