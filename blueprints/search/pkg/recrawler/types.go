// Package recrawler provides a high-throughput recrawler for known URL sets.
// It reads seed URLs from DuckDB, crawls them with maximum parallelism,
// and stores results and state in separate DuckDB files.
package recrawler

import "time"

// SeedURL represents a URL loaded from the seed database.
// Only url and domain are loaded for crawling performance.
type SeedURL struct {
	URL    string
	Domain string
}

// SeedStats holds aggregate stats about the seed database.
type SeedStats struct {
	TotalURLs     int
	UniqueDomains int
	Protocols     map[string]int // HTTP vs HTTPS
	ContentTypes  map[string]int
	TLDs          map[string]int
}

// Result holds the result of recrawling a single URL.
type Result struct {
	URL           string
	StatusCode    int
	ContentType   string
	ContentLength int64
	Title         string
	Description   string
	Language      string
	Domain        string
	RedirectURL   string
	FetchTimeMs   int64
	CrawledAt     time.Time
	Error         string
}

// Config holds configuration for high-throughput recrawling.
type Config struct {
	Workers              int           // Concurrent workers (default: 2000)
	Timeout              time.Duration // Per-request timeout (default: 3s)
	UserAgent            string        // User-Agent header
	HeadOnly             bool          // Only fetch headers, skip body
	StatusOnly           bool          // Only check HTTP status, close body immediately (fastest)
	BatchSize            int           // DB write batch size (default: 5000)
	Resume               bool          // Skip already-crawled URLs
	DNSPrefetch          bool          // Pre-resolve DNS for all domains
	DomainFailThreshold  int           // Failures before marking domain dead (default: 1)
	TransportShards      int           // Number of HTTP transport shards (default: 1)
}

// DefaultConfig returns optimal defaults for high throughput.
func DefaultConfig() Config {
	return Config{
		Workers:         2000,
		Timeout:         3 * time.Second,
		UserAgent:       "MizuCrawler/1.0",
		HeadOnly:        false,
		BatchSize:       5000,
		TransportShards: 1,
	}
}
