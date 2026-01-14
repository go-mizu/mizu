// Package milvus provides a Milvus driver for the vectorize package.
// Import this package to register the "milvus" driver.
// Note: This implementation uses HTTP requests directly for better compatibility.
package milvus

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
	driver.Register("milvus", &Driver{})
}

// Driver implements vectorize.Driver for Milvus.
type Driver struct{}

// Open creates a new Milvus connection.
// DSN format: host:port (e.g., "localhost:19530")
// The driver will use the HTTP API on the same port (v2.4.0+)
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		return nil, vectorize.ErrInvalidDSN
	}

	// Milvus v2.4.0+ uses the same port for gRPC and HTTP API
	baseURL := "http://" + dsn
	if !strings.Contains(dsn, ":") {
		baseURL = "http://" + dsn + ":19530"
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &DB{client: client, baseURL: baseURL}, nil
}

// DB implements vectorize.DB for Milvus.
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
	req.Header.Set("Accept", "application/json")

	return db.client.Do(req)
}

// CreateIndex creates a new collection in Milvus.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	// Check if collection exists
	checkReq := map[string]interface{}{
		"collectionName": index.Name,
	}

	res, err := db.doRequest(ctx, "POST", "/v2/vectordb/collections/has", checkReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var checkResult map[string]interface{}
	json.NewDecoder(res.Body).Decode(&checkResult)
	if data, ok := checkResult["data"].(map[string]interface{}); ok {
		if has, ok := data["has"].(bool); ok && has {
			return vectorize.ErrIndexExists
		}
	}

	// Map metric type
	metricType := "COSINE"
	switch index.Metric {
	case vectorize.Euclidean:
		metricType = "L2"
	case vectorize.DotProduct:
		metricType = "IP"
	}

	// Create collection with schema
	createReq := map[string]interface{}{
		"collectionName": index.Name,
		"description":    index.Description,
		"dimension":      index.Dimensions,
		"metricType":     metricType,
		"primaryField":   "id",
		"vectorField":    "embedding",
	}

	res, err = db.doRequest(ctx, "POST", "/v2/vectordb/collections/create", createReq)
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
	descReq := map[string]interface{}{
		"collectionName": name,
	}

	res, err := db.doRequest(ctx, "POST", "/v2/vectordb/collections/describe", descReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, vectorize.ErrIndexNotFound
	}

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	// Check for error in response
	if code, ok := result["code"].(float64); ok && code != 0 {
		return nil, vectorize.ErrIndexNotFound
	}

	idx := &vectorize.Index{
		Name:   name,
		Metric: vectorize.Cosine,
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		if desc, ok := data["description"].(string); ok {
			idx.Description = desc
		}
		if fields, ok := data["fields"].([]interface{}); ok {
			for _, f := range fields {
				field := f.(map[string]interface{})
				if fieldName, ok := field["name"].(string); ok && fieldName == "embedding" {
					if params, ok := field["params"].(map[string]interface{}); ok {
						if dim, ok := params["dim"].(float64); ok {
							idx.Dimensions = int(dim)
						}
					}
				}
			}
		}
	}

	// Get stats
	statsReq := map[string]interface{}{
		"collectionName": name,
	}
	statsRes, err := db.doRequest(ctx, "POST", "/v2/vectordb/collections/get_stats", statsReq)
	if err == nil {
		defer statsRes.Body.Close()
		var statsResult map[string]interface{}
		json.NewDecoder(statsRes.Body).Decode(&statsResult)
		if data, ok := statsResult["data"].(map[string]interface{}); ok {
			if count, ok := data["rowCount"].(float64); ok {
				idx.VectorCount = int64(count)
			}
		}
	}

	return idx, nil
}

// ListIndexes returns all collections.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	res, err := db.doRequest(ctx, "POST", "/v2/vectordb/collections/list", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	indexes := make([]*vectorize.Index, 0)
	if data, ok := result["data"].([]interface{}); ok {
		for _, name := range data {
			if nameStr, ok := name.(string); ok {
				idx, err := db.GetIndex(ctx, nameStr)
				if err != nil {
					continue
				}
				indexes = append(indexes, idx)
			}
		}
	}

	return indexes, nil
}

// DeleteIndex removes a collection.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	dropReq := map[string]interface{}{
		"collectionName": name,
	}

	res, err := db.doRequest(ctx, "POST", "/v2/vectordb/collections/drop", dropReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	if code, ok := result["code"].(float64); ok && code != 0 {
		return vectorize.ErrIndexNotFound
	}

	return nil
}

