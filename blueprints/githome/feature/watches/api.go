package watches

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("watch not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrInvalidLevel = errors.New("invalid watch level")
)

// Watch levels
const (
	LevelWatching     = "watching"
	LevelReleasesOnly = "releases_only"
	LevelIgnoring     = "ignoring"
)

// Watch represents a repository watch subscription
type Watch struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	RepoID    string    `json:"repo_id"`
	Level     string    `json:"level"`
	CreatedAt time.Time `json:"created_at"`
}

// ListOpts are options for listing watches
type ListOpts struct {
	Limit  int
	Offset int
}

// API is the watches service interface
type API interface {
	Watch(ctx context.Context, userID, repoID string, level string) error
	Unwatch(ctx context.Context, userID, repoID string) error
	GetWatchStatus(ctx context.Context, userID, repoID string) (*Watch, error)
	ListWatchers(ctx context.Context, repoID string, opts *ListOpts) ([]*Watch, int, error)
	ListWatching(ctx context.Context, userID string, opts *ListOpts) ([]*Watch, int, error)
	GetCount(ctx context.Context, repoID string) (int, error)
}

// Store is the watches data store interface
type Store interface {
	Create(ctx context.Context, w *Watch) error
	Update(ctx context.Context, w *Watch) error
	Delete(ctx context.Context, userID, repoID string) error
	Get(ctx context.Context, userID, repoID string) (*Watch, error)
	ListByRepo(ctx context.Context, repoID string, limit, offset int) ([]*Watch, int, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*Watch, int, error)
	Count(ctx context.Context, repoID string) (int, error)
}
