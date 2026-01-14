// Package mizu_vector provides a pure Go in-memory vector database driver.
// Import this package to register the "mizu_vector" driver.
//
// This driver implements multiple state-of-the-art vector search algorithms:
//   - flat: Brute-force exact search (baseline)
//   - ivf: Inverted File Index with k-means clustering
//   - lsh: Locality Sensitive Hashing with random projections
//   - pq: Product Quantization for memory efficiency
//   - hnsw: Hierarchical Navigable Small World graph
//   - vamana: DiskANN's Vamana graph algorithm
//   - rabitq: RaBitQ binary quantization (SIGMOD 2024)
//   - nsg: Navigating Spreading-out Graph (VLDB 2019)
//   - scann: Google's ScaNN with anisotropic quantization (ICML 2020)
//   - acorn: ACORN-1 filter-aware HNSW (Elasticsearch)
//
// All engines use SIMD-optimized distance functions via viterin/vek.
// Engine selection via DSN: "engine=ivf" or "engine=nsg"
package mizu_vector

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
)

func init() {
	driver.Register("mizu_vector", &Driver{})
}

// EngineType represents the search algorithm engine.
type EngineType string

const (
	EngineFlat   EngineType = "flat"   // Brute-force exact search
	EngineIVF    EngineType = "ivf"    // Inverted File Index
	EngineLSH    EngineType = "lsh"    // Locality Sensitive Hashing
	EnginePQ     EngineType = "pq"     // Product Quantization
	EngineHNSW   EngineType = "hnsw"   // Hierarchical Navigable Small World
	EngineVamana EngineType = "vamana" // DiskANN's Vamana graph
	EngineRaBitQ EngineType = "rabitq" // RaBitQ binary quantization
	EngineNSG    EngineType = "nsg"    // Navigating Spreading-out Graph
	EngineScaNN  EngineType = "scann"  // Google's ScaNN
	EngineACORN  EngineType = "acorn"  // ACORN-1 filtered HNSW
)

// Engine is the interface for vector search engines.
type Engine interface {
	// Name returns the engine name.
	Name() string

	// Build builds the index from vectors.
	Build(vectors map[string]*vectorize.Vector, dims int, metric vectorize.DistanceMetric)

	// Insert adds vectors to the index.
	Insert(vectors []*vectorize.Vector)

	// Delete removes vectors from the index.
	Delete(ids []string)

	// Search finds the k nearest neighbors.
	Search(query []float32, k int) []SearchResult

	// NeedsRebuild returns true if the index needs rebuilding.
	NeedsRebuild() bool

	// SetNeedsRebuild marks the index for rebuilding.
	SetNeedsRebuild(v bool)
}

// SearchResult holds a search result with ID and distance.
type SearchResult struct {
	ID       string
	Distance float32
}

// Driver implements vectorize.Driver for mizu_vector.
type Driver struct{}

// Open creates a new mizu_vector database.
// DSN format: "engine=ivf" or "engine=rabitq&nprobe=16"
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	engine := EngineIVF // Default engine

	// Parse DSN for engine selection
	if dsn != "" && dsn != ":memory:" {
		params, err := url.ParseQuery(dsn)
		if err == nil {
			if e := params.Get("engine"); e != "" {
				engine = EngineType(e)
			}
		}
	}

	return &DB{
		indexes:    make(map[string]*Index),
		engineType: engine,
	}, nil
}

// DB implements vectorize.DB for mizu_vector.
type DB struct {
	mu         sync.RWMutex
	indexes    map[string]*Index
	engineType EngineType
}

// Index represents a vector index with a specific engine.
type Index struct {
	mu         sync.RWMutex
	info       *vectorize.Index
	vectors    map[string]*vectorize.Vector
	namespaces map[string]map[string]struct{}
	engine     Engine
	distFunc   DistanceFunc
}

