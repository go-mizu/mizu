package bench

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"

	// Import all drivers
	_ "github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver/chroma"
	_ "github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver/duckdb"
	_ "github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver/elasticsearch"
	_ "github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver/lancedb"
	_ "github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver/milvus"
	_ "github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver/opensearch"
	_ "github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver/pgvector"
	_ "github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver/qdrant"
	_ "github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver/redis"
	_ "github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver/weaviate"
)

// Runner executes benchmarks.
type Runner struct {
	config          *Config
	report          *Report
	dataset         *Dataset
	logger          func(format string, args ...any)
	dockerCollector *DockerStatsCollector
}

// NewRunner creates a benchmark runner.
func NewRunner(cfg *Config) *Runner {
	return &Runner{
		config: cfg,
		report: NewReport(cfg),
		logger: func(format string, args ...any) {
			fmt.Printf(format+"\n", args...)
		},
		dockerCollector: NewDockerStatsCollector("all-"),
	}
}

// SetDockerPrefix sets the docker-compose project prefix for container names.
func (r *Runner) SetDockerPrefix(prefix string) {
	r.dockerCollector = NewDockerStatsCollector(prefix)
}

// SetLogger sets a custom logger.
func (r *Runner) SetLogger(logger func(format string, args ...any)) {
	r.logger = logger
}

// Run executes all benchmarks.
func (r *Runner) Run(ctx context.Context) (*Report, error) {
	r.logger("=== Vectorize Benchmark Suite ===")
	r.logger("Dimensions: %d, Dataset: %d vectors, Batch: %d, Search iterations: %d",
		r.config.Dimensions, r.config.DatasetSize, r.config.BatchSize, r.config.SearchIterations)
	r.logger("")

	// Generate dataset
	r.logger("Generating dataset...")
	r.dataset = GenerateDataset(
		r.config.DatasetSize,
		r.config.Dimensions,
		r.config.SearchIterations,
		42, // Fixed seed for reproducibility
	)
	r.logger("Generated %d vectors and %d query vectors", len(r.dataset.Vectors), len(r.dataset.QueryVectors))
	r.logger("")

	// Get driver configs
	driverConfigs := FilterDrivers(AllDriverConfigs(), r.config.Drivers)
	r.logger("Benchmarking %d drivers...", len(driverConfigs))
	r.logger("")

	// Run benchmarks for each driver
	for _, dcfg := range driverConfigs {
		if err := r.benchmarkDriver(ctx, dcfg); err != nil {
			r.logger("Driver %s failed: %v", dcfg.Name, err)
			// Continue with other drivers
		}
	}

	// Compute final stats
	r.report.ComputeStats()

	// Collect Docker container stats
	r.logger("Collecting Docker container stats...")
	r.collectDockerStats(ctx, driverConfigs)

	return r.report, nil
}

// collectDockerStats collects memory and disk usage from Docker containers.
func (r *Runner) collectDockerStats(ctx context.Context, driverConfigs []DriverConfig) {
	for _, dcfg := range driverConfigs {
		stats, ok := r.report.DriverStats[dcfg.Name]
		if !ok {
			continue
		}

		// Check if embedded driver
		if dcfg.Name == "lancedb" || dcfg.Name == "duckdb" {
			stats.IsEmbedded = true
			r.logger("  %s: embedded driver (no container)", dcfg.Name)
			continue
		}

		// Collect Docker stats
		dockerStats := r.dockerCollector.CollectStats(ctx, dcfg.Name)
		if dockerStats.Available {
			stats.MemoryUsageMB = dockerStats.MemoryUsageMB
			stats.MemoryLimitMB = dockerStats.MemoryLimitMB
			stats.MemoryPercent = dockerStats.MemoryPercent
			stats.CPUPercent = dockerStats.CPUPercent
			stats.DiskUsageMB = dockerStats.DiskUsageMB
			r.logger("  %s: Memory %.1f MB (%.1f%%), CPU %.2f%%, Disk %.1f MB",
				dcfg.Name, dockerStats.MemoryUsageMB, dockerStats.MemoryPercent,
				dockerStats.CPUPercent, dockerStats.DiskUsageMB)
		} else {
			r.logger("  %s: Docker stats unavailable: %s", dcfg.Name, dockerStats.Error)
		}
	}
	r.logger("")
}

