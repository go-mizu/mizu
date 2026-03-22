package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/charmbracelet/fang"

	"now/cli/storage"
)

var (
	Version = "dev"
	Commit  = ""
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cmd := storage.New()

	opts := []fang.Option{
		fang.WithVersion(Version),
	}
	if Commit != "" {
		opts = append(opts, fang.WithCommit(Commit))
	}

	if err := fang.Execute(ctx, cmd, opts...); err != nil {
		os.Exit(1)
	}
}
