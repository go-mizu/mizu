// Command cluster_bench orchestrates 3-node cluster benchmarks for
// MinIO, RustFS, SeaweedFS, and Herd, all via S3 API.
//
// Usage:
//
//	go run ./tools/cluster_bench/              # Full run: start clusters, benchmark, report
//	go run ./tools/cluster_bench/ -systems minio,herd  # Only specific systems
//	go run ./tools/cluster_bench/ -quick       # Quick mode (shorter benchmarks)
//	go run ./tools/cluster_bench/ -skip-start  # Skip cluster startup (already running)
//	go run ./tools/cluster_bench/ -skip-stop   # Don't stop clusters after benchmark
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/liteio-dev/liteio/bench"
)

func main() {
	var (
		systems   = flag.String("systems", "minio,rustfs,seaweedfs,herd", "Comma-separated systems to benchmark")
		quick     = flag.Bool("quick", false, "Quick mode (500ms bench time, fewer sizes)")
		benchTime = flag.Duration("benchtime", 1*time.Second, "Target bench time per operation")
		filter    = flag.String("filter", "", "Filter benchmarks by name substring")
		outputDir = flag.String("output", "./report/cluster", "Output directory for reports")
		skipStart = flag.Bool("skip-start", false, "Skip cluster startup (assume already running)")
		skipStop  = flag.Bool("skip-stop", false, "Don't stop clusters after benchmark")
		large     = flag.Bool("large", false, "Include 100MB object benchmarks")
		verbose   = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	systemList := strings.Split(*systems, ",")

	fmt.Println("=== Cluster S3 Benchmark ===")
	fmt.Printf("Systems: %v\n", systemList)
	fmt.Printf("BenchTime: %v, Quick: %v, Large: %v\n", *benchTime, *quick, *large)
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted, cleaning up...")
		cancel()
		<-sigCh
		os.Exit(1)
	}()

	// Map system names to cluster constructors
	constructors := map[string]func() (*Cluster, error){
		"minio":     NewMinIOCluster,
		"rustfs":    NewRustFSCluster,
		"seaweedfs": NewSeaweedFSCluster,
		"herd":      NewHerdCluster,
	}

	var clusters []*Cluster
	defer func() {
		if !*skipStop {
			for _, c := range clusters {
				fmt.Printf("Stopping %s...\n", c.Name)
				c.Stop()
			}
			// Clean up all data
			os.RemoveAll(baseDir)
		}
	}()

	// Phase 1: Start clusters
	if !*skipStart {
		fmt.Println("=== Phase 1: Starting Clusters ===")
		os.MkdirAll(baseDir, 0o755)

		for _, sys := range systemList {
			sys = strings.TrimSpace(sys)
			constructor, ok := constructors[sys]
			if !ok {
				fmt.Printf("  Unknown system: %s (available: minio, rustfs, seaweedfs, herd)\n", sys)
				continue
			}

			fmt.Printf("  Starting %s cluster...\n", sys)
			cluster, err := constructor()
			if err != nil {
				fmt.Printf("  ERROR starting %s: %v\n", sys, err)
				continue
			}
			clusters = append(clusters, cluster)
		}
		fmt.Println()

		// Wait for all clusters to be ready
		fmt.Println("=== Phase 2: Waiting for Clusters ===")
		var readyClusters []*Cluster
		for _, c := range clusters {
			fmt.Printf("  Waiting for %s (health: %s)...\n", c.Name, c.HealthURL)
			if err := c.WaitReady(ctx, 90*time.Second); err != nil {
				fmt.Printf("  ERROR: %s not ready: %v\n", c.Name, err)
				fmt.Printf("  Check logs: %s/logs/\n", baseDir)
				continue
			}
			fmt.Printf("  %s: ready\n", c.Name)
			readyClusters = append(readyClusters, c)
		}
		clusters = readyClusters
		fmt.Println()

		// Create test buckets
		fmt.Println("=== Phase 3: Creating Test Buckets ===")
		var benchableClusters []*Cluster
		for _, c := range clusters {
			fmt.Printf("  Creating bucket on %s...\n", c.Name)
			var lastErr error
			for attempt := 0; attempt < 5; attempt++ {
				if attempt > 0 {
					time.Sleep(3 * time.Second)
				}
				lastErr = createBucket(c.S3Endpoint(), c.AccessKey, c.SecretKey, "test-bucket")
				if lastErr == nil {
					break
				}
			}
			if lastErr != nil {
				fmt.Printf("  ERROR: %v\n", lastErr)
				continue
			}
			fmt.Printf("  %s: bucket created\n", c.Name)
			benchableClusters = append(benchableClusters, c)
		}
		clusters = benchableClusters
		fmt.Println()
	} else {
		// Build cluster objects for already-running services
		for _, sys := range systemList {
			sys = strings.TrimSpace(sys)
			switch sys {
			case "minio":
				clusters = append(clusters, &Cluster{
					Name: "minio_cluster", S3Port: 9000,
					AccessKey: "minioadmin", SecretKey: "minioadmin",
				})
			case "rustfs":
				clusters = append(clusters, &Cluster{
					Name: "rustfs_cluster", S3Port: 9100,
					AccessKey: "rustfsadmin", SecretKey: "rustfsadmin",
				})
			case "seaweedfs":
				clusters = append(clusters, &Cluster{
					Name: "seaweedfs_cluster", S3Port: 8333,
					AccessKey: "admin", SecretKey: "adminpassword",
				})
			case "herd":
				clusters = append(clusters, &Cluster{
					Name: "herd_cluster", S3Port: 9230,
					AccessKey: "herd", SecretKey: "herd123",
				})
			}
		}
	}

	if len(clusters) == 0 {
		log.Fatal("No clusters available for benchmarking")
	}

	// Phase 4: Run benchmarks
	fmt.Println("=== Phase 4: Running Benchmarks ===")

	// Build driver configs for the bench runner
	var driverConfigs []bench.DriverConfig
	for _, c := range clusters {
		driverConfigs = append(driverConfigs, bench.DriverConfig{
			Name:    c.Name,
			DSN:     c.DSN(),
			Bucket:  "test-bucket",
			Enabled: true,
		})
	}

	// Configure benchmark
	cfg := bench.DefaultConfig()
	if *quick {
		cfg = bench.QuickConfig()
	}
	cfg.BenchTime = *benchTime
	cfg.OutputDir = *outputDir
	cfg.DockerStats = false // No docker containers
	cfg.CleanupDataPaths = false
	cfg.CleanupDockerData = false
	cfg.ResourceTracking = false
	cfg.Verbose = *verbose
	cfg.OutputFormats = []string{"markdown", "json", "csv"}
	cfg.Filter = *filter
	cfg.Progress = true
	if *large {
		cfg.EnableLargeObjects()
	}

	// Override the default drivers with our cluster configs
	cfg.Drivers = make([]string, len(driverConfigs))
	for i, d := range driverConfigs {
		cfg.Drivers[i] = d.Name
	}

	// Create a custom runner that uses our driver configs
	runner := bench.NewRunnerWithDrivers(cfg, driverConfigs)
	runner.SetLogger(func(format string, args ...any) {
		fmt.Printf(format+"\n", args...)
	})

	report, err := runner.Run(ctx)
	if err != nil {
		if ctx.Err() != nil {
			fmt.Println("\nBenchmark interrupted")
			os.Exit(1)
		}
		log.Fatalf("Benchmark failed: %v", err)
	}

	// Phase 5: Save reports
	fmt.Println()
	fmt.Println("=== Phase 5: Generating Reports ===")

	os.MkdirAll(*outputDir, 0o755)

	// Save standard reports
	if err := report.SaveAll(*outputDir, cfg.OutputFormats); err != nil {
		log.Fatalf("Save reports failed: %v", err)
	}

	// Generate cluster comparison report
	generateClusterReport(report, *outputDir)

	fmt.Printf("\nReports saved to %s/\n", *outputDir)

	// Print summary
	fmt.Println()
	fmt.Println("=== Summary ===")
	for _, c := range clusters {
		count := 0
		errors := 0
		for _, m := range report.Results {
			if m.Driver == c.Name {
				count++
				errors += m.Errors
			}
		}
		fmt.Printf("  %s: %d benchmarks, %d errors\n", c.Name, count, errors)
	}
}

