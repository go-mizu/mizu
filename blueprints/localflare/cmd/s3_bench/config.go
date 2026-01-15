// Package main provides configuration for S3 benchmarks.
package main

import (
	"os"
	"path/filepath"
	"time"
)

// Config holds the benchmark configuration.
type Config struct {
	// Thread configuration
	ThreadsMin int // Minimum concurrent threads
	ThreadsMax int // Maximum concurrent threads

	// Payload configuration (powers of 2 in KB)
	PayloadsMin int // Min payload size power (12 = 4MB)
	PayloadsMax int // Max payload size power (14 = 16MB)

	// Test parameters
	Samples  int           // Number of samples per configuration
	Duration time.Duration // Duration-based testing (alternative to samples)

	// Driver selection
	Drivers []string // Specific drivers to test (empty = all)

	// Output
	OutputDir string // Output directory
	Verbose   bool   // Verbose output

	// Modes
	Quick       bool // Quick mode (fewer samples)
	Full        bool // Full comprehensive test
	CleanupOnly bool // Only run cleanup

	// Docker
	ComposeDir string // Docker compose directory
}

// DefaultConfig returns the default benchmark configuration.
func DefaultConfig() *Config {
	return &Config{
		ThreadsMin:  8,
		ThreadsMax:  12,
		PayloadsMin: 12, // 4MB (2^12 KB = 4096 KB)
		PayloadsMax: 14, // 16MB (2^14 KB = 16384 KB)
		Samples:     100,
		OutputDir:   DefaultOutputDir(),
		ComposeDir:  "./docker/s3/all",
	}
}

// DefaultOutputDir returns the default output directory.
func DefaultOutputDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./pkg/storage/report/s3_bench"
	}
	return filepath.Join(home, "github", "go-mizu", "mizu", "blueprints", "localflare", "pkg", "storage", "report", "s3_bench")
}

// QuickConfig returns a quick test configuration.
func QuickConfig() *Config {
	return &Config{
		ThreadsMin:  8,
		ThreadsMax:  10,
		PayloadsMin: 12, // 4MB
		PayloadsMax: 13, // 8MB
		Samples:     20,
		Quick:       true,
		OutputDir:   DefaultOutputDir(),
		ComposeDir:  "./docker/s3/all",
	}
}

// FullConfig returns a comprehensive test configuration.
func FullConfig() *Config {
	return &Config{
		ThreadsMin:  4,
		ThreadsMax:  16,
		PayloadsMin: 10, // 1MB
		PayloadsMax: 15, // 32MB
		Samples:     200,
		Full:        true,
		OutputDir:   DefaultOutputDir(),
		ComposeDir:  "./docker/s3/all",
	}
}

// PayloadSizes returns the list of payload sizes in bytes.
func (c *Config) PayloadSizes() []int {
	var sizes []int
	for p := c.PayloadsMin; p <= c.PayloadsMax; p++ {
		// 2^p KB = 2^(p+10) bytes
		size := 1 << (p + 10) // KB to bytes
		sizes = append(sizes, size)
	}
	return sizes
}

// ThreadCounts returns the list of thread counts to test.
func (c *Config) ThreadCounts() []int {
	var counts []int
	for t := c.ThreadsMin; t <= c.ThreadsMax; t++ {
		counts = append(counts, t)
	}
	return counts
}
