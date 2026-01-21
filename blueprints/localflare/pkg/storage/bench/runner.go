package bench

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	_ "github.com/go-mizu/blueprints/localflare/pkg/storage/driver/devnull"
	_ "github.com/go-mizu/blueprints/localflare/pkg/storage/driver/exp/s3"
	_ "github.com/go-mizu/blueprints/localflare/pkg/storage/driver/rabbit"
	_ "github.com/go-mizu/blueprints/localflare/pkg/storage/driver/usagi"
)

// Runner orchestrates benchmark execution.
type Runner struct {
	config            *Config
	drivers           []DriverConfig
	results           []*Metrics
	skippedBenchmarks []SkippedBenchmark
	dockerStats       map[string]*DockerStats
	logger            func(format string, args ...any)
	resultsMu         sync.Mutex
	keyCounter        uint64
	dockerCollector   *DockerStatsCollector
	// Progress tracking
	progressMu      sync.Mutex
	currentOp       string
	currentIter     int64
	currentDuration time.Duration
}

// NewRunner creates a new benchmark runner.
func NewRunner(cfg *Config) *Runner {
	return &Runner{
		config:          cfg,
		drivers:         FilterDrivers(AllDriverConfigs(), cfg.Drivers),
		results:         make([]*Metrics, 0),
		dockerStats:     make(map[string]*DockerStats),
		logger:          func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
		dockerCollector: NewDockerStatsCollector("all-"),
	}
}

// SetLogger sets a custom logger.
func (r *Runner) SetLogger(fn func(format string, args ...any)) {
	r.logger = fn
}

// showLiveProgress prints live progress during benchmark execution.
// Returns a cleanup function that must be called when the benchmark completes.
// The cleanup function is safe to call multiple times.
func (r *Runner) showLiveProgress(operation string, targetDuration time.Duration) func() {
	stopCh := make(chan struct{})
	startTime := time.Now()
	var once sync.Once
	spinnerChars := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
	spinnerIdx := 0

	// Print initial status immediately
	fmt.Printf("\r    %c %s: running...  ", spinnerChars[0], operation)

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond) // Faster updates for smoother spinner
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				r.progressMu.Lock()
				iters := r.currentIter
				elapsed := time.Since(startTime)
				r.progressMu.Unlock()

				// Calculate progress percentage based on elapsed time vs target
				progress := float64(elapsed) / float64(targetDuration)
				if progress > 1.0 {
					progress = 1.0
				}

				// Calculate iterations per second
				var ipsStr string
				if elapsed.Seconds() > 0 {
					ips := float64(iters) / elapsed.Seconds()
					if ips >= 1000 {
						ipsStr = fmt.Sprintf("%.1fk/s", ips/1000)
					} else {
						ipsStr = fmt.Sprintf("%.0f/s", ips)
					}
				} else {
					ipsStr = "-/s"
				}

				// Spinner
				spinnerIdx = (spinnerIdx + 1) % len(spinnerChars)
				spinner := spinnerChars[spinnerIdx]

				// Build progress bar (20 chars to fit better)
				barWidth := 20
				filled := int(progress * float64(barWidth))
				bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

				// Print progress on same line
				fmt.Printf("\r    %c %s [%s] %3.0f%% | %d iters | %s | %v  ",
					spinner, operation, bar, progress*100, iters, ipsStr,
					elapsed.Round(100*time.Millisecond))
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(stopCh)
			// Clear the progress line
			fmt.Print("\r" + strings.Repeat(" ", 100) + "\r")
		})
	}
}

// updateProgress updates the live progress counters.
func (r *Runner) updateProgress(iters int64) {
	r.progressMu.Lock()
	r.currentIter = iters
	r.progressMu.Unlock()
}

// Run executes all benchmarks.
func (r *Runner) Run(ctx context.Context) (*Report, error) {
	r.logger("=== Storage Benchmark Suite ===")
	r.logger("Drivers: %d configured", len(r.drivers))
	r.logger("BenchTime: %v (warmup: %d, min iters: %d)", r.config.BenchTime, r.config.WarmupIterations, r.config.MinBenchIterations)
	r.logger("Concurrency: %d", r.config.Concurrency)
	r.logger("Object sizes: %v", formatSizes(r.config.ObjectSizes))
	r.logger("")

	// Detect available drivers
	available := r.detectDrivers(ctx)
	if len(available) == 0 {
		return nil, fmt.Errorf("no storage drivers available")
	}

	r.logger("Available drivers: %d", len(available))
	for _, d := range available {
		r.logger("  - %s", d.Name)
	}
	r.logger("")

	// Run benchmarks for each driver
	for i, driver := range available {
		// Check for context cancellation before starting next driver
		select {
		case <-ctx.Done():
			r.logger("Benchmark cancelled")
			return r.generateReport(), ctx.Err()
		default:
		}

		r.logger("=== [%d/%d] Benchmarking %s ===", i+1, len(available), driver.Name)

		// Collect Docker stats before benchmarks (to show growth)
		if r.config.DockerStats && driver.Container != "" {
			r.logger("  Collecting initial Docker stats...")
			stats, err := r.dockerCollector.GetStatsWithDataPath(ctx, driver.Container, driver.DataPath)
			if err == nil {
				r.logger("  Initial: Memory=%.1fMB, Volume=%.1fMB", stats.MemoryUsageMB, stats.VolumeSize)
			}
		}

		if err := r.benchmarkDriver(ctx, driver); err != nil {
			// Check if error is due to context cancellation
			if ctx.Err() != nil {
				r.logger("Driver %s cancelled", driver.Name)
				return r.generateReport(), ctx.Err()
			}
			r.logger("Driver %s failed: %v", driver.Name, err)
			continue
		}

		// Check for context cancellation after benchmark
		select {
		case <-ctx.Done():
			r.logger("Benchmark cancelled after %s", driver.Name)
			return r.generateReport(), ctx.Err()
		default:
		}

		// Collect Docker stats after benchmarks
		if r.config.DockerStats && driver.Container != "" {
			r.logger("  Collecting final Docker stats...")
			stats, err := r.dockerCollector.GetStatsWithDataPath(ctx, driver.Container, driver.DataPath)
			if err == nil {
				r.dockerStats[driver.Name] = stats
				r.logger("  Final: Memory=%.1fMB, Volume=%.1fMB", stats.MemoryUsageMB, stats.VolumeSize)
			}

			// Cleanup container to reset state for next benchmark
			r.logger("  Cleaning up %s container...", driver.Name)
			if err := r.dockerCollector.CleanupContainer(ctx, driver.Container); err != nil {
				r.logger("  Warning: cleanup failed: %v", err)
			} else {
				r.logger("  Container restarted and healthy")
			}
		}

		r.logger("")
	}

	// Generate report
	return r.generateReport(), nil
}

