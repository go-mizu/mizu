// Command bench runs vectorize driver benchmarks.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/bench"
)

func main() {
	cfg := bench.DefaultConfig()

	// Parse flags
	flag.IntVar(&cfg.Dimensions, "dimensions", cfg.Dimensions, "Vector dimensions")
	flag.IntVar(&cfg.DatasetSize, "dataset-size", cfg.DatasetSize, "Number of vectors to generate")
	flag.IntVar(&cfg.BatchSize, "batch-size", cfg.BatchSize, "Batch size for inserts")
	flag.IntVar(&cfg.SearchIterations, "search-iterations", cfg.SearchIterations, "Number of search queries")
	flag.IntVar(&cfg.WarmupIterations, "warmup", cfg.WarmupIterations, "Warmup iterations")
	flag.IntVar(&cfg.TopK, "topk", cfg.TopK, "Number of results to return")
	flag.StringVar(&cfg.OutputDir, "output", cfg.OutputDir, "Output directory for reports")
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Operation timeout")

	var driversFlag string
	flag.StringVar(&driversFlag, "drivers", "", "Comma-separated list of drivers to benchmark (empty = all)")

	flag.Parse()

	// Parse drivers
	if driversFlag != "" {
		cfg.Drivers = strings.Split(driversFlag, ",")
		for i := range cfg.Drivers {
			cfg.Drivers[i] = strings.TrimSpace(cfg.Drivers[i])
		}
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted, cleaning up...")
		cancel()
	}()

	// Run benchmarks
	runner := bench.NewRunner(cfg)

	startTime := time.Now()
	report, err := runner.Run(ctx)
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("Benchmark failed: %v\n", err)
		os.Exit(1)
	}

	// Save reports
	fmt.Printf("\n=== Saving Reports ===\n")
	fmt.Printf("Output directory: %s\n", cfg.OutputDir)

	if err := report.SaveJSON(cfg.OutputDir); err != nil {
		fmt.Printf("Failed to save JSON report: %v\n", err)
	} else {
		fmt.Printf("Saved: %s/raw_results.json\n", cfg.OutputDir)
	}

	if err := report.SaveMarkdown(cfg.OutputDir); err != nil {
		fmt.Printf("Failed to save Markdown report: %v\n", err)
	} else {
		fmt.Printf("Saved: %s/benchmark_report.md\n", cfg.OutputDir)
	}

	fmt.Printf("\nTotal benchmark time: %v\n", elapsed)
}
