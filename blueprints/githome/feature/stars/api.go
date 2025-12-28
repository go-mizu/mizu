package stars

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound       = errors.New("star not found")
	ErrAlreadyStarred = errors.New("repository already starred")
	ErrInvalidInput   = errors.New("invalid input")
)

// Star represents a repository star
type Star struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	RepoID    string    `json:"repo_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ListOpts are options for listing stars
type ListOpts struct {
	Sort      string // created
	Direction string // asc, desc
	Limit     int
	Offset    int
}

// API is the stars service interface
type API interface {
	Star(ctx context.Context, userID, repoID string) error
	Unstar(ctx context.Context, userID, repoID string) error
	IsStarred(ctx context.Context, userID, repoID string) (bool, error)
	ListStargazers(ctx context.Context, repoID string, opts *ListOpts) ([]*Star, int, error)
	ListStarred(ctx context.Context, userID string, opts *ListOpts) ([]*Star, int, error)
	GetCount(ctx context.Context, repoID string) (int, error)
}

// Store is the stars data store interface
type Store interface {
	Create(ctx context.Context, s *Star) error
	Delete(ctx context.Context, userID, repoID string) error
	Get(ctx context.Context, userID, repoID string) (*Star, error)
	ListByRepo(ctx context.Context, repoID string, limit, offset int) ([]*Star, int, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*Star, int, error)
	Count(ctx context.Context, repoID string) (int, error)
}
