package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// BenchmarkRunner orchestrates benchmark execution.
type BenchmarkRunner struct {
	config  *Config
	drivers []*DriverConfig
	results []*BenchmarkResult

	// Callbacks for UI updates
	onPhaseChange      func(phase string, driver string)
	onProgress         func(current, total int, message string)
	onResult           func(result *BenchmarkResult)
	onLog              func(message string)
	onTableRow         func(result *BenchmarkResult)
	onSectionHeader    func(objectSize int, driver string)
	onThroughputSample func(driver string, throughput float64, timestamp time.Time)
	onDriverProgress   func(driver string, completed, total int, throughput float64)
	onConfigChange     func(objectSize, threads int)
	onLatencySample    func(driver string, ttfb, ttlb time.Duration)
	onDriverError      func(driver string, err error)

	// Track failed drivers
	failedDrivers map[string]bool

	mu sync.Mutex
}

// NewBenchmarkRunner creates a new benchmark runner.
func NewBenchmarkRunner(cfg *Config) *BenchmarkRunner {
	return &BenchmarkRunner{
		config:        cfg,
		drivers:       FilterDrivers(DefaultDrivers(), cfg.Drivers),
		results:       make([]*BenchmarkResult, 0),
		failedDrivers: make(map[string]bool),
	}
}

// SetCallbacks sets the UI callback functions.
func (r *BenchmarkRunner) SetCallbacks(
	onPhaseChange func(phase string, driver string),
	onProgress func(current, total int, message string),
	onResult func(result *BenchmarkResult),
	onLog func(message string),
	onTableRow func(result *BenchmarkResult),
	onSectionHeader func(objectSize int, driver string),
) {
	r.onPhaseChange = onPhaseChange
	r.onProgress = onProgress
	r.onResult = onResult
	r.onLog = onLog
	r.onTableRow = onTableRow
	r.onSectionHeader = onSectionHeader
}

// SetDashboardCallbacks sets additional callbacks for the dashboard.
func (r *BenchmarkRunner) SetDashboardCallbacks(
	onThroughputSample func(driver string, throughput float64, timestamp time.Time),
	onDriverProgress func(driver string, completed, total int, throughput float64),
	onConfigChange func(objectSize, threads int),
) {
	r.onThroughputSample = onThroughputSample
	r.onDriverProgress = onDriverProgress
	r.onConfigChange = onConfigChange
}

// SetLatencyCallback sets the callback for latency samples.
func (r *BenchmarkRunner) SetLatencyCallback(fn func(driver string, ttfb, ttlb time.Duration)) {
	r.onLatencySample = fn
}

// SetDriverErrorCallback sets the callback for driver errors.
func (r *BenchmarkRunner) SetDriverErrorCallback(fn func(driver string, err error)) {
	r.onDriverError = fn
}

func (r *BenchmarkRunner) latencySample(driver string, ttfb, ttlb time.Duration) {
	if r.onLatencySample != nil {
		r.onLatencySample(driver, ttfb, ttlb)
	}
}

func (r *BenchmarkRunner) log(msg string) {
	if r.onLog != nil {
		r.onLog(msg)
	}
}

func (r *BenchmarkRunner) progress(current, total int, msg string) {
	if r.onProgress != nil {
		r.onProgress(current, total, msg)
	}
}

func (r *BenchmarkRunner) throughputSample(driver string, throughput float64) {
	if r.onThroughputSample != nil {
		r.onThroughputSample(driver, throughput, time.Now())
	}
}

func (r *BenchmarkRunner) driverProgress(driver string, completed, total int, throughput float64) {
	if r.onDriverProgress != nil {
		r.onDriverProgress(driver, completed, total, throughput)
	}
}

func (r *BenchmarkRunner) configChange(objectSize, threads int) {
	if r.onConfigChange != nil {
		r.onConfigChange(objectSize, threads)
	}
}

func (r *BenchmarkRunner) driverError(driver string, err error) {
	r.mu.Lock()
	r.failedDrivers[driver] = true
	r.mu.Unlock()
	if r.onDriverError != nil {
		r.onDriverError(driver, err)
	}
}

