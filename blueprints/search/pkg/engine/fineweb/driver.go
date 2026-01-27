package fineweb

import (
	"context"
	"iter"
)

// Driver is the minimal interface all search backends must implement.
type Driver interface {
	// Name returns the driver identifier (e.g., "duckdb", "bleve").
	Name() string

	// Search performs full-text search and returns ranked results.
	Search(ctx context.Context, query string, limit, offset int) (*SearchResult, error)

	// Close releases resources.
	Close() error
}

// Indexer is implemented by drivers that support document ingestion.
type Indexer interface {
	// Import ingests documents with progress reporting.
	// The iterator allows streaming from parquet without loading all into memory.
	Import(ctx context.Context, docs iter.Seq2[Document, error], progress ProgressFunc) error
}

// Stats is implemented by drivers that expose index statistics.
type Stats interface {
	// Count returns the number of indexed documents.
	Count(ctx context.Context) (int64, error)
}

// ProgressFunc reports import progress.
type ProgressFunc func(imported, total int64)

// DriverInfo contains metadata about a driver.
type DriverInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	External    bool     `json:"external"` // Requires external service (Docker)
}

// GetDriverInfo returns driver metadata if the driver implements it.
func GetDriverInfo(d Driver) *DriverInfo {
	if info, ok := d.(interface{ Info() *DriverInfo }); ok {
		return info.Info()
	}
	return &DriverInfo{Name: d.Name()}
}

// AsIndexer returns the driver as an Indexer if it implements the interface.
func AsIndexer(d Driver) (Indexer, bool) {
	i, ok := d.(Indexer)
	return i, ok
}

// AsStats returns the driver as Stats if it implements the interface.
func AsStats(d Driver) (Stats, bool) {
	s, ok := d.(Stats)
	return s, ok
}
