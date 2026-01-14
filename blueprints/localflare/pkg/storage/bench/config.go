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

	// Duration-based benchmark mode
	// Duration is the target duration for each benchmark (0 = iteration-based).
	Duration time.Duration
	// MinIterations is the minimum iterations even in duration mode.
	MinIterations int

	// OutputFormats specifies output formats (markdown, json, csv).
	OutputFormats []string

	// CompareBaseline is the path to baseline results for comparison.
	CompareBaseline string
	// SaveBaseline saves results as baseline for future comparisons.
	SaveBaseline string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Iterations:        100,
		WarmupIterations:  10,
		Concurrency:       200,
		ConcurrencyLevels: []int{1, 10, 25, 50, 100, 200}, // Multiple concurrency levels to test
		ObjectSizes:       []int{1024, 64 * 1024, 1024 * 1024, 10 * 1024 * 1024, 100 * 1024 * 1024}, // 1KB, 64KB, 1MB, 10MB, 100MB
		OutputDir:         "./pkg/storage/report",
		Drivers:           nil, // nil means all
		Timeout:           60 * time.Second,
		ParallelTimeout:   120 * time.Second, // Longer timeout for parallel ops
		Quick:             false,
		Large:             false,
		DockerStats:       true,
		Verbose:           false,
		Duration:          0,  // Iteration-based by default
		MinIterations:     10, // Minimum iterations in duration mode
		OutputFormats:     []string{"markdown", "json"}, // Default outputs
	}
}

// QuickConfig returns config for quick benchmark runs.
func QuickConfig() *Config {
	cfg := DefaultConfig()
	cfg.Iterations = 20
	cfg.WarmupIterations = 5
	cfg.ConcurrencyLevels = []int{1, 10, 50} // Fewer levels for quick runs
	cfg.ObjectSizes = []int{1024, 64 * 1024, 1024 * 1024, 10 * 1024 * 1024} // Up to 10MB for quick
	cfg.Quick = true
	return cfg
}

// IterationsForSize returns adaptive iterations based on object size.
// Larger files need fewer iterations to get meaningful results.
func (c *Config) IterationsForSize(size int) int {
	base := c.Iterations

	switch {
	case size >= 100*1024*1024: // 100MB+
		return max(5, base/20) // 5 iterations for 100MB
	case size >= 10*1024*1024: // 10MB+
		return max(10, base/10) // 10 iterations for 10MB
	case size >= 1*1024*1024: // 1MB+
		return max(20, base/5) // 20 iterations for 1MB
	case size >= 64*1024: // 64KB+
		return max(50, base/2) // 50 iterations for 64KB
	default:
		return base // Full iterations for small files
	}
}

// WarmupForSize returns adaptive warmup iterations based on object size.
func (c *Config) WarmupForSize(size int) int {
	base := c.WarmupIterations

	switch {
	case size >= 100*1024*1024: // 100MB+
		return max(1, base/5) // 1-2 warmup for 100MB
	case size >= 10*1024*1024: // 10MB+
		return max(2, base/4) // 2-3 warmup for 10MB
	case size >= 1*1024*1024: // 1MB+
		return max(3, base/3) // 3+ warmup for 1MB
	default:
		return base // Full warmup for small files
	}
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
