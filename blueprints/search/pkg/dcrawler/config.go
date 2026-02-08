package dcrawler

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds crawler configuration.
type Config struct {
	Domain           string
	SeedURLs         []string
	Workers          int
	MaxConns         int
	MaxIdleConns     int
	Timeout          time.Duration
	MaxDepth         int
	MaxPages         int
	MaxBodySize      int64
	UserAgent        string
	DataDir          string
	ShardCount       int
	BatchSize        int
	StoreBody        bool
	StoreLinks       bool
	RespectRobots    bool
	FollowSitemap    bool
	Resume           bool
	FrontierSize     int
	BloomCapacity    uint
	BloomFPR         float64
	RateLimit        int
	IncludeSubdomain bool
	ForceHTTP1       bool
	TransportShards  int
	SeedFile         string
	Continuous       bool          // Run non-stop, re-seed when frontier drains
	ReseedInterval   time.Duration // Min interval between re-seeds (default 30s)
	UseRod           bool          // Use headless Chrome via rod for JS-rendered pages
	RodWorkers       int           // Number of browser pages (default 8)
	RodHeadless      bool          // Run rod in headless mode (default true)
	ScrollCount      int           // Browser mode: scroll N times for infinite scroll (0=no scroll)
	ExtractImages    bool          // Extract <img> URLs and store in links table
}

// DefaultConfig returns optimal defaults for high-throughput single-domain crawling.
// Targets 10K+ pages/sec via HTTP/2 multiplexing.
func DefaultConfig() Config {
	return Config{
		Workers:       1000,
		MaxConns:      200,           // ~200 TCP conns × ~250 H2 streams = 50K concurrent
		MaxIdleConns:  500,
		Timeout:       10 * time.Second,
		MaxBodySize:   512 * 1024,    // 512KB
		UserAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		DataDir:       defaultDataDir(),
		ShardCount:    8,
		BatchSize:     500,
		StoreLinks:    true,
		RespectRobots: true,
		FollowSitemap: true,
		FrontierSize:  4_000_000,
		BloomCapacity:   50_000_000,
		BloomFPR:        0.001,
		TransportShards: 16,
	}
}

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "crawler")
}

// NormalizeDomain extracts and normalizes the domain from a URL or domain string.
// Strips www. prefix, lowercases, and handles full URLs (extracts hostname only).
func NormalizeDomain(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	// If it looks like a URL, parse it to extract hostname
	if strings.Contains(input, "/") || strings.HasPrefix(input, "http") {
		if !strings.Contains(input, "://") {
			input = "https://" + input
		}
		if u, err := url.Parse(input); err == nil && u.Hostname() != "" {
			input = u.Hostname()
		}
	}
	input = strings.TrimPrefix(input, "www.")
	return input
}

// ExtractSeedURL extracts a seed URL from user input.
// If input is a full URL, returns it as-is. Otherwise builds https://{domain}/.
func ExtractSeedURL(input string) string {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return input
	}
	if strings.Contains(input, "/") {
		return "https://" + input
	}
	return ""
}

// DomainDir returns the directory for storing crawl data for a domain.
func (c *Config) DomainDir() string {
	return filepath.Join(c.DataDir, NormalizeDomain(c.Domain))
}

// ResultDir returns the directory for sharded result DuckDB files.
func (c *Config) ResultDir() string {
	return filepath.Join(c.DomainDir(), "results")
}
