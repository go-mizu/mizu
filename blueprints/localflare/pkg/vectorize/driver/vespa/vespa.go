// Package vespa provides a Vespa vector database driver for the vectorize package.
// Import this package to register the "vespa" driver.
// Vespa is a production-ready search and vector database.
package vespa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
)

func init() {
	driver.Register("vespa", &Driver{})
}

// Driver implements vectorize.Driver for Vespa.
type Driver struct{}

// Open creates a new Vespa connection.
// DSN format: http://host:port (e.g., "http://localhost:8080")
// Can also include config server: http://host:8080,http://host:19071
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		dsn = "http://localhost:8080"
	}

	// Parse DSN - format: query_endpoint,config_endpoint or just query_endpoint
	parts := strings.Split(dsn, ",")
	queryEndpoint := strings.TrimRight(parts[0], "/")
	configEndpoint := queryEndpoint
	if len(parts) > 1 {
		configEndpoint = strings.TrimRight(parts[1], "/")
	} else {
		// Default config server port
		if strings.Contains(queryEndpoint, ":8080") {
			configEndpoint = strings.Replace(queryEndpoint, ":8080", ":19071", 1)
		}
	}

	return &DB{
		queryEndpoint:  queryEndpoint,
		configEndpoint: configEndpoint,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		indexes: make(map[string]*indexInfo),
	}, nil
}

// indexInfo stores index metadata.
type indexInfo struct {
	dimensions  int
	metric      vectorize.DistanceMetric
	description string
	vectorCount int64
	createdAt   time.Time
	deployed    bool
}

// DB implements vectorize.DB for Vespa.
type DB struct {
	mu             sync.RWMutex
	queryEndpoint  string
	configEndpoint string
	client         *http.Client
	indexes        map[string]*indexInfo
}

// schemaName is the fixed Vespa schema name (must be pre-deployed).
const schemaName = "vectors"

// CreateIndex creates a virtual index (Vespa schemas must be pre-deployed).
// We use ID prefixes to separate different "indexes" within the same schema.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.indexes[index.Name]; exists {
		return vectorize.ErrIndexExists
	}

	// Store index info locally
	db.indexes[index.Name] = &indexInfo{
		dimensions:  index.Dimensions,
		metric:      index.Metric,
		description: index.Description,
		createdAt:   time.Now(),
		deployed:    true, // Schema is pre-deployed
	}

	return nil
}

// makeDocID creates a prefixed document ID for index separation.
func makeDocID(indexName, id string) string {
	return indexName + ":" + id
}

// parseDocID extracts the original ID from a prefixed document ID.
func parseDocID(prefixedID, indexName string) string {
	prefix := indexName + ":"
	if len(prefixedID) > len(prefix) && prefixedID[:len(prefix)] == prefix {
		return prefixedID[len(prefix):]
	}
	return prefixedID
}


// GetIndex retrieves index information.
func (db *DB) GetIndex(ctx context.Context, name string) (*vectorize.Index, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	info, ok := db.indexes[name]
	if !ok {
		return nil, vectorize.ErrIndexNotFound
	}

	return &vectorize.Index{
		Name:        name,
		Dimensions:  info.dimensions,
		Metric:      info.metric,
		Description: info.description,
		VectorCount: info.vectorCount,
		CreatedAt:   info.createdAt,
	}, nil
}

// ListIndexes returns all indexes.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var indexes []*vectorize.Index
	for name, info := range db.indexes {
		indexes = append(indexes, &vectorize.Index{
			Name:        name,
			Dimensions:  info.dimensions,
			Metric:      info.metric,
			Description: info.description,
			VectorCount: info.vectorCount,
			CreatedAt:   info.createdAt,
		})
	}

	return indexes, nil
}

// DeleteIndex removes an index.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, ok := db.indexes[name]; !ok {
		return vectorize.ErrIndexNotFound
	}

	// Delete all documents with the index prefix (using selection query)
	// Note: This is a best-effort cleanup; Vespa selection syntax may vary
	deleteURL := fmt.Sprintf("%s/document/v1/%s/%s/docid?selection=id.user=%%27%s%%3A*%%27&cluster=content",
		db.queryEndpoint, schemaName, schemaName, name)

	req, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
	if err != nil {
		return err
	}

	resp, err := db.client.Do(req)
	if err != nil {
		// Ignore errors - documents might not exist
	} else {
		resp.Body.Close()
	}

	delete(db.indexes, name)
	return nil
}

// Insert adds vectors to Vespa.
func (db *DB) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	db.mu.RLock()
	info, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return vectorize.ErrIndexNotFound
	}

	for _, v := range vectors {
		if len(v.Values) != info.dimensions {
			return vectorize.ErrDimensionMismatch
		}

		// Create document with prefixed ID
		docID := makeDocID(indexName, v.ID)
		metadataJSON, _ := json.Marshal(v.Metadata)

		doc := map[string]any{
			"fields": map[string]any{
				"id":        docID,
				"embedding": map[string]any{"values": v.Values},
				"namespace": v.Namespace,
				"metadata":  string(metadataJSON),
			},
		}

		body, err := json.Marshal(doc)
		if err != nil {
			return err
		}

		// PUT document using fixed schema name
		putURL := fmt.Sprintf("%s/document/v1/%s/%s/docid/%s",
			db.queryEndpoint, schemaName, schemaName, url.PathEscape(docID))

		req, err := http.NewRequestWithContext(ctx, "POST", putURL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := db.client.Do(req)
		if err != nil {
			return fmt.Errorf("insert failed for %s: %w", v.ID, err)
		}
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			return fmt.Errorf("insert failed for %s: status %d", v.ID, resp.StatusCode)
		}
	}

	// Update vector count
	db.mu.Lock()
	info.vectorCount += int64(len(vectors))
	db.mu.Unlock()

	return nil
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	// Vespa's PUT is an upsert
	return db.Insert(ctx, indexName, vectors)
}

