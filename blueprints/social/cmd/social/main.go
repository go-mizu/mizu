// Package main is the entry point for the Social application.
package main

import (
	"fmt"
	"os"

	"github.com/go-mizu/blueprints/social/cli"
)

// Version is set at build time.
var Version = "dev"

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
