// Package collections provides collection management functionality.
package collections

import (
	"context"

	"github.com/go-mizu/blueprints/cms/config"
	"github.com/go-mizu/blueprints/cms/store/duckdb"
)

// Document represents a collection document.
type Document struct {
	ID        string         `json:"id"`
	Data      map[string]any `json:"-"`
	Status    string         `json:"_status,omitempty"`
	Version   int            `json:"_version,omitempty"`
	CreatedAt string         `json:"createdAt"`
	UpdatedAt string         `json:"updatedAt"`
}

// FindInput holds input for find operations.
type FindInput struct {
	Where          map[string]any
	Sort           string
	Limit          int
	Page           int
	Depth          int
	Locale         string
	FallbackLocale string
	Select         map[string]bool
	Populate       map[string]bool
}

// FindResult holds the result of a find operation.
type FindResult struct {
	Docs          []map[string]any `json:"docs"`
	TotalDocs     int              `json:"totalDocs"`
	Limit         int              `json:"limit"`
	TotalPages    int              `json:"totalPages"`
	Page          int              `json:"page"`
	PagingCounter int              `json:"pagingCounter"`
	HasPrevPage   bool             `json:"hasPrevPage"`
	HasNextPage   bool             `json:"hasNextPage"`
	PrevPage      *int             `json:"prevPage"`
	NextPage      *int             `json:"nextPage"`
}

// CreateInput holds input for create operations.
type CreateInput struct {
	Data   map[string]any
	Depth  int
	Locale string
	Draft  bool
}

// UpdateInput holds input for update operations.
type UpdateInput struct {
	Data      map[string]any
	Depth     int
	Locale    string
	Draft     bool
	Autosave  bool
	Overwrite bool
}

// DeleteResult holds the result of a delete operation.
type DeleteResult struct {
	ID      string         `json:"id"`
	Deleted bool           `json:"deleted"`
	Doc     map[string]any `json:"doc,omitempty"`
}

// API defines the collections service interface.
type API interface {
	// Find finds documents matching the query.
	Find(ctx context.Context, collection string, input *FindInput) (*FindResult, error)

	// FindByID finds a document by ID.
	FindByID(ctx context.Context, collection, id string, depth int, locale string) (map[string]any, error)

	// Count counts documents matching the query.
	Count(ctx context.Context, collection string, where map[string]any) (int, error)

	// Create creates a new document.
	Create(ctx context.Context, collection string, input *CreateInput) (map[string]any, error)

	// UpdateByID updates a document by ID.
	UpdateByID(ctx context.Context, collection, id string, input *UpdateInput) (map[string]any, error)

	// Update updates documents matching the query.
	Update(ctx context.Context, collection string, where map[string]any, input *UpdateInput) ([]map[string]any, error)

	// DeleteByID deletes a document by ID.
	DeleteByID(ctx context.Context, collection, id string) (*DeleteResult, error)

	// Delete deletes documents matching the query.
	Delete(ctx context.Context, collection string, where map[string]any) ([]DeleteResult, error)

	// GetConfig returns the collection configuration.
	GetConfig(collection string) *config.CollectionConfig
}

// Store defines the store interface for collections.
type Store interface {
	Create(ctx context.Context, collection string, data map[string]any) (*duckdb.Document, error)
	FindByID(ctx context.Context, collection, id string) (*duckdb.Document, error)
	Find(ctx context.Context, collection string, opts *duckdb.FindOptions) (*duckdb.FindResult, error)
	Count(ctx context.Context, collection string, where map[string]any) (int, error)
	UpdateByID(ctx context.Context, collection, id string, data map[string]any) (*duckdb.Document, error)
	Update(ctx context.Context, collection string, where map[string]any, data map[string]any) (int64, error)
	DeleteByID(ctx context.Context, collection, id string) (bool, error)
	Delete(ctx context.Context, collection string, where map[string]any) (int64, error)
}
