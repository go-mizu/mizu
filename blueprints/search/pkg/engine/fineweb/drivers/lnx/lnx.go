// Package lnx provides a Lnx (Tantivy REST server) driver for fineweb full-text search.
package lnx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"os"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func init() {
	fineweb.Register("lnx", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// DefaultHost is the default Lnx address.
const DefaultHost = "http://localhost:8000"

// DefaultIndex is the default index name.
const DefaultIndex = "fineweb"

// DefaultAPIKey is the default API key for Lnx.
const DefaultAPIKey = "fineweb-benchmark-key"

// Driver implements the fineweb.Driver interface using Lnx.
type Driver struct {
	client    *http.Client
	host      string
	apiKey    string
	indexName string
	language  string
}

// LnxDocument is the document structure for Lnx.
type LnxDocument struct {
	ID            string  `json:"id"`
	URL           string  `json:"url"`
	Text          string  `json:"text"`
	Dump          string  `json:"dump"`
	Date          string  `json:"date"`
	Language      string  `json:"language"`
	LanguageScore float64 `json:"language_score"`
}

// SchemaField defines a field in the Lnx index schema.
type SchemaField struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Stored bool   `json:"stored"`
}

// New creates a new Lnx driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	host := cfg.GetString("host", DefaultHost)
	apiKey := cfg.GetString("api_key", DefaultAPIKey)

	indexName := cfg.GetString("index", DefaultIndex)
	if cfg.Language != "" {
		indexName = cfg.Language
	}

	d := &Driver{
		client:    &http.Client{Timeout: 2 * time.Minute}, // Longer timeout for bulk ops
		host:      host,
		apiKey:    apiKey,
		indexName: indexName,
		language:  cfg.Language,
	}

	// Ensure index exists
	if err := d.ensureIndex(); err != nil {
		return nil, fmt.Errorf("ensuring index: %w", err)
	}

	return d, nil
}

func (d *Driver) ensureIndex() error {
	// Check if index exists by trying to get it
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/indexes/%s", d.host, d.indexName), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", d.apiKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("checking index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil // Index exists
	}

	// Create index with schema
	schema := []SchemaField{
		{Name: "id", Type: "string", Stored: true},
		{Name: "url", Type: "string", Stored: true},
		{Name: "text", Type: "text", Stored: true},
		{Name: "dump", Type: "string", Stored: true},
		{Name: "date", Type: "string", Stored: true},
		{Name: "language", Type: "string", Stored: true},
		{Name: "language_score", Type: "f64", Stored: true},
	}

	indexConfig := map[string]any{
		"name":   d.indexName,
		"schema": schema,
	}

	body, _ := json.Marshal(indexConfig)
	req, err = http.NewRequest("POST", fmt.Sprintf("%s/indexes", d.host), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", d.apiKey)

	resp, err = d.client.Do(req)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create index (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "lnx"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "lnx",
		Description: "Lnx search engine (Tantivy REST server)",
		Features:    []string{"tantivy-based", "rust-performance", "rest-api"},
		External:    true,
	}
}

// Search performs full-text search.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Build Lnx search query
	searchQuery := map[string]any{
		"query": query,
		"limit": limit,
	}

	body, _ := json.Marshal(searchQuery)
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/indexes/%s/search", d.host, d.indexName),
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", d.apiKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Hits []struct {
			Doc struct {
				ID            string  `json:"id"`
				URL           string  `json:"url"`
				Text          string  `json:"text"`
				Dump          string  `json:"dump"`
				Date          string  `json:"date"`
				Language      string  `json:"language"`
				LanguageScore float64 `json:"language_score"`
			} `json:"doc"`
			Score float64 `json:"score"`
		} `json:"hits"`
		Count int64 `json:"count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	docs := make([]fineweb.Document, 0, len(result.Hits))
	for _, hit := range result.Hits {
		docs = append(docs, fineweb.Document{
			ID:            hit.Doc.ID,
			URL:           hit.Doc.URL,
			Text:          hit.Doc.Text,
			Dump:          hit.Doc.Dump,
			Date:          hit.Doc.Date,
			Language:      hit.Doc.Language,
			LanguageScore: hit.Doc.LanguageScore,
			Score:         hit.Score,
		})
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "lnx",
		Total:     result.Count,
	}, nil
}

// Import ingests documents from an iterator.
// Uses large batches for high throughput.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	batchSize := 5000 // Larger batches for better throughput
	batch := make([]LnxDocument, 0, batchSize)
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

		batch = append(batch, LnxDocument{
			ID:            doc.ID,
			URL:           doc.URL,
			Text:          doc.Text,
			Dump:          doc.Dump,
			Date:          doc.Date,
			Language:      doc.Language,
			LanguageScore: doc.LanguageScore,
		})

		if len(batch) >= batchSize {
			if err := d.bulkIndex(ctx, batch); err != nil {
				return fmt.Errorf("bulk indexing: %w", err)
			}

			imported += int64(len(batch))
			batch = batch[:0]

			if progress != nil {
				progress(imported, 0)
			}
		}
	}

	// Index remaining documents
	if len(batch) > 0 {
		if err := d.bulkIndex(ctx, batch); err != nil {
			return fmt.Errorf("bulk indexing final batch: %w", err)
		}
		imported += int64(len(batch))
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

func (d *Driver) bulkIndex(ctx context.Context, docs []LnxDocument) error {
	body, _ := json.Marshal(docs)
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/indexes/%s/documents", d.host, d.indexName),
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", d.apiKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bulk index failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/indexes/%s", d.host, d.indexName), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", d.apiKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("getting index info: %w", err)
	}
	defer resp.Body.Close()

	var info struct {
		NumDocs int64 `json:"num_docs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return 0, fmt.Errorf("decoding response: %w", err)
	}

	return info.NumDocs, nil
}

// Close is a no-op for Lnx (HTTP client).
func (d *Driver) Close() error {
	return nil
}

// WaitForService waits for Lnx to be ready.
func WaitForService(ctx context.Context, host string, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, _ := http.NewRequest("GET", host+"/", nil)
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("lnx not ready after %v", timeout)
}

// IsServiceAvailable checks if Lnx is reachable.
func IsServiceAvailable(host string) bool {
	if host == "" {
		host = DefaultHost
	}
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET", host+"/", nil)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// NewWithEnv creates a driver using environment variables.
func NewWithEnv(cfg fineweb.DriverConfig) (*Driver, error) {
	if cfg.Options == nil {
		cfg.Options = make(map[string]any)
	}
	if host := os.Getenv("LNX_URL"); host != "" {
		cfg.Options["host"] = host
	}
	if apiKey := os.Getenv("LNX_API_KEY"); apiKey != "" {
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
