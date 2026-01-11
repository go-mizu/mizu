// Package storebench provides benchmarking tools for comparing storage backends.
package storebench

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Config holds benchmark configuration.
type Config struct {
	Backends    []string // Backends to test: duckdb, postgres, sqlite
	Scenarios   []string // Scenarios to run: records, batch, query, fields, concurrent
	Iterations  int      // Base iteration count
	Concurrency int      // Max concurrency for load tests
	WarmupIters int      // Warmup iterations
	OutputDir   string   // Output directory for reports
	DataDir     string   // Directory for DuckDB/SQLite files
	PostgresURL string   // PostgreSQL connection URL
	Verbose     bool     // Verbose output
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Backends:    []string{"duckdb", "sqlite"},
		Scenarios:   []string{"records", "batch", "query", "fields", "concurrent"},
		Iterations:  100,
		Concurrency: 50,
		WarmupIters: 10,
		OutputDir:   "./report",
		DataDir:     filepath.Join(os.TempDir(), "storebench"),
		PostgresURL: os.Getenv("STOREBENCH_POSTGRES_URL"),
		Verbose:     os.Getenv("STOREBENCH_VERBOSE") == "true",
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if len(c.Backends) == 0 {
		return fmt.Errorf("at least one backend must be specified")
	}
	for _, b := range c.Backends {
		switch b {
		case "duckdb", "postgres", "sqlite":
			// valid
		default:
			return fmt.Errorf("unknown backend: %s", b)
		}
	}
	if c.Iterations < 1 {
		return fmt.Errorf("iterations must be at least 1")
	}
	if c.Concurrency < 1 {
		return fmt.Errorf("concurrency must be at least 1")
	}
	return nil
}

// Environment holds system environment information.
type Environment struct {
	GoVersion   string
	OS          string
	Arch        string
	NumCPU      int
	Timestamp   time.Time
	Hostname    string
	PostgresURL string
	DataDir     string
}

// GetEnvironment captures the current environment.
func GetEnvironment(cfg *Config) Environment {
	hostname, _ := os.Hostname()
	return Environment{
		GoVersion:   runtime.Version(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		NumCPU:      runtime.NumCPU(),
		Timestamp:   time.Now(),
		Hostname:    hostname,
		PostgresURL: maskPassword(cfg.PostgresURL),
		DataDir:     cfg.DataDir,
	}
}

func maskPassword(url string) string {
	if url == "" {
		return "(not configured)"
	}
	// Simple masking - replace password portion
	return "(configured)"
}
