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
	onPhaseChange    func(phase string, driver string)
	onProgress       func(current, total int, message string)
	onResult         func(result *BenchmarkResult)
	onLog            func(message string)
	onTableRow       func(result *BenchmarkResult)
	onSectionHeader  func(objectSize int, driver string)

	mu sync.Mutex
}

// NewBenchmarkRunner creates a new benchmark runner.
func NewBenchmarkRunner(cfg *Config) *BenchmarkRunner {
	return &BenchmarkRunner{
		config:  cfg,
		drivers: FilterDrivers(DefaultDrivers(), cfg.Drivers),
		results: make([]*BenchmarkResult, 0),
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

// Run executes the full benchmark suite.
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
		return nil, fmt.Errorf("no drivers available")
	}

	payloadSizes := r.config.PayloadSizes()
	threadCounts := r.config.ThreadCounts()

	// Phase 1: Setup - Upload objects for each driver
	if r.onPhaseChange != nil {
		r.onPhaseChange("SETUP", "")
	}

	for _, driver := range availableDrivers {
		if err := r.setupDriver(ctx, driver, payloadSizes); err != nil {
			r.log(fmt.Sprintf("[ERROR] Setup failed for %s: %v", driver.Name, err))
			continue
		}
	}

	// Phase 2: Benchmark
	if r.onPhaseChange != nil {
		r.onPhaseChange("BENCHMARK", "")
	}

	for _, size := range payloadSizes {
		if r.onSectionHeader != nil {
			r.onSectionHeader(size, "")
		}

		for _, driver := range availableDrivers {
			for _, threads := range threadCounts {
				select {
				case <-ctx.Done():
					return r.results, ctx.Err()
				default:
				}

				result, err := r.runBenchmark(ctx, driver, size, threads)
				if err != nil {
					r.log(fmt.Sprintf("[ERROR] Benchmark failed: %v", err))
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
	}

	// Phase 3: Cleanup
	if r.onPhaseChange != nil {
		r.onPhaseChange("CLEANUP", "")
	}

	for _, driver := range availableDrivers {
		if err := r.cleanupDriver(ctx, driver); err != nil {
			r.log(fmt.Sprintf("[WARN] Cleanup failed for %s: %v", driver.Name, err))
		}
	}

	if r.onPhaseChange != nil {
		r.onPhaseChange("DONE", "")
	}

	return r.results, nil
}

func (r *BenchmarkRunner) setupDriver(ctx context.Context, driver *DriverConfig, sizes []int) error {
	client, err := NewS3Client(ctx, driver)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// Check bucket exists
	if err := client.CheckAvailable(ctx); err != nil {
		return fmt.Errorf("bucket not available: %w", err)
	}

	// Upload objects for each size
	for _, size := range sizes {
		r.log(fmt.Sprintf("[%s] Uploading %s objects", driver.Name, FormatSize(size)))

		// Generate random data
		data := make([]byte, size)
		rand.Read(data)

		// Upload samples objects
		for i := 0; i < r.config.Samples; i++ {
			key := objectKey(driver.Name, size, i)

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

			r.progress(i+1, r.config.Samples, fmt.Sprintf("Uploading %s objects", FormatSize(size)))
		}
	}

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

	// Run downloads
	for i := 0; i < r.config.Samples; i++ {
		select {
		case <-ctx.Done():
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
			collector.AddSample(ttfbReader.TTFB(), ttlb, n, err)

			atomic.AddInt64(&completed, 1)
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Calculate statistics
	ttfb, ttlb, throughput, errors := collector.Calculate()

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

func (r *BenchmarkRunner) cleanupDriver(ctx context.Context, driver *DriverConfig) error {
	client, err := NewS3Client(ctx, driver)
	if err != nil {
		return err
	}

	r.log(fmt.Sprintf("[%s] Deleting test objects", driver.Name))

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