func (r *Runner) detectDrivers(ctx context.Context) []DriverConfig {
	var available []DriverConfig

	r.logger("Detecting available drivers...")
	for _, d := range r.drivers {
		detectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		st, err := storage.Open(detectCtx, d.DSN)

		if err != nil {
			cancel()
			r.logger("  %s: not available (%v)", d.Name, err)
			continue
		}

		// Try to list buckets with a fresh context
		listCtx, listCancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err = st.Buckets(listCtx, 1, 0, nil)
		listCancel()
		st.Close()
		cancel()

		if err != nil {
			r.logger("  %s: connection failed (%v)", d.Name, err)
			continue
		}

		r.logger("  %s: available", d.Name)
		available = append(available, d)
	}

	return available
}

func (r *Runner) benchmarkDriver(ctx context.Context, driver DriverConfig) error {
	// Check for context cancellation
	if ctx.Err() != nil {
		return ctx.Err()
	}

	st, err := storage.Open(ctx, driver.DSN)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer st.Close()

	// Ensure bucket exists
	st.CreateBucket(ctx, driver.Bucket, nil)
	bucket := st.Bucket(driver.Bucket)

	// Determine max concurrency for this driver (0 means unlimited)
	maxConc := driver.MaxConcurrency
	if maxConc > 0 {
		r.logger("  Note: %s limited to C%d", driver.Name, maxConc)
	}

	// Run write benchmarks
	for _, size := range r.config.ObjectSizes {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		label := fmt.Sprintf("Write/%s", SizeLabel(size))
		r.runBenchmark(ctx, bucket, label, func() error {
			return r.benchmarkWrite(ctx, bucket, driver.Name, size)
		})
	}

	// Run read benchmarks
	for _, size := range r.config.ObjectSizes {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		label := fmt.Sprintf("Read/%s", SizeLabel(size))
		r.runBenchmark(ctx, bucket, label, func() error {
			return r.benchmarkRead(ctx, bucket, driver.Name, size)
		})
	}

	// Run stat benchmark
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.runBenchmark(ctx, bucket, "Stat", func() error {
		return r.benchmarkStat(ctx, bucket, driver.Name)
	})

	// Run list benchmark
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.runBenchmark(ctx, bucket, "List", func() error {
		return r.benchmarkList(ctx, bucket, driver.Name)
	})

	// Run delete benchmark
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.runBenchmark(ctx, bucket, "Delete", func() error {
		return r.benchmarkDelete(ctx, bucket, driver.Name)
	})

	// Run parallel benchmarks at multiple concurrency levels
	for _, size := range r.config.ObjectSizes[:1] { // Use first size only
		if ctx.Err() != nil {
			return ctx.Err()
		}
		concLevels := r.config.ConcurrencyLevels
		if len(concLevels) == 0 {
			concLevels = []int{r.config.Concurrency}
		}

		for _, conc := range concLevels {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Skip if concurrency exceeds driver's max limit (if set)
			if maxConc > 0 && conc > maxConc {
				r.logger("  Parallel/C%d: skipped (driver %s max=%d)", conc, driver.Name, maxConc)
				// Track skipped benchmarks for reporting
				r.addSkippedBenchmark(driver.Name, fmt.Sprintf("ParallelWrite/%s/C%d", SizeLabel(size), conc),
					fmt.Sprintf("exceeds max concurrency %d", maxConc))
				r.addSkippedBenchmark(driver.Name, fmt.Sprintf("ParallelRead/%s/C%d", SizeLabel(size), conc),
					fmt.Sprintf("exceeds max concurrency %d", maxConc))
				continue
			}

			r.runBenchmark(ctx, bucket, fmt.Sprintf("ParallelWrite/C%d", conc), func() error {
				return r.benchmarkParallelWrite(ctx, bucket, driver.Name, size, conc)
			})
			r.runBenchmark(ctx, bucket, fmt.Sprintf("ParallelRead/C%d", conc), func() error {
				return r.benchmarkParallelRead(ctx, bucket, driver.Name, size, conc)
			})
		}
	}

	// Run range read benchmarks
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.runBenchmark(ctx, bucket, "RangeRead", func() error {
		return r.benchmarkRangeRead(ctx, bucket, driver.Name)
	})

	// Run copy benchmarks
	for _, size := range r.config.ObjectSizes[:1] { // Use first size only
		if ctx.Err() != nil {
			return ctx.Err()
		}
		label := fmt.Sprintf("Copy/%s", SizeLabel(size))
		r.runBenchmark(ctx, bucket, label, func() error {
			return r.benchmarkCopy(ctx, bucket, driver.Name, size)
		})
	}

	if r.config.Verbose {
		r.logger("  [debug] Copy benchmarks done, starting MixedWorkload...")
	}

	// Run mixed workload benchmarks
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.runBenchmark(ctx, bucket, "MixedWorkload", func() error {
		return r.benchmarkMixedWorkload(ctx, bucket, driver.Name, maxConc)
	})

	if r.config.Verbose {
		r.logger("  [debug] MixedWorkload done, starting Multipart...")
	}

	// Run multipart benchmarks
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.runBenchmark(ctx, bucket, "Multipart", func() error {
		return r.benchmarkMultipart(ctx, bucket, driver.Name)
	})

	if r.config.Verbose {
		r.logger("  [debug] Multipart done, starting EdgeCases...")
	}

	// Run edge case benchmarks
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.runBenchmark(ctx, bucket, "EdgeCases", func() error {
		return r.benchmarkEdgeCases(ctx, bucket, driver.Name)
	})

	if r.config.Verbose {
		r.logger("  [debug] EdgeCases done, starting FileCount...")
	}

	// Run file count benchmarks
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if len(r.config.FileCounts) > 0 {
		r.runBenchmark(ctx, bucket, "FileCount", func() error {
			return r.benchmarkFileCount(ctx, bucket, driver.Name)
		})
	}

	if r.config.Verbose {
		r.logger("  [debug] FileCount done, driver %s complete", driver.Name)
	}

	return nil
}