// Run executes the full benchmark suite.
// Uses per-category cleanup: for each object size, we setup, benchmark, then cleanup
// before moving to the next size. This prevents storage bloat during long runs.
func (r *BenchmarkRunner) Run(ctx context.Context) ([]*BenchmarkResult, error) {
	// Check available drivers
	var availableDrivers []*DriverConfig
	for _, driver := range r.drivers {
		if err := driver.CheckConnectivity(); err != nil {
			r.log(fmt.Sprintf("[SKIP] %s: %v", driver.Name, err))
			continue
		}
		r.log(fmt.Sprintf("[OK] %s at %s", driver.Name, driver.Endpoint))
		availableDrivers = append(availableDrivers, driver)
	}

	if len(availableDrivers) == 0 {
		return nil, fmt.Errorf(`no drivers available

None of the S3 storage backends are running. To start them:

  1. Start docker services:
     docker compose -f ./docker/s3/all/docker-compose.yaml up -d --wait

  2. Or use the --docker-up flag:
     go run ./cmd/s3_bench --docker-up --quick

Available drivers and ports:
  - liteio:     localhost:9200  (credentials: liteio / liteio123)
  - minio:      localhost:9000  (credentials: minioadmin / minioadmin)
  - rustfs:     localhost:9100  (credentials: rustfsadmin / rustfsadmin)
  - seaweedfs:  localhost:8333  (credentials: admin / adminpassword)
  - localstack: localhost:4566  (credentials: test / test)
  - liteio_mem: localhost:9201  (credentials: liteio / liteio123)`)
	}

	payloadSizes := r.config.PayloadSizes()
	threadCounts := r.config.ThreadCounts()

	// Per-category benchmark: for each size, setup -> benchmark -> cleanup
	for sizeIdx, size := range payloadSizes {
		select {
		case <-ctx.Done():
			return r.results, ctx.Err()
		default:
		}

		// === SETUP for this size ===
		if r.onPhaseChange != nil {
			r.onPhaseChange("SETUP", "")
		}
		if r.onSectionHeader != nil {
			r.onSectionHeader(size, "")
		}

		r.log(fmt.Sprintf("=== Category %d/%d: %s objects ===", sizeIdx+1, len(payloadSizes), FormatSize(size)))

		// Setup: Upload objects for this size across all drivers
		for i, driver := range availableDrivers {
			// Skip already failed drivers
			if r.failedDrivers[driver.Name] {
				continue
			}

			r.progress(0, r.config.Samples, fmt.Sprintf("[%s] Uploading %s objects (%d/%d drivers)...", driver.Name, FormatSize(size), i+1, len(availableDrivers)))
			r.driverProgress(driver.Name, 0, r.config.Samples, 0)

			if err := r.setupDriverForSize(ctx, driver, size); err != nil {
				errMsg := fmt.Sprintf("[ERROR] Setup failed for %s: %v", driver.Name, err)
				r.log(errMsg)
				r.driverError(driver.Name, err)
				r.progress(0, r.config.Samples, fmt.Sprintf("[%s] FAILED - moving to next driver...", driver.Name))
				continue
			}
		}

		// === BENCHMARK for this size ===
		if r.onPhaseChange != nil {
			r.onPhaseChange("BENCHMARK", "")
		}

		for _, driver := range availableDrivers {
			// Skip drivers that failed setup
			if r.failedDrivers[driver.Name] {
				continue
			}

			for _, threads := range threadCounts {
				select {
				case <-ctx.Done():
					// Cleanup before returning on cancellation
					r.cleanupSizeForAllDrivers(ctx, availableDrivers, size)
					return r.results, ctx.Err()
				default:
				}

				// Notify config change and starting benchmark
				r.configChange(size, threads)
				r.progress(0, r.config.Samples, fmt.Sprintf("[%s] Benchmarking %s @ %d threads...", driver.Name, FormatSize(size), threads))

				result, err := r.runBenchmark(ctx, driver, size, threads)
				if err != nil {
					errMsg := fmt.Sprintf("[ERROR] Benchmark failed for %s: %v", driver.Name, err)
					r.log(errMsg)
					r.progress(0, r.config.Samples, fmt.Sprintf("[%s] FAILED - continuing...", driver.Name))
					continue
				}

				r.mu.Lock()
				r.results = append(r.results, result)
				r.mu.Unlock()

				if r.onTableRow != nil {
					r.onTableRow(result)
				}
				if r.onResult != nil {
					r.onResult(result)
				}
			}
		}

		// === CLEANUP for this size ===
		if r.onPhaseChange != nil {
			r.onPhaseChange("CLEANUP", "")
		}

		r.cleanupSizeForAllDrivers(ctx, availableDrivers, size)
		r.log(fmt.Sprintf("=== Completed category: %s objects ===", FormatSize(size)))
	}

	if r.onPhaseChange != nil {
		r.onPhaseChange("DONE", "")
	}

	return r.results, nil
}

