// Package chroma provides a ChromaDB driver for the vectorize package.
// Import this package to register the "chroma" driver.
// Note: This implementation uses HTTP requests directly for better compatibility.
package chroma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
)

func init() {
	driver.Register("chroma", &Driver{})
}

// Driver implements vectorize.Driver for Chroma.
type Driver struct{}

// Open creates a new Chroma connection.
// DSN format: http://host:port (e.g., "http://localhost:8000")
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		return nil, vectorize.ErrInvalidDSN
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &DB{client: client, baseURL: dsn}, nil
}

// DB implements vectorize.DB for Chroma.
type DB struct {
	client  *http.Client
	baseURL string
}

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

// CreateIndex creates a new collection in Chroma.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	// Check if collection exists
	res, err := db.doRequest(ctx, "GET", "/api/v1/collections/"+index.Name, nil)
	if err == nil {
		res.Body.Close()
		if res.StatusCode == 200 {
			return vectorize.ErrIndexExists
		}
	}

	// Map distance metric
	metric := "cosine"
	switch index.Metric {
	case vectorize.Euclidean:
		metric = "l2"
	case vectorize.DotProduct:
		metric = "ip"
	}

	// Create collection
	createReq := map[string]interface{}{
		"name": index.Name,
		"metadata": map[string]interface{}{
			"dimensions":  index.Dimensions,
			"metric":      metric,
			"description": index.Description,
			"created_at":  time.Now().Format(time.RFC3339),
		},
	}

	res, err = db.doRequest(ctx, "POST", "/api/v1/collections", createReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to create collection: %s", string(body))
	}

	return nil
}

// GetIndex retrieves collection information.
func (db *DB) GetIndex(ctx context.Context, name string) (*vectorize.Index, error) {
	res, err := db.doRequest(ctx, "GET", "/api/v1/collections/"+name, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, vectorize.ErrIndexNotFound
	}

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	idx := &vectorize.Index{
		Name:   name,
		Metric: vectorize.Cosine,
	}

	if metadata, ok := result["metadata"].(map[string]interface{}); ok {
		if dims, ok := metadata["dimensions"].(float64); ok {
			idx.Dimensions = int(dims)
		}
		if desc, ok := metadata["description"].(string); ok {
			idx.Description = desc
		}
		if metric, ok := metadata["metric"].(string); ok {
			switch metric {
			case "l2":
				idx.Metric = vectorize.Euclidean
			case "ip":
				idx.Metric = vectorize.DotProduct
			}
		}
		if createdAt, ok := metadata["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				idx.CreatedAt = t
			}
		}
	}

	// Get count
	countRes, err := db.doRequest(ctx, "GET", "/api/v1/collections/"+name+"/count", nil)
	if err == nil && countRes.StatusCode == 200 {
		defer countRes.Body.Close()
		var countResult int64
		json.NewDecoder(countRes.Body).Decode(&countResult)
		idx.VectorCount = countResult
	} else if countRes != nil {
		countRes.Body.Close()
	}

	return idx, nil
}

// ListIndexes returns all collections.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	res, err := db.doRequest(ctx, "GET", "/api/v1/collections", nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var collections []map[string]interface{}
	json.NewDecoder(res.Body).Decode(&collections)

	indexes := make([]*vectorize.Index, 0, len(collections))
	for _, col := range collections {
		if name, ok := col["name"].(string); ok {
			idx, err := db.GetIndex(ctx, name)
			if err != nil {
				continue
			}
			indexes = append(indexes, idx)
		}
	}

	return indexes, nil
}

// DeleteIndex removes a collection.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	res, err := db.doRequest(ctx, "DELETE", "/api/v1/collections/"+name, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return vectorize.ErrIndexNotFound
	}

	return nil
}

