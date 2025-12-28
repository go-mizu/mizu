package stars

import (
	"context"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Service implements the stars API
type Service struct {
	store     Store
	repoStore repos.Store
	userStore users.Store
	baseURL   string
}

// NewService creates a new stars service
func NewService(store Store, repoStore repos.Store, userStore users.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		userStore: userStore,
		baseURL:   baseURL,
	}
}

// ListStargazers returns users who starred a repo
func (s *Service) ListStargazers(ctx context.Context, owner, repo string, opts *ListOpts) ([]*users.SimpleUser, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.ListStargazers(ctx, r.ID, opts)
}

// ListStargazersWithTimestamps returns stargazers with timestamps
func (s *Service) ListStargazersWithTimestamps(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Stargazer, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.ListStargazersWithTimestamps(ctx, r.ID, opts)
}

// ListForAuthenticatedUser returns starred repos for authenticated user
func (s *Service) ListForAuthenticatedUser(ctx context.Context, userID int64, opts *ListOpts) ([]*Repository, error) {
	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.ListStarredRepos(ctx, userID, opts)
}

// ListForAuthenticatedUserWithTimestamps returns starred repos with timestamps
func (s *Service) ListForAuthenticatedUserWithTimestamps(ctx context.Context, userID int64, opts *ListOpts) ([]*StarredRepo, error) {
	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.ListStarredReposWithTimestamps(ctx, userID, opts)
}

// ListForUser returns starred repos for a user
func (s *Service) ListForUser(ctx context.Context, username string, opts *ListOpts) ([]*Repository, error) {
	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.ListStarredRepos(ctx, user.ID, opts)
}

// ListForUserWithTimestamps returns starred repos with timestamps
func (s *Service) ListForUserWithTimestamps(ctx context.Context, username string, opts *ListOpts) ([]*StarredRepo, error) {
	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.ListStarredReposWithTimestamps(ctx, user.ID, opts)
}

// IsStarred checks if a repo is starred by the authenticated user
func (s *Service) IsStarred(ctx context.Context, userID int64, owner, repo string) (bool, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return false, err
	}
	if r == nil {
		return false, repos.ErrNotFound
	}

	return s.store.Exists(ctx, userID, r.ID)
}

// Star stars a repo
func (s *Service) Star(ctx context.Context, userID int64, owner, repo string) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	// Check if already starred
	exists, err := s.store.Exists(ctx, userID, r.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil // Already starred, no-op
	}

	if err := s.store.Create(ctx, userID, r.ID); err != nil {
		return err
	}

	// Increment stargazers count
	return s.repoStore.IncrementStargazers(ctx, r.ID, 1)
}

// Unstar unstars a repo
func (s *Service) Unstar(ctx context.Context, userID int64, owner, repo string) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	// Check if starred
	exists, err := s.store.Exists(ctx, userID, r.ID)
	if err != nil {
		return err
	}
	if !exists {
		return nil // Not starred, no-op
	}

	if err := s.store.Delete(ctx, userID, r.ID); err != nil {
		return err
	}

	// Decrement stargazers count
	return s.repoStore.IncrementStargazers(ctx, r.ID, -1)
}
