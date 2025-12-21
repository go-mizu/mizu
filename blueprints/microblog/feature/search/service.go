package search

import (
	"context"
	"strings"
)

// Service handles search operations.
// Implements API interface.
type Service struct {
	store Store
}

// NewService creates a new search service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Search performs a combined search across posts, accounts, and hashtags.
func (s *Service) Search(ctx context.Context, query string, types []ResultType, limit int, viewerID string) ([]*Result, error) {
	if query == "" {
		return nil, nil
	}

	query = strings.TrimSpace(query)
	var results []*Result

	// Determine which types to search
	searchPosts := len(types) == 0 || contains(types, ResultTypePost)
	searchAccounts := len(types) == 0 || contains(types, ResultTypeAccount)
	searchHashtags := len(types) == 0 || contains(types, ResultTypeHashtag)

	// Search hashtags (if query starts with #)
	if strings.HasPrefix(query, "#") {
		searchHashtags = true
		searchPosts = false
		searchAccounts = false
		query = strings.TrimPrefix(query, "#")
	}

	// Search accounts (if query starts with @)
	if strings.HasPrefix(query, "@") {
		searchAccounts = true
		searchPosts = false
		searchHashtags = false
		query = strings.TrimPrefix(query, "@")
	}

	// Search accounts
	if searchAccounts {
		accountResults, err := s.store.SearchAccounts(ctx, query, limit)
		if err == nil {
			results = append(results, accountResults...)
		}
	}

	// Search hashtags
	if searchHashtags {
		hashtagResults, err := s.store.SearchHashtags(ctx, query, limit)
		if err == nil {
			results = append(results, hashtagResults...)
		}
	}

	// Search posts
	if searchPosts {
		postResults, err := s.store.SearchPosts(ctx, query, limit)
		if err == nil {
			results = append(results, postResults...)
		}
	}

	// Limit total results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// SearchPosts searches only posts.
func (s *Service) SearchPosts(ctx context.Context, query string, limit int, maxID, sinceID, viewerID string) ([]string, error) {
	return s.store.SearchPostIDs(ctx, query, limit, maxID, sinceID)
}

// SearchAccounts searches only accounts.
func (s *Service) SearchAccounts(ctx context.Context, query string, limit int) ([]string, error) {
	return s.store.SearchAccountIDs(ctx, query, limit)
}

func contains(slice []ResultType, item ResultType) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
