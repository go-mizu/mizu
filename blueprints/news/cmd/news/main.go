package main

import (
	"context"
	"os"

	"github.com/go-mizu/mizu/blueprints/news/cli"
)

// Version information (set via ldflags)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	cli.Version = Version
	cli.Commit = Commit
	cli.BuildTime = BuildTime

	if err := cli.Execute(context.Background()); err != nil {
		os.Exit(1)
	}
}
