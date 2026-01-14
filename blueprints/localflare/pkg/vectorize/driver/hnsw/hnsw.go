// Package hnsw provides a high-performance in-memory driver for the vectorize package.
// Import this package to register the "hnsw" driver.
// This driver uses HNSW (Hierarchical Navigable Small World) for O(log n) approximate nearest neighbor search.
package hnsw

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coder/hnsw"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
)

func init() {
	driver.Register("hnsw", &Driver{})
}

// Driver implements vectorize.Driver for in-memory storage with HNSW indexing.
type Driver struct{}

// Open creates a new in-memory database.
// DSN is ignored for this driver.
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	return &DB{
		indexes: make(map[string]*memIndex),
	}, nil
}

// memIndex represents an in-memory HNSW vector index.
type memIndex struct {
	mu         sync.RWMutex
	info       *vectorize.Index
	graph      *hnsw.Graph[string]
	vectors    map[string]*vectorize.Vector // Store full vector data
	namespaces map[string]map[string]struct{}
}

// DB implements vectorize.DB for in-memory storage.
type DB struct {
	mu      sync.RWMutex
	indexes map[string]*memIndex
}

// CreateIndex creates a new in-memory HNSW index.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.indexes[index.Name]; exists {
		return vectorize.ErrIndexExists
	}

	// Create HNSW graph and configure it
	g := hnsw.NewGraph[string]()
	g.M = 16       // Connections per node
	g.Ml = 0.25    // Level generation factor
	g.EfSearch = 64 // Search breadth (higher = more accurate but slower)

	// Set distance function based on metric
	switch index.Metric {
	case vectorize.Cosine:
		g.Distance = hnsw.CosineDistance
	case vectorize.Euclidean:
		g.Distance = hnsw.EuclideanDistance
	case vectorize.DotProduct:
		g.Distance = negDotProduct
	default:
		g.Distance = hnsw.CosineDistance
	}

	db.indexes[index.Name] = &memIndex{
		info: &vectorize.Index{
			Name:        index.Name,
			Dimensions:  index.Dimensions,
			Metric:      index.Metric,
			Description: index.Description,
			CreatedAt:   time.Now(),
		},
		graph:      g,
		vectors:    make(map[string]*vectorize.Vector),
		namespaces: make(map[string]map[string]struct{}),
	}

	return nil
}

// negDotProduct returns negative dot product as distance (for similarity search).
func negDotProduct(a, b []float32) float32 {
	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return -sum // Negative so higher dot product = lower distance
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

// Insert adds vectors to an index using HNSW.
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

	// Build nodes for batch insert
	nodes := make([]hnsw.Node[string], 0, len(vectors))

	for _, v := range vectors {
		if len(v.Values) != idx.info.Dimensions {
			return vectorize.ErrDimensionMismatch
		}

		// Create a copy of the vector
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

		// Store full vector data
		idx.vectors[v.ID] = vec

		// Create HNSW node
		nodes = append(nodes, hnsw.MakeNode(v.ID, v.Values))

		// Track namespace
		if v.Namespace != "" {
			if idx.namespaces[v.Namespace] == nil {
				idx.namespaces[v.Namespace] = make(map[string]struct{})
			}
			idx.namespaces[v.Namespace][v.ID] = struct{}{}
		}
	}

	// Batch add to HNSW graph
	idx.graph.Add(nodes...)

	return nil
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
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

	nodes := make([]hnsw.Node[string], 0, len(vectors))

	for _, v := range vectors {
		if len(v.Values) != idx.info.Dimensions {
			return vectorize.ErrDimensionMismatch
		}

		// Delete existing if present
		if _, exists := idx.vectors[v.ID]; exists {
			idx.graph.Delete(v.ID)
		}

		// Create a copy
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
		nodes = append(nodes, hnsw.MakeNode(v.ID, v.Values))

		if v.Namespace != "" {
			if idx.namespaces[v.Namespace] == nil {
				idx.namespaces[v.Namespace] = make(map[string]struct{})
			}
			idx.namespaces[v.Namespace][v.ID] = struct{}{}
		}
	}

	idx.graph.Add(nodes...)
	return nil
}

// Search finds similar vectors using HNSW approximate nearest neighbor search.
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

	// Check if graph is empty
	if len(idx.vectors) == 0 {
		return []*vectorize.Match{}, nil
	}

	// Determine search count - if filtering by namespace, we need more candidates
	searchK := opts.TopK
	if opts.Namespace != "" || len(opts.Filter) > 0 {
		searchK = opts.TopK * 10 // Get more candidates for filtering
		if searchK > len(idx.vectors) {
			searchK = len(idx.vectors)
		}
	}

	// HNSW search - returns results sorted by distance (closest first)
	results := idx.graph.SearchWithDistance(vector, searchK)

	// Convert to matches with filtering
	matches := make([]*vectorize.Match, 0, opts.TopK)
	for _, result := range results {
		v, ok := idx.vectors[result.Key]
		if !ok {
			continue
		}

		// Apply namespace filter
		if opts.Namespace != "" && v.Namespace != opts.Namespace {
			continue
		}

		// Apply metadata filter
		if len(opts.Filter) > 0 && !matchesFilter(v.Metadata, opts.Filter) {
			continue
		}

		// Convert distance to similarity score
		// For cosine distance: similarity = 1 - distance
		// For euclidean: similarity = 1 / (1 + distance)
		// For dot product: similarity = -distance (since we negated it)
		var score float32
		switch idx.info.Metric {
		case vectorize.Cosine:
			score = 1.0 - result.Distance
		case vectorize.Euclidean:
			score = 1.0 / (1.0 + result.Distance)
		case vectorize.DotProduct:
			score = -result.Distance // Undo the negation
		default:
			score = 1.0 - result.Distance
		}

		if opts.ScoreThreshold > 0 && score < opts.ScoreThreshold {
			continue
		}

		match := &vectorize.Match{
			ID:    result.Key,
			Score: score,
		}

		if opts.ReturnValues {
			match.Values = make([]float32, len(v.Values))
			copy(match.Values, v.Values)
		}

		if opts.ReturnMetadata && v.Metadata != nil {
			match.Metadata = make(map[string]any, len(v.Metadata))
			for k, val := range v.Metadata {
				match.Metadata[k] = val
			}
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
			// Delete from HNSW graph
			idx.graph.Delete(id)
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