// generateClusterReport creates a detailed cluster comparison markdown report.
func generateClusterReport(report *bench.Report, outputDir string) {
	path := filepath.Join(outputDir, "cluster_comparison.md")
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("Warning: cannot create cluster report: %v\n", err)
		return
	}
	defer f.Close()

	w := func(format string, args ...any) {
		fmt.Fprintf(f, format+"\n", args...)
	}

	w("# Cluster S3 Benchmark: MinIO vs RustFS vs SeaweedFS vs Herd")
	w("")
	w("**Date**: %s", time.Now().Format("2006-01-02 15:04:05"))
	w("**Mode**: 3-node cluster, all via S3 HTTP API")
	w("")

	// Collect all drivers
	driverSet := map[string]bool{}
	for _, m := range report.Results {
		driverSet[m.Driver] = true
	}
	drivers := make([]string, 0, len(driverSet))
	for d := range driverSet {
		drivers = append(drivers, d)
	}
	sort.Strings(drivers)

	// Group results by operation
	type opResult struct {
		driver     string
		throughput float64
		opsPerSec  float64
		avgLatency time.Duration
		p99Latency time.Duration
		errors     int
	}

	opResults := map[string][]opResult{}
	for _, m := range report.Results {
		opResults[m.Operation] = append(opResults[m.Operation], opResult{
			driver:     m.Driver,
			throughput: m.Throughput,
			opsPerSec:  m.OpsPerSec,
			avgLatency: m.AvgLatency,
			p99Latency: m.P99Latency,
			errors:     m.Errors,
		})
	}

	// Get sorted operation names
	ops := make([]string, 0, len(opResults))
	for op := range opResults {
		ops = append(ops, op)
	}
	sort.Strings(ops)

	// --- Section 1: Sequential Write Throughput ---
	w("## Sequential Write Throughput (MB/s)")
	w("")
	w("| Size | %s |", strings.Join(drivers, " | "))
	w("|------|%s|", strings.Repeat(" --- |", len(drivers)))

	writeOps := filterOps(ops, "Write/")
	for _, op := range writeOps {
		results := opResults[op]
		cells := make([]string, len(drivers))
		best := 0.0
		for _, r := range results {
			if r.throughput > best {
				best = r.throughput
			}
		}
		for i, d := range drivers {
			cells[i] = "-"
			for _, r := range results {
				if r.driver == d {
					val := fmt.Sprintf("%.1f", r.throughput)
					if r.throughput == best && best > 0 {
						val = "**" + val + "**"
					}
					cells[i] = val
					break
				}
			}
		}
		w("| %s | %s |", op, strings.Join(cells, " | "))
	}
	w("")

	// --- Section 2: Sequential Read Throughput ---
	w("## Sequential Read Throughput (MB/s)")
	w("")
	w("| Size | %s |", strings.Join(drivers, " | "))
	w("|------|%s|", strings.Repeat(" --- |", len(drivers)))

	readOps := filterOps(ops, "Read/")
	for _, op := range readOps {
		results := opResults[op]
		cells := make([]string, len(drivers))
		best := 0.0
		for _, r := range results {
			if r.throughput > best {
				best = r.throughput
			}
		}
		for i, d := range drivers {
			cells[i] = "-"
			for _, r := range results {
				if r.driver == d {
					val := fmt.Sprintf("%.1f", r.throughput)
					if r.throughput == best && best > 0 {
						val = "**" + val + "**"
					}
					cells[i] = val
					break
				}
			}
		}
		w("| %s | %s |", op, strings.Join(cells, " | "))
	}
	w("")

	// --- Section 3: Metadata Operations ---
	w("## Metadata Operations (ops/s)")
	w("")
	w("| Operation | %s |", strings.Join(drivers, " | "))
	w("|-----------|%s|", strings.Repeat(" --- |", len(drivers)))

	metaOps := filterOps(ops, "Stat", "List/", "Delete")
	for _, op := range metaOps {
		results := opResults[op]
		cells := make([]string, len(drivers))
		best := 0.0
		for _, r := range results {
			if r.opsPerSec > best {
				best = r.opsPerSec
			}
		}
		for i, d := range drivers {
			cells[i] = "-"
			for _, r := range results {
				if r.driver == d {
					val := formatOps(r.opsPerSec)
					if r.opsPerSec == best && best > 0 {
						val = "**" + val + "**"
					}
					cells[i] = val
					break
				}
			}
		}
		w("| %s | %s |", op, strings.Join(cells, " | "))
	}
	w("")

	// --- Section 4: Concurrency Scaling ---
	w("## Concurrency Scaling — Parallel Write (MB/s)")
	w("")
	w("| Concurrency | %s |", strings.Join(drivers, " | "))
	w("|-------------|%s|", strings.Repeat(" --- |", len(drivers)))

	parallelWriteOps := filterOps(ops, "ParallelWrite/")
	for _, op := range parallelWriteOps {
		results := opResults[op]
		cells := make([]string, len(drivers))
		best := 0.0
		for _, r := range results {
			if r.throughput > best {
				best = r.throughput
			}
		}
		for i, d := range drivers {
			cells[i] = "-"
			for _, r := range results {
				if r.driver == d {
					val := fmt.Sprintf("%.1f", r.throughput)
					if r.throughput == best && best > 0 {
						val = "**" + val + "**"
					}
					cells[i] = val
					break
				}
			}
		}
		w("| %s | %s |", op, strings.Join(cells, " | "))
	}
	w("")

	// --- Section 5: Concurrency Scaling — Parallel Read ---
	w("## Concurrency Scaling — Parallel Read (MB/s)")
	w("")
	w("| Concurrency | %s |", strings.Join(drivers, " | "))
	w("|-------------|%s|", strings.Repeat(" --- |", len(drivers)))

	parallelReadOps := filterOps(ops, "ParallelRead/")
	for _, op := range parallelReadOps {
		results := opResults[op]
		cells := make([]string, len(drivers))
		best := 0.0
		for _, r := range results {
			if r.throughput > best {
				best = r.throughput
			}
		}
		for i, d := range drivers {
			cells[i] = "-"
			for _, r := range results {
				if r.driver == d {
					val := fmt.Sprintf("%.1f", r.throughput)
					if r.throughput == best && best > 0 {
						val = "**" + val + "**"
					}
					cells[i] = val
					break
				}
			}
		}
		w("| %s | %s |", op, strings.Join(cells, " | "))
	}
	w("")

	// --- Section 6: Mixed Workload ---
	w("## Mixed Workload Throughput (MB/s)")
	w("")
	w("| Workload | %s |", strings.Join(drivers, " | "))
	w("|----------|%s|", strings.Repeat(" --- |", len(drivers)))

	mixedOps := filterOps(ops, "MixedWorkload/")
	for _, op := range mixedOps {
		results := opResults[op]
		cells := make([]string, len(drivers))
		best := 0.0
		for _, r := range results {
			if r.throughput > best {
				best = r.throughput
			}
		}
		for i, d := range drivers {
			cells[i] = "-"
			for _, r := range results {
				if r.driver == d {
					val := fmt.Sprintf("%.1f", r.throughput)
					if r.throughput == best && best > 0 {
						val = "**" + val + "**"
					}
					cells[i] = val
					break
				}
			}
		}
		w("| %s | %s |", op, strings.Join(cells, " | "))
	}
	w("")

	// --- Section 7: Latency (P99) ---
	w("## Latency P99 (lower is better)")
	w("")
	w("| Operation | %s |", strings.Join(drivers, " | "))
	w("|-----------|%s|", strings.Repeat(" --- |", len(drivers)))

	latencyOps := filterOps(ops, "Write/1KB", "Read/1KB", "Stat", "List/100", "Delete")
	for _, op := range latencyOps {
		results := opResults[op]
		cells := make([]string, len(drivers))
		best := time.Duration(1<<63 - 1)
		for _, r := range results {
			if r.p99Latency > 0 && r.p99Latency < best {
				best = r.p99Latency
			}
		}
		for i, d := range drivers {
			cells[i] = "-"
			for _, r := range results {
				if r.driver == d && r.p99Latency > 0 {
					val := formatDuration(r.p99Latency)
					if r.p99Latency == best {
						val = "**" + val + "**"
					}
					cells[i] = val
					break
				}
			}
		}
		w("| %s | %s |", op, strings.Join(cells, " | "))
	}
	w("")

	// --- Section 8: Performance Leaders ---
	w("## Performance Leaders")
	w("")
	w("| Category | Winner | Throughput/Ops |")
	w("|----------|--------|----------------|")

	categories := []struct {
		name   string
		prefix string
		metric string // "throughput" or "ops"
	}{
		{"Small Write (1KB)", "Write/1KB", "throughput"},
		{"Medium Write (64KB)", "Write/64KB", "throughput"},
		{"Large Write (1MB)", "Write/1MB", "throughput"},
		{"Small Read (1KB)", "Read/1KB", "throughput"},
		{"Medium Read (64KB)", "Read/64KB", "throughput"},
		{"Large Read (1MB)", "Read/1MB", "throughput"},
		{"Stat (HEAD)", "Stat", "ops"},
		{"List/100", "List/100", "ops"},
		{"Parallel Write C100", "ParallelWrite/1KB/C100", "throughput"},
		{"Parallel Read C100", "ParallelRead/1KB/C100", "throughput"},
	}

	for _, cat := range categories {
		results, ok := opResults[cat.prefix]
		if !ok {
			continue
		}
		var bestDriver string
		var bestVal float64
		for _, r := range results {
			var val float64
			if cat.metric == "throughput" {
				val = r.throughput
			} else {
				val = r.opsPerSec
			}
			if val > bestVal {
				bestVal = val
				bestDriver = r.driver
			}
		}
		if bestDriver != "" {
			var valStr string
			if cat.metric == "throughput" {
				valStr = fmt.Sprintf("%.1f MB/s", bestVal)
			} else {
				valStr = formatOps(bestVal) + " ops/s"
			}
			w("| %s | **%s** | %s |", cat.name, bestDriver, valStr)
		}
	}
	w("")

	// --- Section 9: Errors ---
	w("## Error Summary")
	w("")
	hasErrors := false
	for _, d := range drivers {
		errCount := 0
		for _, m := range report.Results {
			if m.Driver == d {
				errCount += m.Errors
			}
		}
		if errCount > 0 {
			hasErrors = true
			w("- **%s**: %d errors", d, errCount)
		}
	}
	if !hasErrors {
		w("No errors during benchmarks.")
	}
	w("")

	// --- Footer ---
	w("---")
	w("*Generated by cluster_bench on %s*", time.Now().Format("2006-01-02 15:04:05"))
}

