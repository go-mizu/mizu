package stars

import (
	"context"
	"errors"
	"time"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound = errors.New("not found")
)

// Stargazer represents a user who starred a repo
type Stargazer struct {
	StarredAt time.Time         `json:"starred_at"`
	User      *users.SimpleUser `json:"user"`
}

// StarredRepo represents a starred repository
type StarredRepo struct {
	StarredAt  time.Time   `json:"starred_at"`
	Repository *Repository `json:"repo"`
}

// Repository is a minimal repo reference
type Repository struct {
	ID          int64             `json:"id"`
	NodeID      string            `json:"node_id"`
	Name        string            `json:"name"`
	FullName    string            `json:"full_name"`
	Owner       *users.SimpleUser `json:"owner"`
	Private     bool              `json:"private"`
	HTMLURL     string            `json:"html_url"`
	Description string            `json:"description,omitempty"`
	URL         string            `json:"url"`
}

// ListOpts contains pagination options
type ListOpts struct {
	Page      int    `json:"page,omitempty"`
	PerPage   int    `json:"per_page,omitempty"`
	Sort      string `json:"sort,omitempty"`      // created, updated
	Direction string `json:"direction,omitempty"` // asc, desc
}

// API defines the stars service interface
type API interface {
	// ListStargazers returns users who starred a repo
	ListStargazers(ctx context.Context, owner, repo string, opts *ListOpts) ([]*users.SimpleUser, error)

	// ListStargazersWithTimestamps returns stargazers with timestamps
	ListStargazersWithTimestamps(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Stargazer, error)

	// ListForAuthenticatedUser returns starred repos for authenticated user
	ListForAuthenticatedUser(ctx context.Context, userID int64, opts *ListOpts) ([]*Repository, error)

	// ListForAuthenticatedUserWithTimestamps returns starred repos with timestamps
	ListForAuthenticatedUserWithTimestamps(ctx context.Context, userID int64, opts *ListOpts) ([]*StarredRepo, error)

	// ListForUser returns starred repos for a user
	ListForUser(ctx context.Context, username string, opts *ListOpts) ([]*Repository, error)

	// ListForUserWithTimestamps returns starred repos with timestamps
	ListForUserWithTimestamps(ctx context.Context, username string, opts *ListOpts) ([]*StarredRepo, error)

	// IsStarred checks if a repo is starred by the authenticated user
	IsStarred(ctx context.Context, userID int64, owner, repo string) (bool, error)

	// Star stars a repo
	Star(ctx context.Context, userID int64, owner, repo string) error

	// Unstar unstars a repo
	Unstar(ctx context.Context, userID int64, owner, repo string) error
}

// Store defines the data access interface for stars
type Store interface {
	Create(ctx context.Context, userID, repoID int64) error
	Delete(ctx context.Context, userID, repoID int64) error
	Exists(ctx context.Context, userID, repoID int64) (bool, error)
	ListStargazers(ctx context.Context, repoID int64, opts *ListOpts) ([]*users.SimpleUser, error)
	ListStargazersWithTimestamps(ctx context.Context, repoID int64, opts *ListOpts) ([]*Stargazer, error)
	ListStarredRepos(ctx context.Context, userID int64, opts *ListOpts) ([]*Repository, error)
	ListStarredReposWithTimestamps(ctx context.Context, userID int64, opts *ListOpts) ([]*StarredRepo, error)
}
