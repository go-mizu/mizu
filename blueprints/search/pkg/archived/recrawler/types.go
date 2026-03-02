// Package recrawler is deprecated. Use pkg/crawl and pkg/crawl/store instead.
// This file re-exports types for backward compatibility.
package recrawler

import (
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
)

// Type aliases pointing to their new homes in pkg/crawl.
type SeedURL = crawl.SeedURL
type Result = crawl.Result
type FailedURL = crawl.FailedURL
type FailedDomain = crawl.FailedDomain

// SeedStats holds aggregate stats about the seed database.
// Kept here to avoid breaking callers; mirrors store.SeedStats.
type SeedStats struct {
	TotalURLs     int
	UniqueDomains int
	Protocols     map[string]int // HTTP vs HTTPS
	ContentTypes  map[string]int
	TLDs          map[string]int
}

// Config holds configuration for high-throughput recrawling.
type Config struct {
	Workers             int           // Concurrent HTTP fetch workers (default: 2000)
	DNSWorkers          int           // Concurrent DNS workers (default: 2000)
	DNSTimeout          time.Duration // DNS lookup timeout (default: 2s)
	Timeout             time.Duration // Per-request HTTP timeout (default: 3s)
	UserAgent           string        // User-Agent header
	HeadOnly            bool          // Only fetch headers, skip body
	StatusOnly          bool          // Only check HTTP status, close body immediately (fastest)
	BatchSize           int           // DB write batch size (default: 5000)
	Resume              bool          // Skip already-crawled URLs
	DNSPrefetch         bool          // Pre-resolve DNS for all domains
	DomainFailThreshold int           // Failures before marking domain dead (default: 3)
	TransportShards     int           // Number of HTTP transport shards (default: 64)
	MaxConnsPerDomain   int           // Max concurrent connections per domain (0=unlimited, default: 8)
	TwoPass             bool          // Enable two-pass: probe domains before full fetch
	ProbeTimeout        time.Duration // Adaptive probe timeout for unknown domains (default: 1.5s)
}

// DefaultConfig returns optimal defaults for high throughput.
func DefaultConfig() Config {
	return Config{
		Workers:         200,
		DNSWorkers:      2000,
		DNSTimeout:      2 * time.Second,
		Timeout:         5 * time.Second,
		UserAgent:       "MizuCrawler/1.0",
		BatchSize:       5000,
		TransportShards: 64,
		DNSPrefetch:     true,
	}
}
