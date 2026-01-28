// Package quickwit provides a QuickWit-based driver for fineweb full-text search.
package quickwit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func init() {
	fineweb.Register("quickwit", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// DefaultHost is the default QuickWit address.
const DefaultHost = "http://localhost:7280"

// DefaultIndex is the default index name.
const DefaultIndex = "fineweb"

// Driver implements the fineweb.Driver interface using QuickWit.
type Driver struct {
	client    *http.Client
	host      string
	indexName string
	language  string
}

// Document is the document structure for QuickWit.
type Document struct {
	ID            string  `json:"id"`
	URL           string  `json:"url"`
	Text          string  `json:"text"`
	Dump          string  `json:"dump"`
	Date          string  `json:"date"`
	Language      string  `json:"language"`
	LanguageScore float64 `json:"language_score"`
}

// New creates a new QuickWit driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	host := cfg.GetString("host", DefaultHost)

	indexName := cfg.GetString("index", DefaultIndex)
	if cfg.Language != "" {
		indexName = strings.ToLower(strings.ReplaceAll(cfg.Language, "_", "-"))
	}

	d := &Driver{
		client:    &http.Client{Timeout: 2 * time.Minute}, // Longer timeout for bulk ops
		host:      host,
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
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/indexes/%s", d.host, d.indexName), nil)
	if err != nil {
		return err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("checking index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil // Index exists
	}

	// Create index with QuickWit JSON config
	indexConfig := map[string]any{
		"version":  "0.7",
		"index_id": d.indexName,
		"doc_mapping": map[string]any{
			"field_mappings": []map[string]any{
				{"name": "id", "type": "text", "fast": true, "stored": true},
				{"name": "url", "type": "text", "stored": true},
				{"name": "text", "type": "text", "tokenizer": "default", "stored": true, "record": "position"},
				{"name": "dump", "type": "text", "stored": true},
				{"name": "date", "type": "text", "stored": true},
				{"name": "language", "type": "text", "stored": true},
				{"name": "language_score", "type": "f64", "fast": true, "stored": true},
			},
			"mode": "lenient",
		},
		"search_settings": map[string]any{
			"default_search_fields": []string{"text"},
		},
		"indexing_settings": map[string]any{
			"commit_timeout_secs": 30,
		},
	}

	body, _ := json.Marshal(indexConfig)
	req, err = http.NewRequest("POST", fmt.Sprintf("%s/api/v1/indexes", d.host), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = d.client.Do(req)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		// Ignore "already exists" error
		if strings.Contains(string(respBody), "already exists") {
			return nil
		}
		return fmt.Errorf("failed to create index (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "quickwit"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "quickwit",
		Description: "QuickWit sub-second search on cloud storage",
		Features:    []string{"cloud-native", "sub-second-search", "schemaless", "s3-compatible"},
		External:    true,
	}
}

// Search performs full-text search.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Build QuickWit search query
	searchQuery := map[string]any{
		"query":        query,
		"max_hits":     limit,
		"start_offset": offset,
	}

	body, _ := json.Marshal(searchQuery)
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/v1/%s/search", d.host, d.indexName),
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

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
		NumHits           int64             `json:"num_hits"`
		Hits              []json.RawMessage `json:"hits"`
		ElapsedTimeMicros int64             `json:"elapsed_time_micros"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	docs := make([]fineweb.Document, 0, len(result.Hits))
	for i, hit := range result.Hits {
		var doc Document
		if err := json.Unmarshal(hit, &doc); err != nil {
			continue
		}
		docs = append(docs, fineweb.Document{
			ID:            doc.ID,
			URL:           doc.URL,
			Text:          doc.Text,
			Dump:          doc.Dump,
			Date:          doc.Date,
			Language:      doc.Language,
			LanguageScore: doc.LanguageScore,
			Score:         1.0 / float64(i+1), // QuickWit returns results in relevance order
		})
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "quickwit",
		Total:     result.NumHits,
	}, nil
}

// Import ingests documents from an iterator.
// Uses NDJSON format for bulk indexing.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	batchSize := 100 // Smaller batch size due to large document content
	batch := make([]Document, 0, batchSize)
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

		batch = append(batch, Document{
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

func (d *Driver) bulkIndex(ctx context.Context, docs []Document) error {
	// QuickWit expects NDJSON format for ingest
	var buf bytes.Buffer
	for _, doc := range docs {
		data, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshaling document: %w", err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/v1/%s/ingest", d.host, d.indexName),
		&buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bulk index failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/v1/indexes/%s", d.host, d.indexName), nil)
	if err != nil {
		return 0, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("getting index info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get index info (status %d)", resp.StatusCode)
	}

	var info struct {
		IndexConfig struct {
			IndexID string `json:"index_id"`
		} `json:"index_config"`
		// Try to get doc count from splits or search
	}

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return 0, fmt.Errorf("decoding response: %w", err)
	}

	// QuickWit doesn't expose doc count directly in index info,
	// so we do a count query using search with max_hits=0
	countQuery := map[string]any{
		"query":    "*",
		"max_hits": 0,
	}

	body, _ := json.Marshal(countQuery)
	req, err = http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/v1/%s/search", d.host, d.indexName),
		bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = d.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("counting documents: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		NumHits int64 `json:"num_hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decoding count response: %w", err)
	}

	return result.NumHits, nil
}

// Close is a no-op for QuickWit (HTTP client).
func (d *Driver) Close() error {
	return nil
}

// WaitForService waits for QuickWit to be ready.
func WaitForService(ctx context.Context, host string, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, _ := http.NewRequest("GET", host+"/health/livez", nil)
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

	return fmt.Errorf("quickwit not ready after %v", timeout)
}

// IsServiceAvailable checks if QuickWit is reachable.
func IsServiceAvailable(host string) bool {
	if host == "" {
		host = DefaultHost
	}
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET", host+"/health/livez", nil)
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
	if host := os.Getenv("QUICKWIT_URL"); host != "" {
		cfg.Options["host"] = host
	}
	return New(cfg)
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
