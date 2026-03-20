package main

import (
	"context"
	"os"

	"now/cli/storage"
)

var (
	Version   = "1.0.0"
	Commit    = "unknown"
	BuildTime = ""
)

func main() {
	ctx := context.Background()
	cmd := storage.New(Version)
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
