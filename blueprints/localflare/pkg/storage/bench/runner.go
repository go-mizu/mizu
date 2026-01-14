package bench

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	_ "github.com/go-mizu/blueprints/localflare/pkg/storage/driver/exp/s3"
)

// Runner orchestrates benchmark execution.
type Runner struct {
	config       *Config
	drivers      []DriverConfig
	results      []*Metrics
	dockerStats  map[string]*DockerStats
	logger       func(format string, args ...any)
	resultsMu    sync.Mutex
	keyCounter   uint64
	dockerCollector *DockerStatsCollector
}

// NewRunner creates a new benchmark runner.
func NewRunner(cfg *Config) *Runner {
	return &Runner{
		config:       cfg,
		drivers:      FilterDrivers(AllDriverConfigs(), cfg.Drivers),
		results:      make([]*Metrics, 0),
		dockerStats:  make(map[string]*DockerStats),
		logger:       func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
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

		if err := r.benchmarkDriver(ctx, driver); err != nil {
			r.logger("Driver %s failed: %v", driver.Name, err)
			continue
		}

		// Collect Docker stats after benchmarks
		if r.config.DockerStats && driver.Container != "" {
			stats, err := r.dockerCollector.GetStats(ctx, driver.Container)
			if err == nil {
				r.dockerStats[driver.Name] = stats
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
		if err := r.benchmarkWrite(ctx, bucket, driver.Name, size); err != nil {
			r.logger("  Write/%s failed: %v", SizeLabel(size), err)
		}
	}

	// Run read benchmarks
	for _, size := range r.config.ObjectSizes {
		if err := r.benchmarkRead(ctx, bucket, driver.Name, size); err != nil {
			r.logger("  Read/%s failed: %v", SizeLabel(size), err)
		}
	}

	// Run stat benchmark
	if err := r.benchmarkStat(ctx, bucket, driver.Name); err != nil {
		r.logger("  Stat failed: %v", err)
	}

	// Run list benchmark
	if err := r.benchmarkList(ctx, bucket, driver.Name); err != nil {
		r.logger("  List failed: %v", err)
	}

	// Run delete benchmark
	if err := r.benchmarkDelete(ctx, bucket, driver.Name); err != nil {
		r.logger("  Delete failed: %v", err)
	}

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
				continue
			}

			if err := r.benchmarkParallelWrite(ctx, bucket, driver.Name, size, conc); err != nil {
				r.logger("  ParallelWrite/C%d failed: %v", conc, err)
			}
			if err := r.benchmarkParallelRead(ctx, bucket, driver.Name, size, conc); err != nil {
				r.logger("  ParallelRead/C%d failed: %v", conc, err)
			}
		}
	}

	// Run range read benchmarks
	if err := r.benchmarkRangeRead(ctx, bucket, driver.Name); err != nil {
		r.logger("  RangeRead failed: %v", err)
	}

	// Run copy benchmarks
	for _, size := range r.config.ObjectSizes[:1] { // Use first size only
		if err := r.benchmarkCopy(ctx, bucket, driver.Name, size); err != nil {
			r.logger("  Copy/%s failed: %v", SizeLabel(size), err)
		}
	}

	// Run mixed workload benchmarks
	if err := r.benchmarkMixedWorkload(ctx, bucket, driver.Name, maxConc); err != nil {
		r.logger("  MixedWorkload failed: %v", err)
	}

	// Run multipart benchmarks
	if err := r.benchmarkMultipart(ctx, bucket, driver.Name); err != nil {
		r.logger("  Multipart failed: %v", err)
	}

	// Run edge case benchmarks
	if err := r.benchmarkEdgeCases(ctx, bucket, driver.Name); err != nil {
		r.logger("  EdgeCases failed: %v", err)
	}

	// Cleanup
	r.cleanupBucket(ctx, bucket)

	return nil
}

