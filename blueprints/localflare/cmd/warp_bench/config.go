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
	Duration      time.Duration
	Concurrent    int
	Objects       int
	ObjectSizes   []string
	Operations    []string
	ListObjects   int
	ListMaxKeys   int
	Drivers       []string // Filter to specific drivers (empty = all)
	OutputDir     string
	Verbose       bool
	Quick         bool
	DockerClean   bool   // Enable Docker cleanup before/after each driver
	ComposeDir    string // Path to docker-compose directory
	WorkDir       string // Working directory for warp temp files (empty = auto)
	KeepWorkDir   bool   // Keep work dir after run (for debugging)
	NoClear       bool   // Do not clear bucket between warp runs (uses prefix per run)
	Prefix        string // Base prefix for warp objects (empty = auto)
	Lookup        string // Force path or host lookup style
	DisableSHA256 bool
	AutoTerm      bool
	AutoTermDur   time.Duration
	AutoTermPct   float64
	WarpPath      string // Resolved warp binary path
	WarpVersion   string // Resolved warp version
	RunDir        string // Actual run directory used (auto)
	ProgressEvery time.Duration
	DeleteObjects int
	DeleteBatch   int
}

// DefaultConfig returns the default benchmark configuration.
func DefaultConfig() *Config {
	return &Config{
		Duration:      30 * time.Second,
		Concurrent:    20,
		Objects:       200,
		ObjectSizes:   []string{"1MiB", "10MiB"},
		Operations:    []string{"put", "get", "stat", "list", "mixed"},
		ListObjects:   1000,
		ListMaxKeys:   100,
		OutputDir:     "./pkg/storage/report/warp_bench",
		Verbose:       false,
		Quick:         false,
		DockerClean:   false,
		ComposeDir:    "./docker/s3/all",
		NoClear:       true,
		Prefix:        "",
		Lookup:        "path",
		DisableSHA256: true,
		AutoTerm:      true,
		AutoTermDur:   15 * time.Second,
		AutoTermPct:   7.5,
		ProgressEvery: 5 * time.Second,
		DeleteObjects: 1000,
		DeleteBatch:   100,
	}
}

// QuickConfig returns a faster configuration for quick testing.
func QuickConfig() *Config {
	return &Config{
		Duration:      8 * time.Second,
		Concurrent:    10,
		Objects:       50,
		ObjectSizes:   []string{"1MiB"},
		Operations:    []string{"put", "get", "stat"},
		ListObjects:   200,
		ListMaxKeys:   100,
		OutputDir:     "./pkg/storage/report/warp_bench",
		Verbose:       false,
		Quick:         true,
		DockerClean:   false,
		ComposeDir:    "./docker/s3/all",
		NoClear:       true,
		Prefix:        "",
		Lookup:        "path",
		DisableSHA256: true,
		AutoTerm:      true,
		AutoTermDur:   10 * time.Second,
		AutoTermPct:   10,
		ProgressEvery: 5 * time.Second,
		DeleteObjects: 500,
		DeleteBatch:   50,
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
			Name:      "usagi_s3",
			Endpoint:  "localhost:9301",
			AccessKey: "usagi",
			SecretKey: "usagi123",
			Bucket:    "test-bucket",
			Container: "all-usagi_s3-1",
			Enabled:   true,
		},
		{
			Name:      "devnull_s3",
			Endpoint:  "localhost:9302",
			AccessKey: "devnull",
			SecretKey: "devnull123",
			Bucket:    "test-bucket",
			Container: "all-devnull_s3-1",
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
