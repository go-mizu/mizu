// Package redis provides a Redis Stack driver for the vectorize package.
// Import this package to register the "redis" driver.
package redis

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
	"github.com/redis/go-redis/v9"
)

func init() {
	driver.Register("redis", &Driver{})
}

// Driver implements vectorize.Driver for Redis Stack.
type Driver struct{}

// Open creates a new Redis connection.
// DSN format: redis://host:port or redis://user:pass@host:port
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		return nil, vectorize.ErrInvalidDSN
	}

	opts, err := redis.ParseURL(dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", vectorize.ErrInvalidDSN, err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", vectorize.ErrConnectionFailed, err)
	}

	return &DB{client: client}, nil
}

// DB implements vectorize.DB for Redis Stack.
type DB struct {
	client *redis.Client
}

const (
	indexKeyPrefix  = "vectorize:index:"
	vectorKeyPrefix = "vectorize:vec:"
)

// CreateIndex creates a new vector index using RediSearch.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	indexKey := indexKeyPrefix + index.Name

	// Check if index exists
	exists, err := db.client.Exists(ctx, indexKey).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		return vectorize.ErrIndexExists
	}

	// Store index metadata
	metadata := map[string]interface{}{
		"name":        index.Name,
		"dimensions":  index.Dimensions,
		"metric":      string(index.Metric),
		"description": index.Description,
		"created_at":  time.Now().Format(time.RFC3339),
	}
	metadataJSON, _ := json.Marshal(metadata)
	if err := db.client.Set(ctx, indexKey, metadataJSON, 0).Err(); err != nil {
		return err
	}

	// Create RediSearch index for vectors
	distanceMetric := "COSINE"
	switch index.Metric {
	case vectorize.Euclidean:
		distanceMetric = "L2"
	case vectorize.DotProduct:
		distanceMetric = "IP"
	}

	ftIndexName := "idx:" + index.Name
	createCmd := []interface{}{
		"FT.CREATE", ftIndexName,
		"ON", "HASH",
		"PREFIX", "1", vectorKeyPrefix + index.Name + ":",
		"SCHEMA",
		"embedding", "VECTOR", "FLAT", "6",
		"TYPE", "FLOAT32",
		"DIM", index.Dimensions,
		"DISTANCE_METRIC", distanceMetric,
		"namespace", "TAG",
		"metadata", "TEXT",
	}

	if err := db.client.Do(ctx, createCmd...).Err(); err != nil {
		// Clean up on failure
		db.client.Del(ctx, indexKey)
		return err
	}

	return nil
}

// GetIndex retrieves index information.
func (db *DB) GetIndex(ctx context.Context, name string) (*vectorize.Index, error) {
	indexKey := indexKeyPrefix + name

	metadataJSON, err := db.client.Get(ctx, indexKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, vectorize.ErrIndexNotFound
		}
		return nil, err
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, err
	}

	idx := &vectorize.Index{
		Name: name,
	}

	if dims, ok := metadata["dimensions"].(float64); ok {
		idx.Dimensions = int(dims)
	}
	if metric, ok := metadata["metric"].(string); ok {
		idx.Metric = vectorize.DistanceMetric(metric)
	}
	if desc, ok := metadata["description"].(string); ok {
		idx.Description = desc
	}
	if createdAt, ok := metadata["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			idx.CreatedAt = t
		}
	}

	// Count vectors
	pattern := vectorKeyPrefix + name + ":*"
	keys, _ := db.client.Keys(ctx, pattern).Result()
	idx.VectorCount = int64(len(keys))

	return idx, nil
}

// ListIndexes returns all indexes.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	pattern := indexKeyPrefix + "*"
	keys, err := db.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	indexes := make([]*vectorize.Index, 0, len(keys))
	for _, key := range keys {
		name := strings.TrimPrefix(key, indexKeyPrefix)
		idx, err := db.GetIndex(ctx, name)
		if err != nil {
			continue
		}
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

// DeleteIndex removes an index and all its vectors.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	indexKey := indexKeyPrefix + name

	// Check if index exists
	exists, err := db.client.Exists(ctx, indexKey).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return vectorize.ErrIndexNotFound
	}

	// Delete RediSearch index
	ftIndexName := "idx:" + name
	db.client.Do(ctx, "FT.DROPINDEX", ftIndexName, "DD")

	// Delete all vectors
	pattern := vectorKeyPrefix + name + ":*"
	keys, _ := db.client.Keys(ctx, pattern).Result()
	if len(keys) > 0 {
		db.client.Del(ctx, keys...)
	}

	// Delete index metadata
	db.client.Del(ctx, indexKey)

	return nil
}

