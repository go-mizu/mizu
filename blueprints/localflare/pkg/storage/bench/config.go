package bench

import (
	"fmt"
	"time"
)

// Benchmark configuration constants.
// These values have been optimized for comprehensive performance testing.
const (
	// Default benchmark parameters
	defaultIterations       = 100
	defaultWarmupIterations = 10
	defaultConcurrency      = 200
	defaultTimeout          = 60 * time.Second
	defaultParallelTimeout  = 120 * time.Second
	defaultMinIterations    = 10

	// Adaptive benchmark defaults (Go-style)
	defaultBenchTime          = 1 * time.Second // Same as Go's default
	defaultMinBenchIterations = 3               // Minimum for statistics
	defaultMaxBenchIterations = 1_000_000_000   // 1e9 safety limit

	// Object sizes
	sizeSmall   = 1024              // 1KB
	sizeMedium  = 64 * 1024         // 64KB
	sizeLarge   = 1024 * 1024       // 1MB
	sizeXLarge  = 10 * 1024 * 1024  // 10MB
	sizeXXLarge = 100 * 1024 * 1024 // 100MB

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

	// Duration-based benchmark mode (legacy)
	// Duration is the target duration for each benchmark (0 = iteration-based).
	// DEPRECATED: Use BenchTime instead for Go-style adaptive benchmarking.
	Duration time.Duration
	// MinIterations is the minimum iterations even in duration mode.
	// DEPRECATED: Use MinBenchIterations instead.
	MinIterations int

	// Adaptive benchmark settings (Go-style)
	// BenchTime is the target duration for each benchmark operation.
	// The benchmark will auto-scale iterations to meet this duration.
	// Default: 1s (same as Go's testing.B)
	BenchTime time.Duration
	// MinBenchIterations is the minimum iterations for statistical significance.
	MinBenchIterations int
	// MaxBenchIterations is the safety limit for iterations (default: 1e9).
	MaxBenchIterations int

	// OutputFormats specifies output formats (markdown, json, csv).
	OutputFormats []string

	// CompareBaseline is the path to baseline results for comparison.
	CompareBaseline string
	// SaveBaseline saves results as baseline for future comparisons.
	SaveBaseline string

	// FileCounts is the list of file counts to benchmark (e.g., 1, 10, 100, 1000, 10000, 100000).
	FileCounts []int

	// Filter is a substring filter for benchmark names (e.g., "MixedWorkload").
	// Only benchmarks containing this string will run. Empty means all.
	Filter string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Iterations:        defaultIterations,
		WarmupIterations:  defaultWarmupIterations,
		Concurrency:       defaultConcurrency,
		ConcurrencyLevels: []int{1, 10, 25, 50, 100, 200}, // Multiple concurrency levels to test
		ObjectSizes:       []int{sizeSmall, sizeMedium, sizeLarge, sizeXLarge, sizeXXLarge},
		OutputDir:         "./pkg/storage/report",
		Drivers:           nil, // nil means all
		Timeout:           defaultTimeout,
		ParallelTimeout:   defaultParallelTimeout,
		Quick:             false,
		Large:             false,
		DockerStats:       true,
		Verbose:           false,
		Duration:          0, // Legacy, not used with adaptive
		MinIterations:     defaultMinIterations,
		// Adaptive benchmark settings (Go-style)
		BenchTime:          defaultBenchTime,
		MinBenchIterations: defaultMinBenchIterations,
		MaxBenchIterations: defaultMaxBenchIterations,
		OutputFormats:      []string{"markdown", "json"}, // Default outputs
		FileCounts:         []int{1, 10, 100, 1000, 10000}, // File count benchmarks
	}
}

// QuickConfig returns config for quick benchmark runs.
func QuickConfig() *Config {
	cfg := DefaultConfig()
	cfg.Iterations = 20
	cfg.WarmupIterations = 5
	cfg.ConcurrencyLevels = []int{1, 10, 50} // Fewer levels for quick runs
	cfg.ObjectSizes = []int{sizeSmall, sizeMedium, sizeLarge, sizeXLarge} // Up to 10MB for quick
	cfg.Quick = true
	cfg.BenchTime = 500 * time.Millisecond // Shorter target for quick runs
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

// BenchTimeForSize returns adaptive benchmark duration based on object size.
// Larger files need shorter bench time to avoid excessive benchmark duration.
func (c *Config) BenchTimeForSize(size int) time.Duration {
	base := c.BenchTime

	switch {
	case size >= 100*1024*1024: // 100MB+
		// 100MB+ files: cap at 5s since each op is slow
		if base > 5*time.Second {
			return 5 * time.Second
		}
		return base
	case size >= 10*1024*1024: // 10MB+
		// 10MB files: cap at 10s
		if base > 10*time.Second {
			return 10 * time.Second
		}
		return base
	default:
		return base // Full bench time for smaller files
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
			Name:      "rustfs",
			DSN:       "s3://rustfsadmin:rustfsadmin@localhost:9100/test-bucket?insecure=true&force_path_style=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-rustfs-1",
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
		{
			Name:      "devnull",
			DSN:       "devnull://test-bucket",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "", // No container - pure in-process baseline
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
