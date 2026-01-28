package benchmark

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

// Runner orchestrates benchmarks across drivers.
type Runner struct {
	DataDir            string
	ParquetPath        string
	Drivers            []string
	Queries            []Query
	Concurrency        []int
	Iterations         int
	ThroughputDuration time.Duration // Duration for each throughput test
	SkipColdStart      bool          // Skip cold start test
	SkipPerQuery       bool          // Skip per-query stats
	FreshIndex         bool          // Force fresh indexing (delete existing)
	TestIncremental    bool          // Test incremental indexing
	IncrementalDocs    int64         // Number of docs for incremental test
	Logger             *log.Logger
}

// NewRunner creates a benchmark runner with defaults.
func NewRunner(dataDir, parquetPath string) *Runner {
	return &Runner{
		DataDir:            dataDir,
		ParquetPath:        parquetPath,
		Drivers:            fineweb.List(),
		Queries:            VietnameseQueries,
		Concurrency:        []int{1, 10, 50, 100},
		Iterations:         100,
		ThroughputDuration: 10 * time.Second,
		Logger:             log.Default(),
	}
}

// Report contains full benchmark results.
type Report struct {
	StartTime time.Time       `json:"start_time"`
	EndTime   time.Time       `json:"end_time"`
	System    SystemInfo      `json:"system"`
	Results   []*DriverResult `json:"results"`
}

// DriverResult contains benchmark results for a single driver.
type DriverResult struct {
	Name        string                        `json:"name"`
	Error       string                        `json:"error,omitempty"`
	Indexing    *IndexingMetrics              `json:"indexing,omitempty"`
	Incremental *IncrementalMetrics           `json:"incremental,omitempty"`
	IndexSize   int64                         `json:"index_size"`
	DocCount    int64                         `json:"doc_count"`
	Latency     *LatencyMetrics               `json:"latency,omitempty"`
	Throughput  *ThroughputMetrics            `json:"throughput,omitempty"`
	Concurrency map[int]*ThroughputMetrics    `json:"concurrency,omitempty"`
	ColdStart   time.Duration                 `json:"cold_start"`
	Memory      *MemoryMetrics                `json:"memory,omitempty"`
	QueryStats  map[string]*QueryMetrics      `json:"query_stats,omitempty"`
}

// Run executes the full benchmark suite.
func (r *Runner) Run(ctx context.Context) (*Report, error) {
	report := &Report{
		StartTime: time.Now(),
		System:    CollectSystemInfo(),
	}

	for _, driverName := range r.Drivers {
		select {
		case <-ctx.Done():
			return report, ctx.Err()
		default:
		}

		r.log("Benchmarking driver: %s", driverName)

		result, err := r.runDriver(ctx, driverName)
		if err != nil {
			result = &DriverResult{
				Name:  driverName,
				Error: err.Error(),
			}
			r.log("  Error: %v", err)
		}

		report.Results = append(report.Results, result)
	}

	report.EndTime = time.Now()
	return report, nil
}

// RunSingle benchmarks a single driver.
func (r *Runner) RunSingle(ctx context.Context, driverName string) (*DriverResult, error) {
	return r.runDriver(ctx, driverName)
}

