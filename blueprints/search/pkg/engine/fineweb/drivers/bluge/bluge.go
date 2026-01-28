// Package bluge provides a Bluge-based driver for fineweb full-text search.
package bluge

import (
	"context"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/analysis"
	"github.com/blugelabs/bluge/analysis/analyzer"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/tokenizer"
)

func init() {
	fineweb.Register("bluge", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// Driver implements the fineweb.Driver interface using Bluge.
type Driver struct {
	writer    *bluge.Writer
	reader    *bluge.Reader
	indexDir  string
	dataDir   string
	language  string
	analyzer  *analysis.Analyzer
	tokenizer *tokenizer.Vietnamese
	mu        sync.RWMutex // Protects reader access
}

// New creates a new Bluge driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	indexName := "fineweb.bluge"
	if cfg.Language != "" {
		indexName = cfg.Language + ".bluge"
	}
	indexDir := filepath.Join(dataDir, indexName)

	// Create Vietnamese tokenizer
	vietTokenizer := tokenizer.NewVietnamese()

	// Create custom analyzer using Vietnamese tokenizer
	customAnalyzer := analyzer.NewStandardAnalyzer()

	config := bluge.DefaultConfig(indexDir)

	writer, err := bluge.OpenWriter(config)
	if err != nil {
		return nil, fmt.Errorf("opening writer: %w", err)
	}

	reader, err := writer.Reader()
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("opening reader: %w", err)
	}

	d := &Driver{
		writer:    writer,
		reader:    reader,
		indexDir:  indexDir,
		dataDir:   dataDir,
		language:  cfg.Language,
		analyzer:  customAnalyzer,
		tokenizer: vietTokenizer,
	}

	return d, nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "bluge"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "bluge",
		Description: "Bluge full-text search with BM25 ranking (modern Bleve successor)",
		Features:    []string{"bm25", "aggregations", "highlighting", "modern-api"},
		External:    false,
	}
}

// Search performs full-text search.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Get cached reader (thread-safe)
	d.mu.RLock()
	reader := d.reader
	d.mu.RUnlock()

	// Build query
	q := bluge.NewMatchQuery(query).SetField("text")

	// Create search request
	req := bluge.NewTopNSearch(limit, q).
		SetFrom(offset).
		WithStandardAggregations()

	// Execute search
	dmi, err := reader.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("executing search: %w", err)
	}

	// Collect results
	docs := make([]fineweb.Document, 0, limit)
	next, err := dmi.Next()
	for err == nil && next != nil {
		doc := fineweb.Document{
			Score: next.Score,
		}

		// Load stored fields
		err = next.VisitStoredFields(func(field string, value []byte) bool {
			switch field {
			case "_id":
				doc.ID = string(value)
			case "url":
				doc.URL = string(value)
			case "text":
				doc.Text = string(value)
			case "dump":
				doc.Dump = string(value)
			case "date":
				doc.Date = string(value)
			case "language":
				doc.Language = string(value)
			}
			return true
		})
		if err != nil {
			return nil, fmt.Errorf("loading stored fields: %w", err)
		}

		docs = append(docs, doc)
		next, err = dmi.Next()
	}
	if err != nil {
		return nil, fmt.Errorf("iterating results: %w", err)
	}

	// Get aggregation results
	aggs := dmi.Aggregations()
	total := aggs.Count()

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "bluge",
		Total:     int64(total),
	}, nil
}

// Import ingests documents from an iterator.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	batch := bluge.NewBatch()
	batchSize := 1000
	count := 0
	var imported int64

	for doc, err := range docs {
		if err != nil {
			return fmt.Errorf("reading document: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Create Bluge document
		blugeDoc := bluge.NewDocument(doc.ID).
			AddField(bluge.NewTextField("url", doc.URL).StoreValue()).
			AddField(bluge.NewTextField("text", doc.Text).StoreValue().HighlightMatches()).
			AddField(bluge.NewTextField("dump", doc.Dump).StoreValue()).
			AddField(bluge.NewTextField("date", doc.Date).StoreValue()).
			AddField(bluge.NewTextField("language", doc.Language).StoreValue()).
			AddField(bluge.NewNumericField("language_score", doc.LanguageScore))

		batch.Update(blugeDoc.ID(), blugeDoc)

		count++
		imported++

		if count >= batchSize {
			if err := d.writer.Batch(batch); err != nil {
				return fmt.Errorf("committing batch: %w", err)
			}
			batch = bluge.NewBatch()
			count = 0

			if progress != nil {
				progress(imported, 0)
			}
		}
	}

	// Commit remaining
	if count > 0 {
		if err := d.writer.Batch(batch); err != nil {
			return fmt.Errorf("committing final batch: %w", err)
		}
	}

	// Refresh reader to see newly indexed documents
	if err := d.RefreshReader(); err != nil {
		return fmt.Errorf("refreshing reader after import: %w", err)
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

// RefreshReader refreshes the reader to see latest changes.
// Call this after indexing to make new documents searchable.
func (d *Driver) RefreshReader() error {
	newReader, err := d.writer.Reader()
	if err != nil {
		return fmt.Errorf("refreshing reader: %w", err)
	}

	d.mu.Lock()
	oldReader := d.reader
	d.reader = newReader
	d.mu.Unlock()

	if oldReader != nil {
		oldReader.Close()
	}
	return nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	d.mu.RLock()
	reader := d.reader
	d.mu.RUnlock()

	// Count all documents
	q := bluge.NewMatchAllQuery()
	req := bluge.NewAllMatches(q).WithStandardAggregations()

	dmi, err := reader.Search(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("counting documents: %w", err)
	}

	aggs := dmi.Aggregations()
	count := aggs.Count()

	return int64(count), nil
}

// Close closes the index.
func (d *Driver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.reader != nil {
		d.reader.Close()
		d.reader = nil
	}
	if d.writer != nil {
		return d.writer.Close()
	}
	return nil
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
