// Package mem provides an in-memory driver for the vectorize package.
// Import this package to register the "mem" driver.
// This driver uses efficient in-memory indexing with heap-based top-k selection.
package mem

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
)

func init() {
	driver.Register("mem", &Driver{})
}

// Driver implements vectorize.Driver for in-memory storage.
type Driver struct{}

// Open creates a new in-memory database.
// DSN is ignored for this driver.
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	return &DB{
		indexes: make(map[string]*memIndex),
	}, nil
}

// memIndex represents an in-memory vector index.
type memIndex struct {
	mu         sync.RWMutex
	info       *vectorize.Index
	vectors    map[string]*vectorize.Vector
	namespaces map[string]map[string]struct{} // namespace -> set of vector IDs
}

// DB implements vectorize.DB for in-memory storage.
type DB struct {
	mu      sync.RWMutex
	indexes map[string]*memIndex
}

// CreateIndex creates a new in-memory index.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.indexes[index.Name]; exists {
		return vectorize.ErrIndexExists
	}

	db.indexes[index.Name] = &memIndex{
		info: &vectorize.Index{
			Name:        index.Name,
			Dimensions:  index.Dimensions,
			Metric:      index.Metric,
			Description: index.Description,
			CreatedAt:   time.Now(),
		},
		vectors:    make(map[string]*vectorize.Vector),
		namespaces: make(map[string]map[string]struct{}),
	}

	return nil
}

// GetIndex retrieves index information.
func (db *DB) GetIndex(ctx context.Context, name string) (*vectorize.Index, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	idx, ok := db.indexes[name]
	if !ok {
		return nil, vectorize.ErrIndexNotFound
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Return a copy with current vector count
	return &vectorize.Index{
		Name:        idx.info.Name,
		Dimensions:  idx.info.Dimensions,
		Metric:      idx.info.Metric,
		Description: idx.info.Description,
		VectorCount: int64(len(idx.vectors)),
		CreatedAt:   idx.info.CreatedAt,
	}, nil
}

// ListIndexes returns all indexes.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	indexes := make([]*vectorize.Index, 0, len(db.indexes))
	for _, idx := range db.indexes {
		idx.mu.RLock()
		indexes = append(indexes, &vectorize.Index{
			Name:        idx.info.Name,
			Dimensions:  idx.info.Dimensions,
			Metric:      idx.info.Metric,
			Description: idx.info.Description,
			VectorCount: int64(len(idx.vectors)),
			CreatedAt:   idx.info.CreatedAt,
		})
		idx.mu.RUnlock()
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

	delete(db.indexes, name)
	return nil
}

// Insert adds vectors to an index.
func (db *DB) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	db.mu.RLock()
	idx, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return vectorize.ErrIndexNotFound
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	for _, v := range vectors {
		if len(v.Values) != idx.info.Dimensions {
			return vectorize.ErrDimensionMismatch
		}

		// Create a copy to avoid external modification
		vec := &vectorize.Vector{
			ID:        v.ID,
			Values:    make([]float32, len(v.Values)),
			Namespace: v.Namespace,
		}
		copy(vec.Values, v.Values)

		if v.Metadata != nil {
			vec.Metadata = make(map[string]any, len(v.Metadata))
			for k, val := range v.Metadata {
				vec.Metadata[k] = val
			}
		}

		idx.vectors[v.ID] = vec

		// Track namespace
		if v.Namespace != "" {
			if idx.namespaces[v.Namespace] == nil {
				idx.namespaces[v.Namespace] = make(map[string]struct{})
			}
			idx.namespaces[v.Namespace][v.ID] = struct{}{}
		}
	}

	return nil
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	// For in-memory, insert and upsert are the same
	return db.Insert(ctx, indexName, vectors)
}

// scoredVector holds a vector with its similarity score for heap operations.
type scoredVector struct {
	id       string
	score    float32
	vector   *vectorize.Vector
}

