// Package lancedb provides a LanceDB embedded driver for the vectorize package.
// Import this package to register the "lancedb" driver.
// Note: LanceDB Go bindings are limited, this is a placeholder implementation.
package lancedb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
)

func init() {
	driver.Register("lancedb", &Driver{})
}

// Driver implements vectorize.Driver for LanceDB.
type Driver struct{}

// Open creates a new LanceDB connection.
// DSN format: /path/to/data (directory path)
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		return nil, vectorize.ErrInvalidDSN
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dsn, 0755); err != nil {
		return nil, fmt.Errorf("%w: %v", vectorize.ErrConnectionFailed, err)
	}

	return &DB{
		dataDir: dsn,
		indexes: make(map[string]*indexData),
	}, nil
}

// DB implements vectorize.DB for LanceDB.
// This is a file-based implementation since LanceDB Go bindings are limited.
type DB struct {
	dataDir string
	mu      sync.RWMutex
	indexes map[string]*indexData
}

type indexData struct {
	Index   *vectorize.Index
	Vectors map[string]*vectorize.Vector
}

func (db *DB) indexPath(name string) string {
	return filepath.Join(db.dataDir, name+".json")
}

func (db *DB) loadIndex(name string) (*indexData, error) {
	db.mu.RLock()
	if idx, ok := db.indexes[name]; ok {
		db.mu.RUnlock()
		return idx, nil
	}
	db.mu.RUnlock()

	// Load from file
	data, err := os.ReadFile(db.indexPath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, vectorize.ErrIndexNotFound
		}
		return nil, err
	}

	var idx indexData
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}

	db.mu.Lock()
	db.indexes[name] = &idx
	db.mu.Unlock()

	return &idx, nil
}

func (db *DB) saveIndex(idx *indexData) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(db.indexPath(idx.Index.Name), data, 0644)
}

// CreateIndex creates a new index.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Check if exists
	if _, err := os.Stat(db.indexPath(index.Name)); err == nil {
		return vectorize.ErrIndexExists
	}

	idx := &indexData{
		Index: &vectorize.Index{
			Name:        index.Name,
			Dimensions:  index.Dimensions,
			Metric:      index.Metric,
			Description: index.Description,
			VectorCount: 0,
			CreatedAt:   time.Now(),
		},
		Vectors: make(map[string]*vectorize.Vector),
	}

	db.indexes[index.Name] = idx
	return db.saveIndex(idx)
}

// GetIndex retrieves index information.
func (db *DB) GetIndex(ctx context.Context, name string) (*vectorize.Index, error) {
	idx, err := db.loadIndex(name)
	if err != nil {
		return nil, err
	}

	return &vectorize.Index{
		Name:        idx.Index.Name,
		Dimensions:  idx.Index.Dimensions,
		Metric:      idx.Index.Metric,
		Description: idx.Index.Description,
		VectorCount: int64(len(idx.Vectors)),
		CreatedAt:   idx.Index.CreatedAt,
	}, nil
}

// ListIndexes returns all indexes.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	files, err := filepath.Glob(filepath.Join(db.dataDir, "*.json"))
	if err != nil {
		return nil, err
	}

	indexes := make([]*vectorize.Index, 0, len(files))
	for _, f := range files {
		name := filepath.Base(f)
		name = name[:len(name)-5] // Remove .json
		idx, err := db.GetIndex(ctx, name)
		if err != nil {
			continue
		}
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

// DeleteIndex removes an index.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	path := db.indexPath(name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return vectorize.ErrIndexNotFound
	}

	delete(db.indexes, name)
	return os.Remove(path)
}

// Insert adds vectors to an index.
func (db *DB) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	idx, err := db.loadIndex(indexName)
	if err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	for _, v := range vectors {
		if len(v.Values) != idx.Index.Dimensions {
			return vectorize.ErrDimensionMismatch
		}
		idx.Vectors[v.ID] = &vectorize.Vector{
			ID:        v.ID,
			Values:    v.Values,
			Namespace: v.Namespace,
			Metadata:  v.Metadata,
		}
	}

	return db.saveIndex(idx)
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	return db.Insert(ctx, indexName, vectors)
}

// Search finds similar vectors.
func (db *DB) Search(ctx context.Context, indexName string, vector []float32, opts *vectorize.SearchOptions) ([]*vectorize.Match, error) {
	if opts == nil {
		opts = &vectorize.SearchOptions{TopK: 10}
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	idx, err := db.loadIndex(indexName)
	if err != nil {
		return nil, err
	}

	if len(vector) != idx.Index.Dimensions {
		return nil, vectorize.ErrDimensionMismatch
	}

	type scored struct {
		id    string
		score float32
		vec   *vectorize.Vector
	}

	var candidates []scored
	for _, v := range idx.Vectors {
		// Apply namespace filter
		if opts.Namespace != "" && v.Namespace != opts.Namespace {
			continue
		}

		// Apply metadata filter
		if len(opts.Filter) > 0 && !matchesFilter(v.Metadata, opts.Filter) {
			continue
		}

		score := vectorize.ComputeScore(vector, v.Values, idx.Index.Metric)
		candidates = append(candidates, scored{id: v.ID, score: score, vec: v})
	}

	// Sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Take top K
	if len(candidates) > opts.TopK {
		candidates = candidates[:opts.TopK]
	}

	matches := make([]*vectorize.Match, 0, len(candidates))
	for _, c := range candidates {
		if opts.ScoreThreshold > 0 && c.score < opts.ScoreThreshold {
			continue
		}

		match := &vectorize.Match{
			ID:    c.id,
			Score: c.score,
		}

		if opts.ReturnValues {
			match.Values = c.vec.Values
		}
		if opts.ReturnMetadata {
			match.Metadata = c.vec.Metadata
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

	idx, err := db.loadIndex(indexName)
	if err != nil {
		return nil, err
	}

	vectors := make([]*vectorize.Vector, 0, len(ids))
	for _, id := range ids {
		if v, ok := idx.Vectors[id]; ok {
			vectors = append(vectors, v)
		}
	}

	return vectors, nil
}

// Delete removes vectors by IDs.
func (db *DB) Delete(ctx context.Context, indexName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	idx, err := db.loadIndex(indexName)
	if err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	for _, id := range ids {
		delete(idx.Vectors, id)
	}

	return db.saveIndex(idx)
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	_, err := os.Stat(db.dataDir)
	return err
}

// Close releases resources.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Save all loaded indexes
	for _, idx := range db.indexes {
		db.saveIndex(idx)
	}

	db.indexes = make(map[string]*indexData)
	return nil
}

func matchesFilter(metadata map[string]any, filter map[string]any) bool {
	for k, expected := range filter {
		actual, ok := metadata[k]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
			return false
		}
	}
	return true
}
