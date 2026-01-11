// Package bases provides base (database container) management functionality.
package bases

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound = errors.New("base not found")
)

// Base represents a database container within a workspace.
type Base struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	Color       string    `json:"color"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateIn contains input for creating a base.
type CreateIn struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`
}

// UpdateIn contains input for updating a base.
type UpdateIn struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	Color       *string `json:"color,omitempty"`
}

// API defines the bases service interface.
type API interface {
	Create(ctx context.Context, userID string, in CreateIn) (*Base, error)
	GetByID(ctx context.Context, id string) (*Base, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Base, error)
	Delete(ctx context.Context, id string) error
	Duplicate(ctx context.Context, id string, newName string) (*Base, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Base, error)
}

// Store defines the bases data access interface.
type Store interface {
	Create(ctx context.Context, base *Base) error
	GetByID(ctx context.Context, id string) (*Base, error)
	Update(ctx context.Context, base *Base) error
	Delete(ctx context.Context, id string) error
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Base, error)
}
