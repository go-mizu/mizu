package dcrawler

import (
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
		UserAgent:     "MizuCrawler/1.0",
		DataDir:       defaultDataDir(),
		ShardCount:    8,
		BatchSize:     500,
		StoreLinks:    true,
		RespectRobots: true,
		FollowSitemap: true,
		FrontierSize:  4_000_000,
		BloomCapacity: 50_000_000,
		BloomFPR:      0.001,
	}
}

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "crawler")
}

// NormalizeDomain strips www. prefix and lowercases the domain.
func NormalizeDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimSuffix(domain, "/")
	domain = strings.TrimPrefix(domain, "www.")
	return domain
}

// DomainDir returns the directory for storing crawl data for a domain.
func (c *Config) DomainDir() string {
	return filepath.Join(c.DataDir, NormalizeDomain(c.Domain))
}

// ResultDir returns the directory for sharded result DuckDB files.
func (c *Config) ResultDir() string {
	return filepath.Join(c.DomainDir(), "results")
}
