// Package crawler provides a high-performance web crawler with concurrent
// workers, robots.txt compliance, and resumable state.
package crawler

import "time"

// ScopePolicy controls which URLs are followed during crawling.
type ScopePolicy int

const (
	// ScopeSameDomain only follows URLs on the exact same domain.
	ScopeSameDomain ScopePolicy = iota
	// ScopeSameHost follows URLs on the same host including subdomains.
	ScopeSameHost
	// ScopeSubpath only follows URLs under the start URL's path.
	ScopeSubpath
)

// CrawlResult is the output of crawling a single page.
// It is store-independent — the CLI layer maps CrawlResult to store.Document.
type CrawlResult struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Content     string            `json:"content"`
	Language    string            `json:"language"`
	Domain      string            `json:"domain"`
	StatusCode  int               `json:"status_code"`
	ContentType string            `json:"content_type"`
	Depth       int               `json:"depth"`
	Links       []string          `json:"links,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CrawledAt   time.Time         `json:"crawled_at"`
	FetchTimeMs int64             `json:"fetch_time_ms"`
	Error       error             `json:"-"`
}

// CrawlStats tracks aggregate crawl statistics.
type CrawlStats struct {
	PagesTotal     int           `json:"pages_total"`
	PagesSuccess   int           `json:"pages_success"`
	PagesFailed    int           `json:"pages_failed"`
	PagesSkipped   int           `json:"pages_skipped"`
	BytesTotal     int64         `json:"bytes_total"`
	Duration       time.Duration `json:"duration"`
	PagesPerSecond float64       `json:"pages_per_second"`
}

// URLEntry represents a URL in the crawl frontier.
type URLEntry struct {
	URL      string `json:"url"`
	Depth    int    `json:"depth"`
	Priority int    `json:"priority"` // lower = higher priority
}

// ProgressFn is called with crawl progress updates.
type ProgressFn func(stats CrawlStats)

// ResultFn is called for each successfully crawled page.
type ResultFn func(result CrawlResult)
