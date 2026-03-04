package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/fang"
	"github.com/go-mizu/mizu/blueprints/search/cli"
	"github.com/spf13/cobra"
)

// Version information (set via ldflags)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	cli.Version = Version
	cli.Commit = Commit
	cli.BuildTime = BuildTime

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	root := cli.NewBench()
	root.Use = "bench"
	rewriteBenchExamples(root)
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetVersionTemplate("bench {{.Version}}\n")
	root.Version = Version

	if err := fang.Execute(ctx, root,
		fang.WithVersion(Version),
		fang.WithCommit(Commit),
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rewriteBenchExamples(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	if cmd.Example != "" {
		cmd.Example = strings.ReplaceAll(cmd.Example, "search bench ", "bench ")
	}
	for _, c := range cmd.Commands() {
		rewriteBenchExamples(c)
	}
}
