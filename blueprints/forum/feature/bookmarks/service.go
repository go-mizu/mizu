package bookmarks

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/pkg/ulid"
)

// Service implements the bookmarks API.
type Service struct {
	store Store
}

// NewService creates a new bookmarks service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a bookmark.
func (s *Service) Create(ctx context.Context, accountID, targetType, targetID string) error {
	// Check if already exists
	existing, err := s.store.GetByTarget(ctx, accountID, targetType, targetID)
	if err != nil && err != ErrNotFound {
		return err
	}
	if existing != nil {
		return nil // Already bookmarked
	}

	bookmark := &Bookmark{
		ID:         ulid.New(),
		AccountID:  accountID,
		TargetType: targetType,
		TargetID:   targetID,
		CreatedAt:  time.Now(),
	}

	return s.store.Create(ctx, bookmark)
}

// Delete removes a bookmark.
func (s *Service) Delete(ctx context.Context, accountID, targetType, targetID string) error {
	return s.store.Delete(ctx, accountID, targetType, targetID)
}

// IsBookmarked checks if content is bookmarked.
func (s *Service) IsBookmarked(ctx context.Context, accountID, targetType, targetID string) (bool, error) {
	bookmark, err := s.store.GetByTarget(ctx, accountID, targetType, targetID)
	if err == ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return bookmark != nil, nil
}

// List lists bookmarks.
func (s *Service) List(ctx context.Context, accountID, targetType string, opts ListOpts) ([]*Bookmark, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 25
	}
	return s.store.List(ctx, accountID, targetType, opts)
}

// GetBookmarked checks bookmarks for multiple targets.
func (s *Service) GetBookmarked(ctx context.Context, accountID, targetType string, targetIDs []string) (map[string]bool, error) {
	bookmarks, err := s.store.GetByTargets(ctx, accountID, targetType, targetIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool, len(bookmarks))
	for _, b := range bookmarks {
		result[b.TargetID] = true
	}
	return result, nil
}
