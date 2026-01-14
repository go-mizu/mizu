// Package vectorize provides a unified interface for vector database operations.
// It follows the database/sql pattern with a driver registry and connection pooling.
package vectorize

import (
	"context"
	"time"
)

// DistanceMetric defines supported distance metrics for vector similarity.
type DistanceMetric string

const (
	// Cosine similarity measures the cosine of the angle between vectors.
	// Range: [-1, 1] where 1 means identical direction.
	Cosine DistanceMetric = "cosine"

	// Euclidean distance (L2) measures straight-line distance.
	// Range: [0, inf) where 0 means identical vectors.
	Euclidean DistanceMetric = "euclidean"

	// DotProduct measures the inner product of vectors.
	// Range: (-inf, inf) where higher means more similar for normalized vectors.
	DotProduct DistanceMetric = "dot_product"
)

// Index represents a vector index/collection configuration.
type Index struct {
	// Name is the unique identifier for the index.
	Name string `json:"name"`

	// Dimensions is the number of dimensions in each vector.
	Dimensions int `json:"dimensions"`

	// Metric is the distance metric used for similarity search.
	Metric DistanceMetric `json:"metric"`

	// Description is an optional human-readable description.
	Description string `json:"description,omitempty"`

	// VectorCount is the number of vectors stored in the index.
	VectorCount int64 `json:"vector_count"`

	// CreatedAt is when the index was created.
	CreatedAt time.Time `json:"created_at"`
}

// Vector represents a vector with optional metadata.
type Vector struct {
	// ID is the unique identifier for the vector.
	ID string `json:"id"`

	// Values is the vector embedding as float32 array.
	Values []float32 `json:"values"`

	// Namespace is an optional partition key for multi-tenant scenarios.
	Namespace string `json:"namespace,omitempty"`

	// Metadata is optional key-value data associated with the vector.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Match represents a search result with similarity score.
type Match struct {
	// ID is the unique identifier of the matched vector.
	ID string `json:"id"`

	// Score is the similarity score (interpretation depends on metric).
	Score float32 `json:"score"`

	// Values is the vector values (only if ReturnValues was true).
	Values []float32 `json:"values,omitempty"`

	// Metadata is the vector metadata (only if ReturnMetadata was true).
	Metadata map[string]any `json:"metadata,omitempty"`
}

// SearchOptions configures vector search behavior.
type SearchOptions struct {
	// TopK is the maximum number of results to return.
	TopK int `json:"top_k"`

	// Namespace filters results to a specific namespace.
	Namespace string `json:"namespace,omitempty"`

	// Filter is a metadata filter (database-specific syntax).
	Filter map[string]any `json:"filter,omitempty"`

	// ReturnValues includes vector values in results if true.
	ReturnValues bool `json:"return_values"`

	// ReturnMetadata includes metadata in results if true.
	ReturnMetadata bool `json:"return_metadata"`

	// ScoreThreshold filters results below this score (0 means no threshold).
	ScoreThreshold float32 `json:"score_threshold,omitempty"`
}

// DB is the main interface for vector database operations.
// It provides a unified API across different vector database backends.
type DB interface {
	// CreateIndex creates a new vector index.
	CreateIndex(ctx context.Context, index *Index) error

	// GetIndex retrieves index information by name.
	GetIndex(ctx context.Context, name string) (*Index, error)

	// ListIndexes returns all indexes.
	ListIndexes(ctx context.Context) ([]*Index, error)

	// DeleteIndex removes an index and all its vectors.
	DeleteIndex(ctx context.Context, name string) error

	// Insert adds vectors to an index (fails if ID exists).
	Insert(ctx context.Context, indexName string, vectors []*Vector) error

	// Upsert adds or updates vectors in an index.
	Upsert(ctx context.Context, indexName string, vectors []*Vector) error

	// Search finds the most similar vectors to the query vector.
	Search(ctx context.Context, indexName string, vector []float32, opts *SearchOptions) ([]*Match, error)

	// Get retrieves vectors by their IDs.
	Get(ctx context.Context, indexName string, ids []string) ([]*Vector, error)

	// Delete removes vectors by their IDs.
	Delete(ctx context.Context, indexName string, ids []string) error

	// Ping verifies the connection is alive.
	Ping(ctx context.Context) error

	// Close releases resources and closes connections.
	Close() error
}

// Driver is the interface that vector database drivers must implement.
// Similar to database/sql/driver.Driver.
type Driver interface {
	// Open returns a new connection to the database.
	// The dsn (data source name) format is driver-specific.
	Open(dsn string) (DB, error)
}
