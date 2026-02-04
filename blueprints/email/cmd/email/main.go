package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-mizu/mizu/blueprints/email/cli"
)

// Version information (set via ldflags)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	// Set version info in cli package
	cli.Version = Version
	cli.Commit = Commit
	cli.BuildTime = BuildTime

	// Create context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Run CLI
	if err := cli.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
