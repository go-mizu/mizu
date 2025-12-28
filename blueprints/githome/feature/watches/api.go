package watches

import (
	"context"
	"errors"
	"time"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound = errors.New("not found")
)

// Subscription represents a watch subscription
type Subscription struct {
	Subscribed    bool      `json:"subscribed"`
	Ignored       bool      `json:"ignored"`
	Reason        string    `json:"reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	URL           string    `json:"url"`
	RepositoryURL string    `json:"repository_url"`
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
	Page    int `json:"page,omitempty"`
	PerPage int `json:"per_page,omitempty"`
}

// API defines the watches service interface
type API interface {
	// ListWatchers returns users watching a repo
	ListWatchers(ctx context.Context, owner, repo string, opts *ListOpts) ([]*users.SimpleUser, error)

	// ListForAuthenticatedUser returns watched repos for authenticated user
	ListForAuthenticatedUser(ctx context.Context, userID int64, opts *ListOpts) ([]*Repository, error)

	// ListForUser returns watched repos for a user
	ListForUser(ctx context.Context, username string, opts *ListOpts) ([]*Repository, error)

	// GetSubscription returns the user's subscription for a repo
	GetSubscription(ctx context.Context, userID int64, owner, repo string) (*Subscription, error)

	// SetSubscription sets the user's subscription for a repo
	SetSubscription(ctx context.Context, userID int64, owner, repo string, subscribed, ignored bool) (*Subscription, error)

	// DeleteSubscription removes the user's subscription for a repo
	DeleteSubscription(ctx context.Context, userID int64, owner, repo string) error
}

// Store defines the data access interface for watches
type Store interface {
	Create(ctx context.Context, userID, repoID int64, subscribed, ignored bool) error
	Update(ctx context.Context, userID, repoID int64, subscribed, ignored bool) error
	Delete(ctx context.Context, userID, repoID int64) error
	Get(ctx context.Context, userID, repoID int64) (*Subscription, error)
	ListWatchers(ctx context.Context, repoID int64, opts *ListOpts) ([]*users.SimpleUser, error)
	ListWatchedRepos(ctx context.Context, userID int64, opts *ListOpts) ([]*Repository, error)
}
