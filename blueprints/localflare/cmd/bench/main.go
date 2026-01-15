// Command bench runs storage benchmarks for all configured drivers.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage/bench"
	"github.com/go-mizu/blueprints/localflare/pkg/storage/driver/local"
)

func main() {
	var (
		iterations      = flag.Int("iterations", 100, "Number of iterations per benchmark")
		warmup          = flag.Int("warmup", 10, "Number of warmup iterations")
		timeout         = flag.Duration("timeout", 30*time.Second, "Per-operation timeout")
		outputDir       = flag.String("output", "./pkg/storage/report", "Output directory for reports")
		quick           = flag.Bool("quick", false, "Quick mode (fewer iterations)")
		drivers         = flag.String("drivers", "", "Comma-separated list of drivers to benchmark (empty = all)")
		outputFormats   = flag.String("formats", "markdown,json,csv", "Output formats (markdown,json,csv)")
		duration        = flag.Duration("duration", 0, "Duration-based mode (run each benchmark for this duration)")
		dockerStats     = flag.Bool("docker-stats", true, "Collect Docker container statistics and cleanup after each driver")
		verbose         = flag.Bool("verbose", false, "Verbose output")
		fileCounts      = flag.String("file-counts", "1,10,100,1000,10000", "Comma-separated file counts to benchmark (e.g., 1,10,100,1000,10000,100000)")
		noFsync         = flag.Bool("no-fsync", true, "Skip fsync for maximum write performance (default: enabled for benchmarks)")
		filter          = flag.String("filter", "", "Filter benchmarks by name (substring match, e.g., 'MixedWorkload')")
	)
	flag.Parse()

	// Set NoFsync for local driver (major performance improvement)
	local.NoFsync = *noFsync

	cfg := bench.DefaultConfig()
	cfg.Iterations = *iterations
	cfg.WarmupIterations = *warmup
	cfg.Timeout = *timeout
	cfg.OutputDir = *outputDir
	cfg.Duration = *duration
	cfg.DockerStats = *dockerStats
	cfg.Verbose = *verbose

	if *quick {
		cfg = bench.QuickConfig()
		cfg.OutputDir = *outputDir
		cfg.DockerStats = *dockerStats
		cfg.Verbose = *verbose
	}

	if *drivers != "" {
		cfg.Drivers = strings.Split(*drivers, ",")
	}

	cfg.OutputFormats = strings.Split(*outputFormats, ",")
	cfg.Filter = *filter

	// Parse file counts
	if *fileCounts != "" {
		parts := strings.Split(*fileCounts, ",")
		counts := make([]int, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				counts = append(counts, n)
			}
		}
		if len(counts) > 0 {
			cfg.FileCounts = counts
		}
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted, cleaning up...")
		cancel()
	}()

	runner := bench.NewRunner(cfg)
	runner.SetLogger(func(format string, args ...any) {
		fmt.Printf(format+"\n", args...)
	})

	fmt.Println("=== Storage Benchmark Suite v2 ===")
	fmt.Printf("Iterations: %d, Warmup: %d\n", cfg.Iterations, cfg.WarmupIterations)
	if cfg.Duration > 0 {
		fmt.Printf("Mode: Duration-based (%v per operation)\n", cfg.Duration)
	} else {
		fmt.Printf("Mode: Iteration-based\n")
	}
	fmt.Printf("Output: %s\n", cfg.OutputDir)
	fmt.Printf("Formats: %v\n", cfg.OutputFormats)
	if cfg.Filter != "" {
		fmt.Printf("Filter: %s\n", cfg.Filter)
	}
	fmt.Println()

	report, err := runner.Run(ctx)
	if err != nil {
		log.Fatalf("Benchmark failed: %v", err)
	}

	// Save reports in all configured formats
	if err := report.SaveAll(cfg.OutputDir, cfg.OutputFormats); err != nil {
		log.Fatalf("Save reports failed: %v", err)
	}

	fmt.Println()
	fmt.Printf("Reports saved to %s\n", cfg.OutputDir)

	// Print summary
	fmt.Println()
	fmt.Println("=== Summary ===")
	driverResults := make(map[string]int)
	driverErrors := make(map[string]int)
	driverSkipped := make(map[string]int)
	errorDetails := make(map[string][]*bench.Metrics)
	for _, m := range report.Results {
		driverResults[m.Driver]++
		driverErrors[m.Driver] += m.Errors
		if m.Errors > 0 {
			errorDetails[m.Driver] = append(errorDetails[m.Driver], m)
		}
	}
	for _, skip := range report.SkippedBenchmarks {
		driverSkipped[skip.Driver]++
	}
	for driver, count := range driverResults {
		skipped := driverSkipped[driver]
		if skipped > 0 {
			fmt.Printf("  %s: %d benchmarks, %d errors, %d skipped\n", driver, count, driverErrors[driver], skipped)
		} else {
			fmt.Printf("  %s: %d benchmarks, %d errors\n", driver, count, driverErrors[driver])
		}
	}

	// Exit with error if any driver had errors
	totalErrors := 0
	for _, errs := range driverErrors {
		totalErrors += errs
	}
	if totalErrors > 0 {
		fmt.Printf("\nWarning: %d total errors occurred during benchmarks\n", totalErrors)
	}

	if len(errorDetails) > 0 {
		fmt.Println()
		fmt.Println("=== Error Details ===")
		drivers := make([]string, 0, len(errorDetails))
		for driver := range errorDetails {
			drivers = append(drivers, driver)
		}
		sort.Strings(drivers)
		for _, driver := range drivers {
			details := errorDetails[driver]
			sort.Slice(details, func(i, j int) bool {
				return details[i].Operation < details[j].Operation
			})
			fmt.Printf("  %s:\n", driver)
			for _, m := range details {
				msg := m.LastError
				if msg == "" {
					msg = "unknown error"
				}
				fmt.Printf("    - %s: %d errors (last: %s)\n", m.Operation, m.Errors, msg)
			}
		}
	}

	os.Exit(0)
}
