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

	// Determine max concurrency for this driver
	maxConc := r.config.Concurrency
	if driver.MaxConcurrency > 0 && driver.MaxConcurrency < maxConc {
		maxConc = driver.MaxConcurrency
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

	// Run parallel benchmarks
	for _, size := range r.config.ObjectSizes[:1] { // Use first size only
		if err := r.benchmarkParallelWrite(ctx, bucket, driver.Name, size, maxConc); err != nil {
			r.logger("  ParallelWrite/C%d failed: %v", maxConc, err)
		}
		if err := r.benchmarkParallelRead(ctx, bucket, driver.Name, size, maxConc); err != nil {
			r.logger("  ParallelRead/C%d failed: %v", maxConc, err)
		}
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

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for i := 0; i < r.config.Iterations; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			key := r.uniqueKey("parallel-write")
			timer := NewTimer()

			_, err := bucket.Write(ctx, key, bytes.NewReader(data), int64(size), "application/octet-stream", nil)

			collector.RecordWithError(timer.Elapsed(), err)
			progress.Increment()
		}()
	}

	wg.Wait()
	progress.DoneWithStats(int64(r.config.Iterations) * int64(size))

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

	// Benchmark
	collector := NewCollector()
	progress := NewProgress(operation, r.config.Iterations, true)

	var wg sync.WaitGroup
	var keyIdx uint64
	sem := make(chan struct{}, concurrency)

	for i := 0; i < r.config.Iterations; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			idx := atomic.AddUint64(&keyIdx, 1) % uint64(numObjects)
			timer := NewTimer()

			rc, _, err := bucket.Open(ctx, keys[idx], 0, 0, nil)
			if err == nil {
				io.Copy(io.Discard, rc)
				rc.Close()
			}

			collector.RecordWithError(timer.Elapsed(), err)
			progress.Increment()
		}()
	}

	wg.Wait()
	progress.DoneWithStats(int64(r.config.Iterations) * int64(size))

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
