// Package main is the entry point for the Chat application.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-mizu/blueprints/chat/cli"
)

// Version information (set at build time via ldflags).
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	cli.Version = Version
	cli.Commit = Commit
	cli.BuildTime = BuildTime

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := cli.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
