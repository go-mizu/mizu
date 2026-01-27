// Package tantivy provides a Tantivy-based driver for fineweb full-text search.
// Note: This driver requires CGO and the tantivy-go bindings.
// Build with: CGO_ENABLED=1 go build -tags tantivy
//
//go:build tantivy
// +build tantivy

package tantivy

import (
	"context"
	"fmt"
	"iter"
	"os"
	"path/filepath"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func init() {
	fineweb.Register("tantivy", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// Driver implements the fineweb.Driver interface using Tantivy.
type Driver struct {
	indexDir string
	dataDir  string
	language string
}

// New creates a new Tantivy driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	indexName := "fineweb.tantivy"
	if cfg.Language != "" {
		indexName = cfg.Language + ".tantivy"
	}
	indexDir := filepath.Join(dataDir, indexName)

	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return nil, fmt.Errorf("creating index directory: %w", err)
	}

	return &Driver{
		indexDir: indexDir,
		dataDir:  dataDir,
		language: cfg.Language,
	}, nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "tantivy"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "tantivy",
		Description: "Tantivy full-text search (Rust via CGO, very fast) - REQUIRES CGO",
		Features:    []string{"bm25", "fast", "rust-native", "cgo"},
		External:    false,
	}
}

// Search performs full-text search.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	// TODO: Implement with tantivy-go bindings
	// This requires CGO and complex setup
	return nil, fmt.Errorf("tantivy driver not implemented - requires CGO build")
}

// Import ingests documents from an iterator.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	return fmt.Errorf("tantivy driver not implemented - requires CGO build")
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	return 0, fmt.Errorf("tantivy driver not implemented - requires CGO build")
}

// Close closes the index.
func (d *Driver) Close() error {
	return nil
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
