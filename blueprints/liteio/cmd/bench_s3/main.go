// Command bench_s3 runs S3 protocol benchmarks using the AWS SDK v2 directly.
//
// Unlike cmd/bench which uses the storage.Storage abstraction, this tool
// exercises the raw S3 API path — the same path real applications use.
//
// Usage:
//
//	go run ./cmd/bench_s3/
//	go run ./cmd/bench_s3/ -quick
//	go run ./cmd/bench_s3/ -drivers minio,liteio,herd_s3
//	go run ./cmd/bench_s3/ -filter PutObject
//	go run ./cmd/bench_s3/ -docker-up -docker-down
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
	"strings"
	"syscall"
	"time"

	"github.com/liteio-dev/liteio/bench_s3"
)

func main() {
	var (
		quick     = flag.Bool("quick", false, "Quick mode (500ms per benchmark)")
		benchTime = flag.Duration("benchtime", 1*time.Second, "Target duration per benchmark")
		warmup    = flag.Int("warmup", 10, "Warmup iterations")
		outputDir = flag.String("output", "./report/s3", "Output directory for reports")
		drivers   = flag.String("drivers", "", "Comma-separated driver names to benchmark (empty = all)")
		filter    = flag.String("filter", "", "Filter benchmarks by operation name (substring)")
		formats   = flag.String("formats", "markdown,json", "Output formats")
		verbose   = flag.Bool("verbose", false, "Verbose output")
		progress  = flag.Bool("progress", false, "Live progress output")

		composeDir = flag.String("compose-dir", "./docker/s3/all", "Docker compose directory")
		dockerUp   = flag.Bool("docker-up", false, "Start docker-compose before benchmarks")
		dockerDown = flag.Bool("docker-down", false, "Stop docker-compose after benchmarks")
	)
	flag.Parse()

	cfg := bench_s3.DefaultConfig()
	if *quick {
		cfg = bench_s3.QuickConfig()
	}
	cfg.BenchTime = *benchTime
	cfg.WarmupIters = *warmup
	cfg.OutputDir = *outputDir
	cfg.Filter = *filter
	cfg.Verbose = *verbose
	cfg.Progress = *progress
	cfg.OutputFormats = strings.Split(*formats, ",")

	if *quick {
		cfg.BenchTime = 500 * time.Millisecond
		cfg.WarmupIters = 3
	}

	if *drivers != "" {
		cfg.Drivers = strings.Split(*drivers, ",")
	}

	// Signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interrupted := false
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		fmt.Printf("\nReceived %v, stopping...\n", sig)
		interrupted = true
		cancel()
		select {
		case <-sigCh:
			fmt.Println("\nForce exit")
			os.Exit(1)
		case <-time.After(30 * time.Second):
			os.Exit(1)
		}
	}()

	// Docker lifecycle
	if *dockerUp {
		fmt.Println("=== Starting Docker Services ===")
		if err := dockerCompose(*composeDir, "up", "-d", "--wait"); err != nil {
			log.Fatalf("docker compose up failed: %v", err)
		}
		fmt.Println("Docker services started, waiting 5s for healthy status...")
		time.Sleep(5 * time.Second)
		if interrupted {
			os.Exit(1)
		}
		fmt.Println()
	}

	if *dockerDown {
		defer func() {
			fmt.Println("\nStopping docker services...")
			if err := dockerCompose(*composeDir, "down"); err != nil {
				fmt.Printf("Warning: docker compose down failed: %v\n", err)
			}
		}()
	}

	runner := bench_s3.NewRunner(cfg)
	runner.SetLogger(func(format string, args ...any) {
		fmt.Printf(format+"\n", args...)
	})

	report, err := runner.Run(ctx)
	if err != nil {
		if interrupted {
			fmt.Println("\nBenchmark interrupted")
			os.Exit(1)
		}
		log.Fatalf("Benchmark failed: %v", err)
	}

	if interrupted {
		fmt.Println("\nBenchmark interrupted")
		os.Exit(1)
	}

	// Save reports
	if err := report.SaveAll(cfg.OutputDir, cfg.OutputFormats); err != nil {
		log.Fatalf("Save reports failed: %v", err)
	}

	fmt.Printf("\nReports saved to %s\n", cfg.OutputDir)

	// Summary
	fmt.Println("\n=== Summary ===")
	driverResults := map[string]int{}
	driverErrors := map[string]int{}
	for _, m := range report.Results {
		driverResults[m.Driver]++
		driverErrors[m.Driver] += m.Errors
	}
	for driver, count := range driverResults {
		fmt.Printf("  %s: %d benchmarks, %d errors\n", driver, count, driverErrors[driver])
	}
}

func dockerCompose(composeDir string, args ...string) error {
	absDir, err := filepath.Abs(composeDir)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	composeFile := filepath.Join(absDir, "docker-compose.yaml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.yaml not found at %s", absDir)
	}
	cmdArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmd := exec.Command("docker", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = absDir
	return cmd.Run()
}
