// Package vectorize provides vector index management.
package vectorize

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("index not found")
	ErrNameRequired = errors.New("name is required")
)

// Index represents a vector index.
type Index struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Dimensions  int       `json:"dimensions"`
	Metric      string    `json:"metric"`
	CreatedAt   time.Time `json:"created_at"`
	VectorCount int64     `json:"vector_count"`
}

// Vector represents a vector with metadata.
type Vector struct {
	ID        string                 `json:"id"`
	Values    []float32              `json:"values"`
	Namespace string                 `json:"namespace,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Match represents a query match.
type Match struct {
	ID       string                 `json:"id"`
	Score    float32                `json:"score"`
	Values   []float32              `json:"values,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CreateIndexIn contains input for creating an index.
type CreateIndexIn struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Dimensions  int    `json:"dimensions"`
	Metric      string `json:"metric"`
}

// QueryIn contains input for vector queries.
type QueryIn struct {
	Vector         []float32              `json:"vector"`
	TopK           int                    `json:"topK"`
	Namespace      string                 `json:"namespace,omitempty"`
	ReturnValues   bool                   `json:"returnValues"`
	ReturnMetadata string                 `json:"returnMetadata"`
	Filter         map[string]interface{} `json:"filter,omitempty"`
}

// API defines the Vectorize service contract.
type API interface {
	CreateIndex(ctx context.Context, in *CreateIndexIn) (*Index, error)
	GetIndex(ctx context.Context, name string) (*Index, error)
	ListIndexes(ctx context.Context) ([]*Index, error)
	DeleteIndex(ctx context.Context, name string) error
	InsertVectors(ctx context.Context, indexName string, vectors []*Vector) error
	UpsertVectors(ctx context.Context, indexName string, vectors []*Vector) error
	Query(ctx context.Context, indexName string, in *QueryIn) ([]*Match, error)
	GetByIDs(ctx context.Context, indexName string, ids []string) ([]*Vector, error)
	DeleteByIDs(ctx context.Context, indexName string, ids []string) error
}

// Store defines the data access contract.
type Store interface {
	CreateIndex(ctx context.Context, index *Index) error
	GetIndex(ctx context.Context, name string) (*Index, error)
	ListIndexes(ctx context.Context) ([]*Index, error)
	DeleteIndex(ctx context.Context, name string) error
	Insert(ctx context.Context, indexName string, vectors []*Vector) error
	Upsert(ctx context.Context, indexName string, vectors []*Vector) error
	Query(ctx context.Context, indexName string, vector []float32, opts *QueryOpts) ([]*Match, error)
	GetByIDs(ctx context.Context, indexName string, ids []string) ([]*Vector, error)
	DeleteByIDs(ctx context.Context, indexName string, ids []string) error
}

// QueryOpts for vector queries.
type QueryOpts struct {
	TopK           int
	Namespace      string
	ReturnValues   bool
	ReturnMetadata string
	Filter         map[string]interface{}
}
