// Package bleve provides a Bleve-based driver for fineweb full-text search.
package bleve

import (
	"context"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/token/unicodenorm"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func init() {
	fineweb.Register("bleve", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// Driver implements the fineweb.Driver interface using Bleve.
type Driver struct {
	index    bleve.Index
	indexDir string
	dataDir  string
	language string
}

// BleveDocument is the document structure for Bleve indexing.
type BleveDocument struct {
	ID            string  `json:"id"`
	URL           string  `json:"url"`
	Text          string  `json:"text"`
	Dump          string  `json:"dump"`
	Date          string  `json:"date"`
	Language      string  `json:"language"`
	LanguageScore float64 `json:"language_score"`
}

// New creates a new Bleve driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	indexName := "fineweb.bleve"
	if cfg.Language != "" {
		indexName = cfg.Language + ".bleve"
	}
	indexDir := filepath.Join(dataDir, indexName)

	d := &Driver{
		indexDir: indexDir,
		dataDir:  dataDir,
		language: cfg.Language,
	}

	// Try to open existing index
	index, err := bleve.Open(indexDir)
	if err == bleve.ErrorIndexPathDoesNotExist {
		// Create new index
		index, err = d.createIndex(indexDir)
		if err != nil {
			return nil, fmt.Errorf("creating index: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("opening index: %w", err)
	}

	d.index = index
	return d, nil
}

func (d *Driver) createIndex(path string) (bleve.Index, error) {
	// Create Vietnamese-aware mapping
	indexMapping := d.createMapping()
	return bleve.New(path, indexMapping)
}

func (d *Driver) createMapping() mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()

	// Create custom Vietnamese analyzer
	err := indexMapping.AddCustomAnalyzer("vietnamese", map[string]interface{}{
		"type":         custom.Name,
		"tokenizer":    unicode.Name,
		"token_filters": []string{
			unicodenorm.Name,
			lowercase.Name,
		},
	})
	if err != nil {
		// Fallback to standard analyzer
		indexMapping.DefaultAnalyzer = "standard"
		return indexMapping
	}

	// Document mapping
	docMapping := bleve.NewDocumentMapping()

	// Text field with Vietnamese analyzer
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Analyzer = "vietnamese"
	textFieldMapping.Store = true
	textFieldMapping.IncludeTermVectors = true
	docMapping.AddFieldMappingsAt("text", textFieldMapping)

	// URL field
	urlFieldMapping := bleve.NewTextFieldMapping()
	urlFieldMapping.Analyzer = "standard"
	urlFieldMapping.Store = true
	docMapping.AddFieldMappingsAt("url", urlFieldMapping)

	// ID field (keyword, not analyzed)
	idFieldMapping := bleve.NewTextFieldMapping()
	idFieldMapping.Analyzer = "keyword"
	idFieldMapping.Store = true
	docMapping.AddFieldMappingsAt("id", idFieldMapping)

	// Metadata fields
	dumpMapping := bleve.NewTextFieldMapping()
	dumpMapping.Store = true
	dumpMapping.Index = false
	docMapping.AddFieldMappingsAt("dump", dumpMapping)

	dateMapping := bleve.NewTextFieldMapping()
	dateMapping.Store = true
	dateMapping.Index = false
	docMapping.AddFieldMappingsAt("date", dateMapping)

	langMapping := bleve.NewTextFieldMapping()
	langMapping.Store = true
	langMapping.Index = false
	docMapping.AddFieldMappingsAt("language", langMapping)

	scoreMapping := bleve.NewNumericFieldMapping()
	scoreMapping.Store = true
	scoreMapping.Index = false
	docMapping.AddFieldMappingsAt("language_score", scoreMapping)

	indexMapping.DefaultMapping = docMapping
	indexMapping.DefaultAnalyzer = "vietnamese"

	return indexMapping
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "bleve"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "bleve",
		Description: "Bleve full-text search with TF-IDF/BM25 ranking",
		Features:    []string{"bm25", "tf-idf", "facets", "highlighting", "fuzzy"},
		External:    false,
	}
}

// Search performs full-text search.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Create search request
	searchRequest := bleve.NewSearchRequest(bleve.NewMatchQuery(query))
	searchRequest.Size = limit
	searchRequest.From = offset
	searchRequest.Fields = []string{"*"} // Return all stored fields

	// Execute search
	searchResult, err := d.index.SearchInContext(ctx, searchRequest)
	if err != nil {
		return nil, fmt.Errorf("executing search: %w", err)
	}

	// Convert results
	docs := make([]fineweb.Document, 0, len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		doc := fineweb.Document{
			ID:    hit.ID,
			Score: hit.Score,
		}

		// Extract fields from hit
		if v, ok := hit.Fields["url"].(string); ok {
			doc.URL = v
		}
		if v, ok := hit.Fields["text"].(string); ok {
			doc.Text = v
		}
		if v, ok := hit.Fields["dump"].(string); ok {
			doc.Dump = v
		}
		if v, ok := hit.Fields["date"].(string); ok {
			doc.Date = v
		}
		if v, ok := hit.Fields["language"].(string); ok {
			doc.Language = v
		}
		if v, ok := hit.Fields["language_score"].(float64); ok {
			doc.LanguageScore = v
		}

		docs = append(docs, doc)
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "bleve",
		Total:     int64(searchResult.Total),
	}, nil
}

// Import ingests documents from an iterator.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	batch := d.index.NewBatch()
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

		bleveDoc := BleveDocument{
			ID:            doc.ID,
			URL:           doc.URL,
			Text:          doc.Text,
			Dump:          doc.Dump,
			Date:          doc.Date,
			Language:      doc.Language,
			LanguageScore: doc.LanguageScore,
		}

		if err := batch.Index(doc.ID, bleveDoc); err != nil {
			return fmt.Errorf("indexing document %s: %w", doc.ID, err)
		}

		count++
		imported++

		if count >= batchSize {
			if err := d.index.Batch(batch); err != nil {
				return fmt.Errorf("committing batch: %w", err)
			}
			batch = d.index.NewBatch()
			count = 0

			if progress != nil {
				progress(imported, 0)
			}
		}
	}

	// Commit remaining
	if count > 0 {
		if err := d.index.Batch(batch); err != nil {
			return fmt.Errorf("committing final batch: %w", err)
		}
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	count, err := d.index.DocCount()
	return int64(count), err
}

// Close closes the index.
func (d *Driver) Close() error {
	return d.index.Close()
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
