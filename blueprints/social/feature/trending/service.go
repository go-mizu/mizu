package trending

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/social/feature/posts"
)

const (
	defaultLimit = 10
	trendWindow  = 24 * time.Hour
)

// Service implements the trending API.
type Service struct {
	store Store
	posts posts.API
}

// NewService creates a new trending service.
func NewService(store Store, postsSvc posts.API) *Service {
	return &Service{
		store: store,
		posts: postsSvc,
	}
}

// GetTrendingTags returns trending hashtags.
func (s *Service) GetTrendingTags(ctx context.Context, opts TrendingOpts) ([]*TrendingTag, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	return s.store.GetTrendingTags(ctx, limit, opts.Offset)
}

// GetTrendingPosts returns trending posts.
func (s *Service) GetTrendingPosts(ctx context.Context, opts TrendingOpts) ([]*posts.Post, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	ps, err := s.store.GetTrendingPosts(ctx, limit, opts.Offset)
	if err != nil {
		return nil, err
	}

	if s.posts != nil {
		_ = s.posts.PopulateAccounts(ctx, ps)
	}

	return ps, nil
}

// RefreshTrending recomputes trending content.
func (s *Service) RefreshTrending(ctx context.Context) error {
	// Compute and cache trending tags
	_, err := s.store.ComputeTrendingTags(ctx, trendWindow, 50)
	if err != nil {
		return err
	}

	// Compute and cache trending posts
	_, err = s.store.ComputeTrendingPosts(ctx, trendWindow, 50)
	if err != nil {
		return err
	}

	return nil
}
