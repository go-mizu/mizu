// Package folders provides folder management functionality.
package folders

import (
	"context"
	"time"
)

// Folder represents a folder.
type Folder struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	ParentID    string    `json:"parent_id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Color       string    `json:"color,omitempty"`
	IsStarred   bool      `json:"is_starred"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	TrashedAt   time.Time `json:"trashed_at,omitempty"`
}

// CreateIn contains input for creating a folder.
type CreateIn struct {
	Name        string `json:"name"`
	ParentID    string `json:"parent_id,omitempty"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

// UpdateIn contains input for updating a folder.
type UpdateIn struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
}

// MoveIn contains input for moving a folder.
type MoveIn struct {
	ParentID string `json:"parent_id"`
}

// API defines the folders service contract.
type API interface {
	Create(ctx context.Context, userID string, in *CreateIn) (*Folder, error)
	GetByID(ctx context.Context, id string) (*Folder, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Folder, error)
	Move(ctx context.Context, id string, in *MoveIn) (*Folder, error)
	Delete(ctx context.Context, id string) error
	Trash(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	Star(ctx context.Context, id, userID string) error
	Unstar(ctx context.Context, id, userID string) error
	ListByUser(ctx context.Context, userID string) ([]*Folder, error)
	ListByParent(ctx context.Context, userID, parentID string) ([]*Folder, error)
	ListStarred(ctx context.Context, userID string) ([]*Folder, error)
	ListTrashed(ctx context.Context, userID string) ([]*Folder, error)
	Search(ctx context.Context, userID, query string) ([]*Folder, error)
	GetPath(ctx context.Context, id string) ([]*Folder, error)
}