// cleanupSizeForAllDrivers cleans up objects for a specific size across all drivers.
func (r *BenchmarkRunner) cleanupSizeForAllDrivers(ctx context.Context, drivers []*DriverConfig, size int) {
	for i, driver := range drivers {
		r.progress(i, len(drivers), fmt.Sprintf("[%s] Cleaning up %s objects...", driver.Name, FormatSize(size)))
		if err := r.cleanupDriverForSize(ctx, driver, size); err != nil {
			r.log(fmt.Sprintf("[WARN] Cleanup failed for %s: %v", driver.Name, err))
		}
	}
	r.progress(len(drivers), len(drivers), fmt.Sprintf("Cleanup complete for %s objects", FormatSize(size)))
}

// setupDriverForSize uploads test objects for a single size to a driver.
func (r *BenchmarkRunner) setupDriverForSize(ctx context.Context, driver *DriverConfig, size int) error {
	client, err := NewS3Client(ctx, driver)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// Check bucket exists
	if err := client.CheckAvailable(ctx); err != nil {
		return fmt.Errorf("bucket not available: %w", err)
	}

	r.log(fmt.Sprintf("[%s] Uploading %d x %s objects", driver.Name, r.config.Samples, FormatSize(size)))

	// Generate random data
	data := make([]byte, size)
	rand.Read(data)

	// Track upload throughput
	sizeStart := time.Now()
	var bytesUploaded int64

	// Upload samples objects
	for i := 0; i < r.config.Samples; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		key := objectKey(driver.Name, size, i)
		opStart := time.Now()

		_, err := client.Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:        aws.String(driver.Bucket),
			Key:           aws.String(key),
			Body:          bytes.NewReader(data),
			ContentLength: aws.Int64(int64(size)),
			ContentType:   aws.String("application/octet-stream"),
		})
		if err != nil {
			return fmt.Errorf("upload object: %w", err)
		}

		bytesUploaded += int64(size)

		// Calculate upload throughput
		elapsed := time.Since(sizeStart).Seconds()
		var throughput float64
		if elapsed > 0 {
			throughput = float64(bytesUploaded) / elapsed / 1024 / 1024 // MB/s
		}

		// Report progress with throughput
		opDuration := time.Since(opStart)
		r.progress(i+1, r.config.Samples, fmt.Sprintf("[%s] Uploading %s (%d/%d) - %.1f MB/s - %v/obj",
			driver.Name, FormatSize(size), i+1, r.config.Samples, throughput, opDuration.Round(time.Millisecond)))
		r.driverProgress(driver.Name, i+1, r.config.Samples, throughput)
		r.throughputSample(driver.Name, throughput)
	}

	// Log size completion
	totalElapsed := time.Since(sizeStart)
	avgThroughput := float64(bytesUploaded) / totalElapsed.Seconds() / 1024 / 1024
	r.log(fmt.Sprintf("[%s] Uploaded %d x %s in %v (%.1f MB/s avg)",
		driver.Name, r.config.Samples, FormatSize(size), totalElapsed.Round(time.Millisecond), avgThroughput))

	return nil
}

