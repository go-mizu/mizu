package bench

import (
	"fmt"
	"time"
)

// Config holds benchmark configuration.
type Config struct {
	// Iterations is the number of iterations per benchmark.
	Iterations int
	// WarmupIterations is the number of warmup iterations.
	WarmupIterations int
	// Concurrency is the parallel operation concurrency (default level).
	Concurrency int
	// ConcurrencyLevels is the list of concurrency levels to test.
	// If empty, only Concurrency is used.
	ConcurrencyLevels []int
	// ObjectSizes is the list of object sizes to benchmark.
	ObjectSizes []int
	// OutputDir is the directory for reports.
	OutputDir string
	// Drivers is the list of drivers to benchmark (nil = all).
	Drivers []string
	// Timeout is the per-operation timeout.
	Timeout time.Duration
	// ParallelTimeout is the timeout for parallel operations (longer).
	ParallelTimeout time.Duration
	// Quick enables quick mode (fewer iterations).
	Quick bool
	// Large enables large file benchmarks.
	Large bool
	// DockerStats enables Docker container statistics.
	DockerStats bool
	// Verbose enables verbose output.
	Verbose bool
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Iterations:        100,
		WarmupIterations:  10,
		Concurrency:       10,
		ConcurrencyLevels: []int{1, 5, 10, 25, 50}, // Multiple concurrency levels to test
		ObjectSizes:       []int{1024, 64 * 1024, 1024 * 1024}, // 1KB, 64KB, 1MB
		OutputDir:         "./pkg/storage/report",
		Drivers:           nil, // nil means all
		Timeout:           30 * time.Second,
		ParallelTimeout:   60 * time.Second, // Longer timeout for parallel ops
		Quick:             false,
		Large:             false,
		DockerStats:       true,
		Verbose:           false,
	}
}

// QuickConfig returns config for quick benchmark runs.
func QuickConfig() *Config {
	cfg := DefaultConfig()
	cfg.Iterations = 20
	cfg.WarmupIterations = 5
	cfg.ConcurrencyLevels = []int{1, 5, 10} // Fewer levels for quick runs
	cfg.Quick = true
	return cfg
}

// DriverConfig holds connection info for a driver.
type DriverConfig struct {
	Name           string
	DSN            string
	Bucket         string
	Enabled        bool
	Skip           bool   // Skip this driver
	SkipMsg        string // Reason for skipping
	Container      string // Docker container name for stats
	MaxConcurrency int    // Max concurrency (0 = unlimited)
	Features       map[string]bool
}

// AllDriverConfigs returns configurations for all supported drivers.
func AllDriverConfigs() []DriverConfig {
	return []DriverConfig{
		{
			Name:      "minio",
			DSN:       "s3://minioadmin:minioadmin@localhost:9000/test-bucket?insecure=true&force_path_style=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-minio-1",
		},
		{
			Name:           "rustfs",
			DSN:            "s3://rustfsadmin:rustfsadmin@localhost:9100/test-bucket?insecure=true&force_path_style=true",
			Bucket:         "test-bucket",
			Enabled:        true,
			Container:      "all-rustfs-1",
			MaxConcurrency: 10, // RustFS has HTTP connection issues at C25
		},
		{
			Name:      "seaweedfs",
			DSN:       "s3://admin:adminpassword@localhost:8333/test-bucket?insecure=true&force_path_style=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-seaweedfs-s3-1",
		},
		{
			Name:      "localstack",
			DSN:       "s3://test:test@localhost:4566/test-bucket?insecure=true&force_path_style=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-localstack-1",
		},
		{
			Name:      "liteio",
			DSN:       "s3://liteio:liteio123@localhost:9200/test-bucket?insecure=true&force_path_style=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-liteio-1",
		},
		{
			Name:      "liteio_mem",
			DSN:       "s3://liteio:liteio123@localhost:9201/test-bucket?insecure=true&force_path_style=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-liteio_mem-1",
		},
	}
}

// FilterDrivers filters driver configs by name.
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

// BenchResult holds benchmark results for report generation.
type BenchResult struct {
	Driver     string         `json:"driver"`
	Benchmark  string         `json:"benchmark"`
	Iterations int            `json:"iterations"`
	NsPerOp    float64        `json:"ns_per_op"`
	MBPerSec   float64        `json:"mb_per_sec,omitempty"`
	BytesPerOp int64          `json:"bytes_per_op"`
	AllocsOp   int64          `json:"allocs_per_op"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// SizeLabel returns a human-readable size label.
func SizeLabel(size int) string {
	switch {
	case size >= 1024*1024*1024:
		gb := float64(size) / (1024 * 1024 * 1024)
		if gb == float64(int(gb)) {
			return fmt.Sprintf("%dGB", int(gb))
		}
		return fmt.Sprintf("%.1fGB", gb)
	case size >= 1024*1024:
		mb := float64(size) / (1024 * 1024)
		if mb == float64(int(mb)) {
			return fmt.Sprintf("%dMB", int(mb))
		}
		return fmt.Sprintf("%.1fMB", mb)
	case size >= 1024:
		kb := float64(size) / 1024
		if kb == float64(int(kb)) {
			return fmt.Sprintf("%dKB", int(kb))
		}
		return fmt.Sprintf("%.1fKB", kb)
	default:
		return fmt.Sprintf("%dB", size)
	}
}
