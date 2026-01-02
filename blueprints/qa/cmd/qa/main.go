package main

import (
	"context"
	"os"

	"github.com/go-mizu/mizu/blueprints/qa/cli"
)

func main() {
	if err := cli.Execute(context.Background()); err != nil {
		os.Exit(1)
	}
}