func (r *Runner) benchmarkDriver(ctx context.Context, dcfg DriverConfig) error {
	r.logger("--- %s ---", dcfg.Name)

	// Initialize driver stats
	stats := &DriverStats{
		Driver:    dcfg.Name,
		Available: false,
	}
	r.report.DriverStats[dcfg.Name] = stats

	// Connect
	r.logger("  Connecting to %s...", dcfg.DSN)
	connectCollector := NewCollector()
	timer := NewTimer()

	db, err := driver.Open(dcfg.Name, dcfg.DSN)
	connectTime := timer.Stop()
	connectCollector.Record(connectTime)

	if err != nil {
		connectCollector.RecordError(err)
		r.report.AddResult(connectCollector.Result(dcfg.Name, "connect", "", 1))
		r.logger("  Connection failed: %v", err)
		return err
	}
	defer db.Close()

	// Ping
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	if err := db.Ping(pingCtx); err != nil {
		cancel()
		connectCollector.RecordError(err)
		r.report.AddResult(connectCollector.Result(dcfg.Name, "connect", "", 1))
		r.logger("  Ping failed: %v", err)
		return err
	}
	cancel()

	stats.Available = true
	stats.ConnectionTime = connectTime.Milliseconds()
	r.report.AddResult(connectCollector.Result(dcfg.Name, "connect", "", 1))
	r.logger("  Connected in %v", connectTime)

	// Create unique index name
	indexName := fmt.Sprintf("bench_%s_%d", dcfg.Name, time.Now().UnixNano()%10000)

	// Benchmark operations
	if err := r.benchmarkIndex(ctx, db, dcfg.Name, indexName); err != nil {
		r.logger("  Index operations failed: %v", err)
	}

	if err := r.benchmarkInsert(ctx, db, dcfg.Name, indexName); err != nil {
		r.logger("  Insert operations failed: %v", err)
	}

	if err := r.benchmarkSearch(ctx, db, dcfg.Name, indexName); err != nil {
		r.logger("  Search operations failed: %v", err)
	}

	if err := r.benchmarkGet(ctx, db, dcfg.Name, indexName); err != nil {
		r.logger("  Get operations failed: %v", err)
	}

	// Cleanup
	cleanupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	db.DeleteIndex(cleanupCtx, indexName)
	cancel()

	r.logger("")
	return nil
}

func (r *Runner) benchmarkIndex(ctx context.Context, db vectorize.DB, driverName, indexName string) error {
	collector := NewCollector()

	// CreateIndex
	index := &vectorize.Index{
		Name:        indexName,
		Dimensions:  r.config.Dimensions,
		Metric:      vectorize.Cosine,
		Description: "Benchmark index",
	}

	timer := NewTimer()
	opCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	err := db.CreateIndex(opCtx, index)
	cancel()
	collector.Record(timer.Stop())

	if err != nil {
		collector.RecordError(err)
		r.report.AddResult(collector.Result(driverName, "create_index", "", 1))
		return err
	}

	r.report.AddResult(collector.Result(driverName, "create_index", "", 1))
	r.logger("  CreateIndex: %v", collector.Result(driverName, "create_index", "", 1).AvgLatency)

	// GetIndex
	getCollector := NewCollector()
	for i := 0; i < 10; i++ {
		timer := NewTimer()
		opCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
		_, err := db.GetIndex(opCtx, indexName)
		cancel()
		getCollector.Record(timer.Stop())
		if err != nil {
			getCollector.RecordError(err)
		}
	}
	r.report.AddResult(getCollector.Result(driverName, "get_index", "", 10))
	r.logger("  GetIndex: avg %v", getCollector.Result(driverName, "get_index", "", 10).AvgLatency)

	// ListIndexes
	listCollector := NewCollector()
	for i := 0; i < 10; i++ {
		timer := NewTimer()
		opCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
		_, err := db.ListIndexes(opCtx)
		cancel()
		listCollector.Record(timer.Stop())
		if err != nil {
			listCollector.RecordError(err)
		}
	}
	r.report.AddResult(listCollector.Result(driverName, "list_indexes", "", 10))
	r.logger("  ListIndexes: avg %v", listCollector.Result(driverName, "list_indexes", "", 10).AvgLatency)

	return nil
}

