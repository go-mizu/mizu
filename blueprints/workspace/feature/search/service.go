package search

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

// Service implements the search API.
type Service struct {
	store     Store
	pages     pages.API
	databases databases.API
}

// NewService creates a new search service.
func NewService(store Store, pages pages.API, databases databases.API) *Service {
	return &Service{store: store, pages: pages, databases: databases}
}

// Search performs a full-text search.
func (s *Service) Search(ctx context.Context, workspaceID, query string, opts SearchOpts) (*SearchResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	result := &SearchResult{}

	// Search pages
	searchPages, err := s.store.Search(ctx, workspaceID, query, opts)
	if err != nil {
		return nil, err
	}
	result.Pages = searchPages

	// Search databases if requested
	if len(opts.Types) == 0 || contains(opts.Types, "database") {
		dbs, _ := s.databases.ListByWorkspace(ctx, workspaceID)
		// Filter by query
		for _, db := range dbs {
			if containsIgnoreCase(db.Title, query) {
				result.Databases = append(result.Databases, db)
			}
		}
	}

	result.HasMore = len(result.Pages) >= opts.Limit
	if result.HasMore && len(result.Pages) > 0 {
		result.NextCursor = result.Pages[len(result.Pages)-1].ID
	}

	return result, nil
}

// QuickSearch performs a quick title-only search.
func (s *Service) QuickSearch(ctx context.Context, workspaceID, query string, limit int) ([]*pages.PageRef, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.store.QuickSearch(ctx, workspaceID, query, limit)
}

// GetRecent returns recently accessed pages.
func (s *Service) GetRecent(ctx context.Context, userID, workspaceID string, limit int) ([]*pages.Page, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.store.GetRecent(ctx, userID, workspaceID, limit)
}

// RecordAccess records a page access.
func (s *Service) RecordAccess(ctx context.Context, userID, pageID string) error {
	return s.store.RecordAccess(ctx, userID, pageID, time.Now())
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive contains
	sl := len(s)
	subl := len(substr)
	if subl > sl {
		return false
	}
	for i := 0; i <= sl-subl; i++ {
		match := true
		for j := 0; j < subl; j++ {
			sc := s[i+j]
			subc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if subc >= 'A' && subc <= 'Z' {
				subc += 32
			}
			if sc != subc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
