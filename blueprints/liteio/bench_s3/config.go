// Package bench_s3 provides S3 protocol benchmarks using the AWS SDK v2 directly.
package bench_s3

import "time"

// Object sizes for benchmarks.
const (
	SizeSmall  = 1024            // 1 KB
	SizeMedium = 64 * 1024       // 64 KB
	SizeLarge  = 1024 * 1024     // 1 MB
	SizeXLarge = 10 * 1024 * 1024 // 10 MB
)

// Config holds benchmark configuration.
type Config struct {
	BenchTime      time.Duration // Target duration per benchmark (default 1s)
	MinIterations  int           // Minimum iterations for statistics
	WarmupIters    int           // Warmup iterations
	Bucket         string        // S3 bucket name
	OutputDir      string        // Report output directory
	Drivers        []string      // Filter: only run these drivers (nil = all)
	Filter         string        // Substring filter for benchmark names
	Quick          bool          // Quick mode (500ms benchtime)
	Verbose        bool          // Verbose logging
	Progress       bool          // Live progress output
	OutputFormats  []string      // markdown, json, csv
	Timeout        time.Duration // Per-operation timeout
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		BenchTime:     1 * time.Second,
		MinIterations: 3,
		WarmupIters:   10,
		Bucket:        "test-bucket",
		OutputDir:     "./report/s3",
		OutputFormats: []string{"markdown", "json"},
		Timeout:       30 * time.Second,
	}
}

// QuickConfig returns config for quick runs.
func QuickConfig() *Config {
	cfg := DefaultConfig()
	cfg.BenchTime = 500 * time.Millisecond
	cfg.WarmupIters = 3
	cfg.Quick = true
	return cfg
}

// Endpoint describes an S3-compatible server.
type Endpoint struct {
	Name      string
	Host      string // host:port
	AccessKey string
	SecretKey string
	Container string // Docker container name (for stats)
}

// AllEndpoints returns all known S3 server configurations.
func AllEndpoints() []Endpoint {
	return []Endpoint{
		{
			Name:      "minio",
			Host:      "localhost:9000",
			AccessKey: "minioadmin",
			SecretKey: "minioadmin",
			Container: "all-minio-1",
		},
		{
			Name:      "rustfs",
			Host:      "localhost:9100",
			AccessKey: "rustfsadmin",
			SecretKey: "rustfsadmin",
			Container: "all-rustfs-1",
		},
		{
			Name:      "seaweedfs",
			Host:      "localhost:8333",
			AccessKey: "admin",
			SecretKey: "adminpassword",
			Container: "all-seaweedfs-volume-1",
		},
		{
			Name:      "liteio",
			Host:      "localhost:9200",
			AccessKey: "liteio",
			SecretKey: "liteio123",
			Container: "all-liteio-1",
		},
		{
			Name:      "herd_s3",
			Host:      "localhost:9230",
			AccessKey: "herd",
			SecretKey: "herd123",
			Container: "all-herd-1",
		},
	}
}

// FilterEndpoints filters endpoints by name.
func FilterEndpoints(endpoints []Endpoint, names []string) []Endpoint {
	if len(names) == 0 {
		return endpoints
	}
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	var filtered []Endpoint
	for _, e := range endpoints {
		if nameSet[e.Name] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// WarmupForSize returns adaptive warmup count based on object size.
func (c *Config) WarmupForSize(size int) int {
	switch {
	case size >= 10*1024*1024:
		return max(1, c.WarmupIters/5)
	case size >= 1*1024*1024:
		return max(2, c.WarmupIters/3)
	default:
		return c.WarmupIters
	}
}

// BenchTimeForSize returns adaptive bench time based on object size.
func (c *Config) BenchTimeForSize(size int) time.Duration {
	if size >= 10*1024*1024 && c.BenchTime > 5*time.Second {
		return 5 * time.Second
	}
	return c.BenchTime
}

// MaxIterationsForSize caps iterations for large objects.
func (c *Config) MaxIterationsForSize(size int) int {
	switch {
	case size >= 10*1024*1024:
		return 1_000
	case size >= 1*1024*1024:
		return 10_000
	case size >= 64*1024:
		return 50_000
	default:
		return 1_000_000
	}
}
