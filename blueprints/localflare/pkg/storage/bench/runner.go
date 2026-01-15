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

// Run executes all benchmarks.
func (r *Runner) Run(ctx context.Context) (*Report, error) {
	r.logger("=== Storage Benchmark Suite ===")
	r.logger("Drivers: %d configured", len(r.drivers))
	r.logger("Iterations: %d (warmup: %d)", r.config.Iterations, r.config.WarmupIterations)
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
		r.logger("=== [%d/%d] Benchmarking %s ===", i+1, len(available), driver.Name)

		// Collect Docker stats before benchmarks (to show growth)
		if r.config.DockerStats && driver.Container != "" {
			r.logger("  Collecting initial Docker stats...")
			stats, err := r.dockerCollector.GetStats(ctx, driver.Container)
			if err == nil {
				r.logger("  Initial: Memory=%.1fMB, Volume=%.1fMB", stats.MemoryUsageMB, stats.VolumeSize)
			}
		}

		if err := r.benchmarkDriver(ctx, driver); err != nil {
			r.logger("Driver %s failed: %v", driver.Name, err)
			continue
		}

		// Collect Docker stats after benchmarks
		if r.config.DockerStats && driver.Container != "" {
			r.logger("  Collecting final Docker stats...")
			stats, err := r.dockerCollector.GetStats(ctx, driver.Container)
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
		label := fmt.Sprintf("Write/%s", SizeLabel(size))
		r.runBenchmark(ctx, bucket, label, func() error {
			return r.benchmarkWrite(ctx, bucket, driver.Name, size)
		})
	}

	// Run read benchmarks
	for _, size := range r.config.ObjectSizes {
		label := fmt.Sprintf("Read/%s", SizeLabel(size))
		r.runBenchmark(ctx, bucket, label, func() error {
			return r.benchmarkRead(ctx, bucket, driver.Name, size)
		})
	}

	// Run stat benchmark
	r.runBenchmark(ctx, bucket, "Stat", func() error {
		return r.benchmarkStat(ctx, bucket, driver.Name)
	})

	// Run list benchmark
	r.runBenchmark(ctx, bucket, "List", func() error {
		return r.benchmarkList(ctx, bucket, driver.Name)
	})

	// Run delete benchmark
	r.runBenchmark(ctx, bucket, "Delete", func() error {
		return r.benchmarkDelete(ctx, bucket, driver.Name)
	})

	// Run parallel benchmarks at multiple concurrency levels
	for _, size := range r.config.ObjectSizes[:1] { // Use first size only
		concLevels := r.config.ConcurrencyLevels
		if len(concLevels) == 0 {
			concLevels = []int{r.config.Concurrency}
		}

		for _, conc := range concLevels {
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
	r.runBenchmark(ctx, bucket, "RangeRead", func() error {
		return r.benchmarkRangeRead(ctx, bucket, driver.Name)
	})

	// Run copy benchmarks
	for _, size := range r.config.ObjectSizes[:1] { // Use first size only
		label := fmt.Sprintf("Copy/%s", SizeLabel(size))
		r.runBenchmark(ctx, bucket, label, func() error {
			return r.benchmarkCopy(ctx, bucket, driver.Name, size)
		})
	}

	if r.config.Verbose {
		r.logger("  [debug] Copy benchmarks done, starting MixedWorkload...")
	}

	// Run mixed workload benchmarks
	r.runBenchmark(ctx, bucket, "MixedWorkload", func() error {
		return r.benchmarkMixedWorkload(ctx, bucket, driver.Name, maxConc)
	})

	if r.config.Verbose {
		r.logger("  [debug] MixedWorkload done, starting Multipart...")
	}

	// Run multipart benchmarks
	r.runBenchmark(ctx, bucket, "Multipart", func() error {
		return r.benchmarkMultipart(ctx, bucket, driver.Name)
	})

	if r.config.Verbose {
		r.logger("  [debug] Multipart done, starting EdgeCases...")
	}

	// Run edge case benchmarks
	r.runBenchmark(ctx, bucket, "EdgeCases", func() error {
		return r.benchmarkEdgeCases(ctx, bucket, driver.Name)
	})

	if r.config.Verbose {
		r.logger("  [debug] EdgeCases done, starting FileCount...")
	}

	// Run file count benchmarks
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

	// Use adaptive iterations based on file size
	iterations := r.config.IterationsForSize(size)
	warmup := r.config.WarmupForSize(size)

	// Warmup
	for i := 0; i < warmup; i++ {
		key := r.uniqueKey("warmup")
		opCtx, cancel := r.opContextForSize(ctx, size)
		bucket.Write(opCtx, key, bytes.NewReader(data), int64(size), "application/octet-stream", nil)
		cancel()
	}

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, iterations, true)

	for i := 0; i < iterations; i++ {
		key := r.uniqueKey("write")
		timer := NewTimer()

		opCtx, cancel := r.opContextForSize(ctx, size)
		_, err := bucket.Write(opCtx, key, bytes.NewReader(data), int64(size), "application/octet-stream", nil)
		cancel()

		collector.RecordWithError(timer.Elapsed(), err)
		progress.Increment()
	}

	progress.DoneWithStats(int64(iterations) * int64(size))

	metrics := collector.Metrics(operation, driver, size)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkRead(ctx context.Context, bucket storage.Bucket, driver string, size int) error {
	operation := fmt.Sprintf("Read/%s", SizeLabel(size))
	data := generateRandomData(size)

	// Use adaptive iterations based on file size
	iterations := r.config.IterationsForSize(size)
	warmup := r.config.WarmupForSize(size)

	// Pre-create objects
	keys := make([]string, iterations)
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

	// Benchmark with TTFB tracking
	collector := NewCollector()
	progress := NewProgress(operation, iterations, true)

	for i := 0; i < iterations; i++ {
		start := time.Now()

		opCtx, cancel := r.opContextForSize(ctx, size)
		rc, _, err := bucket.Open(opCtx, keys[i%len(keys)], 0, 0, nil)
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
		progress.Increment()
	}

	progress.DoneWithStats(int64(iterations) * int64(size))

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

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	for i := 0; i < r.config.Iterations; i++ {
		timer := NewTimer()

		opCtx, cancel := r.opContext(ctx)
		_, err := bucket.Stat(opCtx, key, nil)
		cancel()

		collector.RecordWithError(timer.Elapsed(), err)
		progress.Increment()
	}

	progress.Done()

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

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	for i := 0; i < r.config.Iterations; i++ {
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
		progress.Increment()
	}

	progress.Done()

	metrics := collector.Metrics(operation, driver, 0)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkDelete(ctx context.Context, bucket storage.Bucket, driver string) error {
	operation := "Delete"
	data := generateRandomData(1024)

	// Pre-create objects for deletion
	keys := make([]string, r.config.Iterations)
	for i := range keys {
		keys[i] = r.uniqueKey("delete")
		opCtx, cancel := r.opContext(ctx)
		bucket.Write(opCtx, keys[i], bytes.NewReader(data), 1024, "application/octet-stream", nil)
		cancel()
	}

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	for i := 0; i < r.config.Iterations; i++ {
		timer := NewTimer()

		opCtx, cancel := r.opContext(ctx)
		err := bucket.Delete(opCtx, keys[i], nil)
		cancel()

		collector.RecordWithError(timer.Elapsed(), err)
		progress.Increment()
	}

	progress.Done()

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

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	var completed int64

	// Per-operation timeout (e.g. 10 seconds per write)
	opTimeout := 10 * time.Second

	for i := 0; i < r.config.Iterations; i++ {
		// Check if benchmark context is done
		select {
		case <-benchCtx.Done():
			r.logger("  %s: timeout after %d/%d iterations", operation, atomic.LoadInt64(&completed), r.config.Iterations)
			goto done
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			// Create per-operation context with timeout
			opCtx, opCancel := context.WithTimeout(benchCtx, opTimeout)
			defer opCancel()

			key := r.uniqueKey("parallel-write")
			timer := NewTimer()

			_, err := bucket.Write(opCtx, key, bytes.NewReader(data), int64(size), "application/octet-stream", nil)

			collector.RecordWithError(timer.Elapsed(), err)
			atomic.AddInt64(&completed, 1)
			progress.Increment()
		}()
	}

done:
	wg.Wait()
	progress.DoneWithStats(atomic.LoadInt64(&completed) * int64(size))

	metrics := collector.Metrics(operation, driver, size)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkParallelRead(ctx context.Context, bucket storage.Bucket, driver string, size, concurrency int) error {
	operation := fmt.Sprintf("ParallelRead/%s/C%d", SizeLabel(size), concurrency)
	data := generateRandomData(size)

	// Pre-create objects
	numObjects := 20
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

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	var wg sync.WaitGroup
	var keyIdx uint64
	var completed int64
	sem := make(chan struct{}, concurrency)

	// Per-operation timeout
	opTimeout := 10 * time.Second

	for i := 0; i < r.config.Iterations; i++ {
		// Check if benchmark context is done
		select {
		case <-benchCtx.Done():
			r.logger("  %s: timeout after %d/%d iterations", operation, atomic.LoadInt64(&completed), r.config.Iterations)
			goto done
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			// Create per-operation context with timeout
			opCtx, opCancel := context.WithTimeout(benchCtx, opTimeout)
			defer opCancel()

			idx := atomic.AddUint64(&keyIdx, 1) % uint64(numObjects)
			start := time.Now()

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

			atomic.AddInt64(&completed, 1)
			progress.Increment()
		}()
	}

done:
	wg.Wait()
	progress.DoneWithStats(atomic.LoadInt64(&completed) * int64(size))

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

	var dirs []string
	for {
		obj, err := iter.Next()
		if err != nil || obj == nil {
			break
		}
		if obj.IsDir {
			dirs = append(dirs, obj.Key)
			continue
		}
		opCtx, opCancel := r.opContext(cleanupCtx)
		bucket.Delete(opCtx, obj.Key, nil)
		opCancel()
	}

	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})
	for _, key := range dirs {
		opCtx, opCancel := r.opContext(cleanupCtx)
		bucket.Delete(opCtx, key, storage.Options{"recursive": true})
		opCancel()
	}
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
	r.cleanupBucket(ctx, bucket)
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
		progress := NewProgress(operation, r.config.Iterations, true)

		for i := 0; i < r.config.Iterations; i++ {
			timer := NewTimer()

			opCtx, cancel := r.opContextForSize(ctx, int(rng.length))
			rc, _, err := bucket.Open(opCtx, key, rng.offset, rng.length, nil)
			if err == nil {
				io.Copy(io.Discard, rc)
				rc.Close()
			}

			collector.RecordWithError(timer.Elapsed(), err)
			cancel()
			progress.Increment()
		}

		progress.DoneWithStats(int64(r.config.Iterations) * rng.length)
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

	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	for i := 0; i < r.config.Iterations; i++ {
		dstKey := r.uniqueKey("copy-dst")
		timer := NewTimer()

		opCtx, cancel := r.opContextForSize(ctx, size)
		_, err := bucket.Copy(opCtx, dstKey, bucket.Name(), srcKey, nil)
		cancel()

		collector.RecordWithError(timer.Elapsed(), err)
		progress.Increment()
	}

	progress.DoneWithStats(int64(r.config.Iterations) * int64(size))
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
		progress := NewProgress(operation, r.config.Iterations, true)

		var wg sync.WaitGroup
		sem := make(chan struct{}, concurrency)
		var opCounter uint64
		var keyIdx uint64

		for i := 0; i < r.config.Iterations; i++ {
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
				progress.Increment()
			}()
		}

		wg.Wait()
		progress.DoneWithStats(int64(r.config.Iterations) * int64(objectSize))
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

	// Fewer iterations for multipart (it's expensive)
	iterations := r.config.Iterations / 5
	if iterations < 3 {
		iterations = 3
	}

	progress := NewProgress(operation, iterations, true)

	for i := 0; i < iterations; i++ {
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
		progress.Increment()
	}

	progress.DoneWithStats(int64(iterations) * int64(totalSize))
	metrics := collector.Metrics(operation, driver, totalSize)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkEdgeCases(ctx context.Context, bucket storage.Bucket, driver string) error {
	// Empty object write
	{
		operation := "EdgeCase/EmptyObject"
		collector := NewCollector()
		iterations := r.config.Iterations / 2
		progress := NewProgress(operation, iterations, true)

		for i := 0; i < iterations; i++ {
			key := r.uniqueKey("empty")
			timer := NewTimer()

			opCtx, cancel := r.opContext(ctx)
			_, err := bucket.Write(opCtx, key, bytes.NewReader(nil), 0, "application/octet-stream", nil)
			cancel()

			collector.RecordWithError(timer.Elapsed(), err)
			progress.Increment()
		}

		progress.Done()
		metrics := collector.Metrics(operation, driver, 0)
		r.addResult(metrics)
	}

	// Long key names (256 chars)
	{
		operation := "EdgeCase/LongKey256"
		data := generateRandomData(100)
		collector := NewCollector()
		iterations := r.config.Iterations / 2
		progress := NewProgress(operation, iterations, true)

		longPrefix := "prefix/" + string(make([]byte, 200)) // Will be replaced
		for i := range longPrefix[7:] {
			longPrefix = longPrefix[:7+i] + "a" + longPrefix[8+i:]
		}

		for i := 0; i < iterations; i++ {
			key := fmt.Sprintf("prefix/%s/%d", string(bytes.Repeat([]byte("a"), 200)), i)
			timer := NewTimer()

			opCtx, cancel := r.opContext(ctx)
			_, err := bucket.Write(opCtx, key, bytes.NewReader(data), 100, "text/plain", nil)
			cancel()

			collector.RecordWithError(timer.Elapsed(), err)
			progress.Increment()
		}

		progress.Done()
		metrics := collector.Metrics(operation, driver, 100)
		r.addResult(metrics)
	}

	// Deep nesting
	{
		operation := "EdgeCase/DeepNested"
		data := generateRandomData(100)
		collector := NewCollector()
		iterations := r.config.Iterations / 2
		progress := NewProgress(operation, iterations, true)

		for i := 0; i < iterations; i++ {
			key := fmt.Sprintf("a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/%d", i)
			timer := NewTimer()

			opCtx, cancel := r.opContext(ctx)
			_, err := bucket.Write(opCtx, key, bytes.NewReader(data), 100, "text/plain", nil)
			cancel()

			collector.RecordWithError(timer.Elapsed(), err)
			progress.Increment()
		}

		progress.Done()
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

// benchmarkIterator handles both iteration-based and duration-based benchmarking.
type benchmarkIterator struct {
	duration      time.Duration
	iterations    int
	minIterations int
	current       int
	startTime     time.Time
}

func (r *Runner) newBenchmarkIterator() *benchmarkIterator {
	return &benchmarkIterator{
		duration:      r.config.Duration,
		iterations:    r.config.Iterations,
		minIterations: r.config.MinIterations,
		startTime:     time.Now(),
	}
}

// Next returns true if another iteration should be performed.
func (bi *benchmarkIterator) Next() bool {
	bi.current++

	// Duration-based mode
	if bi.duration > 0 {
		elapsed := time.Since(bi.startTime)
		// Continue if under duration OR haven't hit minimum iterations
		if elapsed < bi.duration || bi.current <= bi.minIterations {
			return true
		}
		return false
	}

	// Iteration-based mode
	return bi.current <= bi.iterations
}

// Count returns the current iteration count.
func (bi *benchmarkIterator) Count() int {
	return bi.current
}

// IsDurationMode returns true if running in duration mode.
func (bi *benchmarkIterator) IsDurationMode() bool {
	return bi.duration > 0
}