func (r *Runner) benchmarkInsert(ctx context.Context, db vectorize.DB, driverName, indexName string) error {
	collector := NewCollector()
	batches := r.dataset.Batches(r.config.BatchSize)
	totalVectors := 0

	r.logger("  Inserting %d vectors in %d batches...", len(r.dataset.Vectors), len(batches))

	for _, batch := range batches {
		timer := NewTimer()
		opCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
		err := db.Insert(opCtx, indexName, batch)
		cancel()
		collector.Record(timer.Stop())

		if err != nil {
			collector.RecordError(err)
			// Continue with remaining batches
		} else {
			totalVectors += len(batch)
		}
	}

	result := collector.Result(driverName, "insert", fmt.Sprintf("batch_%d", r.config.BatchSize), totalVectors)
	r.report.AddResult(result)
	r.logger("  Insert: %d vectors, throughput %.0f vec/s, avg batch %v",
		totalVectors, result.Throughput, result.AvgLatency)

	return nil
}

func (r *Runner) benchmarkSearch(ctx context.Context, db vectorize.DB, driverName, indexName string) error {
	opts := &vectorize.SearchOptions{
		TopK:           r.config.TopK,
		ReturnMetadata: true,
	}

	// Warmup
	r.logger("  Search warmup (%d queries)...", r.config.WarmupIterations)
	for i := 0; i < r.config.WarmupIterations && i < len(r.dataset.QueryVectors); i++ {
		opCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
		db.Search(opCtx, indexName, r.dataset.QueryVectors[i], opts)
		cancel()
	}

	// Timed search
	collector := NewCollector()
	r.logger("  Running %d search queries...", r.config.SearchIterations)

	for i := 0; i < r.config.SearchIterations && i < len(r.dataset.QueryVectors); i++ {
		timer := NewTimer()
		opCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
		_, err := db.Search(opCtx, indexName, r.dataset.QueryVectors[i], opts)
		cancel()
		collector.Record(timer.Stop())

		if err != nil {
			collector.RecordError(err)
		}
	}

	result := collector.Result(driverName, "search", fmt.Sprintf("top_%d", r.config.TopK), r.config.SearchIterations)
	r.report.AddResult(result)

	var qps float64
	if result.TotalTime > 0 {
		qps = float64(result.Iterations) / result.TotalTime.Seconds()
	}

	r.logger("  Search: p50=%v, p95=%v, p99=%v, QPS=%.1f, errors=%d",
		result.P50Latency, result.P95Latency, result.P99Latency, qps, result.Errors)

	return nil
}

func (r *Runner) benchmarkGet(ctx context.Context, db vectorize.DB, driverName, indexName string) error {
	// Sample some IDs
	sampleIDs := r.dataset.SampleIDs(100, 123)

	collector := NewCollector()

	// Get single
	for i := 0; i < 100 && i < len(sampleIDs); i++ {
		timer := NewTimer()
		opCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
		_, err := db.Get(opCtx, indexName, []string{sampleIDs[i]})
		cancel()
		collector.Record(timer.Stop())

		if err != nil {
			collector.RecordError(err)
		}
	}

	result := collector.Result(driverName, "get_single", "", 100)
	r.report.AddResult(result)
	r.logger("  Get (single): avg %v, errors=%d", result.AvgLatency, result.Errors)

	// Get batch
	batchCollector := NewCollector()
	for i := 0; i < 10; i++ {
		batchIDs := sampleIDs[:10] // Get 10 at a time
		timer := NewTimer()
		opCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
		_, err := db.Get(opCtx, indexName, batchIDs)
		cancel()
		batchCollector.Record(timer.Stop())

		if err != nil {
			batchCollector.RecordError(err)
		}
	}

	batchResult := batchCollector.Result(driverName, "get_batch", "batch_10", 100)
	r.report.AddResult(batchResult)
	r.logger("  Get (batch of 10): avg %v, errors=%d", batchResult.AvgLatency, batchResult.Errors)

	return nil
}
