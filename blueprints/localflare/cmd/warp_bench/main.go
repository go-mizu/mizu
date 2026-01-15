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
		duration    = flag.Duration("duration", 30*time.Second, "Duration per benchmark operation")
		concurrent  = flag.Int("concurrent", 20, "Number of concurrent operations")
		objects     = flag.Int("objects", 100, "Number of objects to use")
		sizes       = flag.String("sizes", "1KiB,64KiB,1MiB,10MiB", "Comma-separated object sizes")
		operations  = flag.String("operations", "put,get,delete,list,stat,mixed", "Comma-separated operations to benchmark")
		drivers     = flag.String("drivers", "", "Comma-separated drivers to test (empty = all)")
		outputDir   = flag.String("output", "./pkg/storage/report", "Output directory for reports")
		quick       = flag.Bool("quick", false, "Quick mode (shorter duration, fewer sizes)")
		verbose     = flag.Bool("verbose", false, "Show warp output in real-time")
		dockerClean = flag.Bool("docker-clean", true, "Enable Docker cleanup before/after each driver")
		composeDir  = flag.String("compose-dir", "./docker/s3/all", "Path to docker-compose directory")
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
