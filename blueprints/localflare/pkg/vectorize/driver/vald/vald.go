// Package vald provides a Vald vector database driver for the vectorize package.
// Import this package to register the "vald" driver.
// Vald is a highly scalable distributed vector search engine.
package vald

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
	"github.com/vdaas/vald-client-go/v1/payload"
	"github.com/vdaas/vald-client-go/v1/vald"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	driver.Register("vald", &Driver{})
}

// Driver implements vectorize.Driver for Vald.
type Driver struct{}

// Open creates a new Vald connection.
// DSN format: host:port (e.g., "localhost:8081")
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		dsn = "localhost:8081"
	}

	conn, err := grpc.NewClient(dsn, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", vectorize.ErrConnectionFailed, err)
	}

	return &DB{
		conn:    conn,
		client:  vald.NewValdClient(conn),
		indexes: make(map[string]*indexInfo),
	}, nil
}

// indexInfo stores index metadata (Vald doesn't have traditional indexes).
type indexInfo struct {
	dimensions  int
	metric      vectorize.DistanceMetric
	description string
	vectorCount int64
	createdAt   time.Time
}

// DB implements vectorize.DB for Vald.
type DB struct {
	mu      sync.RWMutex
	conn    *grpc.ClientConn
	client  vald.Client
	indexes map[string]*indexInfo
}

// CreateIndex creates a virtual index (Vald doesn't have traditional indexes).
// We use a prefix-based approach to separate vectors by "index".
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.indexes[index.Name]; exists {
		return vectorize.ErrIndexExists
	}

	db.indexes[index.Name] = &indexInfo{
		dimensions:  index.Dimensions,
		metric:      index.Metric,
		description: index.Description,
		createdAt:   time.Now(),
	}

	return nil
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

// DeleteIndex removes an index (deletes all vectors with the index prefix).
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, ok := db.indexes[name]; !ok {
		return vectorize.ErrIndexNotFound
	}

	delete(db.indexes, name)
	return nil
}

// makeVectorID creates a prefixed vector ID for index separation.
func makeVectorID(indexName, id string) string {
	return indexName + ":" + id
}

// parseVectorID extracts the original ID from a prefixed vector ID.
func parseVectorID(prefixedID, indexName string) string {
	prefix := indexName + ":"
	if len(prefixedID) > len(prefix) && prefixedID[:len(prefix)] == prefix {
		return prefixedID[len(prefix):]
	}
	return prefixedID
}

// Insert adds vectors to Vald.
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

		req := &payload.Insert_Request{
			Vector: &payload.Object_Vector{
				Id:     makeVectorID(indexName, v.ID),
				Vector: v.Values,
			},
			Config: &payload.Insert_Config{
				SkipStrictExistCheck: true,
			},
		}

		_, err := db.client.Insert(ctx, req)
		if err != nil {
			return fmt.Errorf("insert failed for %s: %w", v.ID, err)
		}
	}

	// Update vector count
	db.mu.Lock()
	info.vectorCount += int64(len(vectors))
	db.mu.Unlock()

	// Wait for auto-indexing to complete (vald-agent-ngt auto-indexes periodically)
	// The config sets auto_index_check_duration to 1s so we wait briefly
	time.Sleep(2 * time.Second)

	return nil
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
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

		req := &payload.Upsert_Request{
			Vector: &payload.Object_Vector{
				Id:     makeVectorID(indexName, v.ID),
				Vector: v.Values,
			},
			Config: &payload.Upsert_Config{
				SkipStrictExistCheck: true,
			},
		}

		_, err := db.client.Upsert(ctx, req)
		if err != nil {
			return fmt.Errorf("upsert failed for %s: %w", v.ID, err)
		}
	}

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

	db.mu.RLock()
	info, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return nil, vectorize.ErrIndexNotFound
	}

	if len(vector) != info.dimensions {
		return nil, vectorize.ErrDimensionMismatch
	}

	req := &payload.Search_Request{
		Vector: vector,
		Config: &payload.Search_Config{
			Num:     uint32(opts.TopK),
			Radius:  -1, // No radius limit
			Epsilon: 0.1,
			Timeout: 30000000000, // 30 seconds in nanoseconds
		},
	}

	resp, err := db.client.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var matches []*vectorize.Match
	prefix := indexName + ":"
	for _, r := range resp.GetResults() {
		// Filter by index prefix
		if len(r.GetId()) <= len(prefix) || r.GetId()[:len(prefix)] != prefix {
			continue
		}

		// Convert distance to similarity score (assuming L2 distance)
		score := 1.0 / (1.0 + r.GetDistance())

		if opts.ScoreThreshold > 0 && score < opts.ScoreThreshold {
			continue
		}

		match := &vectorize.Match{
			ID:    parseVectorID(r.GetId(), indexName),
			Score: float32(score),
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

	db.mu.RLock()
	_, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return nil, vectorize.ErrIndexNotFound
	}

	var vectors []*vectorize.Vector
	for _, id := range ids {
		req := &payload.Object_VectorRequest{
			Id: &payload.Object_ID{
				Id: makeVectorID(indexName, id),
			},
		}

		resp, err := db.client.GetObject(ctx, req)
		if err != nil {
			continue // Skip missing vectors
		}

		vectors = append(vectors, &vectorize.Vector{
			ID:     id,
			Values: resp.GetVector(),
		})
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
		req := &payload.Remove_Request{
			Id: &payload.Object_ID{
				Id: makeVectorID(indexName, id),
			},
		}

		_, err := db.client.Remove(ctx, req)
		if err == nil {
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
	// Try to get index info as a health check
	_, err := db.client.IndexInfo(ctx, &payload.Empty{})
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	return nil
}

// Close releases resources.
func (db *DB) Close() error {
	return db.conn.Close()
}
