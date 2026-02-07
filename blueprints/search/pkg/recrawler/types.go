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
	Body          string // HTML body (full content mode)
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
	Workers              int           // Concurrent HTTP fetch workers (default: 2000)
	DNSWorkers           int           // Concurrent DNS workers (default: 2000)
	DNSTimeout           time.Duration // DNS lookup timeout (default: 2s)
	Timeout              time.Duration // Per-request HTTP timeout (default: 3s)
	UserAgent            string        // User-Agent header
	HeadOnly             bool          // Only fetch headers, skip body
	StatusOnly           bool          // Only check HTTP status, close body immediately (fastest)
	BatchSize            int           // DB write batch size (default: 5000)
	Resume               bool          // Skip already-crawled URLs
	DNSPrefetch          bool          // Pre-resolve DNS for all domains
	DomainFailThreshold  int           // Failures before marking domain dead (default: 3)
	TransportShards      int           // Number of HTTP transport shards (default: 64)
	MaxConnsPerDomain    int           // Max concurrent connections per domain (0=unlimited, default: 8)
	TwoPass              bool          // Enable two-pass: probe domains before full fetch
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
