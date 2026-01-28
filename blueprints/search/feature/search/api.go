// Package search provides search functionality.
package search

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// API defines the search service contract.
type API interface {
	// Search performs a full-text search with options.
	Search(ctx context.Context, query string, opts store.SearchOptions) (*store.SearchResponse, error)

	// SearchImages searches for images.
	SearchImages(ctx context.Context, query string, opts store.SearchOptions) ([]store.ImageResult, error)

	// SearchVideos searches for videos.
	SearchVideos(ctx context.Context, query string, opts store.SearchOptions) ([]store.VideoResult, error)

	// SearchNews searches for news articles.
	SearchNews(ctx context.Context, query string, opts store.SearchOptions) ([]store.NewsResult, error)
}

// Store defines the data access dependencies for search.
type Store interface {
	Search() store.SearchStore
	Suggest() store.SuggestStore
	Knowledge() store.KnowledgeStore
	History() store.HistoryStore
}
