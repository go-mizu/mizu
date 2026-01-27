// Package meilisearch provides a MeiliSearch-based driver for fineweb full-text search.
package meilisearch

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"time"

	"github.com/meilisearch/meilisearch-go"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

// stringPtr returns a pointer to the given string.
func stringPtr(s string) *string {
	return &s
}

func init() {
	fineweb.Register("meilisearch", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// DefaultHost is the default MeiliSearch address.
const DefaultHost = "http://localhost:7700"

// DefaultIndexName is the default index name.
const DefaultIndexName = "fineweb"

// Driver implements the fineweb.Driver interface using MeiliSearch.
type Driver struct {
	client    meilisearch.ServiceManager
	index     meilisearch.IndexManager
	indexName string
	host      string
	apiKey    string
	language  string
}

// MeiliDocument is the document structure for MeiliSearch.
type MeiliDocument struct {
	ID            string  `json:"id"`
	URL           string  `json:"url"`
	Text          string  `json:"text"`
	Dump          string  `json:"dump"`
	Date          string  `json:"date"`
	Language      string  `json:"language"`
	LanguageScore float64 `json:"language_score"`
}

// New creates a new MeiliSearch driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	host := cfg.GetString("host", DefaultHost)
	apiKey := cfg.GetString("api_key", "")

	indexName := cfg.GetString("index", DefaultIndexName)
	if cfg.Language != "" {
		indexName = cfg.Language
	}

	client := meilisearch.New(host, meilisearch.WithAPIKey(apiKey))

	d := &Driver{
		client:    client,
		indexName: indexName,
		host:      host,
		apiKey:    apiKey,
		language:  cfg.Language,
	}

	// Create or get index
	if err := d.ensureIndex(); err != nil {
		return nil, fmt.Errorf("ensuring index: %w", err)
	}

	d.index = client.Index(indexName)

	return d, nil
}

func (d *Driver) ensureIndex() error {
	// Try to create the index (will fail if exists, which is fine)
	_, err := d.client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        d.indexName,
		PrimaryKey: "id",
	})
	if err != nil {
		// Check if it's just "index already exists" error
		if meiliErr, ok := err.(*meilisearch.Error); ok {
			if meiliErr.MeilisearchApiError.Code == "index_already_exists" {
				return nil
			}
		}
		// Also accept if the error message contains "already exists"
		if err.Error() == "index already exists" {
			return nil
		}
	}

	// Configure index settings for Vietnamese
	_, err = d.client.Index(d.indexName).UpdateSettings(&meilisearch.Settings{
		SearchableAttributes: []string{"text", "url"},
		DisplayedAttributes:  []string{"id", "url", "text", "dump", "date", "language", "language_score"},
		RankingRules: []string{
			"words",
			"typo",
			"proximity",
			"attribute",
			"sort",
			"exactness",
		},
	})

	return err
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "meilisearch"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "meilisearch",
		Description: "MeiliSearch with typo tolerance, faceting, and instant search",
		Features:    []string{"typo-tolerance", "facets", "filters", "instant-search", "vietnamese-support"},
		External:    true,
	}
}

// Search performs full-text search.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Build search request
	searchRes, err := d.index.Search(query, &meilisearch.SearchRequest{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("executing search: %w", err)
	}

	// Convert results
	docs := make([]fineweb.Document, 0, len(searchRes.Hits))
	for i, hit := range searchRes.Hits {
		// Hit is map[string]json.RawMessage, unmarshal each field
		doc := fineweb.Document{
			Score: 1.0 / float64(i+1), // MeiliSearch doesn't return a score
		}

		if raw, exists := hit["id"]; exists {
			json.Unmarshal(raw, &doc.ID)
		}
		if raw, exists := hit["url"]; exists {
			json.Unmarshal(raw, &doc.URL)
		}
		if raw, exists := hit["text"]; exists {
			json.Unmarshal(raw, &doc.Text)
		}
		if raw, exists := hit["dump"]; exists {
			json.Unmarshal(raw, &doc.Dump)
		}
		if raw, exists := hit["date"]; exists {
			json.Unmarshal(raw, &doc.Date)
		}
		if raw, exists := hit["language"]; exists {
			json.Unmarshal(raw, &doc.Language)
		}
		if raw, exists := hit["language_score"]; exists {
			json.Unmarshal(raw, &doc.LanguageScore)
		}

		docs = append(docs, doc)
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "meilisearch",
		Total:     searchRes.EstimatedTotalHits,
	}, nil
}

// Import ingests documents from an iterator.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	batchSize := 1000
	batch := make([]MeiliDocument, 0, batchSize)
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

		batch = append(batch, MeiliDocument{
			ID:            doc.ID,
			URL:           doc.URL,
			Text:          doc.Text,
			Dump:          doc.Dump,
			Date:          doc.Date,
			Language:      doc.Language,
			LanguageScore: doc.LanguageScore,
		})

		if len(batch) >= batchSize {
			task, err := d.index.AddDocuments(batch, &meilisearch.DocumentOptions{
				PrimaryKey: stringPtr("id"),
			})
			if err != nil {
				return fmt.Errorf("adding documents: %w", err)
			}

			// Wait for task to complete
			_, err = d.client.WaitForTask(task.TaskUID, 5*time.Minute)
			if err != nil {
				return fmt.Errorf("waiting for task: %w", err)
			}

			imported += int64(len(batch))
			batch = batch[:0]

			if progress != nil {
				progress(imported, 0)
			}
		}
	}

	// Add remaining documents
	if len(batch) > 0 {
		task, err := d.index.AddDocuments(batch, &meilisearch.DocumentOptions{
			PrimaryKey: stringPtr("id"),
		})
		if err != nil {
			return fmt.Errorf("adding final documents: %w", err)
		}

		_, err = d.client.WaitForTask(task.TaskUID, 5*time.Minute)
		if err != nil {
			return fmt.Errorf("waiting for final task: %w", err)
		}

		imported += int64(len(batch))
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	stats, err := d.index.GetStats()
	if err != nil {
		return 0, fmt.Errorf("getting stats: %w", err)
	}
	return stats.NumberOfDocuments, nil
}

// Close is a no-op for MeiliSearch (HTTP client).
func (d *Driver) Close() error {
	return nil
}

// WaitForService waits for MeiliSearch to be ready.
func WaitForService(ctx context.Context, host string, timeout time.Duration) error {
	client := meilisearch.New(host)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if _, err := client.Health(); err == nil {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("meilisearch not ready after %v", timeout)
}

// IsServiceAvailable checks if MeiliSearch is reachable.
func IsServiceAvailable(host string) bool {
	if host == "" {
		host = DefaultHost
	}
	client := meilisearch.New(host)
	_, err := client.Health()
	return err == nil
}

// NewWithEnv creates a driver using environment variables.
func NewWithEnv(cfg fineweb.DriverConfig) (*Driver, error) {
	if cfg.Options == nil {
		cfg.Options = make(map[string]any)
	}
	if host := os.Getenv("MEILISEARCH_URL"); host != "" {
		cfg.Options["host"] = host
	}
	if apiKey := os.Getenv("MEILISEARCH_API_KEY"); apiKey != "" {
		cfg.Options["api_key"] = apiKey
	}
	return New(cfg)
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
