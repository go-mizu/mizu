package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-mizu/mizu/blueprints/search/cli"

	// Register sub-package providers via init()
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/serp/firecrawl"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/serp/jina"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/serp/parallel"
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

	// Create context with signal handling.
	// First SIGTERM cancels the context (lets long-running commands like
	// --schedule recover via their restart loops). Second signal or SIGINT
	// forces immediate exit.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		cancel()
		if sig == os.Interrupt {
			os.Exit(130) // immediate exit on Ctrl+C
		}
		// SIGTERM: cancel context but let restart loops recover.
		// Second signal forces exit.
		<-sigCh
		os.Exit(143)
	}()

	// Run CLI
	if err := cli.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
