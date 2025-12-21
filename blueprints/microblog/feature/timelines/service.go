package timelines

import (
	"context"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/feature/posts"
)

// Service handles timeline operations.
// Implements API interface.
type Service struct {
	store    Store
	accounts accounts.API
}

// NewService creates a new timelines service.
func NewService(store Store, accounts accounts.API) *Service {
	return &Service{store: store, accounts: accounts}
}

// Home returns the home timeline for an account (posts from followed accounts).
func (s *Service) Home(ctx context.Context, accountID string, limit int, maxID, sinceID string) ([]*posts.Post, error) {
	result, err := s.store.Home(ctx, accountID, limit, maxID, sinceID)
	if err != nil {
		return nil, err
	}
	return s.enrichPosts(ctx, result, accountID), nil
}

// Local returns the local timeline (all public posts on the instance).
func (s *Service) Local(ctx context.Context, viewerID string, limit int, maxID, sinceID string) ([]*posts.Post, error) {
	result, err := s.store.Local(ctx, limit, maxID, sinceID)
	if err != nil {
		return nil, err
	}
	return s.enrichPosts(ctx, result, viewerID), nil
}

// Hashtag returns posts with a specific hashtag.
func (s *Service) Hashtag(ctx context.Context, tag, viewerID string, limit int, maxID, sinceID string) ([]*posts.Post, error) {
	result, err := s.store.Hashtag(ctx, tag, limit, maxID, sinceID)
	if err != nil {
		return nil, err
	}
	return s.enrichPosts(ctx, result, viewerID), nil
}

// Account returns posts by a specific account.
func (s *Service) Account(ctx context.Context, accountID, viewerID string, limit int, maxID string, onlyMedia, excludeReplies bool) ([]*posts.Post, error) {
	isFollowing := false
	if viewerID != "" && viewerID != accountID {
		isFollowing, _ = s.store.IsFollowing(ctx, viewerID, accountID)
	}

	result, err := s.store.Account(ctx, accountID, viewerID, limit, maxID, onlyMedia, excludeReplies, isFollowing)
	if err != nil {
		return nil, err
	}
	return s.enrichPosts(ctx, result, viewerID), nil
}

// List returns posts from accounts in a list.
func (s *Service) List(ctx context.Context, listID, viewerID string, limit int, maxID string) ([]*posts.Post, error) {
	result, err := s.store.List(ctx, listID, limit, maxID)
	if err != nil {
		return nil, err
	}
	return s.enrichPosts(ctx, result, viewerID), nil
}

// Bookmarks returns bookmarked posts for an account.
func (s *Service) Bookmarks(ctx context.Context, accountID string, limit int, maxID string) ([]*posts.Post, error) {
	result, err := s.store.Bookmarks(ctx, accountID, limit, maxID)
	if err != nil {
		return nil, err
	}
	return s.enrichPosts(ctx, result, accountID), nil
}

func (s *Service) enrichPosts(ctx context.Context, result []*posts.Post, viewerID string) []*posts.Post {
	for _, p := range result {
		p.Account, _ = s.accounts.GetByID(ctx, p.AccountID)
		if viewerID != "" {
			s.loadViewerState(ctx, p, viewerID)
		}
	}
	return result
}

func (s *Service) loadViewerState(ctx context.Context, post *posts.Post, viewerID string) {
	post.Liked, _ = s.store.CheckLiked(ctx, viewerID, post.ID)
	post.Reposted, _ = s.store.CheckReposted(ctx, viewerID, post.ID)
	post.Bookmarked, _ = s.store.CheckBookmarked(ctx, viewerID, post.ID)
}
