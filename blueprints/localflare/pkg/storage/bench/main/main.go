// Command bench runs S3 storage driver benchmarks.
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

	"github.com/go-mizu/blueprints/localflare/pkg/storage/bench"
)

func main() {
	cfg := bench.DefaultConfig()

	// Parse flags
	flag.IntVar(&cfg.Iterations, "iterations", cfg.Iterations, "Number of iterations per benchmark")
	flag.IntVar(&cfg.WarmupIterations, "warmup", cfg.WarmupIterations, "Warmup iterations")
	flag.IntVar(&cfg.Concurrency, "concurrency", cfg.Concurrency, "Parallel operation concurrency")
	flag.StringVar(&cfg.OutputDir, "output", cfg.OutputDir, "Output directory for reports")
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Per-operation timeout")
	flag.BoolVar(&cfg.Quick, "quick", cfg.Quick, "Quick mode (fewer iterations)")
	flag.BoolVar(&cfg.Large, "large", cfg.Large, "Include large file benchmarks (10MB+)")
	flag.BoolVar(&cfg.DockerStats, "docker-stats", cfg.DockerStats, "Collect Docker container statistics")
	flag.BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "Verbose output")

	var driversFlag string
	var sizesFlag string
	flag.StringVar(&driversFlag, "drivers", "", "Comma-separated list of drivers to benchmark (empty = all)")
	flag.StringVar(&sizesFlag, "sizes", "", "Comma-separated object sizes (e.g., 1KB,64KB,1MB)")

	flag.Parse()

	// Apply quick mode
	if cfg.Quick {
		cfg.Iterations = 20
		cfg.WarmupIterations = 5
	}

	// Parse drivers
	if driversFlag != "" {
		cfg.Drivers = strings.Split(driversFlag, ",")
		for i := range cfg.Drivers {
			cfg.Drivers[i] = strings.TrimSpace(cfg.Drivers[i])
		}
	}

	// Parse sizes
	if sizesFlag != "" {
		cfg.ObjectSizes = parseSizes(sizesFlag)
	}

	// Add large sizes if requested
	if cfg.Large {
		cfg.ObjectSizes = append(cfg.ObjectSizes, 10*1024*1024, 100*1024*1024)
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

func parseSizes(s string) []int {
	var sizes []int
	parts := strings.Split(s, ",")

	for _, p := range parts {
		p = strings.TrimSpace(strings.ToUpper(p))
		var size int

		switch {
		case strings.HasSuffix(p, "GB"):
			n := parseNum(strings.TrimSuffix(p, "GB"))
			size = n * 1024 * 1024 * 1024
		case strings.HasSuffix(p, "MB"):
			n := parseNum(strings.TrimSuffix(p, "MB"))
			size = n * 1024 * 1024
		case strings.HasSuffix(p, "KB"):
			n := parseNum(strings.TrimSuffix(p, "KB"))
			size = n * 1024
		case strings.HasSuffix(p, "B"):
			size = parseNum(strings.TrimSuffix(p, "B"))
		default:
			size = parseNum(p)
		}

		if size > 0 {
			sizes = append(sizes, size)
		}
	}

	if len(sizes) == 0 {
		return []int{1024, 64 * 1024, 1024 * 1024}
	}
	return sizes
}

func parseNum(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