// Insert adds vectors to a collection.
func (db *DB) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	data := make([]map[string]interface{}, len(vectors))
	for i, v := range vectors {
		data[i] = map[string]interface{}{
			"id":        v.ID,
			"embedding": v.Values,
		}
		if v.Namespace != "" {
			data[i]["namespace"] = v.Namespace
		}
		if v.Metadata != nil {
			metaJSON, _ := json.Marshal(v.Metadata)
			data[i]["metadata"] = string(metaJSON)
		}
	}

	insertReq := map[string]interface{}{
		"collectionName": indexName,
		"data":           data,
	}

	res, err := db.doRequest(ctx, "POST", "/v2/vectordb/entities/insert", insertReq)
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
	if len(vectors) == 0 {
		return nil
	}

	data := make([]map[string]interface{}, len(vectors))
	for i, v := range vectors {
		data[i] = map[string]interface{}{
			"id":        v.ID,
			"embedding": v.Values,
		}
		if v.Namespace != "" {
			data[i]["namespace"] = v.Namespace
		}
		if v.Metadata != nil {
			metaJSON, _ := json.Marshal(v.Metadata)
			data[i]["metadata"] = string(metaJSON)
		}
	}

	upsertReq := map[string]interface{}{
		"collectionName": indexName,
		"data":           data,
	}

	res, err := db.doRequest(ctx, "POST", "/v2/vectordb/entities/upsert", upsertReq)
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

	searchReq := map[string]interface{}{
		"collectionName": indexName,
		"data":           [][]float32{vector},
		"annsField":      "embedding",
		"limit":          opts.TopK,
	}

	if opts.Namespace != "" {
		searchReq["filter"] = fmt.Sprintf("namespace == \"%s\"", opts.Namespace)
	}

	outputFields := []string{"id"}
	if opts.ReturnMetadata {
		outputFields = append(outputFields, "namespace", "metadata")
	}
	searchReq["outputFields"] = outputFields

	res, err := db.doRequest(ctx, "POST", "/v2/vectordb/entities/search", searchReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	matches := make([]*vectorize.Match, 0)
	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		// First result set (for single query vector)
		if resultSet, ok := data[0].([]interface{}); ok {
			for _, item := range resultSet {
				if itemMap, ok := item.(map[string]interface{}); ok {
					match := &vectorize.Match{}

					if id, ok := itemMap["id"].(string); ok {
						match.ID = id
					}
					if distance, ok := itemMap["distance"].(float64); ok {
						// Convert distance to similarity score
						match.Score = float32(1.0 / (1.0 + distance))
					}
					if score, ok := itemMap["score"].(float64); ok {
						match.Score = float32(score)
					}

					if opts.ReturnMetadata {
						if metaStr, ok := itemMap["metadata"].(string); ok && metaStr != "" {
							json.Unmarshal([]byte(metaStr), &match.Metadata)
						}
					}

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

	getReq := map[string]interface{}{
		"collectionName": indexName,
		"id":             ids,
		"outputFields":   []string{"id", "namespace", "metadata", "embedding"},
	}

	res, err := db.doRequest(ctx, "POST", "/v2/vectordb/entities/get", getReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	vectors := make([]*vectorize.Vector, 0)
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if itemMap, ok := item.(map[string]interface{}); ok {
				vec := &vectorize.Vector{}

				if id, ok := itemMap["id"].(string); ok {
					vec.ID = id
				}
				if ns, ok := itemMap["namespace"].(string); ok {
					vec.Namespace = ns
				}
				if metaStr, ok := itemMap["metadata"].(string); ok && metaStr != "" {
					json.Unmarshal([]byte(metaStr), &vec.Metadata)
				}
				if emb, ok := itemMap["embedding"].([]interface{}); ok {
					vec.Values = make([]float32, len(emb))
					for i, v := range emb {
						if f, ok := v.(float64); ok {
							vec.Values[i] = float32(f)
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

	deleteReq := map[string]interface{}{
		"collectionName": indexName,
		"id":             ids,
	}

	res, err := db.doRequest(ctx, "POST", "/v2/vectordb/entities/delete", deleteReq)
	if err != nil {
		return err
	}
	res.Body.Close()

	return nil
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	// Milvus v2.4.0+ requires POST for most endpoints
	res, err := db.doRequest(ctx, "POST", "/v2/vectordb/collections/list", map[string]interface{}{})
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
