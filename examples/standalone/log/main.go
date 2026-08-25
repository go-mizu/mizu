// Command log builds an *slog.Logger from a struct of settings, which is the
// whole of what this package does.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/go-mizu/mizu/log"
)

func main() {
	logger, closer, err := log.New(log.Options{Level: slog.LevelInfo, Format: "json", Output: "stdout"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closer.Close()

	logger.Info("listening", "addr", ":8080")
}
