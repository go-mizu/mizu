package repos

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("repository not found")
	ErrExists       = errors.New("repository already exists")
	ErrInvalidInput = errors.New("invalid input")
	ErrAccessDenied = errors.New("access denied")
	ErrMissingName  = errors.New("repository name is required")
)

// Permission levels
type Permission string

const (
	PermissionRead     Permission = "read"
	PermissionTriage   Permission = "triage"
	PermissionWrite    Permission = "write"
	PermissionMaintain Permission = "maintain"
	PermissionAdmin    Permission = "admin"
)

// Repository represents a Git repository
type Repository struct {
	ID             string    `json:"id"`
	OwnerActorID   string    `json:"owner_actor_id"`           // FK to actors table
	OwnerID        string    `json:"owner_id,omitempty"`       // Populated from actor join
	OwnerType      string    `json:"owner_type,omitempty"`     // user or org, from actor
	OwnerName      string    `json:"owner_name,omitempty"`     // Populated from user/org join
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description"`
	Website        string    `json:"website"`
	DefaultBranch  string    `json:"default_branch"`
	IsPrivate      bool      `json:"is_private"`
	IsArchived     bool      `json:"is_archived"`
	IsTemplate     bool      `json:"is_template"`
	IsFork         bool      `json:"is_fork"`
	ForkedFromID   string    `json:"forked_from_id,omitempty"` // forked_from_repo_id
	StarCount      int       `json:"star_count"`
	ForkCount      int       `json:"fork_count"`
	WatcherCount   int       `json:"watcher_count"`
	OpenIssueCount int       `json:"open_issue_count"`
	OpenPRCount    int       `json:"open_pr_count"`
	SizeKB         int       `json:"size_kb"`
	Topics         []string  `json:"topics"`
	License        string    `json:"license"`
	HasIssues      bool      `json:"has_issues"`
	HasWiki        bool      `json:"has_wiki"`
	HasProjects    bool      `json:"has_projects"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	PushedAt       time.Time `json:"pushed_at,omitempty"`
}

// Collaborator represents a repository collaborator
// Uses composite PK (repo_id, user_id) - no ID field
type Collaborator struct {
	RepoID     string     `json:"repo_id"`
	UserID     string     `json:"user_id"`
	Permission Permission `json:"permission"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Star represents a repository star
// Uses composite PK (user_id, repo_id) - no ID field
type Star struct {
	UserID    string    `json:"user_id"`
	RepoID    string    `json:"repo_id"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateIn is the input for creating a repository
type CreateIn struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	IsPrivate     bool     `json:"is_private"`
	AutoInit      bool     `json:"auto_init"`
	DefaultBranch string   `json:"default_branch"`
	License       string   `json:"license"`
	Topics        []string `json:"topics"`
}

// UpdateIn is the input for updating a repository
type UpdateIn struct {
	Name          *string   `json:"name,omitempty"`
	Description   *string   `json:"description,omitempty"`
	Website       *string   `json:"website,omitempty"`
	IsPrivate     *bool     `json:"is_private,omitempty"`
	IsArchived    *bool     `json:"is_archived,omitempty"`
	IsTemplate    *bool     `json:"is_template,omitempty"`
	DefaultBranch *string   `json:"default_branch,omitempty"`
	HasIssues     *bool     `json:"has_issues,omitempty"`
	HasWiki       *bool     `json:"has_wiki,omitempty"`
	HasProjects   *bool     `json:"has_projects,omitempty"`
	Topics        *[]string `json:"topics,omitempty"`
}

// ForkIn is the input for forking a repository
type ForkIn struct {
	Name string `json:"name,omitempty"`
}

// ListOpts are options for listing repositories
type ListOpts struct {
	Type      string // all, public, private, forks, sources, member
	Sort      string // created, updated, pushed, full_name
	Direction string // asc, desc
	Limit     int
	Offset    int
}

// API is the repositories service interface
type API interface {
	// CRUD
	Create(ctx context.Context, ownerID string, in *CreateIn) (*Repository, error)
	GetByID(ctx context.Context, id string) (*Repository, error)
	GetByOwnerAndName(ctx context.Context, ownerID, ownerType, name string) (*Repository, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Repository, error)
	Delete(ctx context.Context, id string) error

	// Listing
	ListByOwner(ctx context.Context, ownerID, ownerType string, opts *ListOpts) ([]*Repository, error)
	ListPublic(ctx context.Context, opts *ListOpts) ([]*Repository, error)
	ListAccessible(ctx context.Context, userID string, opts *ListOpts) ([]*Repository, error)

	// Collaborators
	AddCollaborator(ctx context.Context, repoID, userID string, perm Permission) error
	RemoveCollaborator(ctx context.Context, repoID, userID string) error
	GetPermission(ctx context.Context, repoID, userID string) (Permission, error)
	CanAccess(ctx context.Context, repoID, userID string, required Permission) bool
	ListCollaborators(ctx context.Context, repoID string) ([]*Collaborator, error)

	// Stars
	Star(ctx context.Context, userID, repoID string) error
	Unstar(ctx context.Context, userID, repoID string) error
	IsStarred(ctx context.Context, userID, repoID string) (bool, error)
	ListStarred(ctx context.Context, userID string, opts *ListOpts) ([]*Repository, error)

	// Forking
	Fork(ctx context.Context, userID, repoID string, in *ForkIn) (*Repository, error)
	ListForks(ctx context.Context, repoID string, opts *ListOpts) ([]*Repository, error)
}

// Store is the repositories data store interface
type Store interface {
	Create(ctx context.Context, r *Repository) error
	GetByID(ctx context.Context, id string) (*Repository, error)
	GetByOwnerAndName(ctx context.Context, ownerID, ownerType, name string) (*Repository, error)
	Update(ctx context.Context, r *Repository) error
	Delete(ctx context.Context, id string) error
	ListByOwner(ctx context.Context, ownerID, ownerType string, limit, offset int) ([]*Repository, error)
	ListPublic(ctx context.Context, limit, offset int) ([]*Repository, error)
	ListByIDs(ctx context.Context, ids []string) ([]*Repository, error)

	// Collaborators
	AddCollaborator(ctx context.Context, c *Collaborator) error
	RemoveCollaborator(ctx context.Context, repoID, userID string) error
	GetCollaborator(ctx context.Context, repoID, userID string) (*Collaborator, error)
	ListCollaborators(ctx context.Context, repoID string) ([]*Collaborator, error)

	// Stars
	Star(ctx context.Context, s *Star) error
	Unstar(ctx context.Context, userID, repoID string) error
	IsStarred(ctx context.Context, userID, repoID string) (bool, error)
	ListStarredByUser(ctx context.Context, userID string, limit, offset int) ([]*Repository, error)
}