// minHeap implements a min-heap for top-k selection.
type minHeap []scoredVector

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].score < h[j].score }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *minHeap) Push(x any) {
	*h = append(*h, x.(scoredVector))
}

func (h *minHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// Search finds similar vectors using heap-based top-k selection.
func (db *DB) Search(ctx context.Context, indexName string, vector []float32, opts *vectorize.SearchOptions) ([]*vectorize.Match, error) {
	if opts == nil {
		opts = &vectorize.SearchOptions{TopK: 10}
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	db.mu.RLock()
	idx, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return nil, vectorize.ErrIndexNotFound
	}

	if len(vector) != idx.info.Dimensions {
		return nil, vectorize.ErrDimensionMismatch
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Use min-heap to efficiently track top-k highest scores
	h := &minHeap{}
	heap.Init(h)

	// Get candidate vectors (filter by namespace if specified)
	var candidates map[string]*vectorize.Vector
	if opts.Namespace != "" {
		candidates = make(map[string]*vectorize.Vector)
		if nsIDs, ok := idx.namespaces[opts.Namespace]; ok {
			for id := range nsIDs {
				if v, ok := idx.vectors[id]; ok {
					candidates[id] = v
				}
			}
		}
	} else {
		candidates = idx.vectors
	}

	for id, v := range candidates {
		// Apply metadata filter
		if len(opts.Filter) > 0 && !matchesFilter(v.Metadata, opts.Filter) {
			continue
		}

		score := vectorize.ComputeScore(vector, v.Values, idx.info.Metric)

		if opts.ScoreThreshold > 0 && score < opts.ScoreThreshold {
			continue
		}

		// Maintain top-k using min-heap
		if h.Len() < opts.TopK {
			heap.Push(h, scoredVector{id: id, score: score, vector: v})
		} else if score > (*h)[0].score {
			heap.Pop(h)
			heap.Push(h, scoredVector{id: id, score: score, vector: v})
		}
	}

	// Extract results in descending order
	results := make([]*vectorize.Match, h.Len())
	for i := len(results) - 1; i >= 0; i-- {
		sv := heap.Pop(h).(scoredVector)
		match := &vectorize.Match{
			ID:    sv.id,
			Score: sv.score,
		}
		if opts.ReturnValues {
			match.Values = make([]float32, len(sv.vector.Values))
			copy(match.Values, sv.vector.Values)
		}
		if opts.ReturnMetadata && sv.vector.Metadata != nil {
			match.Metadata = make(map[string]any, len(sv.vector.Metadata))
			for k, val := range sv.vector.Metadata {
				match.Metadata[k] = val
			}
		}
		results[i] = match
	}

	return results, nil
}

// Get retrieves vectors by IDs.
func (db *DB) Get(ctx context.Context, indexName string, ids []string) ([]*vectorize.Vector, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	db.mu.RLock()
	idx, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return nil, vectorize.ErrIndexNotFound
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	vectors := make([]*vectorize.Vector, 0, len(ids))
	for _, id := range ids {
		if v, ok := idx.vectors[id]; ok {
			// Return a copy
			vec := &vectorize.Vector{
				ID:        v.ID,
				Values:    make([]float32, len(v.Values)),
				Namespace: v.Namespace,
			}
			copy(vec.Values, v.Values)
			if v.Metadata != nil {
				vec.Metadata = make(map[string]any, len(v.Metadata))
				for k, val := range v.Metadata {
					vec.Metadata[k] = val
				}
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

	db.mu.RLock()
	idx, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return vectorize.ErrIndexNotFound
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	for _, id := range ids {
		if v, ok := idx.vectors[id]; ok {
			// Remove from namespace tracking
			if v.Namespace != "" {
				if nsIDs, ok := idx.namespaces[v.Namespace]; ok {
					delete(nsIDs, id)
				}
			}
			delete(idx.vectors, id)
		}
	}

	return nil
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	return nil
}

// Close releases resources.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.indexes = make(map[string]*memIndex)
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
