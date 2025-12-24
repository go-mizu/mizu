package search

import (
	"context"
	"strings"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/posts"
)

const defaultLimit = 20

// Service implements the search API.
type Service struct {
	store    Store
	accounts accounts.API
	posts    posts.API
}

// NewService creates a new search service.
func NewService(store Store, accountsSvc accounts.API, postsSvc posts.API) *Service {
	return &Service{
		store:    store,
		accounts: accountsSvc,
		posts:    postsSvc,
	}
}

// Search performs a unified search.
func (s *Service) Search(ctx context.Context, opts SearchOpts) (*SearchResult, error) {
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		return &SearchResult{
			Accounts: []*accounts.Account{},
			Posts:    []*posts.Post{},
			Hashtags: []*Hashtag{},
		}, nil
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	result := &SearchResult{
		Accounts: []*accounts.Account{},
		Posts:    []*posts.Post{},
		Hashtags: []*Hashtag{},
	}

	// Determine what to search based on type
	searchAccounts := opts.Type == "" || opts.Type == TypeAccounts
	searchPosts := opts.Type == "" || opts.Type == TypePosts
	searchHashtags := opts.Type == "" || opts.Type == TypeHashtags

	// Search accounts
	if searchAccounts {
		accs, err := s.store.SearchAccounts(ctx, query, limit, opts.Offset)
		if err == nil {
			result.Accounts = accs
		}
	}

	// Search posts
	if searchPosts {
		ps, err := s.store.SearchPosts(ctx, query, limit, opts.Offset, opts.MinLikes, opts.MinReposts, opts.HasMedia)
		if err == nil {
			if s.posts != nil {
				_ = s.posts.PopulateAccounts(ctx, ps)
			}
			result.Posts = ps
		}
	}

	// Search hashtags
	if searchHashtags {
		tags, err := s.store.SearchHashtags(ctx, query, limit)
		if err == nil {
			result.Hashtags = tags
		}
	}

	return result, nil
}

// SearchAccounts searches for accounts.
func (s *Service) SearchAccounts(ctx context.Context, query string, limit, offset int, viewerID string) ([]*accounts.Account, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	accs, err := s.store.SearchAccounts(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}

	// Populate relationship info if viewer is authenticated
	if viewerID != "" && s.accounts != nil {
		for _, acc := range accs {
			_ = s.accounts.PopulateRelationship(ctx, acc, viewerID)
		}
	}

	return accs, nil
}

// SearchPosts searches for posts.
func (s *Service) SearchPosts(ctx context.Context, query string, limit, offset int, viewerID string) ([]*posts.Post, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	ps, err := s.store.SearchPosts(ctx, query, limit, offset, 0, 0, false)
	if err != nil {
		return nil, err
	}

	if s.posts != nil {
		_ = s.posts.PopulateAccounts(ctx, ps)
		if viewerID != "" {
			_ = s.posts.PopulateViewerStates(ctx, ps, viewerID)
		}
	}

	return ps, nil
}

// SearchHashtags searches for hashtags.
func (s *Service) SearchHashtags(ctx context.Context, query string, limit int) ([]*Hashtag, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	return s.store.SearchHashtags(ctx, query, limit)
}

// Suggest returns search suggestions.
func (s *Service) Suggest(ctx context.Context, query string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.store.SuggestHashtags(ctx, query, limit)
}
