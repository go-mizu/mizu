// Package chromem provides an in-memory vector database driver using chromem-go.
// Import this package to register the "chromem" driver.
package chromem

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
	chromemgo "github.com/philippgille/chromem-go"
)

func init() {
	driver.Register("chromem", &Driver{})
}

// Driver implements vectorize.Driver for chromem-go.
type Driver struct{}

// Open creates a new chromem-go database.
// DSN is ignored for this in-memory driver.
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	return &DB{
		db:      chromemgo.NewDB(),
		indexes: make(map[string]*indexInfo),
	}, nil
}

// indexInfo stores index metadata.
type indexInfo struct {
	dimensions  int
	metric      vectorize.DistanceMetric
	description string
}

// DB implements vectorize.DB for chromem-go.
type DB struct {
	mu      sync.RWMutex
	db      *chromemgo.DB
	indexes map[string]*indexInfo
}

// CreateIndex creates a new collection in chromem-go.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.indexes[index.Name]; exists {
		return vectorize.ErrIndexExists
	}

	// chromem-go uses cosine similarity by default
	// We store metadata about the index for our purposes
	_, err := db.db.CreateCollection(index.Name, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	db.indexes[index.Name] = &indexInfo{
		dimensions:  index.Dimensions,
		metric:      index.Metric,
		description: index.Description,
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

	col := db.db.GetCollection(name, nil)
	if col == nil {
		return nil, vectorize.ErrIndexNotFound
	}

	return &vectorize.Index{
		Name:        name,
		Dimensions:  info.dimensions,
		Metric:      info.metric,
		Description: info.description,
		VectorCount: int64(col.Count()),
	}, nil
}

// ListIndexes returns all indexes.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var indexes []*vectorize.Index
	for name, info := range db.indexes {
		col := db.db.GetCollection(name, nil)
		var count int64
		if col != nil {
			count = int64(col.Count())
		}
		indexes = append(indexes, &vectorize.Index{
			Name:        name,
			Dimensions:  info.dimensions,
			Metric:      info.metric,
			Description: info.description,
			VectorCount: count,
		})
	}

	return indexes, nil
}

// DeleteIndex removes a collection.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, ok := db.indexes[name]; !ok {
		return vectorize.ErrIndexNotFound
	}

	if err := db.db.DeleteCollection(name); err != nil {
		return err
	}

	delete(db.indexes, name)
	return nil
}

// Insert adds vectors to a collection.
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

	col := db.db.GetCollection(indexName, nil)
	if col == nil {
		return vectorize.ErrIndexNotFound
	}

	// Convert to chromem documents
	docs := make([]chromemgo.Document, len(vectors))
	for i, v := range vectors {
		if len(v.Values) != info.dimensions {
			return vectorize.ErrDimensionMismatch
		}

		// Convert float32 to float64 for chromem-go
		embedding := make([]float32, len(v.Values))
		copy(embedding, v.Values)

		// Build metadata
		metadata := make(map[string]string)
		if v.Namespace != "" {
			metadata["namespace"] = v.Namespace
		}
		for k, val := range v.Metadata {
			metadata[k] = fmt.Sprintf("%v", val)
		}

		docs[i] = chromemgo.Document{
			ID:        v.ID,
			Embedding: embedding,
			Metadata:  metadata,
			Content:   v.ID, // Use ID as content since we need some content
		}
	}

	// Add documents (chromem-go handles embeddings)
	return col.AddDocuments(ctx, docs, 1) // Use 1 goroutine since we provide embeddings
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	// chromem-go doesn't have explicit upsert, but AddDocuments overwrites by ID
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
	info, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return nil, vectorize.ErrIndexNotFound
	}

	if len(vector) != info.dimensions {
		return nil, vectorize.ErrDimensionMismatch
	}

	col := db.db.GetCollection(indexName, nil)
	if col == nil {
		return nil, vectorize.ErrIndexNotFound
	}

	// Build where filter for namespace
	var where map[string]string
	if opts.Namespace != "" {
		where = map[string]string{"namespace": opts.Namespace}
	}

	// chromem-go QueryEmbedding takes the embedding directly
	results, err := col.QueryEmbedding(ctx, vector, opts.TopK, where, nil)
	if err != nil {
		return nil, err
	}

	matches := make([]*vectorize.Match, len(results))
	for i, r := range results {
		match := &vectorize.Match{
			ID:    r.ID,
			Score: r.Similarity,
		}

		if opts.ReturnValues {
			match.Values = make([]float32, len(r.Embedding))
			copy(match.Values, r.Embedding)
		}

		if opts.ReturnMetadata && len(r.Metadata) > 0 {
			match.Metadata = make(map[string]any, len(r.Metadata))
			for k, v := range r.Metadata {
				if k != "namespace" { // Skip internal namespace key
					match.Metadata[k] = v
				}
			}
		}

		matches[i] = match
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

	col := db.db.GetCollection(indexName, nil)
	if col == nil {
		return nil, vectorize.ErrIndexNotFound
	}

	var vectors []*vectorize.Vector
	for _, id := range ids {
		doc, err := col.GetByID(ctx, id)
		if err != nil {
			continue // Skip missing documents
		}
		if doc.ID == "" {
			continue // Empty document
		}

		vec := &vectorize.Vector{
			ID:     doc.ID,
			Values: make([]float32, len(doc.Embedding)),
		}
		copy(vec.Values, doc.Embedding)

		if ns, ok := doc.Metadata["namespace"]; ok {
			vec.Namespace = ns
		}

		if len(doc.Metadata) > 0 {
			vec.Metadata = make(map[string]any, len(doc.Metadata))
			for k, v := range doc.Metadata {
				if k != "namespace" {
					vec.Metadata[k] = v
				}
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

	db.mu.RLock()
	_, ok := db.indexes[indexName]
	db.mu.RUnlock()

	if !ok {
		return vectorize.ErrIndexNotFound
	}

	col := db.db.GetCollection(indexName, nil)
	if col == nil {
		return vectorize.ErrIndexNotFound
	}

	for _, id := range ids {
		if err := col.Delete(ctx, nil, nil, id); err != nil {
			// Continue deleting others even if one fails
			continue
		}
	}

	return nil
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	return nil // In-memory, always available
}

// Close releases resources.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.indexes = make(map[string]*indexInfo)
	return nil
}
