// Package engine provides a generic search engine interface.
package engine

import (
	"context"
)

// Engine defines the interface for a search engine.
type Engine interface {
	// Search performs a search with the given query and options.
	Search(ctx context.Context, query string, opts SearchOptions) (*SearchResponse, error)

	// Categories returns the list of supported categories.
	Categories() []Category

	// Name returns the engine name.
	Name() string
}
