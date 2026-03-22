package main

import (
	"context"
	"os"

	"now/cli"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = ""
)

func main() {
	ctx := context.Background()

	if err := cli.Execute(ctx); err != nil {
		os.Exit(1)
	}
}