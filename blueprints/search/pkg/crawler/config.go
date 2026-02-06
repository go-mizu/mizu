package crawler

import (
	"fmt"
	"time"
)

// Config holds crawler configuration.
type Config struct {
	Workers       int           // Concurrent workers (default: 4)
	MaxDepth      int           // Maximum crawl depth (default: 2)
	MaxPages      int           // Maximum pages to crawl (default: 100)
	Delay         time.Duration // Delay between requests per domain (default: 1s)
	UserAgent     string        // User-Agent header (default: "MizuCrawler/1.0")
	Timeout       time.Duration // Per-request timeout (default: 30s)
	Scope         ScopePolicy   // URL scope policy (default: ScopeSameDomain)
	IncludeGlobs  []string      // URL patterns to include (empty = all)
	ExcludeGlobs  []string      // URL patterns to exclude
	RespectRobots bool          // Honor robots.txt (default: true)
	StateFile     string        // Path for state persistence (empty = no persistence)
	BatchSize     int           // Batch size for result callbacks (default: 10)
}

// DefaultConfig returns the default crawler configuration.
func DefaultConfig() Config {
	return Config{
		Workers:       4,
		MaxDepth:      2,
		MaxPages:      100,
		Delay:         1 * time.Second,
		UserAgent:     "MizuCrawler/1.0",
		Timeout:       30 * time.Second,
		Scope:         ScopeSameDomain,
		RespectRobots: true,
		BatchSize:     10,
	}
}

// Validate checks the config for invalid values.
func (c Config) Validate() error {
	if c.Workers < 1 {
		return fmt.Errorf("workers must be >= 1, got %d", c.Workers)
	}
	if c.MaxDepth < 0 {
		return fmt.Errorf("max depth must be >= 0, got %d", c.MaxDepth)
	}
	if c.MaxPages < 1 {
		return fmt.Errorf("max pages must be >= 1, got %d", c.MaxPages)
	}
	if c.Delay < 0 {
		return fmt.Errorf("delay must be >= 0, got %v", c.Delay)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be > 0, got %v", c.Timeout)
	}
	if c.BatchSize < 1 {
		return fmt.Errorf("batch size must be >= 1, got %d", c.BatchSize)
	}
	return nil
}

// merge fills zero-value fields from DefaultConfig.
func (c Config) merge() Config {
	defaults := DefaultConfig()
	if c.Workers == 0 {
		c.Workers = defaults.Workers
	}
	if c.MaxPages == 0 {
		c.MaxPages = defaults.MaxPages
	}
	if c.Delay == 0 {
		c.Delay = defaults.Delay
	}
	if c.UserAgent == "" {
		c.UserAgent = defaults.UserAgent
	}
	if c.Timeout == 0 {
		c.Timeout = defaults.Timeout
	}
	if c.BatchSize == 0 {
		c.BatchSize = defaults.BatchSize
	}
	return c
}
