// Package warc_md implements a 3-phase pipeline that converts Common Crawl
// .warc.gz files into clean, gzip-compressed Markdown files.
//
// File mode (default): each phase writes to disk; phases run sequentially.
// In-memory mode (InMemory=true): phases connected by channels; no temp files.
package warc_md

import (
	"os"
	"path/filepath"
	"runtime"

	mdpkg "github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
)

// Config configures the WARC → Markdown pipeline.
type Config struct {
	CrawlID     string // e.g. "CC-MAIN-2026-08"
	DataDir     string // base: $HOME/data/common-crawl
	Workers     int    // parallel workers for convert/compress (0 = NumCPU)
	Force       bool   // re-process existing files
	Fast        bool   // use go-readability instead of trafilatura (3–8× faster)
	KeepTemp    bool   // keep warc_single/ and markdown/ after pipeline
	InMemory    bool   // streaming pipeline: no temp files
	MIMEFilter  string // e.g. "text/html" (default)
	StatusCode  int    // HTTP status filter (default: 200)
	MaxBodySize int64  // max HTML body bytes (default: 512 KB)

	// SharedIndex is an already-opened IndexDB shared across parallel pipelines.
	// When non-nil, RunInMemoryPipeline uses it instead of opening its own.
	// The caller is responsible for closing it after all pipelines finish.
	SharedIndex *mdpkg.IndexDB

	// NoIndex disables all DuckDB index writes. Useful for perf benchmarking
	// or when the caller doesn't need the per-document metadata.
	NoIndex bool
}

// DefaultConfig returns sensible defaults for a given crawl ID.
func DefaultConfig(crawlID string) Config {
	home, _ := os.UserHomeDir()
	return Config{
		CrawlID:     crawlID,
		DataDir:     filepath.Join(home, "data", "common-crawl"),
		Workers:     0,
		MIMEFilter:  "text/html",
		StatusCode:  200,
		MaxBodySize: 512 * 1024,
	}
}

// CrawlDir returns the crawl-specific data directory.
func (c Config) CrawlDir() string {
	return filepath.Join(c.DataDir, c.CrawlID)
}

// WARCDir returns the directory containing downloaded .warc.gz files.
func (c Config) WARCDir() string {
	return filepath.Join(c.CrawlDir(), "warc")
}

// WARCSingleDir returns the directory for extracted single-record files.
// Each file contains raw HTML body bytes and is named {recordID}.warc.
func (c Config) WARCSingleDir() string {
	return filepath.Join(c.CrawlDir(), "warc_single")
}

// MarkdownDir returns the directory for converted raw markdown files (Phase 2 temp output).
func (c Config) MarkdownDir() string {
	return filepath.Join(c.CrawlDir(), "markdown_raw")
}

// MarkdownGzDir returns the directory for compressed markdown files (final output).
func (c Config) MarkdownGzDir() string {
	return filepath.Join(c.CrawlDir(), "markdown")
}

// IndexPath returns the DuckDB index path (inside markdown/).
func (c Config) IndexPath() string {
	return filepath.Join(c.MarkdownGzDir(), "index.duckdb")
}

// ConvertWorkers returns the optimal worker count for Phase 2 (HTML→Markdown).
//
// Both trafilatura and go-readability are CPU-bound; the optimal is NumCPU.
// More goroutines than cores adds context-switch overhead with no throughput gain.
//
// Workers field overrides the auto value when > 0.
func (c Config) ConvertWorkers() int {
	if c.Workers > 0 {
		return c.Workers
	}
	return runtime.NumCPU()
}

// CompressWorkers returns the optimal worker count for Phase 3 (gzip compress).
//
// klauspost gzip BestSpeed is fast but still CPU-bound; NumCPU is the right
// default. Workers field overrides when > 0.
func (c Config) CompressWorkers() int {
	if c.Workers > 0 {
		return c.Workers
	}
	return runtime.NumCPU()
}
