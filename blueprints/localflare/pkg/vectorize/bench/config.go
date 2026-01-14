// Package bench provides benchmarking utilities for vectorize drivers.
package bench

import (
	"time"
)

// Config holds benchmark configuration.
type Config struct {
	// Dimensions is the vector dimension size.
	Dimensions int
	// DatasetSize is the number of vectors to generate.
	DatasetSize int
	// BatchSize is the batch size for insert operations.
	BatchSize int
	// SearchIterations is the number of search queries to run.
	SearchIterations int
	// WarmupIterations is the number of warmup iterations.
	WarmupIterations int
	// TopK is the number of results to return from search.
	TopK int
	// OutputDir is the directory for reports.
	OutputDir string
	// Drivers is the list of drivers to benchmark.
	Drivers []string
	// Timeout is the timeout for operations.
	Timeout time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Dimensions:       384,
		DatasetSize:      10000,
		BatchSize:        100,
		SearchIterations: 1000,
		WarmupIterations: 100,
		TopK:             10,
		OutputDir:        "./pkg/vectorize/report",
		Drivers:          nil, // nil means all
		Timeout:          30 * time.Second,
	}
}

// DriverConfig holds connection info for a driver.
type DriverConfig struct {
	Name    string
	DSN     string
	Enabled bool
}

// AllDriverConfigs returns configurations for all supported drivers.
func AllDriverConfigs() []DriverConfig {
	return []DriverConfig{
		// Server-based drivers
		{Name: "qdrant", DSN: "localhost:6334", Enabled: true},
		{Name: "milvus", DSN: "localhost:19530", Enabled: true},
		{Name: "weaviate", DSN: "http://localhost:8080", Enabled: true},
		{Name: "chroma", DSN: "http://localhost:8000", Enabled: true},
		{Name: "pgvector", DSN: "postgres://postgres:password@localhost:5432/vectors?sslmode=disable", Enabled: true},
		{Name: "pgvectorscale", DSN: "postgres://postgres:password@localhost:5433/vectors?sslmode=disable", Enabled: true},
		{Name: "redis", DSN: "redis://localhost:6379", Enabled: true},
		{Name: "opensearch", DSN: "http://localhost:9200", Enabled: true},
		{Name: "elasticsearch", DSN: "http://localhost:9201", Enabled: true},
		{Name: "vald", DSN: "localhost:8081", Enabled: false}, // Disabled: NGT library crashes on arm64
		{Name: "vespa", DSN: "http://localhost:8082", Enabled: false}, // Disabled: Vespa config server crashes
		// Embedded drivers
		{Name: "mizu_vector", DSN: "engine=flat", Enabled: true},
		{Name: "mizu_vector_ivf", DSN: "engine=ivf", Enabled: true},
		{Name: "mizu_vector_lsh", DSN: "engine=lsh", Enabled: true},
		{Name: "mizu_vector_pq", DSN: "engine=pq", Enabled: true},
		{Name: "mizu_vector_hnsw", DSN: "engine=hnsw", Enabled: true},
		{Name: "mizu_vector_vamana", DSN: "engine=vamana", Enabled: true},
		{Name: "mizu_vector_rabitq", DSN: "engine=rabitq", Enabled: true},
		{Name: "mizu_vector_nsg", DSN: "engine=nsg", Enabled: true},
		{Name: "mizu_vector_scann", DSN: "engine=scann", Enabled: true},
		{Name: "mizu_vector_acorn", DSN: "engine=acorn", Enabled: true},
		{Name: "hnsw", DSN: ":memory:", Enabled: true},
		{Name: "chromem", DSN: ":memory:", Enabled: true},
		{Name: "sqlite", DSN: "./data/bench_sqlite/vectors.db", Enabled: true},
		{Name: "lancedb", DSN: "./data/bench_lancedb", Enabled: true},
		{Name: "duckdb", DSN: "./data/bench_duckdb/vectors.db", Enabled: true},
	}
}

// FilterDrivers filters driver configs by name and enabled status.
func FilterDrivers(configs []DriverConfig, names []string) []DriverConfig {
	// First filter out disabled drivers
	var enabled []DriverConfig
	for _, c := range configs {
		if c.Enabled {
			enabled = append(enabled, c)
		}
	}

	// If no names specified, return all enabled drivers
	if len(names) == 0 {
		return enabled
	}

	// Filter by name
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	var filtered []DriverConfig
	for _, c := range enabled {
		if nameSet[c.Name] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
