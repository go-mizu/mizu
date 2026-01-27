package benchmark

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

// Runner orchestrates benchmarks across drivers.
type Runner struct {
	DataDir     string
	ParquetPath string
	Drivers     []string
	Queries     []Query
	Concurrency []int
	Iterations  int
	Logger      *log.Logger
}

// NewRunner creates a benchmark runner with defaults.
func NewRunner(dataDir, parquetPath string) *Runner {
	return &Runner{
		DataDir:     dataDir,
		ParquetPath: parquetPath,
		Drivers:     fineweb.List(),
		Queries:     VietnameseQueries,
		Concurrency: []int{1, 10, 50, 100},
		Iterations:  100,
		Logger:      log.Default(),
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
	IndexSize   int64                         `json:"index_size"`
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

	// Open driver
	driver, err := fineweb.Open(name, cfg)
	if err != nil {
		return result, fmt.Errorf("opening driver: %w", err)
	}
	defer driver.Close()

	// Check if indexing needed
	var docCount int64
	if stats, ok := fineweb.AsStats(driver); ok {
		docCount, _ = stats.Count(ctx)
	}

	// Indexing benchmark (if needed and supported)
	if docCount == 0 && r.ParquetPath != "" {
		if indexer, ok := fineweb.AsIndexer(driver); ok {
			result.Indexing, result.Memory = r.benchmarkIndexing(ctx, indexer, name)
		}
	}

	// Measure index size
	result.IndexSize = r.measureIndexSize(name)

	// Search latency benchmark
	result.Latency = r.benchmarkLatency(ctx, driver)

	// Throughput benchmark
	result.Throughput = r.benchmarkThroughput(ctx, driver, 1)

	// Concurrent search benchmark
	for _, n := range r.Concurrency {
		result.Concurrency[n] = r.benchmarkThroughput(ctx, driver, n)
	}

	// Cold start benchmark
	result.ColdStart = r.benchmarkColdStart(ctx, name, cfg)

	// Per-query stats
	for _, q := range r.Queries {
		qm := r.benchmarkQuery(ctx, driver, q)
		result.QueryStats[q.Text] = qm
	}

	return result, nil
}

func (r *Runner) benchmarkIndexing(ctx context.Context, indexer fineweb.Indexer, name string) (*IndexingMetrics, *MemoryMetrics) {
	r.log("  Indexing...")

	reader := fineweb.NewParquetReader(r.ParquetPath)
	total, err := reader.CountDocuments(ctx)
	if err != nil {
		r.log("    Error counting documents: %v", err)
		return nil, nil
	}

	memTracker := NewMemoryTracker()
	start := time.Now()

	var imported int64
	progress := func(n, _ int64) {
		imported = n
		memTracker.Sample()
		if n%100000 == 0 {
			r.log("    Indexed %d documents...", n)
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
		Duration:   duration,
		DocsPerSec: float64(imported) / duration.Seconds(),
		PeakMemory: memTracker.Peak(),
		TotalDocs:  total,
	}

	memory := &MemoryMetrics{
		IndexingPeak: memTracker.Peak(),
	}

	r.log("    Indexed %d docs in %v (%.0f docs/sec)", imported, duration, indexing.DocsPerSec)

	return indexing, memory
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
	r.log("  Measuring latency...")

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
	}

	metrics := collector.Metrics()
	r.log("    p50=%v p95=%v p99=%v", metrics.P50, metrics.P95, metrics.P99)

	return metrics
}

func (r *Runner) benchmarkThroughput(ctx context.Context, driver fineweb.Driver, goroutines int) *ThroughputMetrics {
	r.log("  Measuring throughput (goroutines=%d)...", goroutines)

	duration := 10 * time.Second
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
	r.log("    %d ops in %v = %.1f QPS", ops, duration, qps)

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
