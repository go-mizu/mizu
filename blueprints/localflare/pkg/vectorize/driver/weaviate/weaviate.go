// Package weaviate provides a Weaviate driver for the vectorize package.
// Import this package to register the "weaviate" driver.
// Note: This implementation uses HTTP requests directly for better compatibility.
package weaviate

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
	driver.Register("weaviate", &Driver{})
}

// Driver implements vectorize.Driver for Weaviate.
type Driver struct{}

// Open creates a new Weaviate connection.
// DSN format: http://host:port (e.g., "http://localhost:8080")
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		return nil, vectorize.ErrInvalidDSN
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &DB{client: client, baseURL: dsn}, nil
}

// DB implements vectorize.DB for Weaviate.
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

func toClassName(name string) string {
	// Weaviate requires class names to start with uppercase
	if len(name) == 0 {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// CreateIndex creates a new class in Weaviate.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	className := toClassName(index.Name)

	// Check if class exists
	res, err := db.doRequest(ctx, "GET", "/v1/schema/"+className, nil)
	if err == nil {
		res.Body.Close()
		if res.StatusCode == 200 {
			return vectorize.ErrIndexExists
		}
	}

	distanceMetric := "cosine"
	switch index.Metric {
	case vectorize.Euclidean:
		distanceMetric = "l2-squared"
	case vectorize.DotProduct:
		distanceMetric = "dot"
	}

	classSchema := map[string]interface{}{
		"class":       className,
		"description": index.Description,
		"vectorIndexConfig": map[string]interface{}{
			"distance": distanceMetric,
		},
		"properties": []map[string]interface{}{
			{
				"name":        "_namespace",
				"dataType":    []string{"text"},
				"description": "Namespace for multi-tenant isolation",
			},
			{
				"name":        "_metadata",
				"dataType":    []string{"text"},
				"description": "JSON-encoded metadata",
			},
			{
				"name":        "_dimensions",
				"dataType":    []string{"int"},
				"description": "Vector dimensions",
			},
		},
		"moduleConfig": map[string]interface{}{
			"vectorizer": "none",
		},
	}

	res, err = db.doRequest(ctx, "POST", "/v1/schema", classSchema)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to create class: %s", string(body))
	}

	return nil
}

// GetIndex retrieves class information.
func (db *DB) GetIndex(ctx context.Context, name string) (*vectorize.Index, error) {
	className := toClassName(name)

	res, err := db.doRequest(ctx, "GET", "/v1/schema/"+className, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, vectorize.ErrIndexNotFound
	}

	var classSchema map[string]interface{}
	json.NewDecoder(res.Body).Decode(&classSchema)

	idx := &vectorize.Index{
		Name:   name,
		Metric: vectorize.Cosine,
	}

	if desc, ok := classSchema["description"].(string); ok {
		idx.Description = desc
	}

	// Extract metric from config
	if cfg, ok := classSchema["vectorIndexConfig"].(map[string]interface{}); ok {
		if dist, ok := cfg["distance"].(string); ok {
			switch dist {
			case "l2-squared":
				idx.Metric = vectorize.Euclidean
			case "dot":
				idx.Metric = vectorize.DotProduct
			}
		}
	}

	// Get object count using GraphQL
	countQuery := map[string]interface{}{
		"query": fmt.Sprintf(`{ Aggregate { %s { meta { count } } } }`, className),
	}

	countRes, err := db.doRequest(ctx, "POST", "/v1/graphql", countQuery)
	if err == nil && countRes.StatusCode == 200 {
		defer countRes.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(countRes.Body).Decode(&result)
		if data, ok := result["data"].(map[string]interface{}); ok {
			if agg, ok := data["Aggregate"].(map[string]interface{}); ok {
				if classData, ok := agg[className].([]interface{}); ok && len(classData) > 0 {
					if meta, ok := classData[0].(map[string]interface{}); ok {
						if metaObj, ok := meta["meta"].(map[string]interface{}); ok {
							if count, ok := metaObj["count"].(float64); ok {
								idx.VectorCount = int64(count)
							}
						}
					}
				}
			}
		}
	} else if countRes != nil {
		countRes.Body.Close()
	}

	return idx, nil
}

// ListIndexes returns all classes.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	res, err := db.doRequest(ctx, "GET", "/v1/schema", nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var schema map[string]interface{}
	json.NewDecoder(res.Body).Decode(&schema)

	indexes := make([]*vectorize.Index, 0)
	if classes, ok := schema["classes"].([]interface{}); ok {
		for _, c := range classes {
			if classObj, ok := c.(map[string]interface{}); ok {
				if className, ok := classObj["class"].(string); ok {
					name := strings.ToLower(className[:1]) + className[1:]
					idx, err := db.GetIndex(ctx, name)
					if err != nil {
						continue
					}
					indexes = append(indexes, idx)
				}
			}
		}
	}

	return indexes, nil
}

// DeleteIndex removes a class.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	className := toClassName(name)

	res, err := db.doRequest(ctx, "DELETE", "/v1/schema/"+className, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return vectorize.ErrIndexNotFound
	}

	return nil
}