func (r *Runner) benchmarkWrite(ctx context.Context, bucket storage.Bucket, driver string, size int) error {
	operation := fmt.Sprintf("Write/%s", SizeLabel(size))
	data := generateRandomData(size)

	warmup := r.config.WarmupForSize(size)

	// Warmup
	for i := 0; i < warmup; i++ {
		key := r.uniqueKey("warmup")
		opCtx, cancel := r.opContextForSize(ctx, size)
		bucket.Write(opCtx, key, bytes.NewReader(data), int64(size), "application/octet-stream", nil)
		cancel()
	}

	// Adaptive benchmark (Go-style)
	collector := NewCollector()
	benchTime := r.config.BenchTimeForSize(size)
	ab := NewAdaptiveBenchmarkWithContext(ctx, benchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)

	// Start live progress display
	r.currentIter = 0
	cleanup := r.showLiveProgress(operation, benchTime)
	defer cleanup()

	// Adaptive scaling loop - runs until target duration is reached
	var totalIters int64
	for ab.ShouldContinue() {
		n := ab.NextN()
		start := time.Now()

		// Run N iterations
		for i := 0; i < n; i++ {
			key := r.uniqueKey("write")
			timer := NewTimer()

			opCtx, cancel := r.opContextForSize(ctx, size)
			_, err := bucket.Write(opCtx, key, bytes.NewReader(data), int64(size), "application/octet-stream", nil)
			cancel()

			collector.RecordWithError(timer.Elapsed(), err)
			totalIters++
			r.updateProgress(totalIters)
		}

		elapsed := time.Since(start)
		ab.RecordRun(n, elapsed)
	}

	cleanup() // Stop progress display before logging result
	r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), benchTime)

	metrics := collector.Metrics(operation, driver, size)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkRead(ctx context.Context, bucket storage.Bucket, driver string, size int) error {
	operation := fmt.Sprintf("Read/%s", SizeLabel(size))
	data := generateRandomData(size)

	warmup := r.config.WarmupForSize(size)

	// Pre-create objects (enough for adaptive benchmark)
	numObjects := 100 // Pool of objects to read from
	keys := make([]string, numObjects)
	for i := range keys {
		keys[i] = r.uniqueKey("read")
		opCtx, cancel := r.opContextForSize(ctx, size)
		bucket.Write(opCtx, keys[i], bytes.NewReader(data), int64(size), "application/octet-stream", nil)
		cancel()
	}

	// Warmup
	for i := 0; i < warmup && i < len(keys); i++ {
		opCtx, cancel := r.opContextForSize(ctx, size)
		rc, _, _ := bucket.Open(opCtx, keys[i], 0, 0, nil)
		if rc != nil {
			io.Copy(io.Discard, rc)
			rc.Close()
		}
		cancel()
	}

	// Adaptive benchmark with TTFB tracking
	collector := NewCollector()
	benchTime := r.config.BenchTimeForSize(size)
	ab := NewAdaptiveBenchmarkWithContext(ctx, benchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)
	var keyIdx uint64

	// Start live progress display
	r.currentIter = 0
	cleanup := r.showLiveProgress(operation, benchTime)
	defer cleanup()

	// Adaptive scaling loop
	var totalIters int64
	for ab.ShouldContinue() {
		n := ab.NextN()
		runStart := time.Now()

		for i := 0; i < n; i++ {
			idx := atomic.AddUint64(&keyIdx, 1) % uint64(numObjects)
			start := time.Now()

			opCtx, cancel := r.opContextForSize(ctx, size)
			rc, _, err := bucket.Open(opCtx, keys[idx], 0, 0, nil)
			if err == nil {
				// Wrap reader to capture TTFB
				ttfbReader := NewTTFBReader(rc, start)
				io.Copy(io.Discard, ttfbReader)
				rc.Close()

				latency := time.Since(start)
				collector.RecordWithTTFB(latency, ttfbReader.TTFB(), nil)
			} else {
				collector.RecordWithError(time.Since(start), err)
			}
			cancel()
			totalIters++
			r.updateProgress(totalIters)
		}

		elapsed := time.Since(runStart)
		ab.RecordRun(n, elapsed)
	}

	cleanup() // Stop progress display before logging result
	r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), benchTime)

	metrics := collector.Metrics(operation, driver, size)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkStat(ctx context.Context, bucket storage.Bucket, driver string) error {
	operation := "Stat"
	data := generateRandomData(1024)

	// Create test object
	key := r.uniqueKey("stat")
	opCtx, cancel := r.opContext(ctx)
	bucket.Write(opCtx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
	cancel()

	// Warmup
	for i := 0; i < r.config.WarmupIterations; i++ {
		opCtx, cancel := r.opContext(ctx)
		bucket.Stat(opCtx, key, nil)
		cancel()
	}

	// Adaptive benchmark
	collector := NewCollector()
	ab := NewAdaptiveBenchmarkWithContext(ctx, r.config.BenchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)

	// Start live progress display
	r.currentIter = 0
	cleanup := r.showLiveProgress(operation, r.config.BenchTime)
	defer cleanup()

	var totalIters int64
	for ab.ShouldContinue() {
		n := ab.NextN()
		start := time.Now()

		for i := 0; i < n; i++ {
			timer := NewTimer()

			opCtx, cancel := r.opContext(ctx)
			_, err := bucket.Stat(opCtx, key, nil)
			cancel()

			collector.RecordWithError(timer.Elapsed(), err)
			totalIters++
			r.updateProgress(totalIters)
		}

		elapsed := time.Since(start)
		ab.RecordRun(n, elapsed)
	}

	cleanup()
	r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), r.config.BenchTime)

	metrics := collector.Metrics(operation, driver, 0)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkList(ctx context.Context, bucket storage.Bucket, driver string) error {
	operation := "List/100"
	data := generateRandomData(100)

	// Create 100 objects
	prefix := r.uniqueKey("list")
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("%s/obj-%05d", prefix, i)
		opCtx, cancel := r.opContext(ctx)
		bucket.Write(opCtx, key, bytes.NewReader(data), 100, "text/plain", nil)
		cancel()
	}

	// Warmup
	for i := 0; i < r.config.WarmupIterations; i++ {
		opCtx, cancel := r.opContext(ctx)
		iter, _ := bucket.List(opCtx, prefix, 0, 0, nil)
		if iter != nil {
			for {
				obj, err := iter.Next()
				if err != nil || obj == nil {
					break
				}
			}
			iter.Close()
		}
		cancel()
	}

	// Adaptive benchmark
	collector := NewCollector()
	ab := NewAdaptiveBenchmarkWithContext(ctx, r.config.BenchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)

	// Start live progress display
	r.currentIter = 0
	cleanup := r.showLiveProgress(operation, r.config.BenchTime)
	defer cleanup()

	var totalIters int64
	for ab.ShouldContinue() {
		n := ab.NextN()
		start := time.Now()

		for i := 0; i < n; i++ {
			timer := NewTimer()

			opCtx, cancel := r.opContext(ctx)
			iter, err := bucket.List(opCtx, prefix, 0, 0, nil)
			if err == nil {
				for {
					obj, err := iter.Next()
					if err != nil || obj == nil {
						break
					}
				}
				iter.Close()
			}

			collector.RecordWithError(timer.Elapsed(), err)
			cancel()
			totalIters++
			r.updateProgress(totalIters)
		}

		elapsed := time.Since(start)
		ab.RecordRun(n, elapsed)
	}

	cleanup()
	r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), r.config.BenchTime)

	metrics := collector.Metrics(operation, driver, 0)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkDelete(ctx context.Context, bucket storage.Bucket, driver string) error {
	operation := "Delete"
	data := generateRandomData(1024)

	// Adaptive benchmark
	collector := NewCollector()
	ab := NewAdaptiveBenchmarkWithContext(ctx, r.config.BenchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)

	// Start live progress display
	r.currentIter = 0
	cleanup := r.showLiveProgress(operation, r.config.BenchTime)
	defer cleanup()

	var totalIters int64
	for ab.ShouldContinue() {
		n := ab.NextN()

		// Pre-create objects for this batch
		keys := make([]string, n)
		for i := range keys {
			keys[i] = r.uniqueKey("delete")
			opCtx, cancel := r.opContext(ctx)
			bucket.Write(opCtx, keys[i], bytes.NewReader(data), 1024, "application/octet-stream", nil)
			cancel()
		}

		// Time only the delete operations
		start := time.Now()
		for i := 0; i < n; i++ {
			timer := NewTimer()

			opCtx, cancel := r.opContext(ctx)
			err := bucket.Delete(opCtx, keys[i], nil)
			cancel()

			collector.RecordWithError(timer.Elapsed(), err)
			totalIters++
			r.updateProgress(totalIters)
		}
		elapsed := time.Since(start)
		ab.RecordRun(n, elapsed)
	}

	cleanup()
	r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), r.config.BenchTime)

	metrics := collector.Metrics(operation, driver, 0)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkParallelWrite(ctx context.Context, bucket storage.Bucket, driver string, size, concurrency int) error {
	operation := fmt.Sprintf("ParallelWrite/%s/C%d", SizeLabel(size), concurrency)
	data := generateRandomData(size)

	// Use parallel timeout if set, otherwise use default
	timeout := r.config.ParallelTimeout
	if timeout == 0 {
		timeout = r.config.Timeout
	}

	// Create a context with overall timeout for the benchmark
	benchCtx, benchCancel := context.WithTimeout(ctx, timeout)
	defer benchCancel()

	// Adaptive benchmark
	collector := NewCollector()
	benchTime := r.config.BenchTimeForSize(size)
	ab := NewAdaptiveBenchmarkWithContext(ctx, benchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)

	sem := make(chan struct{}, concurrency)
	opTimeout := 10 * time.Second

	// Start live progress display
	r.currentIter = 0
	cleanup := r.showLiveProgress(operation, benchTime)
	defer cleanup()

	var totalIters int64
	for ab.ShouldContinue() {
		n := ab.NextN()
		var wg sync.WaitGroup

		start := time.Now()

		for i := 0; i < n; i++ {
			// Check if benchmark context is done
			select {
			case <-benchCtx.Done():
				wg.Wait()
				goto done
			default:
			}

			wg.Add(1)
			sem <- struct{}{}

			go func() {
				defer wg.Done()
				defer func() { <-sem }()

				opCtx, opCancel := context.WithTimeout(benchCtx, opTimeout)
				defer opCancel()

				key := r.uniqueKey("parallel-write")
				timer := NewTimer()

				_, err := bucket.Write(opCtx, key, bytes.NewReader(data), int64(size), "application/octet-stream", nil)
				collector.RecordWithError(timer.Elapsed(), err)
				atomic.AddInt64(&totalIters, 1)
				r.updateProgress(atomic.LoadInt64(&totalIters))
			}()
		}

		wg.Wait()
		elapsed := time.Since(start)
		ab.RecordRun(n, elapsed)
	}

done:
	cleanup() // Stop progress display before logging result
	r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), benchTime)

	metrics := collector.Metrics(operation, driver, size)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkParallelRead(ctx context.Context, bucket storage.Bucket, driver string, size, concurrency int) error {
	operation := fmt.Sprintf("ParallelRead/%s/C%d", SizeLabel(size), concurrency)
	data := generateRandomData(size)

	// Pre-create objects
	numObjects := 50 // Larger pool for adaptive benchmark
	keys := make([]string, numObjects)
	for i := range keys {
		keys[i] = r.uniqueKey("parallel-read")
		opCtx, cancel := r.opContext(ctx)
		bucket.Write(opCtx, keys[i], bytes.NewReader(data), int64(size), "application/octet-stream", nil)
		cancel()
	}

	// Use parallel timeout if set, otherwise use default
	timeout := r.config.ParallelTimeout
	if timeout == 0 {
		timeout = r.config.Timeout
	}

	// Create a context with overall timeout for the benchmark
	benchCtx, benchCancel := context.WithTimeout(ctx, timeout)
	defer benchCancel()

	// Adaptive benchmark
	collector := NewCollector()
	benchTime := r.config.BenchTimeForSize(size)
	ab := NewAdaptiveBenchmarkWithContext(ctx, benchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)

	var keyIdx uint64
	sem := make(chan struct{}, concurrency)
	opTimeout := 10 * time.Second

	// Start live progress display
	r.currentIter = 0
	cleanup := r.showLiveProgress(operation, benchTime)
	defer cleanup()

	var totalIters int64
	for ab.ShouldContinue() {
		n := ab.NextN()
		var wg sync.WaitGroup

		start := time.Now()

		for i := 0; i < n; i++ {
			// Check if benchmark context is done
			select {
			case <-benchCtx.Done():
				wg.Wait()
				goto done
			default:
			}

			wg.Add(1)
			sem <- struct{}{}

			go func() {
				defer wg.Done()
				defer func() { <-sem }()

				opCtx, opCancel := context.WithTimeout(benchCtx, opTimeout)
				defer opCancel()

				idx := atomic.AddUint64(&keyIdx, 1) % uint64(numObjects)
				opStart := time.Now()

				rc, _, err := bucket.Open(opCtx, keys[idx], 0, 0, nil)
				if err == nil {
					ttfbReader := NewTTFBReader(rc, opStart)
					io.Copy(io.Discard, ttfbReader)
					rc.Close()

					latency := time.Since(opStart)
					collector.RecordWithTTFB(latency, ttfbReader.TTFB(), nil)
				} else {
					collector.RecordWithError(time.Since(opStart), err)
				}
				atomic.AddInt64(&totalIters, 1)
				r.updateProgress(atomic.LoadInt64(&totalIters))
			}()
		}

		wg.Wait()
		elapsed := time.Since(start)
		ab.RecordRun(n, elapsed)
	}

done:
	cleanup() // Stop progress display before logging result
	r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), benchTime)

	metrics := collector.Metrics(operation, driver, size)
	r.addResult(metrics)

	return nil
}

