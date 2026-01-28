// Package fineweb provides a DuckDB-based search engine for FineWeb-2 dataset.
package fineweb

import (
	"os"
	"path/filepath"
)

// Config holds engine configuration.
type Config struct {
	// DataDir is the directory for DuckDB databases
	// Default: $HOME/data/blueprints/search/fineweb-2
	DataDir string

	// SourceDir is the directory containing parquet files
	// Default: $HOME/data/fineweb-2
	SourceDir string

	// Languages to enable (empty = all downloaded)
	Languages []string

	// ResultLimit is the maximum results per query
	ResultLimit int

	// ContentSnippetLength is the max length of content snippets
	ContentSnippetLength int
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		DataDir:              filepath.Join(home, "data", "blueprints", "search", "fineweb-2"),
		SourceDir:            filepath.Join(home, "data", "fineweb-2"),
		Languages:            nil, // All downloaded
		ResultLimit:          100,
		ContentSnippetLength: 300,
	}
}

// Document represents a document from FineWeb-2.
type Document struct {
	ID            string  `json:"id"`
	URL           string  `json:"url"`
	Text          string  `json:"text"`
	Dump          string  `json:"dump,omitempty"`
	Date          string  `json:"date,omitempty"`
	Language      string  `json:"language,omitempty"`
	LanguageScore float64 `json:"language_score,omitempty"`
	Score         float64 `json:"score,omitempty"` // BM25 score from search
}

// ImportState tracks import status for a parquet file.
type ImportState struct {
	ParquetFile string `json:"parquet_file"`
	ImportedAt  string `json:"imported_at"`
	RowCount    int64  `json:"row_count"`
}
