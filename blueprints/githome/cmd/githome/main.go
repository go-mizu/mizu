package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-mizu/blueprints/githome/cli"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	if err := cli.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