func (r *Runner) addResult(m *Metrics) {
	r.resultsMu.Lock()
	r.results = append(r.results, m)
	r.resultsMu.Unlock()
}

func (r *Runner) addSkippedBenchmark(driver, operation, reason string) {
	r.resultsMu.Lock()
	r.skippedBenchmarks = append(r.skippedBenchmarks, SkippedBenchmark{
		Driver:    driver,
		Operation: operation,
		Reason:    reason,
	})
	r.resultsMu.Unlock()
}

func (r *Runner) uniqueKey(prefix string) string {
	n := atomic.AddUint64(&r.keyCounter, 1)
	return fmt.Sprintf("%s/%d/%d", prefix, time.Now().UnixNano(), n)
}

func (r *Runner) cleanupBucket(ctx context.Context, bucket storage.Bucket) {
	cleanupCtx := ctx
	if cleanupCtx == nil || cleanupCtx.Err() != nil {
		cleanupCtx = context.Background()
	}
	cleanupTimeout := r.config.Timeout
	if cleanupTimeout <= 0 {
		cleanupTimeout = 30 * time.Second
	}
	listCtx, cancel := context.WithTimeout(cleanupCtx, cleanupTimeout)
	defer cancel()

	iter, err := bucket.List(listCtx, "", 0, 0, nil)
	if err != nil {
		return
	}
	defer iter.Close()

	// Collect all objects and dirs first
	var objects []string
	var dirs []string
	for {
		obj, err := iter.Next()
		if err != nil || obj == nil {
			break
		}
		if obj.IsDir {
			dirs = append(dirs, obj.Key)
		} else {
			objects = append(objects, obj.Key)
		}
	}

	// Skip if nothing to clean
	if len(objects) == 0 && len(dirs) == 0 {
		return
	}

	// Show cleanup progress
	total := len(objects) + len(dirs)
	var deleted int64
	spinnerChars := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
	spinnerIdx := 0
	startTime := time.Now()

	showCleanupProgress := func() {
		spinnerIdx = (spinnerIdx + 1) % len(spinnerChars)
		elapsed := time.Since(startTime)
		fmt.Printf("\r    %c Cleanup [%d/%d] %v  ",
			spinnerChars[spinnerIdx], deleted, total, elapsed.Round(100*time.Millisecond))
	}

	// Delete objects with progress
	for _, key := range objects {
		opCtx, opCancel := r.opContext(cleanupCtx)
		bucket.Delete(opCtx, key, nil)
		opCancel()
		deleted++
		if deleted%10 == 0 || deleted == int64(len(objects)) {
			showCleanupProgress()
		}
	}

	// Delete dirs (deepest first)
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})
	for _, key := range dirs {
		opCtx, opCancel := r.opContext(cleanupCtx)
		bucket.Delete(opCtx, key, storage.Options{"recursive": true})
		opCancel()
		deleted++
		showCleanupProgress()
	}

	// Clear progress line
	fmt.Print("\r" + strings.Repeat(" ", 60) + "\r")
}

