// Package dcrawler provides a high-throughput single-domain web crawler.
// It uses HTTP/2 multiplexing, bloom filter dedup, BFS frontier,
// and sharded DuckDB storage to achieve 10K+ pages/second.
package dcrawler

import "time"

// CrawlItem represents a URL in the frontier waiting to be fetched.
type CrawlItem struct {
	URL   string
	Depth int
}

// Result holds the crawl result for a single page.
type Result struct {
	URL            string
	URLHash        uint64
	Depth          int
	StatusCode     int
	ContentType    string
	ContentLength  int64
	BodyHash       uint64
	BodyCompressed []byte // zstd-compressed body (only if StoreBody)
	Title          string
	Description    string
	Language       string
	Canonical      string
	ETag           string
	LastModified   string
	Server         string
	RedirectURL    string
	LinkCount      int
	FetchTimeMs    int64
	CrawledAt      time.Time
	Error          string
}

// Link represents an extracted link from a page.
type Link struct {
	SourceHash uint64
	TargetURL  string
	AnchorText string
	Rel        string
	IsInternal bool
}
