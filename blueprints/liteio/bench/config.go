package bench

import (
	"fmt"
	"os"
	"time"
)

// Benchmark configuration constants.
const (
	// Default benchmark parameters
	defaultWarmupIterations = 10
	defaultConcurrency      = 200
	defaultTimeout          = 60 * time.Second
	defaultParallelTimeout  = 120 * time.Second
	defaultReadBufferSize   = 256 * 1024

	// Go-style adaptive benchmark defaults (same as 'go test -bench')
	defaultBenchTime          = 1 * time.Second // Same as Go's default
	defaultMinBenchIterations = 3               // Minimum for statistics
	defaultMaxBenchIterations = 1_000_000_000   // 1e9 safety limit

	// Object sizes
	sizeTiny    = 256               // 256B
	sizeSmall   = 1024              // 1KB
	sizeMedium  = 64 * 1024         // 64KB
	sizeLarge   = 1024 * 1024       // 1MB
	sizeXLarge  = 10 * 1024 * 1024  // 10MB
	sizeXXLarge = 100 * 1024 * 1024 // 100MB
)

// Config holds benchmark configuration.
type Config struct {
	// WarmupIterations is the number of warmup iterations before timing begins.
	WarmupIterations int
	// Concurrency is the parallel operation concurrency (default level).
	Concurrency int
	// ConcurrencyLevels is the list of concurrency levels to test.
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
	// Quick enables quick mode (shorter benchmark time).
	Quick bool
	// Large enables large file benchmarks.
	Large bool
	// DockerStats enables Docker container statistics.
	DockerStats bool
	// Verbose enables verbose output.
	Verbose bool
	// LowOverhead enables client-side optimizations to minimize benchmark overhead.
	LowOverhead bool
	// Progress enables live progress output.
	Progress bool
	// ProgressEvery controls progress update frequency (iterations). 0 disables updates.
	ProgressEvery int
	// PerOpTimeouts enables per-operation timeouts (extra client overhead).
	PerOpTimeouts bool
	// ReadBufferSize is the buffer size for read copy operations.
	ReadBufferSize int
	// EnableTTFB captures time-to-first-byte for reads (extra client overhead).
	EnableTTFB bool

	// Go-style adaptive benchmark settings (same as 'go test -bench')
	// BenchTime is the target duration for each benchmark.
	// The benchmark auto-scales iterations to meet this target.
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

	// ScaleCounts is the list of object counts to benchmark in the Scale suite.
	ScaleCounts []int
	// ScaleObjectSize is the object size for the Scale suite.
	ScaleObjectSize int
	// ScaleMaxBytes is the maximum total bytes allowed per Scale test.
	ScaleMaxBytes int64

	// CleanupDataPaths removes local benchmark data paths after each driver run.
	CleanupDataPaths bool
	// CleanupDockerData clears docker volume data paths after each driver run.
	CleanupDockerData bool

	// Filter is a substring filter for benchmark names.
	// Only benchmarks containing this string will run. Empty means all.
	Filter string

	// ResourceTracking enables Go runtime memory and disk usage tracking.
	// Captures snapshots before/after each driver benchmark.
	ResourceTracking bool
}

// DefaultConfig returns sensible defaults.
// Uses Go-style adaptive benchmarking with 1s target duration (same as 'go test -bench').
func DefaultConfig() *Config {
	return &Config{
		WarmupIterations:  defaultWarmupIterations,
		Concurrency:       defaultConcurrency,
		ConcurrencyLevels: []int{1, 10, 25, 50, 100, 200},
		ObjectSizes:       []int{sizeSmall, sizeMedium, sizeLarge, sizeXLarge, sizeXXLarge},
		OutputDir:         "./pkg/storage/report",
		Drivers:           nil, // nil means all
		Timeout:           defaultTimeout,
		ParallelTimeout:   defaultParallelTimeout,
		Quick:             false,
		Large:             false,
		DockerStats:       true,
		Verbose:           false,
		LowOverhead:       true,
		Progress:          false,
		ProgressEvery:     256,
		PerOpTimeouts:     false,
		ReadBufferSize:    defaultReadBufferSize,
		EnableTTFB:        false,
		// Go-style adaptive benchmark settings
		BenchTime:          defaultBenchTime,
		MinBenchIterations: defaultMinBenchIterations,
		MaxBenchIterations: defaultMaxBenchIterations,
		OutputFormats:      []string{"markdown", "json"},
		ScaleCounts:        []int{10, 100, 1000, 10000},
		ScaleObjectSize:    sizeTiny,
		ScaleMaxBytes:      2 * 1024 * 1024 * 1024, // 2GB cap to prevent runaway disk usage
		CleanupDataPaths:   true,
		CleanupDockerData:  true,
		ResourceTracking:   true,
	}
}