func (r *Runner) runDriver(ctx context.Context, name string) (*DriverResult, error) {
	result := &DriverResult{
		Name:        name,
		Concurrency: make(map[int]*ThroughputMetrics),
		QueryStats:  make(map[string]*QueryMetrics),
	}

	cfg := fineweb.DriverConfig{
		DataDir:  r.DataDir,
		Language: "vie_Latn",
	}

	// Delete existing index if fresh indexing requested
	if r.FreshIndex {
		r.logPhase("CLEANUP", name, "Cleaning existing index...")
		r.deleteExistingIndex(name)
	}

	// Open driver
	r.logPhase("OPEN", name, "Opening driver...")
	driver, err := fineweb.Open(name, cfg)
	if err != nil {
		return result, fmt.Errorf("opening driver: %w", err)
	}
	defer driver.Close()

	// Check existing doc count
	var docCount int64
	if stats, ok := fineweb.AsStats(driver); ok {
		docCount, _ = stats.Count(ctx)
		r.log("    Existing docs: %d", docCount)
	}
	result.DocCount = docCount

	// Indexing benchmark (if needed and supported)
	if r.ParquetPath != "" {
		if indexer, ok := fineweb.AsIndexer(driver); ok {
			if docCount == 0 {
				r.logPhase("INDEX", name, "Indexing from scratch...")
				result.Indexing, result.Memory = r.benchmarkIndexing(ctx, indexer, name, true)
				// Update doc count
				if stats, ok := fineweb.AsStats(driver); ok {
					result.DocCount, _ = stats.Count(ctx)
				}
				if result.Indexing != nil {
					r.log("    Result: %d docs, %.0f docs/sec, peak memory %s",
						result.Indexing.TotalDocs,
						result.Indexing.DocsPerSec,
						FormatBytes(result.Indexing.PeakMemory))
				}
			} else {
				r.logPhase("INDEX", name, "Skipping indexing (already indexed: %d docs)", docCount)
			}

			// Incremental indexing test
			if r.TestIncremental && r.IncrementalDocs > 0 {
				r.logPhase("INCREMENTAL", name, "Testing incremental indexing (%d docs)...", r.IncrementalDocs)
				result.Incremental = r.benchmarkIncrementalIndexing(ctx, indexer, name)
			}
		}
	}

	// Measure index size
	r.logPhase("SIZE", name, "Measuring index size...")
	result.IndexSize = r.measureIndexSize(name)
	r.log("    Index size: %s", FormatBytes(result.IndexSize))

	// Search latency benchmark
	r.logPhase("LATENCY", name, "Measuring latency (%d iterations x %d queries = %d ops)...",
		r.Iterations, len(r.Queries), r.Iterations*len(r.Queries))
	result.Latency = r.benchmarkLatency(ctx, driver)
	if result.Latency != nil {
		r.log("    Result: p50=%v p95=%v p99=%v avg=%v",
			result.Latency.P50, result.Latency.P95, result.Latency.P99, result.Latency.Avg)
	}

	// Throughput benchmark (single thread baseline)
	r.logPhase("THROUGHPUT", name, "Measuring single-thread throughput (%v)...", r.ThroughputDuration)
	result.Throughput = r.benchmarkThroughput(ctx, driver, 1)
	if result.Throughput != nil {
		r.log("    Result: %.1f QPS (single-thread baseline)", result.Throughput.QPS)
	}

	// Concurrent search benchmark
	r.logPhase("CONCURRENCY", name, "Measuring concurrent throughput...")
	for i, n := range r.Concurrency {
		r.log("    [%d/%d] Testing %d goroutines...", i+1, len(r.Concurrency), n)
		result.Concurrency[n] = r.benchmarkThroughput(ctx, driver, n)
		if result.Concurrency[n] != nil {
			r.log("      %.1f QPS @ %d goroutines", result.Concurrency[n].QPS, n)
		}
	}

	// Cold start benchmark
	if !r.SkipColdStart {
		r.logPhase("COLD_START", name, "Measuring cold start time...")
		result.ColdStart = r.benchmarkColdStart(ctx, name, cfg)
		r.log("    Cold start: %v", result.ColdStart)
	} else {
		r.logPhase("COLD_START", name, "Skipping (disabled)")
	}

	// Per-query stats
	if !r.SkipPerQuery {
		r.logPhase("QUERY_STATS", name, "Collecting per-query stats (%d queries)...", len(r.Queries))
		for _, q := range r.Queries {
			qm := r.benchmarkQuery(ctx, driver, q)
			result.QueryStats[q.Text] = qm
		}
	}

	r.logPhase("COMPLETE", name, "Benchmark complete")
	return result, nil
}

func (r *Runner) deleteExistingIndex(name string) {
	patterns := []string{
		filepath.Join(r.DataDir, "vie_Latn."+name),
		filepath.Join(r.DataDir, "vie_Latn."+name+"db"),
		filepath.Join(r.DataDir, "fineweb."+name),
	}
	for _, pattern := range patterns {
		if err := os.RemoveAll(pattern); err == nil {
			r.log("    Deleted: %s", pattern)
		}
	}
}

func (r *Runner) benchmarkIndexing(ctx context.Context, indexer fineweb.Indexer, name string, fromScratch bool) (*IndexingMetrics, *MemoryMetrics) {
	reader := fineweb.NewParquetReader(r.ParquetPath)
	total, err := reader.CountDocuments(ctx)
	if err != nil {
		r.log("    Error counting documents: %v", err)
		return nil, nil
	}
	r.log("    Total documents to index: %d", total)

	memTracker := NewMemoryTracker()
	start := time.Now()
	lastLog := start

	var imported int64
	progress := func(n, _ int64) {
		imported = n
		memTracker.Sample()
		// Log every 5 seconds or every 10000 docs
		if time.Since(lastLog) > 5*time.Second || n%10000 == 0 {
			elapsed := time.Since(start)
			rate := float64(n) / elapsed.Seconds()
			remaining := time.Duration(float64(total-n)/rate) * time.Second
			r.log("    Progress: %d/%d docs (%.0f/sec, ETA: %v)", n, total, rate, remaining.Round(time.Second))
			lastLog = time.Now()
		}
	}

	docs := reader.ReadAll(ctx)
	err = indexer.Import(ctx, docs, progress)
	duration := time.Since(start)

	if err != nil {
		r.log("    Error: %v", err)
		return nil, nil
	}

	indexing := &IndexingMetrics{
		Duration:    duration,
		DocsPerSec:  float64(imported) / duration.Seconds(),
		PeakMemory:  memTracker.Peak(),
		TotalDocs:   total,
		FromScratch: fromScratch,
	}

	memory := &MemoryMetrics{
		IndexingPeak: memTracker.Peak(),
	}

	r.log("    Completed: %d docs in %v (%.0f docs/sec)", imported, duration.Round(time.Millisecond), indexing.DocsPerSec)

	return indexing, memory
}

