package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-mizu/blueprints/cms/cli"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	// Set version info from ldflags
	cli.Version = Version
	cli.Commit = Commit
	cli.BuildTime = BuildTime

	// Create context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := cli.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