// Insert adds vectors to a collection.
func (db *DB) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	ids := make([]string, len(vectors))
	embeddings := make([][]float32, len(vectors))
	metadatas := make([]map[string]interface{}, len(vectors))

	for i, v := range vectors {
		ids[i] = v.ID
		embeddings[i] = v.Values

		metadata := make(map[string]interface{})
		if v.Namespace != "" {
			metadata["_namespace"] = v.Namespace
		}
		for k, val := range v.Metadata {
			metadata[k] = val
		}
		metadatas[i] = metadata
	}

	addReq := map[string]interface{}{
		"ids":        ids,
		"embeddings": embeddings,
		"metadatas":  metadatas,
	}

	res, err := db.doRequest(ctx, "POST", "/api/v1/collections/"+indexName+"/add", addReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to add vectors: %s", string(body))
	}

	return nil
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	ids := make([]string, len(vectors))
	embeddings := make([][]float32, len(vectors))
	metadatas := make([]map[string]interface{}, len(vectors))

	for i, v := range vectors {
		ids[i] = v.ID
		embeddings[i] = v.Values

		metadata := make(map[string]interface{})
		if v.Namespace != "" {
			metadata["_namespace"] = v.Namespace
		}
		for k, val := range v.Metadata {
			metadata[k] = val
		}
		metadatas[i] = metadata
	}

	upsertReq := map[string]interface{}{
		"ids":        ids,
		"embeddings": embeddings,
		"metadatas":  metadatas,
	}

	res, err := db.doRequest(ctx, "POST", "/api/v1/collections/"+indexName+"/upsert", upsertReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

// Search finds similar vectors.
func (db *DB) Search(ctx context.Context, indexName string, vector []float32, opts *vectorize.SearchOptions) ([]*vectorize.Match, error) {
	if opts == nil {
		opts = &vectorize.SearchOptions{TopK: 10}
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	queryReq := map[string]interface{}{
		"query_embeddings": [][]float32{vector},
		"n_results":        opts.TopK,
		"include":          []string{"distances", "metadatas"},
	}

	if opts.Namespace != "" {
		queryReq["where"] = map[string]interface{}{
			"_namespace": opts.Namespace,
		}
	}

	res, err := db.doRequest(ctx, "POST", "/api/v1/collections/"+indexName+"/query", queryReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	matches := make([]*vectorize.Match, 0)
	if ids, ok := result["ids"].([]interface{}); ok && len(ids) > 0 {
		if idList, ok := ids[0].([]interface{}); ok {
			distances := make([]float64, 0)
			if dist, ok := result["distances"].([]interface{}); ok && len(dist) > 0 {
				if distList, ok := dist[0].([]interface{}); ok {
					for _, d := range distList {
						if df, ok := d.(float64); ok {
							distances = append(distances, df)
						}
					}
				}
			}

			var metadatas []map[string]interface{}
			if meta, ok := result["metadatas"].([]interface{}); ok && len(meta) > 0 {
				if metaList, ok := meta[0].([]interface{}); ok {
					for _, m := range metaList {
						if mm, ok := m.(map[string]interface{}); ok {
							metadatas = append(metadatas, mm)
						}
					}
				}
			}

			for i, id := range idList {
				idStr, ok := id.(string)
				if !ok {
					continue
				}

				match := &vectorize.Match{
					ID: idStr,
				}

				// Convert distance to similarity
				if i < len(distances) {
					match.Score = float32(1.0 / (1.0 + distances[i]))
				}

				if opts.ReturnMetadata && i < len(metadatas) {
					match.Metadata = metadatas[i]
				}

				if opts.ScoreThreshold > 0 && match.Score < opts.ScoreThreshold {
					continue
				}

				matches = append(matches, match)
			}
		}
	}

	// Sort by score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches, nil
}

// Get retrieves vectors by IDs.
func (db *DB) Get(ctx context.Context, indexName string, ids []string) ([]*vectorize.Vector, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	getReq := map[string]interface{}{
		"ids":     ids,
		"include": []string{"embeddings", "metadatas"},
	}

	res, err := db.doRequest(ctx, "POST", "/api/v1/collections/"+indexName+"/get", getReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	vectors := make([]*vectorize.Vector, 0)
	if idList, ok := result["ids"].([]interface{}); ok {
		metadatas := make([]map[string]interface{}, len(idList))
		if meta, ok := result["metadatas"].([]interface{}); ok {
			for i, m := range meta {
				if mm, ok := m.(map[string]interface{}); ok {
					metadatas[i] = mm
				}
			}
		}

		embeddings := make([][]float32, len(idList))
		if emb, ok := result["embeddings"].([]interface{}); ok {
			for i, e := range emb {
				if embList, ok := e.([]interface{}); ok {
					embeddings[i] = make([]float32, len(embList))
					for j, v := range embList {
						if vf, ok := v.(float64); ok {
							embeddings[i][j] = float32(vf)
						}
					}
				}
			}
		}

		for i, id := range idList {
			idStr, ok := id.(string)
			if !ok {
				continue
			}

			vec := &vectorize.Vector{
				ID: idStr,
			}

			if i < len(metadatas) && metadatas[i] != nil {
				vec.Metadata = metadatas[i]
				if ns, ok := vec.Metadata["_namespace"].(string); ok {
					vec.Namespace = ns
					delete(vec.Metadata, "_namespace")
				}
			}

			if i < len(embeddings) {
				vec.Values = embeddings[i]
			}

			vectors = append(vectors, vec)
		}
	}

	return vectors, nil
}

// Delete removes vectors by IDs.
func (db *DB) Delete(ctx context.Context, indexName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	deleteReq := map[string]interface{}{
		"ids": ids,
	}

	res, err := db.doRequest(ctx, "POST", "/api/v1/collections/"+indexName+"/delete", deleteReq)
	if err != nil {
		return err
	}
	res.Body.Close()

	return nil
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	res, err := db.doRequest(ctx, "GET", "/api/v1/heartbeat", nil)
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

// Close releases resources.
func (db *DB) Close() error {
	// HTTP client doesn't require explicit close
	return nil
}
