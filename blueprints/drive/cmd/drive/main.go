// Drive is a full-featured cloud file storage application.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-mizu/blueprints/drive/cli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := cli.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
