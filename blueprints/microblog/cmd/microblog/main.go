// Package main is the entry point for the microblog CLI.
package main

import (
	"os"

	"github.com/go-mizu/blueprints/microblog/cli"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
