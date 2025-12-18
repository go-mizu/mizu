package main

import (
	"os"

	"github.com/go-mizu/mizu/cmd/cli"
)

func main() {
	os.Exit(cli.Run())
}
