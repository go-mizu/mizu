package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-mizu/mizu/blueprints/lingo/cli"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	cli.Version = Version
	cli.Commit = Commit
	cli.BuildTime = BuildTime

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := cli.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