// CreateIndex creates a new index with the configured engine.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.indexes[index.Name]; exists {
		return vectorize.ErrIndexExists
	}

	// Select distance function (use loop-unrolled versions for best performance)
	var distFunc DistanceFunc
	switch index.Metric {
	case vectorize.Cosine:
		distFunc = CosineDistance
	case vectorize.Euclidean:
		distFunc = EuclideanDistance
	case vectorize.DotProduct:
		distFunc = NegDotProduct
	default:
		distFunc = CosineDistance
	}

	// Create engine based on type
	var engine Engine
	switch db.engineType {
	case EngineFlat:
		engine = NewFlatEngine(distFunc)
	case EngineIVF:
		engine = NewIVFEngine(distFunc)
	case EngineLSH:
		engine = NewLSHEngine(distFunc, index.Dimensions)
	case EnginePQ:
		engine = NewPQEngine(distFunc, index.Dimensions)
	case EngineHNSW:
		engine = NewHNSWEngine(distFunc)
	case EngineVamana:
		engine = NewVamanaEngine(distFunc)
	case EngineRaBitQ:
		engine = NewRaBitQEngine(distFunc, index.Dimensions)
	case EngineNSG:
		engine = NewNSGEngine(distFunc)
	case EngineScaNN:
		engine = NewScaNNEngine(distFunc)
	case EngineACORN:
		engine = NewACORNEngine(distFunc)
	default:
		engine = NewIVFEngine(distFunc)
	}

	db.indexes[index.Name] = &Index{
		info: &vectorize.Index{
			Name:        index.Name,
			Dimensions:  index.Dimensions,
			Metric:      index.Metric,
			Description: index.Description,
			CreatedAt:   time.Now(),
		},
		vectors:    make(map[string]*vectorize.Vector),
		namespaces: make(map[string]map[string]struct{}),
		engine:     engine,
		distFunc:   distFunc,
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

		if v.Namespace != "" {
			if idx.namespaces[v.Namespace] == nil {
				idx.namespaces[v.Namespace] = make(map[string]struct{})
			}
			idx.namespaces[v.Namespace][v.ID] = struct{}{}
		}
	}

	// Mark engine for rebuild
	idx.engine.SetNeedsRebuild(true)

	return nil
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

	db.mu.RLock()
	idx, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return nil, vectorize.ErrIndexNotFound
	}

	if len(vector) != idx.info.Dimensions {
		return nil, vectorize.ErrDimensionMismatch
	}

	// Rebuild index if needed (lazy build)
	idx.mu.Lock()
	if idx.engine.NeedsRebuild() && len(idx.vectors) > 0 {
		idx.engine.Build(idx.vectors, idx.info.Dimensions, idx.info.Metric)
		idx.engine.SetNeedsRebuild(false)
	}
	idx.mu.Unlock()

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(idx.vectors) == 0 {
		return []*vectorize.Match{}, nil
	}

	// Search using engine
	k := opts.TopK * 2 // Get extra for filtering
	results := idx.engine.Search(vector, k)

	// Convert to matches with filtering
	matches := make([]*vectorize.Match, 0, opts.TopK)
	for _, r := range results {
		v := idx.vectors[r.ID]
		if v == nil {
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

		// Convert distance to score
		var score float32
		switch idx.info.Metric {
		case vectorize.Cosine:
			score = 1.0 - r.Distance
		case vectorize.Euclidean:
			score = 1.0 / (1.0 + r.Distance)
		case vectorize.DotProduct:
			score = -r.Distance
		default:
			score = 1.0 - r.Distance
		}

		if opts.ScoreThreshold > 0 && score < opts.ScoreThreshold {
			continue
		}

		match := &vectorize.Match{
			ID:    r.ID,
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
			if v.Namespace != "" {
				if nsIDs, ok := idx.namespaces[v.Namespace]; ok {
					delete(nsIDs, id)
				}
			}
			delete(idx.vectors, id)
		}
	}

	idx.engine.Delete(ids)
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
	db.indexes = make(map[string]*Index)
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