func (r *Runner) runBenchmark(ctx context.Context, bucket storage.Bucket, label string, fn func() error) {
	// Check filter - skip if filter is set and label doesn't match
	if r.config.Filter != "" && !strings.Contains(label, r.config.Filter) {
		if r.config.Verbose {
			r.logger("  %s: skipped (filter: %s)", label, r.config.Filter)
		}
		return
	}

	if err := fn(); err != nil {
		r.logger("  %s failed: %v", label, err)
	}
	// Note: cleanup is done once per driver via Docker container restart, not after each operation
}

func (r *Runner) opContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if r.config.Timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, r.config.Timeout)
}

func (r *Runner) opContextForSize(ctx context.Context, size int) (context.Context, context.CancelFunc) {
	timeout := r.timeoutForSize(size)
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func (r *Runner) timeoutForSize(size int) time.Duration {
	timeout := r.config.Timeout
	if timeout <= 0 {
		return timeout
	}
	switch {
	case size >= 100*1024*1024:
		if timeout < 5*time.Minute {
			return 5 * time.Minute
		}
	case size >= 10*1024*1024:
		if timeout < 2*time.Minute {
			return 2 * time.Minute
		}
	}
	return timeout
}

func (r *Runner) generateReport() *Report {
	return &Report{
		Timestamp:         time.Now(),
		Config:            r.config,
		Results:           r.results,
		DockerStats:       r.dockerStats,
		SkippedBenchmarks: r.skippedBenchmarks,
	}
}

func (r *Runner) benchmarkRangeRead(ctx context.Context, bucket storage.Bucket, driver string) error {
	const totalSize = 1024 * 1024 // 1MB object
	data := generateRandomData(totalSize)

	// Create test object
	key := r.uniqueKey("range")
	opCtx, cancel := r.opContextForSize(ctx, totalSize)
	_, err := bucket.Write(opCtx, key, bytes.NewReader(data), int64(totalSize), "application/octet-stream", nil)
	cancel()
	if err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	ranges := []struct {
		name   string
		offset int64
		length int64
	}{
		{"Start_256KB", 0, 256 * 1024},
		{"Middle_256KB", 512 * 1024, 256 * 1024},
		{"End_256KB", 768 * 1024, 256 * 1024},
	}

	for _, rng := range ranges {
		operation := fmt.Sprintf("RangeRead/%s", rng.name)
		collector := NewCollector()
		ab := NewAdaptiveBenchmarkWithContext(ctx, r.config.BenchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)

		// Start live progress display
		r.currentIter = 0
		cleanup := r.showLiveProgress(operation, r.config.BenchTime)

		var totalIters int64
		for ab.ShouldContinue() {
			n := ab.NextN()
			start := time.Now()

			for i := 0; i < n; i++ {
				timer := NewTimer()

				opCtx, cancel := r.opContextForSize(ctx, int(rng.length))
				rc, _, err := bucket.Open(opCtx, key, rng.offset, rng.length, nil)
				if err == nil {
					io.Copy(io.Discard, rc)
					rc.Close()
				}

				collector.RecordWithError(timer.Elapsed(), err)
				cancel()
				totalIters++
				r.updateProgress(totalIters)
			}

			elapsed := time.Since(start)
			ab.RecordRun(n, elapsed)
		}

		cleanup() // Stop progress display before logging result
		r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), r.config.BenchTime)
		metrics := collector.Metrics(operation, driver, int(rng.length))
		r.addResult(metrics)
	}

	return nil
}

