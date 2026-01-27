// Package zinc provides a Zinc-based driver for fineweb full-text search.
package zinc

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
	fineweb.Register("zinc", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// DefaultHost is the default Zinc address.
const DefaultHost = "http://localhost:4080"

// DefaultIndex is the default index name.
const DefaultIndex = "fineweb"

// Driver implements the fineweb.Driver interface using Zinc.
type Driver struct {
	client    *http.Client
	host      string
	username  string
	password  string
	indexName string
	language  string
}

// ZincDocument is the document structure for Zinc.
type ZincDocument struct {
	ID            string  `json:"id"`
	URL           string  `json:"url"`
	Text          string  `json:"text"`
	Dump          string  `json:"dump"`
	Date          string  `json:"date"`
	Language      string  `json:"language"`
	LanguageScore float64 `json:"language_score"`
}

// New creates a new Zinc driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	host := cfg.GetString("host", DefaultHost)
	username := cfg.GetString("username", "admin")
	password := cfg.GetString("password", "admin123")

	indexName := cfg.GetString("index", DefaultIndex)
	if cfg.Language != "" {
		indexName = cfg.Language
	}

	d := &Driver{
		client:    &http.Client{Timeout: 30 * time.Second},
		host:      host,
		username:  username,
		password:  password,
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
	// Check if index exists
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/index/%s", d.host, d.indexName), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(d.username, d.password)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("checking index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil // Index exists
	}

	// Create index
	indexSettings := map[string]interface{}{
		"name":         d.indexName,
		"storage_type": "disk",
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id":             map[string]string{"type": "keyword"},
				"url":            map[string]string{"type": "text"},
				"text":           map[string]string{"type": "text", "analyzer": "standard"},
				"dump":           map[string]string{"type": "keyword", "index": "false"},
				"date":           map[string]string{"type": "keyword"},
				"language":       map[string]string{"type": "keyword"},
				"language_score": map[string]string{"type": "numeric"},
			},
		},
	}

	body, _ := json.Marshal(indexSettings)
	req, err = http.NewRequest("POST", fmt.Sprintf("%s/api/index", d.host), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(d.username, d.password)

	resp, err = d.client.Do(req)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create index: %s", string(body))
	}

	return nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "zinc"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "zinc",
		Description: "Zinc search engine (Elasticsearch compatible)",
		Features:    []string{"elasticsearch-compatible", "lightweight", "go-native"},
		External:    true,
	}
}

// Search performs full-text search.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Build Zinc search query (Elasticsearch compatible)
	searchQuery := map[string]interface{}{
		"search_type": "match",
		"query": map[string]interface{}{
			"term":  query,
			"field": "text",
		},
		"from":       offset,
		"max_results": limit,
		"_source":    []string{"id", "url", "text", "dump", "date", "language", "language_score"},
	}

	body, _ := json.Marshal(searchQuery)
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/%s/_search", d.host, d.indexName),
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(d.username, d.password)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: %s", string(respBody))
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string  `json:"_id"`
				Score  float64 `json:"_score"`
				Source struct {
					ID            string  `json:"id"`
					URL           string  `json:"url"`
					Text          string  `json:"text"`
					Dump          string  `json:"dump"`
					Date          string  `json:"date"`
					Language      string  `json:"language"`
					LanguageScore float64 `json:"language_score"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	docs := make([]fineweb.Document, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		docs = append(docs, fineweb.Document{
			ID:            hit.Source.ID,
			URL:           hit.Source.URL,
			Text:          hit.Source.Text,
			Dump:          hit.Source.Dump,
			Date:          hit.Source.Date,
			Language:      hit.Source.Language,
			LanguageScore: hit.Source.LanguageScore,
			Score:         hit.Score,
		})
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "zinc",
		Total:     result.Hits.Total.Value,
	}, nil
}

// Import ingests documents from an iterator.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	batchSize := 1000
	batch := make([]ZincDocument, 0, batchSize)
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

		batch = append(batch, ZincDocument{
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

func (d *Driver) bulkIndex(ctx context.Context, docs []ZincDocument) error {
	// Zinc bulk API format
	bulkRequest := map[string]interface{}{
		"index":   d.indexName,
		"records": docs,
	}

	body, _ := json.Marshal(bulkRequest)
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/_bulkv2", d.host),
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(d.username, d.password)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bulk index failed: %s", string(respBody))
	}

	return nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/index/%s", d.host, d.indexName), nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(d.username, d.password)

	resp, err := d.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("getting index info: %w", err)
	}
	defer resp.Body.Close()

	var info struct {
		Stats struct {
			DocNum int64 `json:"doc_num"`
		} `json:"stats"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return 0, fmt.Errorf("decoding response: %w", err)
	}

	return info.Stats.DocNum, nil
}

// Close is a no-op for Zinc (HTTP client).
func (d *Driver) Close() error {
	return nil
}

// WaitForService waits for Zinc to be ready.
func WaitForService(ctx context.Context, host string, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, _ := http.NewRequest("GET", host+"/healthz", nil)
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

	return fmt.Errorf("zinc not ready after %v", timeout)
}

// IsServiceAvailable checks if Zinc is reachable.
func IsServiceAvailable(host string) bool {
	if host == "" {
		host = DefaultHost
	}
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET", host+"/healthz", nil)
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
	if host := os.Getenv("ZINC_URL"); host != "" {
		cfg.Options["host"] = host
	}
	if user := os.Getenv("ZINC_USER"); user != "" {
		cfg.Options["username"] = user
	}
	if pass := os.Getenv("ZINC_PASSWORD"); pass != "" {
		cfg.Options["password"] = pass
	}
	return New(cfg)
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
