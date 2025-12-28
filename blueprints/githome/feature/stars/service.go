package stars

import (
	"context"
	"time"
)

// Service implements the stars API
type Service struct {
	store Store
}

// NewService creates a new stars service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Star stars a repository
func (s *Service) Star(ctx context.Context, userID, repoID string) error {
	if userID == "" || repoID == "" {
		return ErrInvalidInput
	}

	// Check if already starred
	existing, err := s.store.Get(ctx, userID, repoID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrAlreadyStarred
	}

	star := &Star{
		UserID:    userID,
		RepoID:    repoID,
		CreatedAt: time.Now(),
	}

	return s.store.Create(ctx, star)
}

// Unstar unstars a repository
func (s *Service) Unstar(ctx context.Context, userID, repoID string) error {
	if userID == "" || repoID == "" {
		return ErrInvalidInput
	}
	return s.store.Delete(ctx, userID, repoID)
}

// IsStarred checks if a user has starred a repository
func (s *Service) IsStarred(ctx context.Context, userID, repoID string) (bool, error) {
	star, err := s.store.Get(ctx, userID, repoID)
	if err != nil {
		return false, err
	}
	return star != nil, nil
}

// ListStargazers lists users who starred a repository
func (s *Service) ListStargazers(ctx context.Context, repoID string, opts *ListOpts) ([]*Star, int, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListByRepo(ctx, repoID, limit, offset)
}

// ListStarred lists repositories starred by a user
func (s *Service) ListStarred(ctx context.Context, userID string, opts *ListOpts) ([]*Star, int, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListByUser(ctx, userID, limit, offset)
}

// GetCount gets the star count for a repository
func (s *Service) GetCount(ctx context.Context, repoID string) (int, error) {
	return s.store.Count(ctx, repoID)
}

func (s *Service) getPageParams(opts *ListOpts) (int, int) {
	limit := 30
	offset := 0
	if opts != nil {
		if opts.Limit > 0 && opts.Limit <= 100 {
			limit = opts.Limit
		}
		if opts.Offset >= 0 {
			offset = opts.Offset
		}
	}
	return limit, offset
}
