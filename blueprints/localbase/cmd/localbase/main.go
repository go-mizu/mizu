package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-mizu/mizu/blueprints/localbase/cli"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	// Set version info
	cli.Version = Version
	cli.Commit = Commit
	cli.BuildTime = BuildTime

	// Setup context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := cli.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
