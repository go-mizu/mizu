package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
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
	Skipped        bool
	SkipReason     string
}

// Runner executes warp benchmarks against S3 drivers.
type Runner struct {
	config   *Config
	warpPath string
	script   string
	workDir  string
	runID    string
	logger   func(format string, args ...any)
}

// NewRunner creates a new benchmark runner.
func NewRunner(cfg *Config) *Runner {
	return &Runner{
		config: cfg,
		runID:  fmt.Sprintf("run-%d", time.Now().UnixNano()),
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
	r.config.WarpPath = path
	r.logger("Found warp at: %s", path)

	if scriptPath, err := exec.LookPath("script"); err == nil {
		r.script = scriptPath
		r.logger("Using PTY wrapper: %s", scriptPath)
	}

	// Get version
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err == nil {
		r.config.WarpVersion = strings.TrimSpace(string(output))
		r.logger("Warp version: %s", r.config.WarpVersion)
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
	if r.config.Lookup != "" {
		args = append(args, "--lookup="+r.config.Lookup)
	}
	if r.config.DisableSHA256 {
		args = append(args, "--disable-sha256-payload")
	}
	if r.config.AutoTerm {
		args = append(args, "--autoterm")
		if r.config.AutoTermDur > 0 {
			args = append(args, "--autoterm.dur="+r.config.AutoTermDur.String())
		}
		if r.config.AutoTermPct > 0 {
			args = append(args, "--autoterm.pct="+formatFloat(r.config.AutoTermPct))
		}
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
		// delete can be very slow with huge object counts; use configured values
		deleteObjects := r.config.DeleteObjects
		if deleteObjects <= 0 {
			deleteObjects = r.config.Objects
		}
		batch := r.config.DeleteBatch
		if batch <= 0 {
			batch = 100
		}
		minObjects := r.config.Concurrent * batch * 4
		if deleteObjects < minObjects {
			adjusted := deleteObjects / (r.config.Concurrent * 4)
			if adjusted < 1 {
				adjusted = 1
			}
			if adjusted != batch {
				r.logger("  [warn] delete objects (%d) below warp minimum (%d). Adjusting batch %d -> %d",
					deleteObjects, minObjects, batch, adjusted)
				batch = adjusted
				minObjects = r.config.Concurrent * batch * 4
			}
		}
		if deleteObjects < minObjects {
			r.logger("  [warn] raising delete objects %d -> %d to satisfy warp minimum", deleteObjects, minObjects)
			deleteObjects = minObjects
		}
		args = append(args, "--objects="+strconv.Itoa(deleteObjects))
		args = append(args, "--batch="+strconv.Itoa(batch))
		if size != "" {
			if totalBytes := estimateTotalBytes(deleteObjects, size); totalBytes > 0 {
				r.logger("  Delete workload: %d objects x %s (~%s)", deleteObjects, size, formatBytes(totalBytes))
			} else {
				r.logger("  Delete workload: %d objects x %s", deleteObjects, size)
			}
		}

	case "list":
		// list uses --objects to create test objects first
		listObjects := r.config.ListObjects
		if listObjects <= 0 {
			listObjects = r.config.Objects
		}
		args = append(args, "--objects="+strconv.Itoa(listObjects))
		if r.config.ListMaxKeys > 0 {
			args = append(args, "--max-keys="+strconv.Itoa(r.config.ListMaxKeys))
		}
		args = append(args, "--obj.size=1KiB")

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

	if r.config.NoClear {
		args = append(args, "--noclear")
	}
	if r.config.NoClear || r.config.Prefix != "" {
		if prefix := r.prefixForRun(driver, op, size); prefix != "" {
			args = append(args, "--prefix="+prefix)
		}
	}

	cmd := r.buildCommand(ctx, args)
	progressDone := make(chan struct{})
	if r.config.ProgressEvery > 0 {
		ticker := time.NewTicker(r.config.ProgressEvery)
		start := time.Now()
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					elapsed := time.Since(start).Truncate(time.Second)
					sizeStr := size
					if sizeStr == "" {
						sizeStr = "N/A"
					}
					r.logger("  [progress] %s %s %s running for %s", driver.Name, op, sizeStr, elapsed)
				case <-progressDone:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	if r.workDir != "" {
		cmd.Dir = r.workDir
		cmd.Env = append(os.Environ(), "TMPDIR="+r.workDir)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	close(progressDone)
	result.Duration = time.Since(startTime)

	// Combine output
	output := stdout.String() + stderr.String()

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
//   - Average: 3.51 MiB/s, 3591.95 obj/s
//   - Reqs: Avg: 2.9ms, 50%: 2.6ms, 90%: 3.8ms, 99%: 6.1ms, Fastest: 0.8ms, Slowest: 176.1ms
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
	if err := r.prepareWorkDir(); err != nil {
		return nil, err
	}
	defer r.cleanupWorkDir()

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
		driverPrefixes := make(map[string]struct{})

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

				if r.config.NoClear {
					if prefix := r.prefixForRun(driver, op, size); prefix != "" {
						driverPrefixes[prefix] = struct{}{}
					}
				}

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
			var prefixes []string
			if r.config.NoClear && len(driverPrefixes) > 0 {
				prefixes = make([]string, 0, len(driverPrefixes))
				for prefix := range driverPrefixes {
					prefixes = append(prefixes, prefix)
				}
			}
			dockerCleanup.PostBenchmarkCleanup(ctx, driver, prefixes)
		}

		r.logger("")
	}

	return results, nil
}

func estimateTotalBytes(objects int, size string) int64 {
	if objects <= 0 {
		return 0
	}
	bytes := parseSizeBytes(size)
	if bytes <= 0 {
		return 0
	}
	return int64(objects) * bytes
}

func parseSizeBytes(size string) int64 {
	s := strings.TrimSpace(size)
	if s == "" {
		return 0
	}
	s = strings.ToLower(s)
	multiplier := int64(1)
	switch {
	case strings.HasSuffix(s, "kib") || strings.HasSuffix(s, "kb"):
		multiplier = 1024
	case strings.HasSuffix(s, "mib") || strings.HasSuffix(s, "mb"):
		multiplier = 1024 * 1024
	case strings.HasSuffix(s, "gib") || strings.HasSuffix(s, "gb"):
		multiplier = 1024 * 1024 * 1024
	}
	numStr := strings.TrimRight(s, "kibmgab")
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	return int64(num * float64(multiplier))
}

func formatBytes(val int64) string {
	if val < 1024 {
		return fmt.Sprintf("%d B", val)
	}
	kb := float64(val) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1f KiB", kb)
	}
	mb := kb / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.1f MiB", mb)
	}
	gb := mb / 1024
	return fmt.Sprintf("%.2f GiB", gb)
}

func (r *Runner) prefixForRun(driver *DriverConfig, op, size string) string {
	base := r.config.Prefix
	if base == "" {
		base = r.runID
	}
	sizeStr := size
	if sizeStr == "" {
		sizeStr = "na"
	}
	sizeStr = strings.ReplaceAll(sizeStr, "/", "_")
	op = strings.ReplaceAll(op, "/", "_")
	driverName := strings.ReplaceAll(driver.Name, "/", "_")
	return strings.Trim(strings.Join([]string{base, driverName, op, sizeStr}, "/"), "/")
}

func formatFloat(val float64) string {
	s := strconv.FormatFloat(val, 'f', -1, 64)
	return s
}

func (r *Runner) buildCommand(ctx context.Context, args []string) *exec.Cmd {
	if r.script == "" {
		return exec.CommandContext(ctx, r.warpPath, args...)
	}
	scriptArgs := make([]string, 0, len(args)+3)
	scriptArgs = append(scriptArgs, "-q", "/dev/null", r.warpPath)
	scriptArgs = append(scriptArgs, args...)
	return exec.CommandContext(ctx, r.script, scriptArgs...)
}

func (r *Runner) prepareWorkDir() error {
	if r.config.WorkDir != "" {
		r.workDir = r.config.WorkDir
		if err := os.MkdirAll(r.workDir, 0755); err != nil {
			return fmt.Errorf("create work dir: %w", err)
		}
		r.config.RunDir = r.workDir
		r.logger("Using work dir: %s", r.workDir)
		return nil
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil || cacheDir == "" {
		cacheDir = os.TempDir()
	}
	base := filepath.Join(cacheDir, "mizu", "warp_bench")
	if err := os.MkdirAll(base, 0755); err != nil {
		return fmt.Errorf("create work dir base: %w", err)
	}
	runDir, err := os.MkdirTemp(base, "run-")
	if err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}
	r.workDir = runDir
	r.config.RunDir = runDir
	r.logger("Using work dir: %s", r.workDir)
	return nil
}

func (r *Runner) cleanupWorkDir() {
	if r.workDir == "" || r.config.KeepWorkDir {
		return
	}
	if err := os.RemoveAll(r.workDir); err == nil {
		r.logger("Cleaned work dir: %s", r.workDir)
	}
}