func (r *BenchmarkRunner) runBenchmark(ctx context.Context, driver *DriverConfig, objectSize, threads int) (*BenchmarkResult, error) {
	client, err := NewS3Client(ctx, driver)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	collector := NewCollector()
	startTime := time.Now()

	// Create worker pool
	sem := make(chan struct{}, threads)
	var wg sync.WaitGroup
	var completed int64

	// Throughput tracking for real-time updates
	var totalBytesDownloaded int64
	lastReportTime := time.Now()
	reportInterval := 250 * time.Millisecond

	// Goroutine to report throughput samples
	stopReporter := make(chan struct{})
	go func() {
		ticker := time.NewTicker(reportInterval)
		defer ticker.Stop()

		for {
			select {
			case <-stopReporter:
				return
			case <-ticker.C:
				bytes := atomic.LoadInt64(&totalBytesDownloaded)
				elapsed := time.Since(lastReportTime).Seconds()
				if elapsed > 0 && bytes > 0 {
					throughput := float64(bytes) / elapsed / 1024 / 1024
					r.throughputSample(driver.Name, throughput)

					// Update driver progress
					comp := atomic.LoadInt64(&completed)
					r.driverProgress(driver.Name, int(comp), r.config.Samples, throughput)
				}
			}
		}
	}()

	// Run downloads
	for i := 0; i < r.config.Samples; i++ {
		select {
		case <-ctx.Done():
			close(stopReporter)
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			key := objectKey(driver.Name, objectSize, idx)
			opStart := time.Now()

			// Download object
			output, err := client.Client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(driver.Bucket),
				Key:    aws.String(key),
			})

			if err != nil {
				collector.AddSample(0, 0, 0, err)
				return
			}
			defer output.Body.Close()

			// Read with TTFB tracking
			ttfbReader := NewTTFBReader(output.Body, opStart)
			n, err := io.Copy(io.Discard, ttfbReader)

			ttlb := time.Since(opStart)
			ttfb := ttfbReader.TTFB()
			collector.AddSample(ttfb, ttlb, n, err)

			// Send latency sample to dashboard
			r.latencySample(driver.Name, ttfb, ttlb)

			// Update counters
			atomic.AddInt64(&totalBytesDownloaded, n)
			atomic.AddInt64(&completed, 1)

			// Update progress
			comp := atomic.LoadInt64(&completed)
			r.progress(int(comp), r.config.Samples, fmt.Sprintf("[%s] %s @ %d threads", driver.Name, FormatSize(objectSize), threads))
		}(i)
	}

	wg.Wait()
	close(stopReporter)
	duration := time.Since(startTime)

	// Calculate statistics
	ttfb, ttlb, throughput, errors := collector.Calculate()

	// Send final throughput sample
	r.throughputSample(driver.Name, throughput)
	r.driverProgress(driver.Name, r.config.Samples, r.config.Samples, throughput)

	return &BenchmarkResult{
		Driver:     driver.Name,
		ObjectSize: objectSize,
		Threads:    threads,
		Throughput: throughput,
		TTFB:       ttfb,
		TTLB:       ttlb,
		TotalBytes: int64(objectSize) * int64(r.config.Samples),
		Duration:   duration,
		Samples:    r.config.Samples,
		Errors:     errors,
	}, nil
}

// cleanupDriverForSize deletes test objects for a specific size from a driver.
func (r *BenchmarkRunner) cleanupDriverForSize(ctx context.Context, driver *DriverConfig, size int) error {
	client, err := NewS3Client(ctx, driver)
	if err != nil {
		return err
	}

	// Use size-specific prefix for targeted cleanup
	prefix := fmt.Sprintf("s3bench-%s-%s/", driver.Name, FormatSize(size))

	// List and delete objects with this prefix
	paginator := s3.NewListObjectsV2Paginator(client.Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(driver.Bucket),
		Prefix: aws.String(prefix),
	})

	var deleted int
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list objects: %w", err)
		}

		for _, obj := range page.Contents {
			_, err := client.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(driver.Bucket),
				Key:    obj.Key,
			})
			if err != nil {
				r.log(fmt.Sprintf("[WARN] Failed to delete %s: %v", *obj.Key, err))
			}
			deleted++
		}
	}

	if deleted > 0 {
		r.log(fmt.Sprintf("[%s] Deleted %d x %s objects", driver.Name, deleted, FormatSize(size)))
	}
	return nil
}

// cleanupDriver deletes all test objects for a driver (used for full cleanup).
func (r *BenchmarkRunner) cleanupDriver(ctx context.Context, driver *DriverConfig) error {
	client, err := NewS3Client(ctx, driver)
	if err != nil {
		return err
	}

	r.log(fmt.Sprintf("[%s] Deleting all test objects", driver.Name))

	// List and delete all test objects
	paginator := s3.NewListObjectsV2Paginator(client.Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(driver.Bucket),
		Prefix: aws.String(fmt.Sprintf("s3bench-%s-", driver.Name)),
	})

	var deleted int
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list objects: %w", err)
		}

		for _, obj := range page.Contents {
			_, err := client.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(driver.Bucket),
				Key:    obj.Key,
			})
			if err != nil {
				r.log(fmt.Sprintf("[WARN] Failed to delete %s: %v", *obj.Key, err))
			}
			deleted++
			r.progress(deleted, deleted+1, "Deleting objects")
			r.driverProgress(driver.Name, deleted, deleted+1, 0)
		}
	}

	r.log(fmt.Sprintf("[%s] Deleted %d objects", driver.Name, deleted))
	return nil
}

// objectKey generates a unique key for a test object.
func objectKey(driver string, size, index int) string {
	// Create deterministic key based on driver, size, and index
	h := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d-%d", driver, size, index, os.Getpid())))
	return fmt.Sprintf("s3bench-%s-%s/%d/%s", driver, FormatSize(size), index, hex.EncodeToString(h[:8]))
}

// Results returns all benchmark results.
func (r *BenchmarkRunner) Results() []*BenchmarkResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.results
}
