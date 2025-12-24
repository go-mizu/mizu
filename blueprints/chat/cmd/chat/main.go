// Package main is the entry point for the Chat application.
package main

import (
	"fmt"
	"os"

	"github.com/go-mizu/blueprints/chat/cli"
)

// Version is set at build time.
var Version = "dev"

func main() {
	cli.Version = Version
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