func (r *Runner) benchmarkIncrementalIndexing(ctx context.Context, indexer fineweb.Indexer, _ string) *IncrementalMetrics {
	// Get current count
	var startCount int64
	if stats, ok := indexer.(fineweb.Stats); ok {
		startCount, _ = stats.Count(ctx)
	}

	// Create a limited reader for incremental test
	reader := fineweb.NewParquetReader(r.ParquetPath)

	start := time.Now()

	// Read limited docs
	docs := reader.ReadN(ctx, int(r.IncrementalDocs))
	err := indexer.Import(ctx, docs, nil)
	duration := time.Since(start)

	if err != nil {
		r.log("    Incremental error: %v", err)
		return nil
	}

	var endCount int64
	if stats, ok := indexer.(fineweb.Stats); ok {
		endCount, _ = stats.Count(ctx)
	}

	actualAdded := endCount - startCount
	metrics := &IncrementalMetrics{
		Duration:   duration,
		DocsPerSec: float64(actualAdded) / duration.Seconds(),
		DocsAdded:  actualAdded,
		StartCount: startCount,
		EndCount:   endCount,
	}

	r.log("    Incremental: added %d docs in %v (%.0f docs/sec)", actualAdded, duration.Round(time.Millisecond), metrics.DocsPerSec)

	return metrics
}

func (r *Runner) measureIndexSize(name string) int64 {
	// Common index directory patterns
	patterns := []string{
		filepath.Join(r.DataDir, "vie_Latn."+name),
		filepath.Join(r.DataDir, "vie_Latn."+name+"db"),
		filepath.Join(r.DataDir, "fineweb."+name),
	}

	for _, pattern := range patterns {
		size, err := MeasureIndexSize(pattern)
		if err == nil && size > 0 {
			return size
		}
	}

	return 0
}

func (r *Runner) benchmarkLatency(ctx context.Context, driver fineweb.Driver) *LatencyMetrics {
	collector := NewLatencyCollector()

	for i := 0; i < r.Iterations; i++ {
		for _, q := range r.Queries {
			select {
			case <-ctx.Done():
				return collector.Metrics()
			default:
			}

			start := time.Now()
			_, _ = driver.Search(ctx, q.Text, 20, 0)
			collector.Add(time.Since(start))
		}

		// Log progress every 25% for larger tests
		if r.Iterations > 10 && (i+1)%(r.Iterations/4) == 0 {
			r.log("      Progress: %d/%d iterations (%.0f%%)", i+1, r.Iterations, float64(i+1)*100/float64(r.Iterations))
		}
	}

	return collector.Metrics()
}

func (r *Runner) benchmarkThroughput(ctx context.Context, driver fineweb.Driver, goroutines int) *ThroughputMetrics {
	duration := r.ThroughputDuration
	if duration == 0 {
		duration = 10 * time.Second
	}
	deadline := time.Now().Add(duration)

	var ops int64
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			queryIdx := 0
			for time.Now().Before(deadline) {
				select {
				case <-ctx.Done():
					return
				default:
				}

				q := r.Queries[queryIdx%len(r.Queries)]
				_, _ = driver.Search(ctx, q.Text, 20, 0)

				mu.Lock()
				ops++
				mu.Unlock()

				queryIdx++
			}
		}()
	}

	wg.Wait()

	qps := float64(ops) / duration.Seconds()

	return &ThroughputMetrics{
		QPS:        qps,
		Duration:   duration,
		TotalOps:   ops,
		Goroutines: goroutines,
	}
}

func (r *Runner) benchmarkColdStart(ctx context.Context, name string, cfg fineweb.DriverConfig) time.Duration {
	r.log("  Measuring cold start...")

	start := time.Now()

	driver, err := fineweb.Open(name, cfg)
	if err != nil {
		r.log("    Error: %v", err)
		return 0
	}

	// First search
	_, _ = driver.Search(ctx, r.Queries[0].Text, 20, 0)
	duration := time.Since(start)

	driver.Close()

	r.log("    Cold start: %v", duration)
	return duration
}

func (r *Runner) benchmarkQuery(ctx context.Context, driver fineweb.Driver, q Query) *QueryMetrics {
	start := time.Now()
	result, _ := driver.Search(ctx, q.Text, 20, 0)
	duration := time.Since(start)

	results := 0
	if result != nil {
		results = len(result.Documents)
	}

	return &QueryMetrics{
		Query:    q,
		Duration: duration,
		Results:  results,
	}
}

func (r *Runner) log(format string, args ...interface{}) {
	if r.Logger != nil {
		r.Logger.Printf(format, args...)
	}
}

// logPhase logs a benchmark phase with consistent formatting
func (r *Runner) logPhase(phase, driver string, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	r.log("  [%s] %s: %s", phase, driver, msg)
}
