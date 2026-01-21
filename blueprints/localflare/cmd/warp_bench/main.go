// Command warp_bench wraps the minio/warp CLI to benchmark S3 drivers.
//
// Install warp: go install github.com/minio/warp@latest
//
// Usage:
//
//	go run ./cmd/warp_bench --quick --drivers minio,rustfs
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
)

func main() {
	var (
		duration    = flag.Duration("duration", 5*time.Second, "Duration per benchmark operation")
		concurrent  = flag.Int("concurrent", 10, "Number of concurrent operations")
		objects     = flag.Int("objects", 20, "Number of objects to use")
		sizes       = flag.String("sizes", "1KiB,1MiB", "Comma-separated object sizes")
		operations  = flag.String("operations", "put,get,stat", "Comma-separated operations to benchmark")
		drivers     = flag.String("drivers", "", "Comma-separated drivers to test (empty = all)")
		outputDir   = flag.String("output", "./pkg/storage/report/warp_bench", "Output directory for reports")
		quick       = flag.Bool("quick", true, "Quick mode (shorter duration, fewer sizes)")
		verbose     = flag.Bool("verbose", false, "Show warp output in real-time")
		dockerClean = flag.Bool("docker-clean", false, "Enable Docker cleanup before/after each driver")
		composeDir  = flag.String("compose-dir", "./docker/s3/all", "Path to docker-compose directory")
		workDir     = flag.String("work-dir", "", "Working directory for warp temp files (empty = auto)")
		keepWorkDir = flag.Bool("keep-workdir", false, "Keep work directory after run")
	)
	flag.Parse()

	cfg := DefaultConfig()
	if *quick {
		cfg = QuickConfig()
	}

	// Override with flags
	if *duration != 30*time.Second || !*quick {
		cfg.Duration = *duration
	}
	cfg.Concurrent = *concurrent
	cfg.Objects = *objects
	cfg.OutputDir = *outputDir
	cfg.Verbose = *verbose
	cfg.DockerClean = *dockerClean
	cfg.ComposeDir = *composeDir
	cfg.WorkDir = *workDir
	cfg.KeepWorkDir = *keepWorkDir

	if *sizes != "" && !*quick {
		cfg.ObjectSizes = strings.Split(*sizes, ",")
	}
	if *operations != "" {
		cfg.Operations = strings.Split(*operations, ",")
	}
	if *drivers != "" {
		cfg.Drivers = strings.Split(*drivers, ",")
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

	fmt.Println("=== Warp S3 Benchmark Suite ===")
	fmt.Printf("Using minio/warp CLI wrapper\n\n")

	runner := NewRunner(cfg)

	results, err := runner.Run(ctx)
	if err != nil && ctx.Err() == nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Generate report even if some benchmarks failed
	if len(results) > 0 {
		report := NewReport(cfg, results)

		if err := report.SaveMarkdown(cfg.OutputDir); err != nil {
			fmt.Printf("Failed to save markdown report: %v\n", err)
		} else {
			fmt.Printf("\nMarkdown report saved to: %s/warp_report.md\n", cfg.OutputDir)
		}

		if err := report.SaveJSON(cfg.OutputDir); err != nil {
			fmt.Printf("Failed to save JSON report: %v\n", err)
		} else {
			fmt.Printf("JSON report saved to: %s/warp_results.json\n", cfg.OutputDir)
		}
	}

	// Print summary
	fmt.Println("\n=== Summary ===")
	successCount := 0
	errorCount := 0
	skippedCount := 0

	for _, res := range results {
		if res.Skipped {
			skippedCount++
		} else if res.Errors > 0 {
			errorCount++
		} else {
			successCount++
		}
	}

	fmt.Printf("  Successful: %d\n", successCount)
	fmt.Printf("  With errors: %d\n", errorCount)
	fmt.Printf("  Skipped: %d\n", skippedCount)

	if errorCount > 0 || ctx.Err() != nil {
		os.Exit(1)
	}
}
