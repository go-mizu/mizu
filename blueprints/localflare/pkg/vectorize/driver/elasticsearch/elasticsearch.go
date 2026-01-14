// Package elasticsearch provides an Elasticsearch driver for the vectorize package.
// Import this package to register the "elasticsearch" driver.
package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
)

func init() {
	driver.Register("elasticsearch", &Driver{})
}

// Driver implements vectorize.Driver for Elasticsearch.
type Driver struct{}

// Open creates a new Elasticsearch connection.
// DSN format: http://user:pass@host:port or https://user:pass@host:port
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		return nil, vectorize.ErrInvalidDSN
	}

	cfg := elasticsearch.Config{
		Addresses: []string{dsn},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", vectorize.ErrConnectionFailed, err)
	}

	return &DB{client: client, basePath: dsn}, nil
}

// DB implements vectorize.DB for Elasticsearch.
type DB struct {
	client   *elasticsearch.Client
	basePath string
}

const indexPrefix = "vectorize_"

// CreateIndex creates a new Elasticsearch index for vectors.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	indexName := indexPrefix + index.Name

	// Check if index exists
	res, err := db.client.Indices.Exists([]string{indexName})
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode == 200 {
		return vectorize.ErrIndexExists
	}

	// Create index with dense_vector mapping
	similarity := "cosine"
	switch index.Metric {
	case vectorize.Euclidean:
		similarity = "l2_norm"
	case vectorize.DotProduct:
		similarity = "dot_product"
	}

	mapping := map[string]interface{}{
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
					"type":       "dense_vector",
					"dims":       index.Dimensions,
					"index":      true,
					"similarity": similarity,
				},
				"_dimensions": map[string]interface{}{
					"type": "integer",
				},
				"_description": map[string]interface{}{
					"type": "text",
				},
				"_created_at": map[string]interface{}{
					"type": "date",
				},
			},
		},
	}

	body, _ := json.Marshal(mapping)
	res, err = db.client.Indices.Create(indexName, db.client.Indices.Create.WithBody(bytes.NewReader(body)))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to create index: %s", string(bodyBytes))
	}

	// Store index metadata
	metaDoc := map[string]interface{}{
		"_dimensions":  index.Dimensions,
		"_description": index.Description,
		"_created_at":  time.Now().Format(time.RFC3339),
	}
	metaBody, _ := json.Marshal(metaDoc)

	res, err = db.client.Index(indexName, bytes.NewReader(metaBody), db.client.Index.WithDocumentID("_meta"))
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
	res, err := db.client.Indices.Exists([]string{indexName})
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
	res, err = db.client.Get(indexName, "_meta")
	if err == nil && !res.IsError() {
		defer res.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(res.Body).Decode(&result)
		if source, ok := result["_source"].(map[string]interface{}); ok {
			if dims, ok := source["_dimensions"].(float64); ok {
				idx.Dimensions = int(dims)
			}
			if desc, ok := source["_description"].(string); ok {
				idx.Description = desc
			}
			if createdAt, ok := source["_created_at"].(string); ok {
				if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
					idx.CreatedAt = t
				}
			}
		}
	}

	// Get count
	res, err = db.client.Count(db.client.Count.WithIndex(indexName))
	if err == nil && !res.IsError() {
		defer res.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(res.Body).Decode(&result)
		if count, ok := result["count"].(float64); ok {
			idx.VectorCount = int64(count) - 1 // Subtract metadata doc
		}
	}

	return idx, nil
}

// ListIndexes returns all vector indexes.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	res, err := db.client.Cat.Indices(
		db.client.Cat.Indices.WithIndex(indexPrefix+"*"),
		db.client.Cat.Indices.WithFormat("json"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var indices []map[string]interface{}
	json.NewDecoder(res.Body).Decode(&indices)

	indexes := make([]*vectorize.Index, 0, len(indices))
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

	res, err := db.client.Indices.Delete([]string{indexName})
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

	esIndexName := indexPrefix + indexName

	// Use bulk API
	var buf bytes.Buffer
	for _, v := range vectors {
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": esIndexName,
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

	res, err := db.client.Bulk(bytes.NewReader(buf.Bytes()))
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

	esIndexName := indexPrefix + indexName

	// Convert float32 to float64 for JSON
	queryVector := make([]float64, len(vector))
	for i, v := range vector {
		queryVector[i] = float64(v)
	}

	query := map[string]interface{}{
		"size": opts.TopK,
		"knn": map[string]interface{}{
			"field":          "embedding",
			"query_vector":   queryVector,
			"k":              opts.TopK,
			"num_candidates": opts.TopK * 2,
		},
	}

	// Add namespace filter
	if opts.Namespace != "" {
		query["knn"].(map[string]interface{})["filter"] = map[string]interface{}{
			"term": map[string]interface{}{
				"namespace": opts.Namespace,
			},
		}
	}

	body, _ := json.Marshal(query)
	res, err := db.client.Search(
		db.client.Search.WithIndex(esIndexName),
		db.client.Search.WithBody(bytes.NewReader(body)),
	)
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

	esIndexName := indexPrefix + indexName

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"ids": map[string]interface{}{
				"values": ids,
			},
		},
	}

	body, _ := json.Marshal(query)
	res, err := db.client.Search(
		db.client.Search.WithIndex(esIndexName),
		db.client.Search.WithBody(bytes.NewReader(body)),
	)
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

	esIndexName := indexPrefix + indexName

	// Use bulk delete
	var buf bytes.Buffer
	for _, id := range ids {
		action := map[string]interface{}{
			"delete": map[string]interface{}{
				"_index": esIndexName,
				"_id":    id,
			},
		}
		actionJSON, _ := json.Marshal(action)
		buf.Write(actionJSON)
		buf.WriteByte('\n')
	}

	res, err := db.client.Bulk(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}
	res.Body.Close()

	return nil
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	res, err := db.client.Ping()
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

// Close releases resources.
func (db *DB) Close() error {
	// Elasticsearch client doesn't require explicit close
	return nil
}