// filterOps returns ops matching any of the given prefixes.
func filterOps(ops []string, prefixes ...string) []string {
	var result []string
	for _, op := range ops {
		for _, prefix := range prefixes {
			if strings.HasPrefix(op, prefix) || op == prefix {
				result = append(result, op)
				break
			}
		}
	}
	return result
}

func formatOps(ops float64) string {
	switch {
	case ops >= 1_000_000:
		return fmt.Sprintf("%.2fM", ops/1_000_000)
	case ops >= 1_000:
		return fmt.Sprintf("%.1fK", ops/1_000)
	default:
		return fmt.Sprintf("%.0f", ops)
	}
}

func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.2fms", float64(d)/float64(time.Millisecond))
	default:
		return fmt.Sprintf("%.1fus", float64(d)/float64(time.Microsecond))
	}
}

// writeCSVSummary exports a CSV comparison file.
func writeCSVSummary(report *bench.Report, outputDir string) {
	path := filepath.Join(outputDir, "cluster_comparison.csv")
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"Operation", "Driver", "Throughput_MBs", "OpsPerSec", "AvgLatency_us", "P99Latency_us", "Errors"})
	for _, m := range report.Results {
		w.Write([]string{
			m.Operation,
			m.Driver,
			fmt.Sprintf("%.2f", m.Throughput),
			fmt.Sprintf("%.2f", m.OpsPerSec),
			fmt.Sprintf("%.2f", float64(m.AvgLatency)/float64(time.Microsecond)),
			fmt.Sprintf("%.2f", float64(m.P99Latency)/float64(time.Microsecond)),
			fmt.Sprintf("%d", m.Errors),
		})
	}
}

// writeJSONResults exports a JSON file with all results.
func writeJSONResults(report *bench.Report, outputDir string) {
	path := filepath.Join(outputDir, "cluster_results.json")
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.Encode(report.Results)
}
