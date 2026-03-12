package store

import (
	"context"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
)

// ResultStore writes crawl results to persistent storage.
type ResultStore interface {
	// Add queues a result for batch writing. Never blocks.
	Add(r crawl.Result)
	// Flush sends all pending batches to the underlying writer. Blocks until done.
	Flush(ctx context.Context) error
	// FlushedCount returns the number of results successfully written.
	FlushedCount() int64
	// PendingCount returns the number of results waiting to be flushed.
	PendingCount() int
	// Close flushes remaining results and releases all resources.
	Close() error
}

// FailedStore records failed domains and URLs from a crawl run.
type FailedStore interface {
	AddDomain(d crawl.FailedDomain)
	AddURL(u crawl.FailedURL)
	AddURLBatch(urls []crawl.SeedURL, reason string)
	DomainCount() int64
	URLCount() int64
	Close() error
}

// Compile-time interface checks.
var (
	_ ResultStore = (*ResultDB)(nil)
	_ FailedStore = (*FailedDB)(nil)
)
