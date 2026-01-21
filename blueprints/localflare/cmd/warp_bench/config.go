// Package main provides configuration for warp benchmarks.
package main

import "time"

// DriverConfig holds the configuration for a single S3 driver.
type DriverConfig struct {
	Name      string
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Container string // Docker container name for health checks
	Enabled   bool
}

// Config holds the benchmark configuration.
type Config struct {
	Duration    time.Duration
	Concurrent  int
	Objects     int
	ObjectSizes []string
	Operations  []string
	Drivers     []string // Filter to specific drivers (empty = all)
	OutputDir   string
	Verbose     bool
	Quick       bool
	DockerClean bool   // Enable Docker cleanup before/after each driver
	ComposeDir  string // Path to docker-compose directory
	WorkDir     string // Working directory for warp temp files (empty = auto)
	KeepWorkDir bool   // Keep work dir after run (for debugging)
	WarpPath    string // Resolved warp binary path
	WarpVersion string // Resolved warp version
	RunDir      string // Actual run directory used (auto)
}

// DefaultConfig returns the default benchmark configuration.
func DefaultConfig() *Config {
	return &Config{
		Duration:    5 * time.Second,
		Concurrent:  10,
		Objects:     20,
		ObjectSizes: []string{"1KiB", "1MiB"},
		Operations:  []string{"put", "get", "stat"},
		OutputDir:   "./pkg/storage/report/warp_bench",
		Verbose:     false,
		Quick:       true,
		DockerClean: false,
		ComposeDir:  "./docker/s3/all",
	}
}

// QuickConfig returns a faster configuration for quick testing.
func QuickConfig() *Config {
	return &Config{
		Duration:    10 * time.Second,
		Concurrent:  10,
		Objects:     50,
		ObjectSizes: []string{"1KiB", "1MiB"},
		Operations:  []string{"put", "get", "mixed"},
		OutputDir:   "./pkg/storage/report/warp_bench",
		Verbose:     false,
		Quick:       true,
		DockerClean: true,
		ComposeDir:  "./docker/s3/all",
	}
}

// DefaultDrivers returns all configured S3 drivers.
func DefaultDrivers() []*DriverConfig {
	return []*DriverConfig{
		{
			Name:      "minio",
			Endpoint:  "localhost:9000",
			AccessKey: "minioadmin",
			SecretKey: "minioadmin",
			Bucket:    "test-bucket",
			Container: "all-minio-1",
			Enabled:   true,
		},
		{
			Name:      "rustfs",
			Endpoint:  "localhost:9100",
			AccessKey: "rustfsadmin",
			SecretKey: "rustfsadmin",
			Bucket:    "test-bucket",
			Container: "all-rustfs-1",
			Enabled:   true,
		},
		{
			Name:      "seaweedfs",
			Endpoint:  "localhost:8333",
			AccessKey: "admin",
			SecretKey: "adminpassword",
			Bucket:    "test-bucket",
			Container: "all-seaweedfs-s3-1",
			Enabled:   true,
		},
		{
			Name:      "localstack",
			Endpoint:  "localhost:4566",
			AccessKey: "test",
			SecretKey: "test",
			Bucket:    "test-bucket",
			Container: "all-localstack-1",
			Enabled:   true,
		},
		{
			Name:      "liteio",
			Endpoint:  "localhost:9200",
			AccessKey: "liteio",
			SecretKey: "liteio123",
			Bucket:    "test-bucket",
			Container: "all-liteio-1",
			Enabled:   true,
		},
		{
			Name:      "liteio_mem",
			Endpoint:  "localhost:9201",
			AccessKey: "liteio",
			SecretKey: "liteio123",
			Bucket:    "test-bucket",
			Container: "all-liteio_mem-1",
			Enabled:   true,
		},
	}
}

// FilterDrivers returns only the drivers matching the given names.
// If names is empty, returns all enabled drivers.
func FilterDrivers(drivers []*DriverConfig, names []string) []*DriverConfig {
	if len(names) == 0 {
		var enabled []*DriverConfig
		for _, d := range drivers {
			if d.Enabled {
				enabled = append(enabled, d)
			}
		}
		return enabled
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	var filtered []*DriverConfig
	for _, d := range drivers {
		if nameSet[d.Name] && d.Enabled {
			filtered = append(filtered, d)
		}
	}
	return filtered
}
