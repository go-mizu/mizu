// Command s3_bench benchmarks S3-compatible storage backends.
//
// Usage:
//
//	go run ./cmd/s3_bench --drivers liteio,minio,rustfs
//	go run ./cmd/s3_bench --quick
//	go run ./cmd/s3_bench --full
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-mizu/blueprints/localflare/cmd/s3_bench/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var cfg Config

	root := &cobra.Command{
		Use:   "s3_bench",
		Short: "S3 Benchmark - Compare S3-compatible storage backends",
		Long: `S3 Benchmark tool inspired by dvassallo/s3-benchmark.

Benchmarks download performance (throughput, TTFB, TTLB) across multiple
S3-compatible storage backends including MinIO, RustFS, and LiteIO.

Examples:
  # Run with default settings
  s3_bench

  # Quick test
  s3_bench --quick

  # Test specific drivers
  s3_bench --drivers liteio,minio,rustfs

  # Custom thread range
  s3_bench --threads-min 4 --threads-max 16

  # Full comprehensive test
  s3_bench --full

Docker Integration:
  # Start docker services, run benchmark, then stop
  s3_bench --docker-up --docker-down --quick

  # Start services from custom compose directory
  s3_bench --docker-up --compose-dir ./docker/s3/all

  # Manually start docker services
  docker compose -f ./docker/s3/all/docker-compose.yaml up -d --wait`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBenchmark(&cfg)
		},
	}

	// Global flags
	flags := root.Flags()
	flags.IntVar(&cfg.ThreadsMin, "threads-min", 8, "Minimum concurrent threads")
	flags.IntVar(&cfg.ThreadsMax, "threads-max", 12, "Maximum concurrent threads")
	flags.IntVar(&cfg.PayloadsMin, "payloads-min", 12, "Min payload size power (2^n KB, 12=4MB)")
	flags.IntVar(&cfg.PayloadsMax, "payloads-max", 14, "Max payload size power (2^n KB, 14=16MB)")
	flags.IntVar(&cfg.Samples, "samples", 100, "Samples per configuration")
	flags.StringSliceVar(&cfg.Drivers, "drivers", nil, "Comma-separated drivers to test (empty=all)")
	flags.StringVar(&cfg.OutputDir, "output", DefaultOutputDir(), "Output directory for reports")
	flags.BoolVar(&cfg.Quick, "quick", false, "Quick mode (fewer samples, smaller range)")
	flags.BoolVar(&cfg.Full, "full", false, "Full comprehensive test")
	flags.BoolVar(&cfg.Verbose, "verbose", false, "Verbose output")
	flags.BoolVar(&cfg.CleanupOnly, "cleanup-only", false, "Only run cleanup")
	flags.StringVar(&cfg.ComposeDir, "compose-dir", "./docker/s3/all", "Docker compose directory")
	flags.BoolVar(&cfg.DockerUp, "docker-up", false, "Start docker-compose services before benchmark")
	flags.BoolVar(&cfg.DockerDown, "docker-down", false, "Stop docker-compose services after benchmark")

	return root.Execute()
}

func runBenchmark(cfg *Config) error {
	// Handle docker-compose up
	if cfg.DockerUp {
		if err := dockerCompose(cfg.ComposeDir, "up", "-d", "--wait"); err != nil {
			return fmt.Errorf("docker-compose up: %w", err)
		}
		fmt.Println("Docker services started, waiting for healthy status...")
		// Give services time to fully initialize
		time.Sleep(5 * time.Second)
	}

	// Handle docker-compose down on exit
	if cfg.DockerDown {
		defer func() {
			fmt.Println("\nStopping docker services...")
			if err := dockerCompose(cfg.ComposeDir, "down"); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: docker-compose down failed: %v\n", err)
			}
		}()
	}

	// Apply mode presets
	if cfg.Quick {
		quickCfg := QuickConfig()
		if cfg.Drivers == nil {
			cfg.Drivers = quickCfg.Drivers
		}
		cfg.ThreadsMin = quickCfg.ThreadsMin
		cfg.ThreadsMax = quickCfg.ThreadsMax
		cfg.PayloadsMin = quickCfg.PayloadsMin
		cfg.PayloadsMax = quickCfg.PayloadsMax
		cfg.Samples = quickCfg.Samples
	} else if cfg.Full {
		fullCfg := FullConfig()
		if cfg.Drivers == nil {
			cfg.Drivers = fullCfg.Drivers
		}
		cfg.ThreadsMin = fullCfg.ThreadsMin
		cfg.ThreadsMax = fullCfg.ThreadsMax
		cfg.PayloadsMin = fullCfg.PayloadsMin
		cfg.PayloadsMax = fullCfg.PayloadsMax
		cfg.Samples = fullCfg.Samples
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

	// Create benchmark runner
	runner := NewBenchmarkRunner(cfg)

	// Check if we should use interactive UI or simple output
	if isTerminal() && !cfg.Verbose {
		return runInteractive(ctx, runner, cfg)
	}
	return runSimple(ctx, runner, cfg)
}

