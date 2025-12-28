package watches

import (
	"context"
	"time"
)

// Service implements the watches API
type Service struct {
	store Store
}

// NewService creates a new watches service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Watch subscribes to a repository
func (s *Service) Watch(ctx context.Context, userID, repoID string, level string) error {
	if userID == "" || repoID == "" {
		return ErrInvalidInput
	}

	// Validate level
	if level != LevelWatching && level != LevelReleasesOnly && level != LevelIgnoring {
		return ErrInvalidLevel
	}

	// Check if already watching
	existing, err := s.store.Get(ctx, userID, repoID)
	if err != nil {
		return err
	}

	if existing != nil {
		// Update existing watch
		existing.Level = level
		return s.store.Update(ctx, existing)
	}

	watch := &Watch{
		UserID:    userID,
		RepoID:    repoID,
		Level:     level,
		CreatedAt: time.Now(),
	}

	return s.store.Create(ctx, watch)
}

// Unwatch unsubscribes from a repository
func (s *Service) Unwatch(ctx context.Context, userID, repoID string) error {
	if userID == "" || repoID == "" {
		return ErrInvalidInput
	}
	return s.store.Delete(ctx, userID, repoID)
}

// GetWatchStatus gets the watch status for a user and repository
func (s *Service) GetWatchStatus(ctx context.Context, userID, repoID string) (*Watch, error) {
	watch, err := s.store.Get(ctx, userID, repoID)
	if err != nil {
		return nil, err
	}
	if watch == nil {
		return nil, ErrNotFound
	}
	return watch, nil
}

// ListWatchers lists users watching a repository
func (s *Service) ListWatchers(ctx context.Context, repoID string, opts *ListOpts) ([]*Watch, int, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListByRepo(ctx, repoID, limit, offset)
}

// ListWatching lists repositories a user is watching
func (s *Service) ListWatching(ctx context.Context, userID string, opts *ListOpts) ([]*Watch, int, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListByUser(ctx, userID, limit, offset)
}

// GetCount gets the watcher count for a repository
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