func (r *Runner) benchmarkCopy(ctx context.Context, bucket storage.Bucket, driver string, size int) error {
	operation := fmt.Sprintf("Copy/%s", SizeLabel(size))
	data := generateRandomData(size)

	// Create source object
	srcKey := r.uniqueKey("copy-src")
	opCtx, cancel := r.opContextForSize(ctx, size)
	_, err := bucket.Write(opCtx, srcKey, bytes.NewReader(data), int64(size), "application/octet-stream", nil)
	cancel()
	if err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	// Adaptive benchmark
	collector := NewCollector()
	benchTime := r.config.BenchTimeForSize(size)
	ab := NewAdaptiveBenchmarkWithContext(ctx, benchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)

	// Start live progress display
	r.currentIter = 0
	cleanup := r.showLiveProgress(operation, benchTime)
	defer cleanup()

	var totalIters int64
	for ab.ShouldContinue() {
		n := ab.NextN()
		start := time.Now()

		for i := 0; i < n; i++ {
			dstKey := r.uniqueKey("copy-dst")
			timer := NewTimer()

			opCtx, cancel := r.opContextForSize(ctx, size)
			_, err := bucket.Copy(opCtx, dstKey, bucket.Name(), srcKey, nil)
			cancel()

			collector.RecordWithError(timer.Elapsed(), err)
			totalIters++
			r.updateProgress(totalIters)
		}

		elapsed := time.Since(start)
		ab.RecordRun(n, elapsed)
	}

	cleanup() // Stop progress display before logging result
	r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), benchTime)
	metrics := collector.Metrics(operation, driver, size)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkMixedWorkload(ctx context.Context, bucket storage.Bucket, driver string, maxConcurrency int) error {
	objectSize := 16 * 1024 // 16KB
	data := generateRandomData(objectSize)

	// Use config concurrency if maxConcurrency is 0 (unlimited)
	concurrency := maxConcurrency
	if concurrency <= 0 {
		concurrency = r.config.Concurrency
	}

	// Pre-create objects for reading
	numObjects := 50
	keys := make([]string, numObjects)
	for i := 0; i < numObjects; i++ {
		keys[i] = r.uniqueKey("mixed")
		opCtx, cancel := r.opContext(ctx)
		bucket.Write(opCtx, keys[i], bytes.NewReader(data), int64(objectSize), "application/octet-stream", nil)
		cancel()
	}

	workloads := []struct {
		name       string
		readRatio  int
		writeRatio int
	}{
		{"ReadHeavy_90_10", 90, 10},
		{"Balanced_50_50", 50, 50},
		{"WriteHeavy_10_90", 10, 90},
	}

	for _, wl := range workloads {
		operation := fmt.Sprintf("MixedWorkload/%s", wl.name)
		collector := NewCollector()
		ab := NewAdaptiveBenchmarkWithContext(ctx, r.config.BenchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)

		sem := make(chan struct{}, concurrency)
		var opCounter uint64
		var keyIdx uint64

		// Start live progress display
		r.currentIter = 0
		cleanup := r.showLiveProgress(operation, r.config.BenchTime)

		var totalIters int64
		for ab.ShouldContinue() {
			n := ab.NextN()
			var wg sync.WaitGroup

			start := time.Now()

			for i := 0; i < n; i++ {
				wg.Add(1)
				sem <- struct{}{}

				go func() {
					defer wg.Done()
					defer func() { <-sem }()

					timer := NewTimer()
					var err error

					op := atomic.AddUint64(&opCounter, 1) % 100
					if int(op) < wl.readRatio {
						// Read operation
						idx := atomic.AddUint64(&keyIdx, 1) % uint64(len(keys))
						opCtx, cancel := r.opContext(ctx)
						rc, _, e := bucket.Open(opCtx, keys[idx], 0, 0, nil)
						if e == nil {
							io.Copy(io.Discard, rc)
							rc.Close()
						}
						cancel()
						err = e
					} else {
						// Write operation
						key := r.uniqueKey("mixed-write")
						opCtx, cancel := r.opContext(ctx)
						_, err = bucket.Write(opCtx, key, bytes.NewReader(data), int64(objectSize), "application/octet-stream", nil)
						cancel()
					}

					collector.RecordWithError(timer.Elapsed(), err)
					atomic.AddInt64(&totalIters, 1)
					r.updateProgress(atomic.LoadInt64(&totalIters))
				}()
			}

			wg.Wait()
			elapsed := time.Since(start)
			ab.RecordRun(n, elapsed)
		}

		cleanup() // Stop progress display before logging result
		r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), r.config.BenchTime)
		metrics := collector.Metrics(operation, driver, objectSize)
		r.addResult(metrics)
	}

	return nil
}