// Insert adds vectors to a class.
func (db *DB) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	className := toClassName(indexName)

	objects := make([]map[string]interface{}, len(vectors))
	for i, v := range vectors {
		props := map[string]interface{}{
			"_namespace":  v.Namespace,
			"_dimensions": len(v.Values),
		}

		// Encode metadata as JSON
		if v.Metadata != nil {
			metaJSON, _ := json.Marshal(v.Metadata)
			props["_metadata"] = string(metaJSON)
		}

		objects[i] = map[string]interface{}{
			"class":      className,
			"id":         v.ID,
			"properties": props,
			"vector":     v.Values,
		}
	}

	batchReq := map[string]interface{}{
		"objects": objects,
	}

	res, err := db.doRequest(ctx, "POST", "/v1/batch/objects", batchReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to insert vectors: %s", string(body))
	}

	return nil
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	return db.Insert(ctx, indexName, vectors) // Weaviate handles upsert automatically
}

// Search finds similar vectors.
func (db *DB) Search(ctx context.Context, indexName string, vector []float32, opts *vectorize.SearchOptions) ([]*vectorize.Match, error) {
	if opts == nil {
		opts = &vectorize.SearchOptions{TopK: 10}
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	className := toClassName(indexName)

	// Build GraphQL query
	fields := "_additional { id distance"
	if opts.ReturnValues {
		fields += " vector"
	}
	fields += " }"

	if opts.ReturnMetadata {
		fields += " _namespace _metadata"
	}

	whereClause := ""
	if opts.Namespace != "" {
		whereClause = fmt.Sprintf(`where: { path: ["_namespace"], operator: Equal, valueText: "%s" }`, opts.Namespace)
	}

	// Convert vector to JSON array
	vectorJSON, _ := json.Marshal(vector)

	query := fmt.Sprintf(`{
		Get {
			%s(
				nearVector: { vector: %s }
				limit: %d
				%s
			) {
				%s
			}
		}
	}`, className, string(vectorJSON), opts.TopK, whereClause, fields)

	queryReq := map[string]interface{}{
		"query": query,
	}

	res, err := db.doRequest(ctx, "POST", "/v1/graphql", queryReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	matches := make([]*vectorize.Match, 0)
	if data, ok := result["data"].(map[string]interface{}); ok {
		if getData, ok := data["Get"].(map[string]interface{}); ok {
			if classData, ok := getData[className].([]interface{}); ok {
				for _, item := range classData {
					obj, ok := item.(map[string]interface{})
					if !ok {
						continue
					}

					match := &vectorize.Match{}

					// Extract ID and distance from _additional
					if additional, ok := obj["_additional"].(map[string]interface{}); ok {
						if id, ok := additional["id"].(string); ok {
							match.ID = id
						}
						if dist, ok := additional["distance"].(float64); ok {
							// Convert distance to similarity score
							match.Score = float32(1.0 - dist)
						}
						if opts.ReturnValues {
							if vec, ok := additional["vector"].([]interface{}); ok {
								match.Values = make([]float32, len(vec))
								for i, v := range vec {
									if f, ok := v.(float64); ok {
										match.Values[i] = float32(f)
									}
								}
							}
						}
					}

					// Extract metadata
					if opts.ReturnMetadata {
						if metaStr, ok := obj["_metadata"].(string); ok && metaStr != "" {
							json.Unmarshal([]byte(metaStr), &match.Metadata)
						}
					}

					// Apply score threshold
					if opts.ScoreThreshold > 0 && match.Score < opts.ScoreThreshold {
						continue
					}

					matches = append(matches, match)
				}
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

	className := toClassName(indexName)
	vectors := make([]*vectorize.Vector, 0, len(ids))

	for _, id := range ids {
		res, err := db.doRequest(ctx, "GET", fmt.Sprintf("/v1/objects/%s/%s?include=vector", className, id), nil)
		if err != nil || res.StatusCode != 200 {
			if res != nil {
				res.Body.Close()
			}
			continue
		}

		var obj map[string]interface{}
		json.NewDecoder(res.Body).Decode(&obj)
		res.Body.Close()

		vec := &vectorize.Vector{
			ID: id,
		}

		if vector, ok := obj["vector"].([]interface{}); ok {
			vec.Values = make([]float32, len(vector))
			for i, v := range vector {
				if f, ok := v.(float64); ok {
					vec.Values[i] = float32(f)
				}
			}
		}

		if props, ok := obj["properties"].(map[string]interface{}); ok {
			if ns, ok := props["_namespace"].(string); ok {
				vec.Namespace = ns
			}
			if metaStr, ok := props["_metadata"].(string); ok && metaStr != "" {
				json.Unmarshal([]byte(metaStr), &vec.Metadata)
			}
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

	className := toClassName(indexName)

	for _, id := range ids {
		res, err := db.doRequest(ctx, "DELETE", fmt.Sprintf("/v1/objects/%s/%s", className, id), nil)
		if err != nil {
			return err
		}
		res.Body.Close()
	}

	return nil
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	res, err := db.doRequest(ctx, "GET", "/v1/.well-known/ready", nil)
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