// QuickConfig returns config for quick benchmark runs.
// Uses 500ms target duration instead of the default 1s.
func QuickConfig() *Config {
	cfg := DefaultConfig()
	cfg.WarmupIterations = 5
	cfg.ConcurrencyLevels = []int{1, 10, 50}                              // Fewer levels for quick runs
	cfg.ObjectSizes = []int{sizeSmall, sizeMedium, sizeLarge, sizeXLarge} // Up to 10MB for quick
	cfg.Quick = true
	cfg.BenchTime = 500 * time.Millisecond // Shorter target for quick runs
	cfg.ScaleCounts = []int{1, 10, 100, 1000}
	return cfg
}

// EnableLargeObjects enables 100MB benchmarks for large object testing.
func (c *Config) EnableLargeObjects() {
	c.Large = true
	for _, size := range c.ObjectSizes {
		if size == sizeXXLarge {
			return
		}
	}
	c.ObjectSizes = append(c.ObjectSizes, sizeXXLarge)
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

// MaxIterationsForSize caps the adaptive benchmark iteration count based on object size.
// Prevents the adaptive algorithm from overshooting on operations like Read/1MB where
// initial iterations are fast (page cache hot) but the algorithm commits to millions.
func (c *Config) MaxIterationsForSize(size int) int {
	switch {
	case size >= 100*1024*1024: // 100MB+: cap at 100 (10 GB max)
		return 100
	case size >= 10*1024*1024: // 10MB+: cap at 1K (10 GB max)
		return 1_000
	case size >= 1*1024*1024: // 1MB+: cap at 10K (10 GB max)
		return 10_000
	case size >= 64*1024: // 64KB+: cap at 50K (3.2 GB max)
		return 50_000
	default:
		return c.MaxBenchIterations
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
	DataPath       string // Data path inside container for volume size calculation
	MaxConcurrency int    // Max concurrency (0 = unlimited)
	Features       map[string]bool
}

// garageDriverConfig returns the Garage driver config.
// Garage uses dynamically generated API keys, so credentials come from env vars.
func garageDriverConfig() DriverConfig {
	accessKey := os.Getenv("GARAGE_BENCH_ACCESS_KEY")
	secretKey := os.Getenv("GARAGE_BENCH_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		return DriverConfig{
			Name:    "garage",
			Enabled: true,
			Skip:    true,
			SkipMsg: "Garage credentials not set (GARAGE_BENCH_ACCESS_KEY/GARAGE_BENCH_SECRET_KEY)",
		}
	}
	return DriverConfig{
		Name:    "garage",
		DSN:     fmt.Sprintf("s3://%s:%s@localhost:3900/test-bucket?insecure=true&force_path_style=true", accessKey, secretKey),
		Bucket:  "test-bucket",
		Enabled: true,
	}
}

// AllDriverConfigs returns configurations for all supported drivers.
func AllDriverConfigs() []DriverConfig {
	return []DriverConfig{
		{
			Name:      "minio",
			DSN:       "s3://minioadmin:minioadmin@localhost:9000/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-minio-1",
			DataPath:  "/data",
		},
		{
			Name:      "rustfs",
			DSN:       "s3://rustfsadmin:rustfsadmin@localhost:9100/test-bucket?insecure=true&force_path_style=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-rustfs-1",
			DataPath:  "/data",
		},
		{
			Name:      "seaweedfs",
			DSN:       "s3://admin:adminpassword@localhost:8333/test-bucket?insecure=true&force_path_style=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-seaweedfs-volume-1", // Use volume container for data size
			DataPath:  "/data",
		},
		garageDriverConfig(),
		{
			Name:      "localstack",
			DSN:       "s3://test:test@localhost:4566/test-bucket?insecure=true&force_path_style=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-localstack-1",
			DataPath:  "/var/lib/localstack",
		},
		{
			Name:      "liteio",
			DSN:       "s3://liteio:liteio123@localhost:9200/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-liteio-1",
			DataPath:  "/data",
		},
		{
			Name:    "local_s3",
			DSN:     "s3://local:local123@localhost:9213/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:    "horse_s3",
			DSN:     "s3://horse:horse123@localhost:9210/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:    "bee3net_s3",
			DSN:     "s3://bee3:bee3123@localhost:9211/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:    "bee5net_s3",
			DSN:     "s3://bee5:bee5123@localhost:9212/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:      "rabbit_s3",
			DSN:       "s3://rabbit:rabbit123@localhost:9300/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-rabbit_s3-1",
			DataPath:  "/data",
		},
		{
			Name:      "usagi_s3",
			DSN:       "s3://usagi:usagi123@localhost:9301/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-usagi_s3-1",
			DataPath:  "/data",
		},
		{
			Name:      "devnull_s3",
			DSN:       "s3://devnull:devnull123@localhost:9302/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-devnull_s3-1",
			DataPath:  "",
		},
		{
			Name:      "devnull",
			DSN:       "devnull://test-bucket",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "", // No container - pure in-process baseline
		},
		{
			Name:      "rabbit",
			DSN:       "rabbit:///tmp/rabbit-bench?nofsync=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "", // No container - pure in-process driver
			DataPath:  "/tmp/rabbit-bench",
		},
		{
			Name:      "usagi",
			DSN:       "usagi:///tmp/usagi-bench?nofsync=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "", // No container - pure in-process driver
			DataPath:  "/tmp/usagi-bench",
		},
		{
			Name:     "local",
			DSN:      "local:///tmp/local-bench",
			Bucket:   "test-bucket",
			Enabled:  true,
			DataPath: "/tmp/local-bench",
		},
		{
			Name:     "horse",
			DSN:      "horse:///tmp/horse-bench?sync=none&prealloc=2048",
			Bucket:   "test-bucket",
			Enabled:  true,
			DataPath: "/tmp/horse-bench",
		},
		{
			Name:     "pony",
			DSN:      "pony:///tmp/pony-bench?sync=none&prealloc=256&slots=65536",
			Bucket:   "test-bucket",
			Enabled:  true,
			DataPath: "/tmp/pony-bench",
		},
		{
			Name:     "bee3",
			DSN:      "bee:///tmp/bee3-bench?nodes=3&replicas=3&w=1&r=1&sync=none&inline_kb=64&repair=true&repair_workers=6&repair_max_kb=256",
			Bucket:   "test-bucket",
			Enabled:  true,
			DataPath: "/tmp/bee3-bench",
		},
		{
			Name:     "bee5",
			DSN:      "bee:///tmp/bee5-bench?nodes=5&replicas=3&w=1&r=1&sync=none&inline_kb=64&repair=true&repair_workers=10&repair_max_kb=256",
			Bucket:   "test-bucket",
			Enabled:  true,
			DataPath: "/tmp/bee5-bench",
		},
		{
			Name:    "bee3net",
			DSN:     "bee:///?peers=http://127.0.0.1:9401,http://127.0.0.1:9402,http://127.0.0.1:9403&replicas=3&w=1&r=1&repair=true&repair_workers=6&repair_max_kb=256",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:    "bee5net",
			DSN:     "bee:///?peers=http://127.0.0.1:9501,http://127.0.0.1:9502,http://127.0.0.1:9503,http://127.0.0.1:9504,http://127.0.0.1:9505&replicas=3&w=1&r=1&repair=true&repair_workers=10&repair_max_kb=256",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:     "herd",
			DSN:      "herd:///tmp/herd-bench?stripes=16&sync=none&inline_kb=8&prealloc=1024&bufsize=8388608",
			Bucket:   "test-bucket",
			Enabled:  true,
			DataPath: "/tmp/herd-bench",
		},
		{
			Name:     "herd3",
			DSN:      "herd:///tmp/herd3-bench?nodes=3&stripes=16&sync=none&inline_kb=8&prealloc=1024&bufsize=8388608",
			Bucket:   "test-bucket",
			Enabled:  true,
			DataPath: "/tmp/herd3-bench",
		},
		{
			Name:     "herd5",
			DSN:      "herd:///tmp/herd5-bench?nodes=5&stripes=16&sync=none&inline_kb=8&prealloc=1024&bufsize=8388608",
			Bucket:   "test-bucket",
			Enabled:  true,
			DataPath: "/tmp/herd5-bench",
		},
		{
			Name:      "herd_s3",
			DSN:       "s3://herd:herd123@localhost:9230/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:    "test-bucket",
			Enabled:   true,
			Container: "all-herd-1",
			DataPath:  "/data",
		},
		{
			Name:    "herd3net",
			DSN:     "herd:///?peers=127.0.0.1:9241,127.0.0.1:9242,127.0.0.1:9243&replicas=1",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:    "herd5net",
			DSN:     "herd:///?peers=127.0.0.1:9241,127.0.0.1:9242,127.0.0.1:9243,127.0.0.1:9244,127.0.0.1:9245&replicas=1",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:    "herd3net_s3",
			DSN:     "s3://herd3:herd3123@localhost:9231/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:     "zebra",
			DSN:      "zebra:///tmp/zebra-bench?stripes=8&sync=none&inline_kb=4&prealloc=1024",
			Bucket:   "test-bucket",
			Enabled:  true,
			DataPath: "/tmp/zebra-bench",
		},
		{
			Name:    "zebra_s3",
			DSN:     "s3://zebra:zebra123@localhost:9220/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:    "zebra3net",
			DSN:     "zebra:///?peers=127.0.0.1:9601,127.0.0.1:9602,127.0.0.1:9603&replicas=1&w=1&r=1",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:    "zebra5net",
			DSN:     "zebra:///?peers=127.0.0.1:9601,127.0.0.1:9602,127.0.0.1:9603,127.0.0.1:9604,127.0.0.1:9605&replicas=1&w=1&r=1",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:    "zebra3net_s3",
			DSN:     "s3://zebra3:zebra3123@localhost:9221/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:  "test-bucket",
			Enabled: true,
		},
		{
			Name:    "zebra5net_s3",
			DSN:     "s3://zebra5:zebra5123@localhost:9222/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
			Bucket:  "test-bucket",
			Enabled: true,
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
