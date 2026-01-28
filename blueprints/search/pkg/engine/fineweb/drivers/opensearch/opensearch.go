// Package opensearch provides an OpenSearch-based driver for fineweb full-text search.
package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func init() {
	fineweb.Register("opensearch", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// DefaultHost is the default OpenSearch address.
const DefaultHost = "http://localhost:9200"

// DefaultIndexName is the default index name.
const DefaultIndexName = "fineweb"

// Driver implements the fineweb.Driver interface using OpenSearch.
type Driver struct {
	client    *opensearch.Client
	indexName string
	host      string
	username  string
	password  string
	language  string
}

// Document is the document structure for OpenSearch.
type Document struct {
	ID            string  `json:"id"`
	URL           string  `json:"url"`
	Text          string  `json:"text"`
	Dump          string  `json:"dump"`
	Date          string  `json:"date"`
	Language      string  `json:"language"`
	LanguageScore float64 `json:"language_score"`
}

// New creates a new OpenSearch driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	host := cfg.GetString("host", DefaultHost)
	username := cfg.GetString("username", "")
	password := cfg.GetString("password", "")

	indexName := cfg.GetString("index", DefaultIndexName)
	if cfg.Language != "" {
		indexName = strings.ToLower(cfg.Language)
	}

	// Configure OpenSearch client
	osConfig := opensearch.Config{
		Addresses: []string{host},
	}
	if username != "" && password != "" {
		osConfig.Username = username
		osConfig.Password = password
	}

	client, err := opensearch.NewClient(osConfig)
	if err != nil {
		return nil, fmt.Errorf("creating opensearch client: %w", err)
	}

	d := &Driver{
		client:    client,
		indexName: indexName,
		host:      host,
		username:  username,
		password:  password,
		language:  cfg.Language,
	}

	// Create or verify index exists
	if err := d.ensureIndex(context.Background()); err != nil {
		return nil, fmt.Errorf("ensuring index: %w", err)
	}

	return d, nil
}

func (d *Driver) ensureIndex(ctx context.Context) error {
	// Check if index exists
	existsReq := opensearchapi.IndicesExistsRequest{
		Index: []string{d.indexName},
	}
	existsRes, err := existsReq.Do(ctx, d.client)
	if err != nil {
		return fmt.Errorf("checking index existence: %w", err)
	}
	defer existsRes.Body.Close()

	if existsRes.StatusCode == 200 {
		// Index already exists
		return nil
	}

	// Create index with mapping optimized for BM25 search
	mapping := map[string]any{
		"settings": map[string]any{
			"number_of_shards":   1,
			"number_of_replicas": 0,
			"index": map[string]any{
				"refresh_interval": "30s", // Delay refresh for faster indexing
			},
		},
		"mappings": map[string]any{
			"properties": map[string]any{
				"id": map[string]any{
					"type": "keyword",
				},
				"url": map[string]any{
					"type": "keyword",
				},
				"text": map[string]any{
					"type": "text",
				},
				"dump": map[string]any{
					"type": "keyword",
				},
				"date": map[string]any{
					"type": "keyword",
				},
				"language": map[string]any{
					"type": "keyword",
				},
				"language_score": map[string]any{
					"type": "float",
				},
			},
		},
	}

	mappingJSON, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("marshaling index mapping: %w", err)
	}

	createReq := opensearchapi.IndicesCreateRequest{
		Index: d.indexName,
		Body:  bytes.NewReader(mappingJSON),
	}
	createRes, err := createReq.Do(ctx, d.client)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		return fmt.Errorf("index creation failed: %s", createRes.String())
	}

	return nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "opensearch"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "opensearch",
		Description: "OpenSearch with BM25 scoring, full-text search, and distributed capabilities",
		Features:    []string{"bm25", "full-text", "distributed", "aggregations", "filters"},
		External:    true,
	}
}

// Search performs full-text search using BM25 scoring.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Build match query on text field
	searchQuery := map[string]any{
		"query": map[string]any{
			"match": map[string]any{
				"text": query,
			},
		},
		"from": offset,
		"size": limit,
	}

	queryJSON, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("marshaling search query: %w", err)
	}

	searchReq := opensearchapi.SearchRequest{
		Index: []string{d.indexName},
		Body:  bytes.NewReader(queryJSON),
	}
	searchRes, err := searchReq.Do(ctx, d.client)
	if err != nil {
		return nil, fmt.Errorf("executing search: %w", err)
	}
	defer searchRes.Body.Close()

	if searchRes.IsError() {
		return nil, fmt.Errorf("search failed: %s", searchRes.String())
	}

	// Parse response
	var result searchResponse
	if err := json.NewDecoder(searchRes.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding search response: %w", err)
	}

	// Convert results
	docs := make([]fineweb.Document, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		doc := fineweb.Document{
			ID:            hit.Source.ID,
			URL:           hit.Source.URL,
			Text:          hit.Source.Text,
			Dump:          hit.Source.Dump,
			Date:          hit.Source.Date,
			Language:      hit.Source.Language,
			LanguageScore: hit.Source.LanguageScore,
			Score:         hit.Score,
		}
		docs = append(docs, doc)
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "opensearch",
		Total:     result.Hits.Total.Value,
	}, nil
}

