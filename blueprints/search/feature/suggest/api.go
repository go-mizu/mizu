// Package suggest provides autocomplete suggestion functionality.
package suggest

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// API defines the suggestion service contract.
type API interface {
	// GetSuggestions returns autocomplete suggestions for a prefix.
	GetSuggestions(ctx context.Context, prefix string, limit int) ([]store.Suggestion, error)

	// RecordQuery records a query for future suggestions.
	RecordQuery(ctx context.Context, query string) error

	// GetTrending returns trending/popular queries.
	GetTrending(ctx context.Context, limit int) ([]string, error)
}

// Store defines the data access contract for suggestions.
type Store interface {
	Suggest() store.SuggestStore
}
