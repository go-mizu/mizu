// Package fts_zig provides Go bindings for the fts_zig high-performance
// full-text search library written in Zig.
//
// Three integration modes are supported:
//   - CGO: Direct library calls via CGO (lowest latency)
//   - IPC: Unix socket communication (no CGO dependency)
//   - Mmap: Memory-mapped shared segments (zero-copy reads)
package fts_zig

import (
	"errors"
	"iter"
)

// Profile represents the search profile to use.
type Profile int

const (
	// ProfileSpeed uses raw arrays with no compression for <1ms p99 latency.
	ProfileSpeed Profile = iota
	// ProfileBalanced uses Block-Max WAND with VByte for 1-10ms p99 latency.
	ProfileBalanced
	// ProfileCompact uses Elias-Fano encoding for 10-50ms p99 latency.
	ProfileCompact
)

// SearchResult represents a search result.
type SearchResult struct {
	DocID uint32
	Score float32
}

// Stats represents index statistics.
type Stats struct {
	DocCount    uint32
	TermCount   uint32
	MemoryBytes uint64
}

// Document represents a document to index.
type Document struct {
	ID   uint32
	Text string
}

// Driver is the interface for all fts_zig integration modes.
type Driver interface {
	// AddDocument adds a document to the index.
	AddDocument(text string) (uint32, error)

	// AddDocuments adds multiple documents to the index.
	AddDocuments(texts []string) error

	// Build finalizes the index for searching.
	Build() error

	// Search performs a search query and returns results.
	Search(query string, limit int) ([]SearchResult, error)

	// Stats returns index statistics.
	Stats() (Stats, error)

	// Close releases resources.
	Close() error
}

// Errors
var (
	ErrNotInitialized = errors.New("fts_zig: driver not initialized")
	ErrAlreadyBuilt   = errors.New("fts_zig: index already built")
	ErrNotBuilt       = errors.New("fts_zig: index not built yet")
	ErrInvalidHandle  = errors.New("fts_zig: invalid handle")
	ErrCGODisabled    = errors.New("fts_zig: CGO is disabled")
)

// Config holds configuration for creating a driver.
type Config struct {
	// Profile to use (speed, balanced, compact)
	Profile Profile

	// BasePath for segment storage (IPC and Mmap modes)
	BasePath string

	// IPCSocketPath for IPC mode
	IPCSocketPath string

	// FlushThreshold for streaming indexing
	FlushThreshold uint32
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Profile:        ProfileBalanced,
		BasePath:       "/tmp/fts_zig",
		FlushThreshold: 64 * 1024,
	}
}

// NewCGODriver creates a new CGO-based driver.
// This requires the fts_zig shared library to be available.
func NewCGODriver(cfg Config) (Driver, error) {
	return newCGODriver(cfg)
}

// NewIPCDriver creates a new IPC-based driver.
// This communicates with a separate fts_zig_server process.
func NewIPCDriver(cfg Config) (Driver, error) {
	return newIPCDriver(cfg)
}

// NewMmapDriver creates a new memory-mapped driver.
// This reads segments directly from disk without CGO.
func NewMmapDriver(cfg Config) (Driver, error) {
	return newMmapDriver(cfg)
}

// ImportFromParquet indexes all documents from a pre-loaded text slice.
// Returns the number of documents indexed.
func ImportFromParquet(driver Driver, texts []string) (int, error) {
	if err := driver.AddDocuments(texts); err != nil {
		return 0, err
	}
	return len(texts), nil
}

// ImportFromParquetIter indexes documents streamed from a batch iterator.
// This is the preferred method for large datasets — it avoids loading all texts
// into memory at once. The iterator should yield batches of text strings, e.g.
// from ParquetReader.ReadPureTextsParallel().
//
// Usage:
//
//	reader := fineweb.NewParquetReader(parquetDir)
//	n, err := fts_zig.ImportFromParquetIter(driver,
//	    reader.ReadPureTextsParallel(ctx, runtime.NumCPU()))
func ImportFromParquetIter(driver Driver, batches iter.Seq2[[]string, error]) (int, error) {
	total := 0
	for batch, err := range batches {
		if err != nil {
			return total, err
		}
		if err := driver.AddDocuments(batch); err != nil {
			return total, err
		}
		total += len(batch)
	}
	return total, nil
}
