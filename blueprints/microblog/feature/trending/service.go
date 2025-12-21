package trending

import (
	"context"
)

// Service handles trending calculations.
// Implements API interface.
type Service struct {
	store Store
}

// NewService creates a new trending service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Tags returns trending hashtags.
func (s *Service) Tags(ctx context.Context, limit int) ([]*TrendingTag, error) {
	return s.store.Tags(ctx, limit)
}

// Posts returns trending posts.
func (s *Service) Posts(ctx context.Context, limit int) ([]string, error) {
	return s.store.Posts(ctx, limit)
}

// SuggestedAccounts returns suggested accounts to follow.
func (s *Service) SuggestedAccounts(ctx context.Context, accountID string, limit int) ([]string, error) {
	return s.store.SuggestedAccounts(ctx, accountID, limit)
}
