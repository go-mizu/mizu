package collaborators

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("collaborator not found")
	ErrExists       = errors.New("collaborator already exists")
	ErrInvalidInput = errors.New("invalid input")
	ErrAccessDenied = errors.New("access denied")
	ErrSelfRemove   = errors.New("cannot remove yourself as owner")
)

// Permission levels
const (
	PermissionRead     = "read"
	PermissionTriage   = "triage"
	PermissionWrite    = "write"
	PermissionMaintain = "maintain"
	PermissionAdmin    = "admin"
)

// Collaborator represents a repository collaborator
// Uses composite PK (repo_id, user_id) - no ID field
type Collaborator struct {
	RepoID     string    `json:"repo_id"`
	UserID     string    `json:"user_id"`
	Permission string    `json:"permission"`
	CreatedAt  time.Time `json:"created_at"`
}

// AddIn is the input for adding a collaborator
type AddIn struct {
	UserID     string `json:"user_id"`
	Permission string `json:"permission"`
}

// ListOpts are options for listing collaborators
type ListOpts struct {
	Limit  int
	Offset int
}

// API is the collaborators service interface
type API interface {
	Add(ctx context.Context, repoID, userID string, permission string) error
	Remove(ctx context.Context, repoID, userID string) error
	Update(ctx context.Context, repoID, userID string, permission string) error
	Get(ctx context.Context, repoID, userID string) (*Collaborator, error)
	List(ctx context.Context, repoID string, opts *ListOpts) ([]*Collaborator, error)
	ListUserRepos(ctx context.Context, userID string, opts *ListOpts) ([]string, error)
	GetPermission(ctx context.Context, repoID, userID string) (string, error)
	HasPermission(ctx context.Context, repoID, userID string, required string) (bool, error)
}

// Store is the collaborators data store interface
type Store interface {
	Create(ctx context.Context, c *Collaborator) error
	Delete(ctx context.Context, repoID, userID string) error
	Update(ctx context.Context, c *Collaborator) error
	Get(ctx context.Context, repoID, userID string) (*Collaborator, error)
	List(ctx context.Context, repoID string, limit, offset int) ([]*Collaborator, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]string, error)
}
