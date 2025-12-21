// Package api provides a contract-based REST API for FineWiki.
package api

import (
	"context"

	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/feature/view"
)

// WikiAPI defines the FineWiki REST API contract.
// Methods follow the contract pattern: (ctx, *In) (*Out, error)
type WikiAPI interface {
	// GetPage retrieves a wiki page by ID or by (wikiname, title).
	GetPage(ctx context.Context, in *GetPageIn) (*view.Page, error)

	// Search searches for pages by title prefix with optional filters.
	Search(ctx context.Context, in *SearchIn) (*SearchOut, error)
}

// GetPageIn is the input for GetPage.
// Either ID or (WikiName + Title) must be provided.
type GetPageIn struct {
	// ID is the page ID like "viwiki/123456"
	ID string `json:"id,omitempty"`

	// WikiName is the wiki name like "viwiki" or "enwiki"
	WikiName string `json:"wikiname,omitempty"`

	// Title is the page title
	Title string `json:"title,omitempty"`
}

// SearchIn is the input for Search.
type SearchIn struct {
	// Q is the search query text (required, minimum 2 characters)
	Q string `json:"q"`

	// WikiName filters results by wiki name
	WikiName string `json:"wikiname,omitempty"`

	// InLanguage filters results by language code
	InLanguage string `json:"in_language,omitempty"`

	// Limit is the maximum number of results (default 20, max 200)
	Limit int `json:"limit,omitempty"`

	// EnableFTS enables full-text search fallback
	EnableFTS bool `json:"fts,omitempty"`
}

// SearchOut is the output for Search.
type SearchOut struct {
	// Results is the list of matching pages
	Results []search.Result `json:"results"`

	// Count is the number of results returned
	Count int `json:"count"`
}
