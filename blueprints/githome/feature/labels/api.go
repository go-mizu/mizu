package labels

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("label not found")
	ErrExists       = errors.New("label already exists")
	ErrInvalidInput = errors.New("invalid input")
	ErrMissingName  = errors.New("label name is required")
)

// Label represents a repository label
type Label struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateIn is the input for creating a label
type CreateIn struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// UpdateIn is the input for updating a label
type UpdateIn struct {
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

// API is the labels service interface
type API interface {
	Create(ctx context.Context, repoID string, in *CreateIn) (*Label, error)
	GetByID(ctx context.Context, id string) (*Label, error)
	GetByName(ctx context.Context, repoID, name string) (*Label, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Label, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, repoID string) ([]*Label, error)
}

// Store is the labels data store interface
type Store interface {
	Create(ctx context.Context, l *Label) error
	GetByID(ctx context.Context, id string) (*Label, error)
	GetByName(ctx context.Context, repoID, name string) (*Label, error)
	Update(ctx context.Context, l *Label) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, repoID string) ([]*Label, error)
	ListByIDs(ctx context.Context, ids []string) ([]*Label, error)
}
