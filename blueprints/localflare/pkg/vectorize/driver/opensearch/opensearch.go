// Package opensearch provides an OpenSearch driver for the vectorize package.
// Import this package to register the "opensearch" driver.
// Note: This implementation uses HTTP requests directly for better compatibility.
package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
)

func init() {
	driver.Register("opensearch", &Driver{})
}

// Driver implements vectorize.Driver for OpenSearch.
type Driver struct{}

// Open creates a new OpenSearch connection.
// DSN format: http://host:port or https://host:port
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		return nil, vectorize.ErrInvalidDSN
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &DB{client: client, baseURL: dsn}, nil
}

// DB implements vectorize.DB for OpenSearch.
type DB struct {
	client  *http.Client
	baseURL string
}

const indexPrefix = "vectorize_"

func (db *DB) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := db.baseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return db.client.Do(req)
}

// CreateIndex creates a new OpenSearch index for vectors.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	indexName := indexPrefix + index.Name

	// Check if index exists
	res, err := db.doRequest(ctx, "HEAD", "/"+indexName, nil)
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode == 200 {
		return vectorize.ErrIndexExists
	}

	// Create index with vector mapping
	spaceType := "cosinesimil"
	switch index.Metric {
	case vectorize.Euclidean:
		spaceType = "l2"
	case vectorize.DotProduct:
		spaceType = "innerproduct"
	}

	mapping := map[string]interface{}{
		"settings": map[string]interface{}{
			"index": map[string]interface{}{
				"knn": true,
			},
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "keyword",
				},
				"namespace": map[string]interface{}{
					"type": "keyword",
				},
				"metadata": map[string]interface{}{
					"type": "object",
				},
				"embedding": map[string]interface{}{
					"type":       "knn_vector",
					"dimension":  index.Dimensions,
					"space_type": spaceType,
				},
			},
		},
	}

	res, err = db.doRequest(ctx, "PUT", "/"+indexName, mapping)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to create index: %s", string(body))
	}

	// Store metadata document
	metaDoc := map[string]interface{}{
		"_meta":        true,
		"dimensions":   index.Dimensions,
		"description":  index.Description,
		"created_at":   time.Now().Format(time.RFC3339),
		"is_meta_only": true,
	}

	res, err = db.doRequest(ctx, "PUT", "/"+indexName+"/_doc/_meta", metaDoc)
	if err != nil {
		return err
	}
	res.Body.Close()

	return nil
}

// GetIndex retrieves index information.
func (db *DB) GetIndex(ctx context.Context, name string) (*vectorize.Index, error) {
	indexName := indexPrefix + name

	// Check if index exists
	res, err := db.doRequest(ctx, "HEAD", "/"+indexName, nil)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	if res.StatusCode == 404 {
		return nil, vectorize.ErrIndexNotFound
	}

	idx := &vectorize.Index{
		Name:   name,
		Metric: vectorize.Cosine,
	}

	// Get metadata
	res, err = db.doRequest(ctx, "GET", "/"+indexName+"/_doc/_meta", nil)
	if err == nil && res.StatusCode == 200 {
		defer res.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(res.Body).Decode(&result)
		if source, ok := result["_source"].(map[string]interface{}); ok {
			if dims, ok := source["dimensions"].(float64); ok {
				idx.Dimensions = int(dims)
			}
			if desc, ok := source["description"].(string); ok {
				idx.Description = desc
			}
			if createdAt, ok := source["created_at"].(string); ok {
				if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
					idx.CreatedAt = t
				}
			}
		}
	} else if res != nil {
		res.Body.Close()
	}

	// Get count
	res, err = db.doRequest(ctx, "GET", "/"+indexName+"/_count", nil)
	if err == nil && res.StatusCode == 200 {
		defer res.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(res.Body).Decode(&result)
		if count, ok := result["count"].(float64); ok {
			idx.VectorCount = int64(count) - 1 // Subtract metadata doc
		}
	} else if res != nil {
		res.Body.Close()
	}

	return idx, nil
}

// ListIndexes returns all vector indexes.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	res, err := db.doRequest(ctx, "GET", "/_cat/indices/"+indexPrefix+"*?format=json", nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var indices []map[string]interface{}
	json.NewDecoder(res.Body).Decode(&indices)

	indexes := make([]*vectorize.Index, 0)
	for _, idx := range indices {
		if name, ok := idx["index"].(string); ok {
			name = strings.TrimPrefix(name, indexPrefix)
			index, err := db.GetIndex(ctx, name)
			if err != nil {
				continue
			}
			indexes = append(indexes, index)
		}
	}

	return indexes, nil
}

// DeleteIndex removes an index.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	indexName := indexPrefix + name

	res, err := db.doRequest(ctx, "DELETE", "/"+indexName, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return vectorize.ErrIndexNotFound
	}

	return nil
}

