package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Result holds the benchmark results for a single operation.
type Result struct {
	Driver         string
	Operation      string
	ObjectSize     string
	ThroughputMBps float64
	OpsPerSec      float64
	LatencyAvgMs   float64
	LatencyP50Ms   float64
	LatencyP99Ms   float64
	TotalRequests  int
	Errors         int
	Duration       time.Duration
	RawOutput      string
	Skipped        bool
	SkipReason     string
}

// Runner executes warp benchmarks against S3 drivers.
type Runner struct {
	config   *Config
	warpPath string
	logger   func(format string, args ...any)
}

// NewRunner creates a new benchmark runner.
func NewRunner(cfg *Config) *Runner {
	return &Runner{
		config: cfg,
		logger: func(format string, args ...any) {
			fmt.Printf(format+"\n", args...)
		},
	}
}

// SetLogger sets the logger function.
func (r *Runner) SetLogger(logger func(format string, args ...any)) {
	r.logger = logger
}

// CheckWarp verifies warp is installed and returns the path.
func (r *Runner) CheckWarp() error {
	path, err := exec.LookPath("warp")
	if err != nil {
		return fmt.Errorf("warp not found in PATH. Install with: go install github.com/minio/warp@latest")
	}
	r.warpPath = path
	r.logger("Found warp at: %s", path)

	// Get version
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err == nil {
		r.logger("Warp version: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// CheckDriver verifies a driver is accessible.
func (r *Runner) CheckDriver(ctx context.Context, driver *DriverConfig) error {
	conn, err := net.DialTimeout("tcp", driver.Endpoint, 5*time.Second)
	if err != nil {
		return fmt.Errorf("cannot connect to %s at %s: %w", driver.Name, driver.Endpoint, err)
	}
	conn.Close()
	return nil
}

// RunBenchmark executes a single warp benchmark.
func (r *Runner) RunBenchmark(ctx context.Context, driver *DriverConfig, op string, size string) (*Result, error) {
	result := &Result{
		Driver:     driver.Name,
		Operation:  op,
		ObjectSize: size,
	}

	// Build warp command arguments
	args := []string{
		op,
		"--host=" + driver.Endpoint,
		"--access-key=" + driver.AccessKey,
		"--secret-key=" + driver.SecretKey,
		"--bucket=" + driver.Bucket,
		"--duration=" + r.config.Duration.String(),
		"--concurrent=" + strconv.Itoa(r.config.Concurrent),
		"--no-color",
		"--insecure", // Allow HTTP endpoints
	}

	// Add object size for data operations (not for list)
	if size != "" && op != "list" {
		args = append(args, "--obj.size="+size)
	}

	// Handle operation-specific flags
	switch op {
	case "put":
		// put doesn't take --objects, it just runs continuously
		// No additional flags needed

	case "get", "stat":
		// get and stat take --objects
		args = append(args, "--objects="+strconv.Itoa(r.config.Objects))

	case "delete":
		// delete needs many objects: batch (100) * concurrent * 4 = 8000 minimum
		// Use a reasonable number for benchmarking
		deleteObjects := r.config.Concurrent * 100 * 4
		if deleteObjects < 8000 {
			deleteObjects = 8000
		}
		args = append(args, "--objects="+strconv.Itoa(deleteObjects))
		args = append(args, "--batch=100")

	case "list":
		// list uses --objects to create test objects first
		args = append(args, "--objects="+strconv.Itoa(r.config.Objects))

	case "mixed":
		// mixed workload with distribution
		args = append(args,
			"--objects="+strconv.Itoa(r.config.Objects),
			"--get-distrib=45",
			"--put-distrib=25",
			"--delete-distrib=15",
			"--stat-distrib=15",
		)

	case "multipart":
		// multipart upload benchmark
		args = append(args,
			"--parts=3",
			"--part.size=5MiB",
		)
	}

	if r.config.Verbose {
		r.logger("  Running: warp %s", strings.Join(args, " "))
	}

	cmd := exec.CommandContext(ctx, r.warpPath, args...)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	result.Duration = time.Since(startTime)

	// Combine output
	output := stdout.String() + stderr.String()
	result.RawOutput = output

	if err != nil {
		// Check for context cancellation
		if ctx.Err() != nil {
			result.Skipped = true
			result.SkipReason = "cancelled"
			return result, nil
		}
		// Parse any partial results
		r.parseOutput(result, output)
		result.Errors++
		return result, fmt.Errorf("warp %s failed: %w\nOutput: %s", op, err, output)
	}

	// Parse successful output
	r.parseOutput(result, output)
	return result, nil
}

// parseOutput extracts metrics from warp output.
// Example output:
// Report: PUT. Concurrency: 10. Ran: 2s
//   * Average: 3.51 MiB/s, 3591.95 obj/s
//   * Reqs: Avg: 2.9ms, 50%: 2.6ms, 90%: 3.8ms, 99%: 6.1ms, Fastest: 0.8ms, Slowest: 176.1ms
func (r *Runner) parseOutput(result *Result, output string) {
	// Parse throughput: "Average: 3.51 MiB/s, 3591.95 obj/s"
	avgRe := regexp.MustCompile(`Average:\s*([0-9.]+)\s*(MiB|MB|KiB|KB|GiB|GB)/s,\s*([0-9.]+)\s*obj/s`)
	if m := avgRe.FindStringSubmatch(output); len(m) >= 4 {
		val, _ := strconv.ParseFloat(m[1], 64)
		// Normalize to MB/s (MiB is close enough for reporting)
		switch strings.ToLower(m[2]) {
		case "kib", "kb":
			val /= 1024
		case "gib", "gb":
			val *= 1024
		}
		result.ThroughputMBps = val
		result.OpsPerSec, _ = strconv.ParseFloat(m[3], 64)
	}

	// Parse latencies: "Reqs: Avg: 2.9ms, 50%: 2.6ms, 90%: 3.8ms, 99%: 6.1ms"
	// Or: "* Reqs: Avg: 2.9ms, 50%: 2.6ms, ..."
	latencyLineRe := regexp.MustCompile(`Reqs:\s*Avg:\s*([0-9.]+)(ms|s|µs|us),\s*50%:\s*([0-9.]+)(ms|s|µs|us)`)
	if m := latencyLineRe.FindStringSubmatch(output); len(m) >= 5 {
		result.LatencyAvgMs = normalizeLatency(parseFloat(m[1]), m[2])
		result.LatencyP50Ms = normalizeLatency(parseFloat(m[3]), m[4])
	}

	// Parse P99 separately as it might be in a different position
	p99Re := regexp.MustCompile(`99%:\s*([0-9.]+)(ms|s|µs|us)`)
	if m := p99Re.FindStringSubmatch(output); len(m) >= 3 {
		result.LatencyP99Ms = normalizeLatency(parseFloat(m[1]), m[2])
	}

	// If ops/s wasn't found in the main line, try to find it elsewhere
	if result.OpsPerSec == 0 {
		opsRe := regexp.MustCompile(`([0-9.]+)\s*obj/s`)
		if m := opsRe.FindStringSubmatch(output); len(m) >= 2 {
			result.OpsPerSec, _ = strconv.ParseFloat(m[1], 64)
		}
	}

	// If throughput wasn't found, try alternative patterns
	if result.ThroughputMBps == 0 {
		// Try "Throughput: 123.45 MiB/s"
		throughputRe := regexp.MustCompile(`Throughput[:\s]+([0-9.]+)\s*(MiB|MB|KiB|KB|GiB|GB)/s`)
		if m := throughputRe.FindStringSubmatch(output); len(m) >= 3 {
			val, _ := strconv.ParseFloat(m[1], 64)
			switch strings.ToLower(m[2]) {
			case "kib", "kb":
				val /= 1024
			case "gib", "gb":
				val *= 1024
			}
			result.ThroughputMBps = val
		}
	}
}

// normalizeLatency converts latency to milliseconds.
func normalizeLatency(val float64, unit string) float64 {
	switch strings.ToLower(unit) {
	case "s":
		return val * 1000
	case "µs", "us":
		return val / 1000
	default: // ms
		return val
	}
}

// parseFloat parses a float, returning 0 on error.
func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// Run executes all benchmarks and returns results.
func (r *Runner) Run(ctx context.Context) ([]*Result, error) {
	if err := r.CheckWarp(); err != nil {
		return nil, err
	}

	drivers := FilterDrivers(DefaultDrivers(), r.config.Drivers)
	if len(drivers) == 0 {
		return nil, fmt.Errorf("no drivers configured")
	}

	// Setup Docker cleanup if enabled
	var dockerCleanup *DockerCleanup
	if r.config.DockerClean {
		dockerCleanup = NewDockerCleanup(r.config.ComposeDir)
		dockerCleanup.SetLogger(r.logger)
	}

	var results []*Result

	// Check which drivers are available
	var availableDrivers []*DriverConfig
	r.logger("\nChecking driver availability...")
	for _, driver := range drivers {
		if err := r.CheckDriver(ctx, driver); err != nil {
			r.logger("  [SKIP] %s: %v", driver.Name, err)
			results = append(results, &Result{
				Driver:     driver.Name,
				Skipped:    true,
				SkipReason: err.Error(),
			})
		} else {
			r.logger("  [OK] %s at %s", driver.Name, driver.Endpoint)
			availableDrivers = append(availableDrivers, driver)
		}
	}

	if len(availableDrivers) == 0 {
		return results, fmt.Errorf("no drivers available")
	}

	r.logger("\nStarting benchmarks...")
	r.logger("Duration: %v per operation", r.config.Duration)
	r.logger("Concurrent: %d", r.config.Concurrent)
	r.logger("Objects: %d", r.config.Objects)
	r.logger("Sizes: %v", r.config.ObjectSizes)
	r.logger("Operations: %v", r.config.Operations)
	if r.config.DockerClean {
		r.logger("Docker cleanup: enabled")
	}
	r.logger("")

	// Calculate total operations (some ops don't need size variations)
	totalOps := 0
	for _, op := range r.config.Operations {
		if op == "list" {
			totalOps += len(availableDrivers) // list runs once per driver
		} else {
			totalOps += len(availableDrivers) * len(r.config.ObjectSizes)
		}
	}
	currentOp := 0

	for _, driver := range availableDrivers {
		r.logger("=== Driver: %s ===", driver.Name)

		// Pre-benchmark cleanup: recreate container with fresh volumes
		if dockerCleanup != nil {
			if err := dockerCleanup.RecreateContainer(ctx, driver); err != nil {
				r.logger("  [WARN] Pre-cleanup failed: %v", err)
			}
			// Re-check driver availability after restart
			if err := r.CheckDriver(ctx, driver); err != nil {
				r.logger("  [SKIP] %s not available after cleanup: %v", driver.Name, err)
				results = append(results, &Result{
					Driver:     driver.Name,
					Skipped:    true,
					SkipReason: "unavailable after cleanup: " + err.Error(),
				})
				continue
			}
		}

		for _, op := range r.config.Operations {
			// Some operations don't need size variations
			sizes := r.config.ObjectSizes
			if op == "list" {
				sizes = []string{""} // list doesn't use object size
			}

			for _, size := range sizes {
				currentOp++
				sizeStr := size
				if sizeStr == "" {
					sizeStr = "N/A"
				}
				r.logger("[%d/%d] %s %s %s...", currentOp, totalOps, driver.Name, op, sizeStr)

				result, err := r.RunBenchmark(ctx, driver, op, size)
				if err != nil {
					r.logger("  Error: %v", err)
				} else {
					r.logger("  Throughput: %.2f MB/s, Ops: %.2f/s, Latency: %.2fms avg, %.2fms p99",
						result.ThroughputMBps, result.OpsPerSec, result.LatencyAvgMs, result.LatencyP99Ms)
				}
				results = append(results, result)

				// Check for cancellation
				if ctx.Err() != nil {
					r.logger("\nBenchmark cancelled")
					return results, ctx.Err()
				}
			}
		}

		// Post-benchmark cleanup: clear bucket data
		if dockerCleanup != nil {
			dockerCleanup.PostBenchmarkCleanup(ctx, driver)
		}

		r.logger("")
	}

	return results, nil
}
