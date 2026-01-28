// Package elasticsearch provides an Elasticsearch-based driver for fineweb full-text search.
package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func init() {
	fineweb.Register("elasticsearch", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// DefaultHost is the default Elasticsearch address.
const DefaultHost = "http://localhost:9201"

// DefaultIndexName is the default index name.
const DefaultIndexName = "fineweb"

// Driver implements the fineweb.Driver interface using Elasticsearch.
type Driver struct {
	client    *elasticsearch.Client
	indexName string
	host      string
	language  string
}

// ESDocument is the document structure for Elasticsearch.
type ESDocument struct {
	ID            string  `json:"id"`
	URL           string  `json:"url"`
	Text          string  `json:"text"`
	Dump          string  `json:"dump"`
	Date          string  `json:"date"`
	Language      string  `json:"language"`
	LanguageScore float64 `json:"language_score"`
}

// New creates a new Elasticsearch driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	host := cfg.GetString("host", DefaultHost)

	indexName := cfg.GetString("index", DefaultIndexName)
	if cfg.Language != "" {
		indexName = strings.ToLower(cfg.Language)
	}

	esCfg := elasticsearch.Config{
		Addresses: []string{host},
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("creating elasticsearch client: %w", err)
	}

	d := &Driver{
		client:    client,
		indexName: indexName,
		host:      host,
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
	res, err := d.client.Indices.Exists([]string{d.indexName})
	if err != nil {
		return fmt.Errorf("checking index existence: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		return nil // Index exists
	}

	// Create index with optimized settings for BM25 text search
	mapping := map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   1,
			"number_of_replicas": 0,
			"index": map[string]interface{}{
				"refresh_interval": "30s", // Less frequent refresh during indexing
			},
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "keyword",
				},
				"url": map[string]interface{}{
					"type": "keyword",
				},
				"text": map[string]interface{}{
					"type": "text",
				},
				"dump": map[string]interface{}{
					"type": "keyword",
				},
				"date": map[string]interface{}{
					"type": "keyword",
				},
				"language": map[string]interface{}{
					"type": "keyword",
				},
				"language_score": map[string]interface{}{
					"type": "float",
				},
			},
		},
	}

	body, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("marshaling mapping: %w", err)
	}

	res, err = d.client.Indices.Create(
		d.indexName,
		d.client.Indices.Create.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to create index: %s", res.String())
	}

	return nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "elasticsearch"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "elasticsearch",
		Description: "Elasticsearch with BM25 scoring, distributed search, and rich query DSL",
		Features:    []string{"bm25", "distributed", "aggregations", "filters", "fuzzy-search"},
		External:    true,
	}
}

// Search performs full-text search using match query with BM25 scoring.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Build Elasticsearch search query with match on text field (uses BM25 by default)
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"text": query,
			},
		},
		"from": offset,
		"size": limit,
		"_source": []string{
			"id", "url", "text", "dump", "date", "language", "language_score",
		},
	}

	body, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("marshaling search query: %w", err)
	}

	res, err := d.client.Search(
		d.client.Search.WithContext(ctx),
		d.client.Search.WithIndex(d.indexName),
		d.client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("executing search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search failed: %s", res.String())
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

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
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
		Method:    "elasticsearch",
		Total:     result.Hits.Total.Value,
	}, nil
}

// Import ingests documents from an iterator using the bulk API.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	batchSize := 1000
	batch := make([]ESDocument, 0, batchSize)
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

		batch = append(batch, ESDocument{
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

	// Refresh index to make documents searchable
	_, err := d.client.Indices.Refresh(d.client.Indices.Refresh.WithIndex(d.indexName))
	if err != nil {
		return fmt.Errorf("refreshing index: %w", err)
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

func (d *Driver) bulkIndex(ctx context.Context, docs []ESDocument) error {
	var buf bytes.Buffer

	for _, doc := range docs {
		// Action line
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": d.indexName,
				"_id":    doc.ID,
			},
		}
		if err := json.NewEncoder(&buf).Encode(meta); err != nil {
			return fmt.Errorf("encoding bulk meta: %w", err)
		}

		// Document line
		if err := json.NewEncoder(&buf).Encode(doc); err != nil {
			return fmt.Errorf("encoding document: %w", err)
		}
	}

	req := esapi.BulkRequest{
		Body:    &buf,
		Refresh: "false", // Don't refresh after each bulk request for better performance
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return fmt.Errorf("executing bulk request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk index failed: %s", res.String())
	}

	// Check for item-level errors
	var bulkResponse struct {
		Errors bool `json:"errors"`
		Items  []struct {
			Index struct {
				Error struct {
					Type   string `json:"type"`
					Reason string `json:"reason"`
				} `json:"error"`
			} `json:"index"`
		} `json:"items"`
	}

	if err := json.NewDecoder(res.Body).Decode(&bulkResponse); err != nil {
		return fmt.Errorf("decoding bulk response: %w", err)
	}

	if bulkResponse.Errors {
		// Find first error for reporting
		for _, item := range bulkResponse.Items {
			if item.Index.Error.Type != "" {
				return fmt.Errorf("bulk index error: %s - %s",
					item.Index.Error.Type, item.Index.Error.Reason)
			}
		}
	}

	return nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	res, err := d.client.Count(
		d.client.Count.WithContext(ctx),
		d.client.Count.WithIndex(d.indexName),
	)
	if err != nil {
		return 0, fmt.Errorf("counting documents: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return 0, fmt.Errorf("count failed: %s", res.String())
	}

	var result struct {
		Count int64 `json:"count"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decoding count response: %w", err)
	}

	return result.Count, nil
}

// Close is a no-op for Elasticsearch (HTTP client).
func (d *Driver) Close() error {
	return nil
}

// WaitForService waits for Elasticsearch to be ready.
func WaitForService(ctx context.Context, host string, timeout time.Duration) error {
	cfg := elasticsearch.Config{
		Addresses: []string{host},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		res, err := client.Cluster.Health(
			client.Cluster.Health.WithContext(ctx),
			client.Cluster.Health.WithWaitForStatus("yellow"),
			client.Cluster.Health.WithTimeout(time.Second),
		)
		if err == nil && !res.IsError() {
			res.Body.Close()
			return nil
		}
		if res != nil {
			res.Body.Close()
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("elasticsearch not ready after %v", timeout)
}

// IsServiceAvailable checks if Elasticsearch is reachable.
func IsServiceAvailable(host string) bool {
	if host == "" {
		host = DefaultHost
	}

	cfg := elasticsearch.Config{
		Addresses: []string{host},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return false
	}

	res, err := client.Ping()
	if err != nil {
		return false
	}
	defer res.Body.Close()

	return !res.IsError()
}

// NewWithEnv creates a driver using environment variables.
func NewWithEnv(cfg fineweb.DriverConfig) (*Driver, error) {
	if cfg.Options == nil {
		cfg.Options = make(map[string]any)
	}
	if host := os.Getenv("ELASTICSEARCH_URL"); host != "" {
		cfg.Options["host"] = host
	}
	// Also check for comma-separated hosts
	if hosts := os.Getenv("ELASTICSEARCH_HOSTS"); hosts != "" {
		// Use the first host
		cfg.Options["host"] = strings.Split(hosts, ",")[0]
	}
	return New(cfg)
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
