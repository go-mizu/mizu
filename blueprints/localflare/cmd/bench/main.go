// Command bench runs storage benchmarks for all configured drivers.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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
		warmup        = flag.Int("warmup", 10, "Number of warmup iterations")
		timeout       = flag.Duration("timeout", 30*time.Second, "Per-operation timeout")
		outputDir     = flag.String("output", "./pkg/storage/report", "Output directory for reports")
		quick         = flag.Bool("quick", false, "Quick mode (shorter benchmark time)")
		drivers       = flag.String("drivers", "", "Comma-separated list of drivers to benchmark (empty = all)")
		outputFormats = flag.String("formats", "markdown,json,csv", "Output formats (markdown,json,csv)")
		dockerStats   = flag.Bool("docker-stats", true, "Collect Docker container statistics and cleanup after each driver")
		verbose       = flag.Bool("verbose", false, "Verbose output")
		fileCounts    = flag.String("file-counts", "1,10,100,1000,10000", "Comma-separated file counts to benchmark (e.g., 1,10,100,1000,10000,100000)")
		noFsync       = flag.Bool("no-fsync", true, "Skip fsync for maximum write performance (default: enabled for benchmarks)")
		filter        = flag.String("filter", "", "Filter benchmarks by name (substring match, e.g., 'MixedWorkload')")
		inMemory      = flag.Bool("in-memory", false, "Use in-memory storage mode for liteio (maximum performance, no persistence)")
		// Go-style adaptive benchmark settings (same defaults as 'go test -bench')
		benchTime     = flag.Duration("benchtime", 1*time.Second, "Target duration for each benchmark (e.g., 1s, 500ms, 2s)")
		minIters      = flag.Int("min-iters", 3, "Minimum iterations for statistical significance")
		// Docker compose settings
		composeDir    = flag.String("compose-dir", "./docker/s3/all", "Docker compose directory for S3 services")
		restartDocker = flag.Bool("restart-docker", true, "Restart docker-compose services before running benchmarks")
	)
	flag.Parse()

	// Enable in-memory mode for liteio if requested
	if *inMemory {
		local.EnableInMemoryMode()
		fmt.Println("In-memory mode: ENABLED for liteio")
	}

	// Set NoFsync for local driver (major performance improvement)
	local.NoFsync = *noFsync

	cfg := bench.DefaultConfig()
	cfg.WarmupIterations = *warmup
	cfg.Timeout = *timeout
	cfg.OutputDir = *outputDir
	cfg.DockerStats = *dockerStats
	cfg.Verbose = *verbose
	cfg.BenchTime = *benchTime
	cfg.MinBenchIterations = *minIters

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

	// Track if we were interrupted
	interrupted := false

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		fmt.Printf("\nReceived %v, stopping benchmark...\n", sig)
		interrupted = true
		cancel()
		// Give a moment for cleanup, then force exit on second signal
		select {
		case <-sigCh:
			fmt.Println("\nForce exit")
			os.Exit(1)
		case <-time.After(30 * time.Second):
			fmt.Println("\nCleanup timeout, force exit")
			os.Exit(1)
		}
	}()

	// Restart docker-compose services if requested
	if *restartDocker {
		fmt.Println("=== Restarting Docker Services ===")
		absComposeDir, err := filepath.Abs(*composeDir)
		if err != nil {
			log.Fatalf("Invalid compose directory: %v", err)
		}

		// Check if docker-compose file exists
		composeFile := filepath.Join(absComposeDir, "docker-compose.yaml")
		if _, err := os.Stat(composeFile); os.IsNotExist(err) {
			log.Fatalf("docker-compose.yaml not found at %s", composeFile)
		}

		fmt.Printf("Compose directory: %s\n", absComposeDir)

		// Stop and remove containers, volumes
		fmt.Println("Stopping existing containers...")
		stopCmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "down", "-v", "--remove-orphans")
		stopCmd.Dir = absComposeDir
		stopCmd.Stdout = os.Stdout
		stopCmd.Stderr = os.Stderr
		if err := stopCmd.Run(); err != nil {
			fmt.Printf("Warning: docker compose down failed: %v\n", err)
		}

		// Check if interrupted
		if interrupted {
			fmt.Println("Benchmark cancelled during docker restart")
			os.Exit(1)
		}

		// Start services fresh
		fmt.Println("Starting fresh containers...")
		startCmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "up", "-d", "--wait", "--build")
		startCmd.Dir = absComposeDir
		startCmd.Stdout = os.Stdout
		startCmd.Stderr = os.Stderr
		if err := startCmd.Run(); err != nil {
			log.Fatalf("docker compose up failed: %v", err)
		}

		// Check if interrupted
		if interrupted {
			fmt.Println("Benchmark cancelled during docker startup")
			os.Exit(1)
		}

		fmt.Println("Docker services ready")
		fmt.Println()
	}

	runner := bench.NewRunner(cfg)
	runner.SetLogger(func(format string, args ...any) {
		fmt.Printf(format+"\n", args...)
	})

	fmt.Println("=== Storage Benchmark Suite v2 ===")
	fmt.Printf("Mode: Adaptive (Go-style, target: %v per benchmark)\n", cfg.BenchTime)
	fmt.Printf("Min iterations: %d, Warmup: %d\n", cfg.MinBenchIterations, cfg.WarmupIterations)
	fmt.Printf("Output: %s\n", cfg.OutputDir)
	fmt.Printf("Formats: %v\n", cfg.OutputFormats)
	if cfg.Filter != "" {
		fmt.Printf("Filter: %s\n", cfg.Filter)
	}
	fmt.Println()

	report, err := runner.Run(ctx)
	if err != nil {
		if interrupted || ctx.Err() != nil {
			fmt.Println("\nBenchmark interrupted by user")
			os.Exit(1)
		}
		log.Fatalf("Benchmark failed: %v", err)
	}

	// Check if interrupted during benchmark
	if interrupted {
		fmt.Println("\nBenchmark interrupted by user")
		os.Exit(1)
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