func runInteractive(ctx context.Context, runner *BenchmarkRunner, cfg *Config) error {
	// Create Bubbletea program
	model := ui.NewModel()
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Channel for results
	resultsChan := make(chan *BenchmarkResult, 100)
	doneChan := make(chan error, 1)

	// Set callbacks to send messages to the UI
	runner.SetCallbacks(
		func(phase string, driver string) {
			var uiPhase ui.Phase
			switch phase {
			case "SETUP":
				uiPhase = ui.PhaseSetup
			case "BENCHMARK":
				uiPhase = ui.PhaseBenchmark
			case "CLEANUP":
				uiPhase = ui.PhaseCleanup
			case "DONE":
				uiPhase = ui.PhaseDone
			}
			p.Send(ui.PhaseChangeMsg{Phase: uiPhase, Driver: driver})
		},
		func(current, total int, message string) {
			p.Send(ui.ProgressMsg{Current: current, Total: total, Message: message})
		},
		func(result *BenchmarkResult) {
			resultsChan <- result
		},
		func(message string) {
			p.Send(ui.LogMsg{Message: message})
		},
		func(result *BenchmarkResult) {
			p.Send(ui.BenchmarkResultMsg{
				Driver:     result.Driver,
				ObjectSize: result.ObjectSize,
				Threads:    result.Threads,
				Throughput: result.Throughput,
				TTFBAvg:    result.TTFB.Avg,
				TTFBMin:    result.TTFB.Min,
				TTFBP25:    result.TTFB.P25,
				TTFBP50:    result.TTFB.P50,
				TTFBP75:    result.TTFB.P75,
				TTFBP90:    result.TTFB.P90,
				TTFBP99:    result.TTFB.P99,
				TTFBMax:    result.TTFB.Max,
				TTLBAvg:    result.TTLB.Avg,
				TTLBMin:    result.TTLB.Min,
				TTLBP25:    result.TTLB.P25,
				TTLBP50:    result.TTLB.P50,
				TTLBP75:    result.TTLB.P75,
				TTLBP90:    result.TTLB.P90,
				TTLBP99:    result.TTLB.P99,
				TTLBMax:    result.TTLB.Max,
			})
		},
		func(objectSize int, driver string) {
			p.Send(ui.SectionHeaderMsg{ObjectSize: objectSize})
		},
	)

	// Set dashboard-specific callbacks
	runner.SetDashboardCallbacks(
		func(driver string, throughput float64, timestamp time.Time) {
			p.Send(ui.ThroughputSampleMsg{
				Driver:     driver,
				Throughput: throughput,
				Timestamp:  timestamp,
			})
		},
		func(driver string, completed, total int, throughput float64) {
			p.Send(ui.DriverProgressMsg{
				Driver:     driver,
				Completed:  completed,
				Total:      total,
				Throughput: throughput,
			})
		},
		func(objectSize, threads int) {
			p.Send(ui.ConfigChangeMsg{
				ObjectSize: objectSize,
				Threads:    threads,
			})
		},
	)

	// Set latency callback
	runner.SetLatencyCallback(func(driver string, ttfb, ttlb time.Duration) {
		p.Send(ui.LatencySampleMsg{
			Driver: driver,
			TTFB:   ttfb,
			TTLB:   ttlb,
		})
	})

	// Set driver error callback
	runner.SetDriverErrorCallback(func(driver string, err error) {
		p.Send(ui.DriverErrorMsg{
			Driver: driver,
			Error:  err.Error(),
		})
	})

	// Run benchmark in background
	go func() {
		_, err := runner.Run(ctx)
		doneChan <- err
		p.Send(ui.QuitMsg{})
	}()

	// Run the UI
	if _, err := p.Run(); err != nil {
		return err
	}

	// Wait for benchmark to complete
	if err := <-doneChan; err != nil && ctx.Err() == nil {
		return err
	}

	// Generate reports
	results := runner.Results()
	if len(results) > 0 {
		report := NewReport(cfg, results)
		if err := report.SaveAll(cfg.OutputDir); err != nil {
			return fmt.Errorf("save reports: %w", err)
		}
		fmt.Printf("\nReports saved to: %s\n", cfg.OutputDir)
	}

	return nil
}