func (r *Runner) benchmarkWrite(ctx context.Context, bucket storage.Bucket, driver string, size int) error {
	operation := fmt.Sprintf("Write/%s", SizeLabel(size))
	data := generateRandomData(size)

	// Warmup
	for i := 0; i < r.config.WarmupIterations; i++ {
		key := r.uniqueKey("warmup")
		bucket.Write(ctx, key, bytes.NewReader(data), int64(size), "application/octet-stream", nil)
	}

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	for i := 0; i < r.config.Iterations; i++ {
		key := r.uniqueKey("write")
		timer := NewTimer()

		_, err := bucket.Write(ctx, key, bytes.NewReader(data), int64(size), "application/octet-stream", nil)

		collector.RecordWithError(timer.Elapsed(), err)
		progress.Increment()
	}

	progress.DoneWithStats(int64(r.config.Iterations) * int64(size))

	metrics := collector.Metrics(operation, driver, size)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkRead(ctx context.Context, bucket storage.Bucket, driver string, size int) error {
	operation := fmt.Sprintf("Read/%s", SizeLabel(size))
	data := generateRandomData(size)

	// Pre-create objects
	keys := make([]string, r.config.Iterations)
	for i := range keys {
		keys[i] = r.uniqueKey("read")
		bucket.Write(ctx, keys[i], bytes.NewReader(data), int64(size), "application/octet-stream", nil)
	}

	// Warmup
	for i := 0; i < r.config.WarmupIterations && i < len(keys); i++ {
		rc, _, _ := bucket.Open(ctx, keys[i], 0, 0, nil)
		if rc != nil {
			io.Copy(io.Discard, rc)
			rc.Close()
		}
	}

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	for i := 0; i < r.config.Iterations; i++ {
		timer := NewTimer()

		rc, _, err := bucket.Open(ctx, keys[i%len(keys)], 0, 0, nil)
		if err == nil {
			io.Copy(io.Discard, rc)
			rc.Close()
		}

		collector.RecordWithError(timer.Elapsed(), err)
		progress.Increment()
	}

	progress.DoneWithStats(int64(r.config.Iterations) * int64(size))

	metrics := collector.Metrics(operation, driver, size)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkStat(ctx context.Context, bucket storage.Bucket, driver string) error {
	operation := "Stat"
	data := generateRandomData(1024)

	// Create test object
	key := r.uniqueKey("stat")
	bucket.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)

	// Warmup
	for i := 0; i < r.config.WarmupIterations; i++ {
		bucket.Stat(ctx, key, nil)
	}

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	for i := 0; i < r.config.Iterations; i++ {
		timer := NewTimer()

		_, err := bucket.Stat(ctx, key, nil)

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
		bucket.Write(ctx, key, bytes.NewReader(data), 100, "text/plain", nil)
	}

	// Warmup
	for i := 0; i < r.config.WarmupIterations; i++ {
		iter, _ := bucket.List(ctx, prefix, 0, 0, nil)
		if iter != nil {
			for {
				obj, err := iter.Next()
				if err != nil || obj == nil {
					break
				}
			}
			iter.Close()
		}
	}

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	for i := 0; i < r.config.Iterations; i++ {
		timer := NewTimer()

		iter, err := bucket.List(ctx, prefix, 0, 0, nil)
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
		bucket.Write(ctx, keys[i], bytes.NewReader(data), 1024, "application/octet-stream", nil)
	}

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	for i := 0; i < r.config.Iterations; i++ {
		timer := NewTimer()

		err := bucket.Delete(ctx, keys[i], nil)

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
		bucket.Write(ctx, keys[i], bytes.NewReader(data), int64(size), "application/octet-stream", nil)
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
			timer := NewTimer()

			rc, _, err := bucket.Open(opCtx, keys[idx], 0, 0, nil)
			if err == nil {
				io.Copy(io.Discard, rc)
				rc.Close()
			}

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

func (r *Runner) addResult(m *Metrics) {
	r.resultsMu.Lock()
	r.results = append(r.results, m)
	r.resultsMu.Unlock()
}

func (r *Runner) uniqueKey(prefix string) string {
	n := atomic.AddUint64(&r.keyCounter, 1)
	return fmt.Sprintf("%s/%d/%d", prefix, time.Now().UnixNano(), n)
}

func (r *Runner) cleanupBucket(ctx context.Context, bucket storage.Bucket) {
	iter, err := bucket.List(ctx, "", 1000, 0, nil)
	if err != nil {
		return
	}
	defer iter.Close()

	for {
		obj, err := iter.Next()
		if err != nil || obj == nil {
			break
		}
		bucket.Delete(ctx, obj.Key, nil)
	}
}

func (r *Runner) generateReport() *Report {
	return &Report{
		Timestamp:   time.Now(),
		Config:      r.config,
		Results:     r.results,
		DockerStats: r.dockerStats,
	}
}

func (r *Runner) benchmarkRangeRead(ctx context.Context, bucket storage.Bucket, driver string) error {
	const totalSize = 1024 * 1024 // 1MB object
	data := generateRandomData(totalSize)

	// Create test object
	key := r.uniqueKey("range")
	_, err := bucket.Write(ctx, key, bytes.NewReader(data), int64(totalSize), "application/octet-stream", nil)
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

			rc, _, err := bucket.Open(ctx, key, rng.offset, rng.length, nil)
			if err == nil {
				io.Copy(io.Discard, rc)
				rc.Close()
			}

			collector.RecordWithError(timer.Elapsed(), err)
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
	_, err := bucket.Write(ctx, srcKey, bytes.NewReader(data), int64(size), "application/octet-stream", nil)
	if err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	for i := 0; i < r.config.Iterations; i++ {
		dstKey := r.uniqueKey("copy-dst")
		timer := NewTimer()

		_, err := bucket.Copy(ctx, dstKey, bucket.Name(), srcKey, nil)

		collector.RecordWithError(timer.Elapsed(), err)
		progress.Increment()
	}

	progress.DoneWithStats(int64(r.config.Iterations) * int64(size))
	metrics := collector.Metrics(operation, driver, size)
	r.addResult(metrics)

	return nil
}

func (r *Runner) benchmarkMixedWorkload(ctx context.Context, bucket storage.Bucket, driver string, concurrency int) error {
	objectSize := 16 * 1024 // 16KB
	data := generateRandomData(objectSize)

	// Pre-create objects for reading
	numObjects := 50
	keys := make([]string, numObjects)
	for i := 0; i < numObjects; i++ {
		keys[i] = r.uniqueKey("mixed")
		bucket.Write(ctx, keys[i], bytes.NewReader(data), int64(objectSize), "application/octet-stream", nil)
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
					rc, _, e := bucket.Open(ctx, keys[idx], 0, 0, nil)
					if e == nil {
						io.Copy(io.Discard, rc)
						rc.Close()
					}
					err = e
				} else {
					// Write operation
					key := r.uniqueKey("mixed-write")
					_, err = bucket.Write(ctx, key, bytes.NewReader(data), int64(objectSize), "application/octet-stream", nil)
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
	partSize := 5 * 1024 * 1024  // 5MB
	partCount := 3               // 15MB total
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
		mu, e := mp.InitMultipart(ctx, key, "application/octet-stream", nil)
		if e != nil {
			err = e
		} else {
			// Upload parts
			parts := make([]*storage.PartInfo, partCount)
			for p := 0; p < partCount && err == nil; p++ {
				part, e := mp.UploadPart(ctx, mu, p+1, bytes.NewReader(partData), int64(partSize), nil)
				if e != nil {
					mp.AbortMultipart(ctx, mu, nil)
					err = e
					break
				}
				parts[p] = part
			}

			// Complete
			if err == nil {
				_, err = mp.CompleteMultipart(ctx, mu, parts, nil)
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

			_, err := bucket.Write(ctx, key, bytes.NewReader(nil), 0, "application/octet-stream", nil)

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

			_, err := bucket.Write(ctx, key, bytes.NewReader(data), 100, "text/plain", nil)

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

			_, err := bucket.Write(ctx, key, bytes.NewReader(data), 100, "text/plain", nil)

			collector.RecordWithError(timer.Elapsed(), err)
			progress.Increment()
		}

		progress.Done()
		metrics := collector.Metrics(operation, driver, 100)
		r.addResult(metrics)
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