// Insert adds vectors to an index.
func (db *DB) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	// Verify index exists
	_, err := db.GetIndex(ctx, indexName)
	if err != nil {
		return err
	}

	pipe := db.client.Pipeline()
	for _, v := range vectors {
		key := vectorKeyPrefix + indexName + ":" + v.ID

		metadataJSON, _ := json.Marshal(v.Metadata)

		// Convert vector to bytes
		embBytes := vectorToBytes(v.Values)

		fields := map[string]interface{}{
			"embedding": embBytes,
			"namespace": v.Namespace,
			"metadata":  string(metadataJSON),
		}

		pipe.HSet(ctx, key, fields)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	return db.Insert(ctx, indexName, vectors)
}

// Search finds similar vectors using RediSearch.
func (db *DB) Search(ctx context.Context, indexName string, vector []float32, opts *vectorize.SearchOptions) ([]*vectorize.Match, error) {
	if opts == nil {
		opts = &vectorize.SearchOptions{TopK: 10}
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	ftIndexName := "idx:" + indexName

	// Build KNN query
	query := fmt.Sprintf("*=>[KNN %d @embedding $vec]", opts.TopK)

	// Add namespace filter
	if opts.Namespace != "" {
		query = fmt.Sprintf("@namespace:{%s} ", opts.Namespace) + query
	}

	// Prepare search command
	searchCmd := []interface{}{
		"FT.SEARCH", ftIndexName, query,
		"PARAMS", "2", "vec", vectorToBytes(vector),
		"SORTBY", "__embedding_score",
		"DIALECT", "2",
		"RETURN", "3", "metadata", "namespace", "__embedding_score",
	}

	result, err := db.client.Do(ctx, searchCmd...).Result()
	if err != nil {
		return nil, err
	}

	// Parse results
	results, ok := result.([]interface{})
	if !ok || len(results) < 1 {
		return []*vectorize.Match{}, nil
	}

	// First element is total count
	matches := make([]*vectorize.Match, 0)
	for i := 1; i < len(results); i += 2 {
		if i+1 >= len(results) {
			break
		}

		key, ok := results[i].(string)
		if !ok {
			continue
		}

		// Extract ID from key
		id := strings.TrimPrefix(key, vectorKeyPrefix+indexName+":")

		match := &vectorize.Match{
			ID: id,
		}

		// Parse fields
		fields, ok := results[i+1].([]interface{})
		if ok {
			for j := 0; j < len(fields)-1; j += 2 {
				fieldName, _ := fields[j].(string)
				fieldValue := fields[j+1]

				switch fieldName {
				case "__embedding_score":
					if scoreStr, ok := fieldValue.(string); ok {
						score, _ := strconv.ParseFloat(scoreStr, 32)
						// Redis returns distance, convert to similarity
						match.Score = float32(1.0 / (1.0 + score))
					}
				case "metadata":
					if opts.ReturnMetadata {
						if metadataStr, ok := fieldValue.(string); ok {
							json.Unmarshal([]byte(metadataStr), &match.Metadata)
						}
					}
				}
			}
		}

		// Apply score threshold
		if opts.ScoreThreshold > 0 && match.Score < opts.ScoreThreshold {
			continue
		}

		matches = append(matches, match)
	}

	return matches, nil
}

// Get retrieves vectors by IDs.
func (db *DB) Get(ctx context.Context, indexName string, ids []string) ([]*vectorize.Vector, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	pipe := db.client.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(ids))

	for i, id := range ids {
		key := vectorKeyPrefix + indexName + ":" + id
		cmds[i] = pipe.HGetAll(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	vectors := make([]*vectorize.Vector, 0, len(ids))
	for i, cmd := range cmds {
		fields, err := cmd.Result()
		if err != nil || len(fields) == 0 {
			continue
		}

		vec := &vectorize.Vector{
			ID:        ids[i],
			Namespace: fields["namespace"],
		}

		if embStr, ok := fields["embedding"]; ok {
			vec.Values = bytesToVector([]byte(embStr))
		}

		if metadataStr, ok := fields["metadata"]; ok && metadataStr != "" {
			json.Unmarshal([]byte(metadataStr), &vec.Metadata)
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

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = vectorKeyPrefix + indexName + ":" + id
	}

	return db.client.Del(ctx, keys...).Err()
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	return db.client.Ping(ctx).Err()
}

// Close releases resources.
func (db *DB) Close() error {
	return db.client.Close()
}

// Helper functions

func vectorToBytes(v []float32) []byte {
	bytes := make([]byte, len(v)*4)
	for i, val := range v {
		binary.LittleEndian.PutUint32(bytes[i*4:], math.Float32bits(val))
	}
	return bytes
}

func bytesToVector(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}
	v := make([]float32, len(b)/4)
	for i := 0; i < len(v); i++ {
		bits := binary.LittleEndian.Uint32(b[i*4:])
		v[i] = math.Float32frombits(bits)
	}
	return v
}