// Insert adds vectors to an index.
func (db *DB) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	osIndexName := indexPrefix + indexName

	// Use bulk API
	var buf bytes.Buffer
	for _, v := range vectors {
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": osIndexName,
				"_id":    v.ID,
			},
		}
		doc := map[string]interface{}{
			"id":        v.ID,
			"namespace": v.Namespace,
			"metadata":  v.Metadata,
			"embedding": v.Values,
		}

		actionJSON, _ := json.Marshal(action)
		docJSON, _ := json.Marshal(doc)
		buf.Write(actionJSON)
		buf.WriteByte('\n')
		buf.Write(docJSON)
		buf.WriteByte('\n')
	}

	req, err := http.NewRequestWithContext(ctx, "POST", db.baseURL+"/_bulk", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")

	res, err := db.client.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()

	return nil
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	return db.Insert(ctx, indexName, vectors)
}

// Search finds similar vectors using kNN.
func (db *DB) Search(ctx context.Context, indexName string, vector []float32, opts *vectorize.SearchOptions) ([]*vectorize.Match, error) {
	if opts == nil {
		opts = &vectorize.SearchOptions{TopK: 10}
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	osIndexName := indexPrefix + indexName

	query := map[string]interface{}{
		"size": opts.TopK,
		"query": map[string]interface{}{
			"knn": map[string]interface{}{
				"embedding": map[string]interface{}{
					"vector": vector,
					"k":      opts.TopK,
				},
			},
		},
	}

	// Add namespace filter
	if opts.Namespace != "" {
		query["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"knn": map[string]interface{}{
							"embedding": map[string]interface{}{
								"vector": vector,
								"k":      opts.TopK,
							},
						},
					},
				},
				"filter": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"namespace": opts.Namespace,
						},
					},
				},
			},
		}
	}

	res, err := db.doRequest(ctx, "POST", "/"+osIndexName+"/_search", query)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	matches := make([]*vectorize.Match, 0)
	if hits, ok := result["hits"].(map[string]interface{}); ok {
		if hitList, ok := hits["hits"].([]interface{}); ok {
			for _, h := range hitList {
				hit := h.(map[string]interface{})
				match := &vectorize.Match{
					ID: hit["_id"].(string),
				}

				if score, ok := hit["_score"].(float64); ok {
					match.Score = float32(score)
				}

				if opts.ReturnMetadata || opts.ReturnValues {
					if source, ok := hit["_source"].(map[string]interface{}); ok {
						if opts.ReturnMetadata {
							if meta, ok := source["metadata"].(map[string]interface{}); ok {
								match.Metadata = meta
							}
						}
						if opts.ReturnValues {
							if emb, ok := source["embedding"].([]interface{}); ok {
								match.Values = make([]float32, len(emb))
								for i, v := range emb {
									match.Values[i] = float32(v.(float64))
								}
							}
						}
					}
				}

				if opts.ScoreThreshold > 0 && match.Score < opts.ScoreThreshold {
					continue
				}

				matches = append(matches, match)
			}
		}
	}

	return matches, nil
}

// Get retrieves vectors by IDs.
func (db *DB) Get(ctx context.Context, indexName string, ids []string) ([]*vectorize.Vector, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	osIndexName := indexPrefix + indexName

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"ids": map[string]interface{}{
				"values": ids,
			},
		},
	}

	res, err := db.doRequest(ctx, "POST", "/"+osIndexName+"/_search", query)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	vectors := make([]*vectorize.Vector, 0)
	if hits, ok := result["hits"].(map[string]interface{}); ok {
		if hitList, ok := hits["hits"].([]interface{}); ok {
			for _, h := range hitList {
				hit := h.(map[string]interface{})
				vec := &vectorize.Vector{
					ID: hit["_id"].(string),
				}

				if source, ok := hit["_source"].(map[string]interface{}); ok {
					if ns, ok := source["namespace"].(string); ok {
						vec.Namespace = ns
					}
					if meta, ok := source["metadata"].(map[string]interface{}); ok {
						vec.Metadata = meta
					}
					if emb, ok := source["embedding"].([]interface{}); ok {
						vec.Values = make([]float32, len(emb))
						for i, v := range emb {
							vec.Values[i] = float32(v.(float64))
						}
					}
				}

				vectors = append(vectors, vec)
			}
		}
	}

	return vectors, nil
}

// Delete removes vectors by IDs.
func (db *DB) Delete(ctx context.Context, indexName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	osIndexName := indexPrefix + indexName

	// Use bulk delete
	var buf bytes.Buffer
	for _, id := range ids {
		action := map[string]interface{}{
			"delete": map[string]interface{}{
				"_index": osIndexName,
				"_id":    id,
			},
		}
		actionJSON, _ := json.Marshal(action)
		buf.Write(actionJSON)
		buf.WriteByte('\n')
	}

	req, err := http.NewRequestWithContext(ctx, "POST", db.baseURL+"/_bulk", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")

	res, err := db.client.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()

	return nil
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	res, err := db.doRequest(ctx, "GET", "/", nil)
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("ping failed: status %d", res.StatusCode)
	}
	return nil
}

// Close releases resources.
func (db *DB) Close() error {
	// HTTP client doesn't require explicit close
	return nil
}
