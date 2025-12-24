package timelines

import (
	"context"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/posts"
)

const defaultLimit = 20

// Service implements the timelines API.
type Service struct {
	store    Store
	accounts accounts.API
	posts    posts.API
}

// NewService creates a new timelines service.
func NewService(store Store, accountsSvc accounts.API, postsSvc posts.API) *Service {
	return &Service{
		store:    store,
		accounts: accountsSvc,
		posts:    postsSvc,
	}
}

// Home returns the home timeline for an account.
func (s *Service) Home(ctx context.Context, accountID string, opts TimelineOpts) ([]*posts.Post, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	feed, err := s.store.GetHomeFeed(ctx, accountID, limit, opts.MaxID, opts.MinID)
	if err != nil {
		return nil, err
	}

	if err := s.posts.PopulateAccounts(ctx, feed); err != nil {
		return nil, err
	}

	if err := s.posts.PopulateViewerStates(ctx, feed, accountID); err != nil {
		return nil, err
	}

	return feed, nil
}

// Public returns the public timeline.
func (s *Service) Public(ctx context.Context, opts TimelineOpts) ([]*posts.Post, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	feed, err := s.store.GetPublicFeed(ctx, limit, opts.MaxID, opts.MinID, opts.OnlyMedia)
	if err != nil {
		return nil, err
	}

	if err := s.posts.PopulateAccounts(ctx, feed); err != nil {
		return nil, err
	}

	return feed, nil
}

// User returns the timeline for a specific user.
func (s *Service) User(ctx context.Context, userID string, opts TimelineOpts, includeReplies bool) ([]*posts.Post, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	feed, err := s.store.GetUserFeed(ctx, userID, limit, opts.MaxID, opts.MinID, includeReplies, opts.OnlyMedia)
	if err != nil {
		return nil, err
	}

	if err := s.posts.PopulateAccounts(ctx, feed); err != nil {
		return nil, err
	}

	return feed, nil
}

// Hashtag returns posts with a specific hashtag.
func (s *Service) Hashtag(ctx context.Context, tag string, opts TimelineOpts) ([]*posts.Post, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	feed, err := s.store.GetHashtagFeed(ctx, tag, limit, opts.MaxID, opts.MinID)
	if err != nil {
		return nil, err
	}

	if err := s.posts.PopulateAccounts(ctx, feed); err != nil {
		return nil, err
	}

	return feed, nil
}

// List returns the timeline for a list.
func (s *Service) List(ctx context.Context, accountID, listID string, opts TimelineOpts) ([]*posts.Post, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	feed, err := s.store.GetListFeed(ctx, listID, limit, opts.MaxID, opts.MinID)
	if err != nil {
		return nil, err
	}

	if err := s.posts.PopulateAccounts(ctx, feed); err != nil {
		return nil, err
	}

	if err := s.posts.PopulateViewerStates(ctx, feed, accountID); err != nil {
		return nil, err
	}

	return feed, nil
}

// Bookmarks returns bookmarked posts for an account.
func (s *Service) Bookmarks(ctx context.Context, accountID string, opts TimelineOpts) ([]*posts.Post, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	feed, err := s.store.GetBookmarksFeed(ctx, accountID, limit, opts.MaxID, opts.MinID)
	if err != nil {
		return nil, err
	}

	if err := s.posts.PopulateAccounts(ctx, feed); err != nil {
		return nil, err
	}

	if err := s.posts.PopulateViewerStates(ctx, feed, accountID); err != nil {
		return nil, err
	}

	return feed, nil
}

// Likes returns liked posts for an account.
func (s *Service) Likes(ctx context.Context, accountID string, opts TimelineOpts) ([]*posts.Post, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	feed, err := s.store.GetLikesFeed(ctx, accountID, limit, opts.MaxID, opts.MinID)
	if err != nil {
		return nil, err
	}

	if err := s.posts.PopulateAccounts(ctx, feed); err != nil {
		return nil, err
	}

	if err := s.posts.PopulateViewerStates(ctx, feed, accountID); err != nil {
		return nil, err
	}

	return feed, nil
}