// searchResponse is the OpenSearch search response structure.
type searchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Score  float64  `json:"_score"`
			Source Document `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// Import ingests documents from an iterator using bulk API.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	const batchSize = 1000
	var batch []Document
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

	// Refresh index to make documents searchable
	refreshReq := opensearchapi.IndicesRefreshRequest{
		Index: []string{d.indexName},
	}
	refreshRes, err := refreshReq.Do(ctx, d.client)
	if err != nil {
		return fmt.Errorf("refreshing index: %w", err)
	}
	defer refreshRes.Body.Close()

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

// bulkIndex indexes a batch of documents using the bulk API.
func (d *Driver) bulkIndex(ctx context.Context, docs []Document) error {
	var buf bytes.Buffer

	for _, doc := range docs {
		// Action line
		action := map[string]any{
			"index": map[string]any{
				"_index": d.indexName,
				"_id":    doc.ID,
			},
		}
		actionJSON, err := json.Marshal(action)
		if err != nil {
			return fmt.Errorf("marshaling action: %w", err)
		}
		buf.Write(actionJSON)
		buf.WriteByte('\n')

		// Document line
		docJSON, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshaling document: %w", err)
		}
		buf.Write(docJSON)
		buf.WriteByte('\n')
	}

	bulkReq := opensearchapi.BulkRequest{
		Body: strings.NewReader(buf.String()),
	}
	bulkRes, err := bulkReq.Do(ctx, d.client)
	if err != nil {
		return fmt.Errorf("executing bulk request: %w", err)
	}
	defer bulkRes.Body.Close()

	if bulkRes.IsError() {
		return fmt.Errorf("bulk request failed: %s", bulkRes.String())
	}

	// Check for individual errors in the response
	var bulkResponse struct {
		Errors bool `json:"errors"`
		Items  []struct {
			Index struct {
				Error *struct {
					Type   string `json:"type"`
					Reason string `json:"reason"`
				} `json:"error,omitempty"`
			} `json:"index"`
		} `json:"items"`
	}
	if err := json.NewDecoder(bulkRes.Body).Decode(&bulkResponse); err != nil {
		return fmt.Errorf("decoding bulk response: %w", err)
	}

	if bulkResponse.Errors {
		// Find first error
		for _, item := range bulkResponse.Items {
			if item.Index.Error != nil {
				return fmt.Errorf("bulk index error: %s - %s", item.Index.Error.Type, item.Index.Error.Reason)
			}
		}
	}

	return nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	countReq := opensearchapi.CountRequest{
		Index: []string{d.indexName},
	}
	countRes, err := countReq.Do(ctx, d.client)
	if err != nil {
		return 0, fmt.Errorf("counting documents: %w", err)
	}
	defer countRes.Body.Close()

	if countRes.IsError() {
		return 0, fmt.Errorf("count failed: %s", countRes.String())
	}

	var result struct {
		Count int64 `json:"count"`
	}
	if err := json.NewDecoder(countRes.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decoding count response: %w", err)
	}

	return result.Count, nil
}

// Close is a no-op for OpenSearch (HTTP client).
func (d *Driver) Close() error {
	return nil
}

// WaitForService waits for OpenSearch to be ready.
func WaitForService(ctx context.Context, host string, timeout time.Duration) error {
	cfg := opensearch.Config{
		Addresses: []string{host},
	}
	client, err := opensearch.NewClient(cfg)
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
			client.Cluster.Health.WithTimeout(5*time.Second),
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

	return fmt.Errorf("opensearch not ready after %v", timeout)
}

// IsServiceAvailable checks if OpenSearch is reachable.
func IsServiceAvailable(host string) bool {
	if host == "" {
		host = DefaultHost
	}
	cfg := opensearch.Config{
		Addresses: []string{host},
	}
	client, err := opensearch.NewClient(cfg)
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
	if host := os.Getenv("OPENSEARCH_URL"); host != "" {
		cfg.Options["host"] = host
	}
	if username := os.Getenv("OPENSEARCH_USERNAME"); username != "" {
		cfg.Options["username"] = username
	}
	if password := os.Getenv("OPENSEARCH_PASSWORD"); password != "" {
		cfg.Options["password"] = password
	}
	return New(cfg)
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
