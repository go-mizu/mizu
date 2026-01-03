// Package rows provides database row management.
package rows

import (
	"context"
	"time"
)

// Row represents a single row in a database.
type Row struct {
	ID         string                 `json:"id"`
	DatabaseID string                 `json:"database_id"`
	Properties map[string]interface{} `json:"properties"`
	CreatedBy  string                 `json:"created_by"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedBy  string                 `json:"updated_by"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// Filter represents a single filter condition.
type Filter struct {
	Property string      `json:"property"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// FilterGroup allows AND/OR combinations of filters.
type FilterGroup struct {
	Operator string        `json:"operator"` // "and" | "or"
	Filters  []interface{} `json:"filters"`  // Filter or FilterGroup
}

// Sort represents a sort condition.
type Sort struct {
	Property  string `json:"property"`
	Direction string `json:"direction"` // "asc" | "desc"
}

// ListIn contains input for listing rows.
type ListIn struct {
	DatabaseID string
	Filters    []Filter
	Sorts      []Sort
	Limit      int
	Cursor     string
}

// ListResult contains the result of listing rows.
type ListResult struct {
	Rows       []*Row `json:"rows"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// CreateIn contains input for creating a row.
type CreateIn struct {
	DatabaseID string                 `json:"database_id"`
	Properties map[string]interface{} `json:"properties"`
	CreatedBy  string                 `json:"-"`
}

// UpdateIn contains input for updating a row.
type UpdateIn struct {
	Properties map[string]interface{} `json:"properties"`
	UpdatedBy  string                 `json:"-"`
}

// API defines the rows service contract.
type API interface {
	// CRUD
	Create(ctx context.Context, in *CreateIn) (*Row, error)
	GetByID(ctx context.Context, id string) (*Row, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Row, error)
	Delete(ctx context.Context, id string) error

	// List
	List(ctx context.Context, in *ListIn) (*ListResult, error)

	// Bulk operations
	DeleteByDatabase(ctx context.Context, databaseID string) error
	DuplicateRow(ctx context.Context, id string, userID string) (*Row, error)
}

// Store defines the data access contract for rows.
type Store interface {
	Create(ctx context.Context, row *Row) error
	GetByID(ctx context.Context, id string) (*Row, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, in *ListIn) ([]*Row, error)
	Count(ctx context.Context, databaseID string, filters []Filter) (int, error)
	DeleteByDatabase(ctx context.Context, databaseID string) error
}
