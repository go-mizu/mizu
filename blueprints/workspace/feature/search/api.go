// Package search provides search functionality.
package search

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

// SearchOpts contains options for searching.
type SearchOpts struct {
	Types    []string // page, database
	Cursor   string
	Limit    int
}

// SearchResult holds search results.
type SearchResult struct {
	Pages      []*pages.Page         `json:"pages,omitempty"`
	Databases  []*databases.Database `json:"databases,omitempty"`
	NextCursor string                `json:"next_cursor,omitempty"`
	HasMore    bool                  `json:"has_more"`
}

// API defines the search service contract.
type API interface {
	// Full-text search
	Search(ctx context.Context, workspaceID, query string, opts SearchOpts) (*SearchResult, error)

	// Quick search (titles only)
	QuickSearch(ctx context.Context, workspaceID, query string, limit int) ([]*pages.PageRef, error)

	// Recent
	GetRecent(ctx context.Context, userID, workspaceID string, limit int) ([]*pages.Page, error)

	// Record access (for recent pages)
	RecordAccess(ctx context.Context, userID, pageID string) error
}

// Store defines the data access contract for search.
type Store interface {
	Search(ctx context.Context, workspaceID, query string, opts SearchOpts) ([]*pages.Page, error)
	QuickSearch(ctx context.Context, workspaceID, query string, limit int) ([]*pages.PageRef, error)
	RecordAccess(ctx context.Context, userID, pageID string, accessedAt time.Time) error
	GetRecent(ctx context.Context, userID, workspaceID string, limit int) ([]*pages.Page, error)
}