func runSimple(ctx context.Context, runner *BenchmarkRunner, cfg *Config) error {
	fmt.Println("S3 Benchmark - Comparing S3-compatible storage backends")
	fmt.Println()

	// Configuration summary
	drivers := cfg.Drivers
	if len(drivers) == 0 {
		drivers = []string{"all available"}
	}
	fmt.Printf("Drivers: %s\n", strings.Join(drivers, ", "))
	fmt.Printf("Object sizes: %s - %s\n",
		FormatSize(1<<(cfg.PayloadsMin+10)),
		FormatSize(1<<(cfg.PayloadsMax+10)))
	fmt.Printf("Threads: %d - %d\n", cfg.ThreadsMin, cfg.ThreadsMax)
	fmt.Printf("Samples: %d per configuration\n", cfg.Samples)
	fmt.Println()

	// Set simple callbacks
	runner.SetCallbacks(
		func(phase string, driver string) {
			fmt.Printf("\n--- %s ", phase)
			fmt.Println(strings.Repeat("-", 70))
			fmt.Println()
		},
		func(current, total int, message string) {
			// Simple progress
			if current == total || current%10 == 0 {
				fmt.Printf("\r%s: %d/%d", message, current, total)
				if current == total {
					fmt.Println()
				}
			}
		},
		func(result *BenchmarkResult) {
			// Result added
		},
		func(message string) {
			fmt.Println(message)
		},
		func(result *BenchmarkResult) {
			// Print result row
			fmt.Printf("| %8s | %3d threads | %8.1f MB/s | TTFB: %4d/%4d/%4d ms | TTLB: %4d/%4d/%4d ms |\n",
				result.Driver, result.Threads, result.Throughput,
				result.TTFB.P50.Milliseconds(), result.TTFB.P90.Milliseconds(), result.TTFB.P99.Milliseconds(),
				result.TTLB.P50.Milliseconds(), result.TTLB.P90.Milliseconds(), result.TTLB.P99.Milliseconds())
		},
		func(objectSize int, driver string) {
			fmt.Printf("\nDownload performance with %s objects\n", FormatSize(objectSize))
			fmt.Println(strings.Repeat("-", 100))
		},
	)

	// Run benchmark
	startTime := time.Now()
	results, err := runner.Run(ctx)
	elapsed := time.Since(startTime)

	if err != nil && ctx.Err() == nil {
		return err
	}

	// Summary
	fmt.Println()
	fmt.Printf("Completed in %s\n", elapsed.Round(time.Second))

	// Generate reports
	if len(results) > 0 {
		report := NewReport(cfg, results)
		if err := report.SaveAll(cfg.OutputDir); err != nil {
			return fmt.Errorf("save reports: %w", err)
		}
		fmt.Printf("Reports saved to: %s\n", cfg.OutputDir)

		// Print best results
		var bestThroughput *BenchmarkResult
		for _, res := range results {
			if bestThroughput == nil || res.Throughput > bestThroughput.Throughput {
				bestThroughput = res
			}
		}
		if bestThroughput != nil {
			fmt.Printf("\nBest throughput: %s (%.1f MB/s with %s, %d threads)\n",
				bestThroughput.Driver, bestThroughput.Throughput,
				FormatSize(bestThroughput.ObjectSize), bestThroughput.Threads)
		}
	}

	return nil
}

func isTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// dockerCompose runs docker-compose with the given arguments.
func dockerCompose(composeDir string, args ...string) error {
	// Resolve compose directory to absolute path
	absDir, err := filepath.Abs(composeDir)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Check if compose file exists
	composeFile := filepath.Join(absDir, "docker-compose.yaml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.yaml not found at %s", absDir)
	}

	// Build command arguments
	cmdArgs := append([]string{"-f", composeFile}, args...)
	cmd := exec.Command("docker", append([]string{"compose"}, cmdArgs...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = absDir

	return cmd.Run()
}
