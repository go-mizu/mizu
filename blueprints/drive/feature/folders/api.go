// Package folders provides folder management.
package folders

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("folder not found")
	ErrNameTaken     = errors.New("folder name already exists")
	ErrInvalidParent = errors.New("invalid parent folder")
	ErrCannotMove    = errors.New("cannot move folder into itself")
	ErrNotOwner      = errors.New("not folder owner")
)

// Folder represents a folder.
type Folder struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Depth     int       `json:"depth"`
	Color     string    `json:"color,omitempty"`
	IsRoot    bool      `json:"is_root"`
	Starred   bool      `json:"starred"`
	Trashed   bool      `json:"trashed"`
	TrashedAt *time.Time `json:"trashed_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateIn contains folder creation input.
type CreateIn struct {
	Name     string `json:"name"`
	ParentID string `json:"parent_id,omitempty"`
	Color    string `json:"color,omitempty"`
}

// UpdateIn contains folder update input.
type UpdateIn struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
}

// ListIn contains folder listing input.
type ListIn struct {
	ParentID string
	Starred  *bool
	Trashed  bool
	Limit    int
	Offset   int
	OrderBy  string
	Order    string
}

// TreeNode represents a folder in the tree.
type TreeNode struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Children []*TreeNode `json:"children,omitempty"`
}

// API defines the folders service contract.
type API interface {
	Create(ctx context.Context, ownerID string, in *CreateIn) (*Folder, error)
	GetByID(ctx context.Context, id string) (*Folder, error)
	GetRoot(ctx context.Context, ownerID string) (*Folder, error)
	List(ctx context.Context, ownerID string, in *ListIn) ([]*Folder, error)
	Update(ctx context.Context, id, ownerID string, in *UpdateIn) (*Folder, error)
	Move(ctx context.Context, id, ownerID, newParentID string) (*Folder, error)
	Copy(ctx context.Context, id, ownerID, destParentID string) (*Folder, error)
	Delete(ctx context.Context, id, ownerID string) error
	Star(ctx context.Context, id, ownerID string, starred bool) error
	GetTree(ctx context.Context, ownerID string, rootID string) (*TreeNode, error)
	EnsureRoot(ctx context.Context, ownerID string) (*Folder, error)
}

// Store defines the data access contract.
type Store interface {
	Create(ctx context.Context, f *Folder) error
	GetByID(ctx context.Context, id string) (*Folder, error)
	GetByOwnerAndParentAndName(ctx context.Context, ownerID, parentID, name string) (*Folder, error)
	GetRoot(ctx context.Context, ownerID string) (*Folder, error)
	List(ctx context.Context, ownerID string, in *ListIn) ([]*Folder, error)
	ListByParent(ctx context.Context, ownerID, parentID string) ([]*Folder, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	UpdatePath(ctx context.Context, id, path string, depth int) error
	UpdateParent(ctx context.Context, id, parentID, path string, depth int) error
	UpdateTrashed(ctx context.Context, id string, trashed bool) error
	UpdateStarred(ctx context.Context, id string, starred bool) error
	Delete(ctx context.Context, id string) error
	ListDescendants(ctx context.Context, id string) ([]*Folder, error)
}
