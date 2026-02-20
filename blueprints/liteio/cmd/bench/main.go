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

	"github.com/liteio-dev/liteio/bench"
)

func main() {
	var (
		warmup        = flag.Int("warmup", 10, "Number of warmup iterations")
		timeout       = flag.Duration("timeout", 30*time.Second, "Per-operation timeout")
		outputDir     = flag.String("output", "./report", "Output directory for reports")
		quick         = flag.Bool("quick", false, "Quick mode (shorter benchmark time)")
		drivers       = flag.String("drivers", "", "Comma-separated list of drivers to benchmark (empty = all)")
		outputFormats = flag.String("formats", "markdown,json,csv", "Output formats (markdown,json,csv)")
		dockerStats   = flag.Bool("docker-stats", true, "Collect Docker container statistics and cleanup after each driver")
		verbose       = flag.Bool("verbose", false, "Verbose output")
		lowOverhead   = flag.Bool("low-overhead", true, "Enable low overhead client mode for benchmarks")
		progress      = flag.Bool("progress", false, "Enable live progress output")
		progressEvery = flag.Int("progress-every", 256, "Progress update frequency (iterations)")
		perOpTimeouts = flag.Bool("per-op-timeouts", false, "Enable per-operation timeouts (adds client overhead)")
		readBufSize   = flag.Int("read-buffer-size", 256*1024, "Read buffer size for io.CopyBuffer")
		enableTTFB    = flag.Bool("enable-ttfb", false, "Capture time-to-first-byte for reads")
		large         = flag.Bool("large", false, "Include 100MB object benchmarks")
		scales        = flag.String("scales", "", "Comma-separated scale counts to benchmark (empty = defaults)")
		objectCounts  = flag.String("object-counts", "", "Deprecated: use --scales (comma-separated object counts)")
		scaleSize     = flag.Int("scale-object-size", 256, "Object size in bytes for Scale benchmarks")
		scaleMaxBytes = flag.Int64("scale-max-bytes", 2*1024*1024*1024, "Max total bytes per Scale test (safety cap)")
		cleanupData   = flag.Bool("cleanup-data", true, "Cleanup local benchmark data paths after each driver run")
		cleanupDocker = flag.Bool("cleanup-docker-data", true, "Cleanup docker volume data paths after each driver run")
		filter           = flag.String("filter", "", "Filter benchmarks by name (substring match, e.g., 'MixedWorkload')")
		resourceTracking = flag.Bool("resource-tracking", true, "Track Go runtime memory and disk usage for embedded drivers")
		// Go-style adaptive benchmark settings (same defaults as 'go test -bench')
		benchTime = flag.Duration("benchtime", 1*time.Second, "Target duration for each benchmark (e.g., 1s, 500ms, 2s)")
		minIters  = flag.Int("min-iters", 3, "Minimum iterations for statistical significance")
		// Docker compose settings
		composeDir = flag.String("compose-dir", "./docker/s3/all", "Docker compose directory for S3 services")
		dockerUp   = flag.Bool("docker-up", false, "Start docker-compose services before benchmark")
		dockerDown = flag.Bool("docker-down", false, "Stop docker-compose services after benchmark")
	)
	flag.Parse()

	// Check for subcommand: "limits"
	args := flag.Args()
	if len(args) > 0 && args[0] == "limits" {
		runLimits(*outputDir, *benchTime)
		return
	}

	cfg := bench.DefaultConfig()
	cfg.WarmupIterations = *warmup
	cfg.Timeout = *timeout
	cfg.OutputDir = *outputDir
	cfg.DockerStats = *dockerStats
	cfg.Verbose = *verbose
	cfg.BenchTime = *benchTime
	cfg.MinBenchIterations = *minIters
	cfg.ScaleObjectSize = *scaleSize
	cfg.ScaleMaxBytes = *scaleMaxBytes
	cfg.CleanupDataPaths = *cleanupData
	cfg.CleanupDockerData = *cleanupDocker
	cfg.LowOverhead = *lowOverhead
	cfg.Progress = *progress
	cfg.ProgressEvery = *progressEvery
	cfg.PerOpTimeouts = *perOpTimeouts
	cfg.ReadBufferSize = *readBufSize
	cfg.EnableTTFB = *enableTTFB
	if *large {
		cfg.EnableLargeObjects()
	}

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
	cfg.ResourceTracking = *resourceTracking

	// Parse scale counts
	countsInput := strings.TrimSpace(*scales)
	if *objectCounts != "" {
		fmt.Println("Warning: --object-counts is deprecated; use --scales")
		countsInput = *objectCounts
	}
	if countsInput != "" {
		parts := strings.Split(countsInput, ",")
		counts := make([]int, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				counts = append(counts, n)
			}
		}
		if len(counts) > 0 {
			cfg.ScaleCounts = counts
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

	// Start docker-compose services if requested
	if *dockerUp {
		fmt.Println("=== Starting Docker Services ===")
		if err := dockerCompose(*composeDir, "up", "-d", "--wait"); err != nil {
			log.Fatalf("docker compose up failed: %v", err)
		}
		fmt.Println("Docker services started, waiting for healthy status...")
		time.Sleep(5 * time.Second)

		// Check if interrupted
		if interrupted {
			fmt.Println("Benchmark cancelled during docker startup")
			os.Exit(1)
		}
		fmt.Println()
	}

	// Handle docker-compose down on exit
	if *dockerDown {
		defer func() {
			fmt.Println("\nStopping docker services...")
			if err := dockerCompose(*composeDir, "down"); err != nil {
				fmt.Printf("Warning: docker compose down failed: %v\n", err)
			}
		}()
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
	fmt.Printf("Scale: counts=%v, size=%dB, cap=%dB\n", cfg.ScaleCounts, cfg.ScaleObjectSize, cfg.ScaleMaxBytes)
	fmt.Printf("Client: low-overhead=%v, per-op-timeouts=%v, progress=%v (every %d), read-buffer=%dB, ttfb=%v\n",
		cfg.LowOverhead, cfg.PerOpTimeouts, cfg.Progress, cfg.ProgressEvery, cfg.ReadBufferSize, cfg.EnableTTFB)
	fmt.Println("Disk note: if you see /Users/apple/Library/Containers/com.docker.docker/Data/log/vm/init.log: no space left on device, reduce --scales or --scale-object-size.")
	fmt.Println("Cleanup: local benchmark data paths (/tmp/usagi-bench, /tmp/rabbit-bench) are removed after each driver run.")
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
		driverNames := make([]string, 0, len(errorDetails))
		for driver := range errorDetails {
			driverNames = append(driverNames, driver)
		}
		sort.Strings(driverNames)
		for _, driver := range driverNames {
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

	// Print runtime resource usage for embedded drivers
	if len(report.ResourceSnapshots) > 0 {
		fmt.Println()
		fmt.Println("=== Resource Usage (Embedded) ===")
		fmt.Printf("  %-12s %10s %10s %10s %10s %8s\n", "Driver", "Peak RSS", "Go Heap", "Go Sys", "Disk", "GC")
		rsDrivers := make([]string, 0, len(report.ResourceSnapshots))
		for d := range report.ResourceSnapshots {
			rsDrivers = append(rsDrivers, d)
		}
		sort.Strings(rsDrivers)
		for _, d := range rsDrivers {
			rs := report.ResourceSnapshots[d]
			fmt.Printf("  %-12s %8.1f MB %8.1f MB %8.1f MB %8.1f MB %8d\n",
				d, rs.PeakRSSMB, rs.PeakHeapMB, rs.PeakSysMB, rs.FinalDiskMB, rs.NumGC)
		}
	}

	// Print server-side metrics for Docker containers
	if len(report.ServerMetrics) > 0 {
		fmt.Println()
		fmt.Println("=== Server-Side Resource Usage ===")
		fmt.Printf("  %-14s %10s %10s %10s %10s %10s %10s %8s\n",
			"Driver", "Memory", "Disk", "Net In", "Net Out", "Block R", "Block W", "CPU%")
		smDrivers := make([]string, 0, len(report.ServerMetrics))
		for d := range report.ServerMetrics {
			smDrivers = append(smDrivers, d)
		}
		sort.Strings(smDrivers)
		for _, d := range smDrivers {
			sm := report.ServerMetrics[d]
			fmt.Printf("  %-14s %8.1f MB %8.1f MB %8.1f MB %8.1f MB %8.1f MB %8.1f MB %7.1f%%\n",
				d, sm.AfterMemoryMB, sm.AfterDiskMB, sm.NetInTotalMB, sm.NetOutTotalMB,
				sm.BlockReadMB, sm.BlockWriteMB, sm.AfterCPU)
		}
	}

	os.Exit(0)
}

// runLimits runs the physical limits benchmark subcommand.
func runLimits(outputDir string, benchTime time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nCancelling...")
		cancel()
		<-sigCh
		os.Exit(1)
	}()

	cfg := bench.LimitsConfig{
		BenchTime: benchTime,
		OutputDir: outputDir,
	}

	if err := bench.RunLimits(ctx, cfg); err != nil {
		log.Fatalf("Physical limits benchmark failed: %v", err)
	}
}

// dockerCompose runs docker-compose with the given arguments.
func dockerCompose(composeDir string, args ...string) error {
	absDir, err := filepath.Abs(composeDir)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	composeFile := filepath.Join(absDir, "docker-compose.yaml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.yaml not found at %s", absDir)
	}

	cmdArgs := append([]string{"-f", composeFile}, args...)
	cmd := exec.Command("docker", append([]string{"compose"}, cmdArgs...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = absDir

	return cmd.Run()
}
