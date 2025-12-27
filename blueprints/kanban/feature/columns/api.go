// Package columns provides kanban column management functionality.
package columns

import (
	"context"
)

// Column represents a kanban board column.
type Column struct {
	ID         string `json:"id"`
	ProjectID  string `json:"project_id"`
	Name       string `json:"name"`
	Position   int    `json:"position"`
	IsDefault  bool   `json:"is_default"`
	IsArchived bool   `json:"is_archived"`
}

// CreateIn contains input for creating a column.
type CreateIn struct {
	Name      string `json:"name"`
	Position  int    `json:"position,omitempty"`
	IsDefault bool   `json:"is_default,omitempty"`
}

// UpdateIn contains input for updating a column.
type UpdateIn struct {
	Name *string `json:"name,omitempty"`
}

// API defines the columns service contract.
type API interface {
	Create(ctx context.Context, projectID string, in *CreateIn) (*Column, error)
	GetByID(ctx context.Context, id string) (*Column, error)
	ListByProject(ctx context.Context, projectID string) ([]*Column, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Column, error)
	UpdatePosition(ctx context.Context, id string, position int) error
	SetDefault(ctx context.Context, projectID, columnID string) error
	Archive(ctx context.Context, id string) error
	Unarchive(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	GetDefault(ctx context.Context, projectID string) (*Column, error)
}

// Store defines the data access contract for columns.
type Store interface {
	Create(ctx context.Context, c *Column) error
	GetByID(ctx context.Context, id string) (*Column, error)
	ListByProject(ctx context.Context, projectID string) ([]*Column, error)
	CountByProject(ctx context.Context, projectID string) (int, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	UpdatePosition(ctx context.Context, id string, position int) error
	SetDefault(ctx context.Context, projectID, columnID string) error
	Archive(ctx context.Context, id string) error
	Unarchive(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	GetDefault(ctx context.Context, projectID string) (*Column, error)
}