func (r *Runner) benchmarkMultipart(ctx context.Context, bucket storage.Bucket, driver string) error {
	// Check if multipart is supported
	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		r.logger("  Multipart: not supported by %s", driver)
		return nil
	}

	// S3 requires minimum 5MB per part (except last part)
	partSize := 5 * 1024 * 1024 // 5MB
	partCount := 3              // 15MB total
	totalSize := partSize * partCount
	partData := generateRandomData(partSize)

	operation := fmt.Sprintf("Multipart/%dMB_%dParts", totalSize/(1024*1024), partCount)
	collector := NewCollector()

	// Use shorter BenchTime for multipart (expensive operation)
	benchTime := r.config.BenchTimeForSize(totalSize)
	ab := NewAdaptiveBenchmarkWithContext(ctx, benchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)

	// Start live progress display
	r.currentIter = 0
	cleanup := r.showLiveProgress(operation, benchTime)
	defer cleanup()

	var totalIters int64
	for ab.ShouldContinue() {
		n := ab.NextN()
		start := time.Now()

		for i := 0; i < n; i++ {
			key := r.uniqueKey("multipart")
			timer := NewTimer()
			var err error

			// Init multipart upload
			opCtx, cancel := r.opContextForSize(ctx, totalSize)
			mu, e := mp.InitMultipart(opCtx, key, "application/octet-stream", nil)
			cancel()
			if e != nil {
				err = e
			} else {
				// Upload parts
				parts := make([]*storage.PartInfo, partCount)
				for p := 0; p < partCount && err == nil; p++ {
					opCtx, cancel := r.opContextForSize(ctx, partSize)
					part, e := mp.UploadPart(opCtx, mu, p+1, bytes.NewReader(partData), int64(partSize), nil)
					cancel()
					if e != nil {
						opCtx, cancel := r.opContextForSize(ctx, partSize)
						mp.AbortMultipart(opCtx, mu, nil)
						cancel()
						err = e
						break
					}
					parts[p] = part
				}

				// Complete
				if err == nil {
					opCtx, cancel := r.opContextForSize(ctx, totalSize)
					_, err = mp.CompleteMultipart(opCtx, mu, parts, nil)
					cancel()
				}
			}

			collector.RecordWithError(timer.Elapsed(), err)
			totalIters++
			r.updateProgress(totalIters)
		}

		elapsed := time.Since(start)
		ab.RecordRun(n, elapsed)
	}

	cleanup() // Stop progress display before logging result
	r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), benchTime)
	metrics := collector.Metrics(operation, driver, totalSize)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkEdgeCases(ctx context.Context, bucket storage.Bucket, driver string) error {
	// Empty object write
	{
		operation := "EdgeCase/EmptyObject"
		collector := NewCollector()
		ab := NewAdaptiveBenchmarkWithContext(ctx, r.config.BenchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)

		// Start live progress display
		r.currentIter = 0
		cleanup := r.showLiveProgress(operation, r.config.BenchTime)

		var totalIters int64
		for ab.ShouldContinue() {
			n := ab.NextN()
			start := time.Now()

			for i := 0; i < n; i++ {
				key := r.uniqueKey("empty")
				timer := NewTimer()

				opCtx, cancel := r.opContext(ctx)
				_, err := bucket.Write(opCtx, key, bytes.NewReader(nil), 0, "application/octet-stream", nil)
				cancel()

				collector.RecordWithError(timer.Elapsed(), err)
				totalIters++
				r.updateProgress(totalIters)
			}

			elapsed := time.Since(start)
			ab.RecordRun(n, elapsed)
		}

		cleanup() // Stop progress display before logging result
		r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), r.config.BenchTime)
		metrics := collector.Metrics(operation, driver, 0)
		r.addResult(metrics)
	}

	// Long key names (256 chars)
	{
		operation := "EdgeCase/LongKey256"
		data := generateRandomData(100)
		collector := NewCollector()
		ab := NewAdaptiveBenchmarkWithContext(ctx, r.config.BenchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)
		var keyCounter uint64

		// Start live progress display
		r.currentIter = 0
		cleanup := r.showLiveProgress(operation, r.config.BenchTime)

		var totalIters int64
		for ab.ShouldContinue() {
			n := ab.NextN()
			start := time.Now()

			for i := 0; i < n; i++ {
				idx := atomic.AddUint64(&keyCounter, 1)
				key := fmt.Sprintf("prefix/%s/%d", string(bytes.Repeat([]byte("a"), 200)), idx)
				timer := NewTimer()

				opCtx, cancel := r.opContext(ctx)
				_, err := bucket.Write(opCtx, key, bytes.NewReader(data), 100, "text/plain", nil)
				cancel()

				collector.RecordWithError(timer.Elapsed(), err)
				totalIters++
				r.updateProgress(totalIters)
			}

			elapsed := time.Since(start)
			ab.RecordRun(n, elapsed)
		}

		cleanup() // Stop progress display before logging result
		r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), r.config.BenchTime)
		metrics := collector.Metrics(operation, driver, 100)
		r.addResult(metrics)
	}

	// Deep nesting
	{
		operation := "EdgeCase/DeepNested"
		data := generateRandomData(100)
		collector := NewCollector()
		ab := NewAdaptiveBenchmarkWithContext(ctx, r.config.BenchTime, r.config.MinBenchIterations, r.config.MaxBenchIterations)
		var keyCounter uint64

		// Start live progress display
		r.currentIter = 0
		cleanup := r.showLiveProgress(operation, r.config.BenchTime)

		var totalIters int64
		for ab.ShouldContinue() {
			n := ab.NextN()
			start := time.Now()

			for i := 0; i < n; i++ {
				idx := atomic.AddUint64(&keyCounter, 1)
				key := fmt.Sprintf("a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/%d", idx)
				timer := NewTimer()

				opCtx, cancel := r.opContext(ctx)
				_, err := bucket.Write(opCtx, key, bytes.NewReader(data), 100, "text/plain", nil)
				cancel()

				collector.RecordWithError(timer.Elapsed(), err)
				totalIters++
				r.updateProgress(totalIters)
			}

			elapsed := time.Since(start)
			ab.RecordRun(n, elapsed)
		}

		cleanup() // Stop progress display before logging result
		r.logger("  %s: %d iterations in %v (target: %v)", operation, ab.TotalIterations(), ab.TotalDuration().Round(time.Millisecond), r.config.BenchTime)
		metrics := collector.Metrics(operation, driver, 100)
		r.addResult(metrics)
	}

	return nil
}

func (r *Runner) benchmarkFileCount(ctx context.Context, bucket storage.Bucket, driver string) error {
	// Test performance with varying numbers of files
	// File counts to test: 1, 10, 100, 1000, 10000, 100000
	fileCounts := r.config.FileCounts
	if len(fileCounts) == 0 {
		fileCounts = []int{1, 10, 100, 1000, 10000} // Default, skip 100k unless explicitly enabled
	}

	objectSize := 1024 // 1KB files for file count tests
	data := generateRandomData(objectSize)

	for _, count := range fileCounts {
		// Skip very large counts if timeout is short
		if count > 10000 && r.config.Timeout < 5*time.Minute {
			r.logger("  FileCount/%d: skipped (requires longer timeout)", count)
			r.addSkippedBenchmark(driver, fmt.Sprintf("FileCount/%d", count), "requires longer timeout")
			continue
		}

		prefix := r.uniqueKey(fmt.Sprintf("filecount-%d", count))

		// Benchmark: Write N files
		{
			operation := fmt.Sprintf("FileCount/Write/%d", count)
			collector := NewCollector()
			progress := NewProgress(operation, count, true)
			timer := NewTimer()

			for i := 0; i < count; i++ {
				key := fmt.Sprintf("%s/%05d", prefix, i)
				opCtx, cancel := r.opContext(ctx)
				_, err := bucket.Write(opCtx, key, bytes.NewReader(data), int64(objectSize), "application/octet-stream", nil)
				cancel()
				if err != nil {
					collector.RecordError(err)
				}
				progress.Increment()

				// Check for context cancellation periodically
				if i%1000 == 0 {
					select {
					case <-ctx.Done():
						r.logger("  %s: cancelled at %d/%d files", operation, i, count)
						goto writeCleanup
					default:
					}
				}
			}

		writeCleanup:
			elapsed := timer.Elapsed()
			if elapsed > 0 {
				// Record total time as a single sample
				collector.Record(elapsed)
			}
			progress.Done()
			metrics := collector.Metrics(operation, driver, objectSize*count)
			metrics.Iterations = count
			r.addResult(metrics)
		}

		// Benchmark: List N files
		{
			operation := fmt.Sprintf("FileCount/List/%d", count)
			collector := NewCollector()
			progress := NewProgress(operation, 1, true)
			timer := NewTimer()

			opCtx, cancel := r.opContext(ctx)
			iter, err := bucket.List(opCtx, prefix, count+100, 0, nil)
			if err == nil {
				listed := 0
				for {
					obj, err := iter.Next()
					if err != nil || obj == nil {
						break
					}
					listed++
				}
				iter.Close()

				elapsed := timer.Elapsed()
				if listed >= count {
					collector.Record(elapsed)
				} else {
					collector.RecordError(fmt.Errorf("listed %d, expected %d", listed, count))
				}
			} else {
				collector.RecordError(err)
			}
			cancel()

			progress.Increment()
			progress.Done()
			metrics := collector.Metrics(operation, driver, 0)
			metrics.Iterations = 1
			r.addResult(metrics)
		}

		// Benchmark: Delete N files (batch)
		{
			operation := fmt.Sprintf("FileCount/Delete/%d", count)
			collector := NewCollector()
			progress := NewProgress(operation, count, true)
			timer := NewTimer()

			for i := 0; i < count; i++ {
				key := fmt.Sprintf("%s/%05d", prefix, i)
				opCtx, cancel := r.opContext(ctx)
				err := bucket.Delete(opCtx, key, nil)
				cancel()
				if err != nil {
					collector.RecordError(err)
				}
				progress.Increment()

				// Check for context cancellation periodically
				if i%1000 == 0 {
					select {
					case <-ctx.Done():
						r.logger("  %s: cancelled at %d/%d files", operation, i, count)
						goto deleteCleanup
					default:
					}
				}
			}

		deleteCleanup:
			elapsed := timer.Elapsed()
			if elapsed > 0 {
				collector.Record(elapsed)
			}
			progress.Done()
			metrics := collector.Metrics(operation, driver, 0)
			metrics.Iterations = count
			r.addResult(metrics)
		}
	}

	return nil
}

