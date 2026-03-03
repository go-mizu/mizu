package warc_md

import (
	"os"
	"path/filepath"
	"runtime"
)

// Config configures the WARC → Markdown pipeline.
type Config struct {
	CrawlID     string // e.g. "CC-MAIN-2026-08"
	DataDir     string // base: $HOME/data/common-crawl
	Workers     int    // parallel workers for convert (0 = NumCPU)
	Force       bool   // re-process existing files
	Fast        bool   // use go-readability instead of trafilatura
	KeepTemp    bool   // keep warc_single/ and markdown_raw/ after pipeline
	MIMEFilter  string // e.g. "text/html" (default)
	StatusCode  int    // HTTP status filter (default: 200)
	MaxBodySize int64  // max HTML body bytes (default: 512 KB)
	// Deprecated: InMemory is no longer used (phase 3 removed). Will be
	// removed in Task 10 CLI rewrite.
	InMemory bool
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

// WARCSingleDir returns the directory for extracted single-record files (Phase 1 temp).
func (c Config) WARCSingleDir() string {
	return filepath.Join(c.CrawlDir(), "warc_single")
}

// MarkdownDir returns the directory for converted raw markdown files (Phase 2 temp output).
func (c Config) MarkdownDir() string {
	return filepath.Join(c.CrawlDir(), "markdown_raw")
}

// MarkdownWarcDir returns the final output directory for one WARC's markdown files.
// warcIdx is the zero-padded 5-digit file index, e.g. "00000".
// Files inside are plain .md (uncompressed), sharded by UUID.
func (c Config) MarkdownWarcDir(warcIdx string) string {
	return filepath.Join(c.CrawlDir(), "markdown", warcIdx)
}

// ConvertWorkers returns the optimal worker count for Phase 2 (HTML→Markdown).
func (c Config) ConvertWorkers() int {
	if c.Workers > 0 {
		return c.Workers
	}
	return runtime.NumCPU()
}

// CompressWorkers is a no-op stub retained for CLI compatibility until Task 10
// removes all phase-3 references from cc_warc_markdown.go.
//
// Deprecated: phase 3 (compress) has been removed. Use ConvertWorkers instead.
func (c Config) CompressWorkers() int {
	return runtime.NumCPU()
}

// MarkdownGzDir is a stub retained for CLI compatibility until Task 10 removes
// all phase-3 references from cc_warc_markdown.go.
//
// Deprecated: phase 3 (compress) has been removed. Output is now plain .md
// under markdown/{warcIdx}/ via MarkdownWarcDir.
func (c Config) MarkdownGzDir() string {
	return filepath.Join(c.CrawlDir(), "markdown")
}