// Search finds similar vectors using nearest neighbor search.
func (db *DB) Search(ctx context.Context, indexName string, vector []float32, opts *vectorize.SearchOptions) ([]*vectorize.Match, error) {
	if opts == nil {
		opts = &vectorize.SearchOptions{TopK: 10}
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	db.mu.RLock()
	info, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return nil, vectorize.ErrIndexNotFound
	}

	if len(vector) != info.dimensions {
		return nil, vectorize.ErrDimensionMismatch
	}

	// Build YQL query using fixed schema name
	// Filter by ID prefix to only match vectors for this "index"
	prefix := indexName + ":"
	yql := fmt.Sprintf("select * from %s where {targetHits:%d}nearestNeighbor(embedding,q) and id matches \"%s*\"",
		schemaName, opts.TopK*2, prefix) // Request more results to account for prefix filtering

	if opts.Namespace != "" {
		yql += fmt.Sprintf(" and namespace contains \"%s\"", opts.Namespace)
	}

	// Build query tensor
	tensorStr := fmt.Sprintf("[%s]", floatsToString(vector))

	// Build query request
	queryParams := url.Values{}
	queryParams.Set("yql", yql)
	queryParams.Set("input.query(q)", tensorStr)
	queryParams.Set("hits", fmt.Sprintf("%d", opts.TopK*2))
	queryParams.Set("ranking.profile", "default")

	searchURL := fmt.Sprintf("%s/search/?%s", db.queryEndpoint, queryParams.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := db.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Root struct {
			Children []struct {
				ID        string  `json:"id"`
				Relevance float32 `json:"relevance"`
				Fields    struct {
					ID        string `json:"id"`
					Namespace string `json:"namespace"`
					Metadata  string `json:"metadata"`
					Embedding struct {
						Values []float32 `json:"values"`
					} `json:"embedding"`
				} `json:"fields"`
			} `json:"children"`
		} `json:"root"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	var matches []*vectorize.Match
	for _, child := range result.Root.Children {
		// Filter by prefix and extract original ID
		if len(child.Fields.ID) <= len(prefix) || child.Fields.ID[:len(prefix)] != prefix {
			continue
		}

		score := child.Relevance
		if opts.ScoreThreshold > 0 && score < opts.ScoreThreshold {
			continue
		}

		match := &vectorize.Match{
			ID:    parseDocID(child.Fields.ID, indexName),
			Score: score,
		}

		if opts.ReturnValues {
			match.Values = child.Fields.Embedding.Values
		}

		if opts.ReturnMetadata && child.Fields.Metadata != "" {
			var metadata map[string]any
			json.Unmarshal([]byte(child.Fields.Metadata), &metadata)
			match.Metadata = metadata
		}

		matches = append(matches, match)
		if len(matches) >= opts.TopK {
			break
		}
	}

	return matches, nil
}

// Get retrieves vectors by IDs.
func (db *DB) Get(ctx context.Context, indexName string, ids []string) ([]*vectorize.Vector, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	db.mu.RLock()
	_, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return nil, vectorize.ErrIndexNotFound
	}

	var vectors []*vectorize.Vector
	for _, id := range ids {
		docID := makeDocID(indexName, id)
		getURL := fmt.Sprintf("%s/document/v1/%s/%s/docid/%s",
			db.queryEndpoint, schemaName, schemaName, url.PathEscape(docID))

		req, err := http.NewRequestWithContext(ctx, "GET", getURL, nil)
		if err != nil {
			continue
		}

		resp, err := db.client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode >= 400 {
			resp.Body.Close()
			continue
		}

		var result struct {
			Fields struct {
				ID        string `json:"id"`
				Namespace string `json:"namespace"`
				Metadata  string `json:"metadata"`
				Embedding struct {
					Values []float32 `json:"values"`
				} `json:"embedding"`
			} `json:"fields"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		vec := &vectorize.Vector{
			ID:        id, // Return original ID, not the prefixed one
			Values:    result.Fields.Embedding.Values,
			Namespace: result.Fields.Namespace,
		}

		if result.Fields.Metadata != "" {
			json.Unmarshal([]byte(result.Fields.Metadata), &vec.Metadata)
		}

		vectors = append(vectors, vec)
	}

	return vectors, nil
}

// Delete removes vectors by IDs.
func (db *DB) Delete(ctx context.Context, indexName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	db.mu.RLock()
	info, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return vectorize.ErrIndexNotFound
	}

	deleted := 0
	for _, id := range ids {
		docID := makeDocID(indexName, id)
		deleteURL := fmt.Sprintf("%s/document/v1/%s/%s/docid/%s",
			db.queryEndpoint, schemaName, schemaName, url.PathEscape(docID))

		req, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
		if err != nil {
			continue
		}

		resp, err := db.client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode < 400 {
			deleted++
		}
	}

	// Update vector count
	db.mu.Lock()
	info.vectorCount -= int64(deleted)
	if info.vectorCount < 0 {
		info.vectorCount = 0
	}
	db.mu.Unlock()

	return nil
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", db.queryEndpoint+"/state/v1/health", nil)
	if err != nil {
		return err
	}

	resp, err := db.client.Do(req)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ping failed: status %d", resp.StatusCode)
	}

	return nil
}

// Close releases resources.
func (db *DB) Close() error {
	db.client.CloseIdleConnections()
	return nil
}

// floatsToString converts a float32 slice to a comma-separated string.
func floatsToString(values []float32) string {
	strs := make([]string, len(values))
	for i, v := range values {
		strs[i] = fmt.Sprintf("%f", v)
	}
	return strings.Join(strs, ",")
}