func generateRandomData(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

func formatSizes(sizes []int) string {
	labels := make([]string, len(sizes))
	for i, s := range sizes {
		labels[i] = SizeLabel(s)
	}
	return fmt.Sprintf("%v", labels)
}

// AdaptiveBenchmark implements Go-style adaptive iteration scaling.
// It automatically scales the number of iterations to achieve a target duration,
// following the exact algorithm used by Go's testing.B framework.
type AdaptiveBenchmark struct {
	// Target duration for the benchmark
	benchTime time.Duration
	// Minimum iterations regardless of duration
	minIterations int
	// Maximum iterations (safety limit)
	maxIterations int64

	// Current state
	totalN        int64         // Total iterations executed
	totalDuration time.Duration // Total benchmark duration
	lastN         int64         // Last run's iteration count
	lastDuration  time.Duration // Last run's duration
	nextN         int64         // Next iteration count to run
	started       bool          // Whether benchmark has started
	ctx           context.Context // Context for cancellation

	// Progress callback (called periodically during benchmark)
	onProgress func(iters int64, elapsed time.Duration)
}

// NewAdaptiveBenchmark creates a new adaptive benchmark controller.
func NewAdaptiveBenchmark(benchTime time.Duration, minIter, maxIter int) *AdaptiveBenchmark {
	return &AdaptiveBenchmark{
		benchTime:     benchTime,
		minIterations: minIter,
		maxIterations: int64(maxIter),
		nextN:         1, // Start with 1 iteration like Go
		ctx:           context.Background(),
	}
}

// NewAdaptiveBenchmarkWithContext creates a new adaptive benchmark controller with context.
func NewAdaptiveBenchmarkWithContext(ctx context.Context, benchTime time.Duration, minIter, maxIter int) *AdaptiveBenchmark {
	return &AdaptiveBenchmark{
		benchTime:     benchTime,
		minIterations: minIter,
		maxIterations: int64(maxIter),
		nextN:         1, // Start with 1 iteration like Go
		ctx:           ctx,
	}
}

// SetProgressCallback sets a callback to be called periodically during the benchmark.
func (ab *AdaptiveBenchmark) SetProgressCallback(fn func(iters int64, elapsed time.Duration)) {
	ab.onProgress = fn
}

// NotifyProgress calls the progress callback if set.
func (ab *AdaptiveBenchmark) NotifyProgress() {
	if ab.onProgress != nil {
		ab.onProgress(ab.totalN, ab.totalDuration)
	}
}

// predictN calculates the next iteration count using Go's exact algorithm.
// This is the core of Go's benchmark auto-scaling: it extrapolates from
// previous runs to estimate how many iterations are needed to reach the
// target duration.
func (ab *AdaptiveBenchmark) predictN(goalns, prevIters, prevns, last int64) int64 {
	if prevns == 0 {
		prevns = 1 // Avoid divide by zero
	}

	// Extrapolate: n = goalns * prevIters / prevns
	// Order matters: multiply first to avoid precision loss
	n := goalns * prevIters / prevns

	// Add 20% buffer for timing variability
	n += n / 5

	// Cap growth at 100x previous (protects against timing errors)
	if n > 100*last {
		n = 100 * last
	}

	// Ensure at least one more iteration than last (guarantee progress)
	if n < last+1 {
		n = last + 1
	}

	// Safety limit
	if n > ab.maxIterations {
		n = ab.maxIterations
	}

	return n
}

// ShouldContinue returns true if more benchmark runs are needed.
func (ab *AdaptiveBenchmark) ShouldContinue() bool {
	// Check for context cancellation first
	if ab.ctx != nil {
		select {
		case <-ab.ctx.Done():
			return false
		default:
		}
	}

	// Always run at least minIterations
	if ab.totalN < int64(ab.minIterations) {
		return true
	}
	// Continue until we reach the target duration or max iterations
	return ab.totalDuration < ab.benchTime && ab.totalN < ab.maxIterations
}

// IsCancelled returns true if the benchmark was cancelled via context.
func (ab *AdaptiveBenchmark) IsCancelled() bool {
	if ab.ctx == nil {
		return false
	}
	select {
	case <-ab.ctx.Done():
		return true
	default:
		return false
	}
}

// NextN returns the number of iterations for the next run.
func (ab *AdaptiveBenchmark) NextN() int {
	if !ab.started {
		ab.started = true
		return 1 // First run: start with 1 iteration
	}
	return int(ab.nextN)
}

// RecordRun records the results of a benchmark run and calculates the next N.
func (ab *AdaptiveBenchmark) RecordRun(n int, elapsed time.Duration) {
	ab.lastN = int64(n)
	ab.lastDuration = elapsed
	ab.totalDuration += elapsed
	ab.totalN += int64(n)

	// Calculate next N using Go's prediction algorithm
	if ab.ShouldContinue() {
		goalns := ab.benchTime.Nanoseconds()
		prevns := elapsed.Nanoseconds()
		ab.nextN = ab.predictN(goalns, ab.lastN, prevns, ab.lastN)
	}
}

// TotalIterations returns the total iterations executed.
func (ab *AdaptiveBenchmark) TotalIterations() int {
	return int(ab.totalN)
}

// TotalDuration returns the total benchmark duration.
func (ab *AdaptiveBenchmark) TotalDuration() time.Duration {
	return ab.totalDuration
}
