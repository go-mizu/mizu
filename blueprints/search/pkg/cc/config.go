package cc

import (
	"os"
	"path/filepath"
	"time"
)

// Config holds configuration for Common Crawl operations.
type Config struct {
	DataDir         string        // Base data directory ($HOME/data/common-crawl)
	CrawlID         string        // Crawl identifier (CC-MAIN-2026-04)
	BaseURL         string        // Base URL for data access (https://data.commoncrawl.org)
	Workers         int           // Concurrent WARC fetch workers
	IndexWorkers    int           // Concurrent index download workers
	Timeout         time.Duration // Per-request HTTP timeout
	IndexTimeout    time.Duration // Per-index-file download timeout
	BatchSize       int           // DB write batch size
	MaxBodySize     int           // Max HTML body to store (bytes)
	TransportShards int           // HTTP transport shard count
	Resume          bool          // Skip already-fetched records
}

// DefaultConfig returns sensible defaults for Common Crawl operations.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		DataDir:         filepath.Join(home, "data", "common-crawl"),
		CrawlID:         "CC-MAIN-2026-04",
		BaseURL:         "https://data.commoncrawl.org",
		Workers:         5000,
		IndexWorkers:    10,
		Timeout:         30 * time.Second,
		IndexTimeout:    10 * time.Minute,
		BatchSize:       5000,
		MaxBodySize:     512 * 1024,
		TransportShards: 32,
	}
}

// CrawlDir returns the directory for a specific crawl's data.
func (c Config) CrawlDir() string {
	return filepath.Join(c.DataDir, c.CrawlID)
}

// IndexDir returns the directory for columnar index parquet files.
func (c Config) IndexDir() string {
	return filepath.Join(c.CrawlDir(), "index")
}

// IndexDBPath returns the path to the imported DuckDB index.
func (c Config) IndexDBPath() string {
	return filepath.Join(c.CrawlDir(), "index.duckdb")
}

// ResultDir returns the directory for result shard files.
func (c Config) ResultDir() string {
	return filepath.Join(c.CrawlDir(), "results")
}

// CDXJDir returns the directory for CDXJ index files.
func (c Config) CDXJDir() string {
	return filepath.Join(c.CrawlDir(), "cdxj")
}

// WARCDir returns the directory for downloaded WARC files.
func (c Config) WARCDir() string {
	return filepath.Join(c.CrawlDir(), "warc")
}

// RecrawlDir returns the directory for recrawl result shard files.
// Separate from ResultDir (WARC fetch results) to avoid confusion.
func (c Config) RecrawlDir() string {
	return filepath.Join(c.CrawlDir(), "recrawl")
}

// DNSCachePath returns the path to the shared DNS cache.
func (c Config) DNSCachePath() string {
	return filepath.Join(c.CrawlDir(), "dns.duckdb")
}

// FailedDBPath returns the path to the failed domains/URLs database.
func (c Config) FailedDBPath() string {
	return filepath.Join(c.RecrawlDir(), "failed.duckdb")
}

// VerifyDBPath returns the path to the verification results database.
func (c Config) VerifyDBPath() string {
	return filepath.Join(c.RecrawlDir(), "verified.duckdb")
}
