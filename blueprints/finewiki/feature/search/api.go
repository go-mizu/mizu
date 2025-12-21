// feature/search/api.go
package search

import "context"

// Query describes a title-only search request.
// MVP intent: fast autocomplete and exact title lookup.
// Text is required; all other fields are optional.
type Query struct {
	Text string `json:"text,omitempty"`

	// WikiName filters by wiki, for example "enwiki".
	WikiName string `json:"wikiname,omitempty"`

	// InLanguage filters by language code, for example "en".
	InLanguage string `json:"in_language,omitempty"`

	// Limit bounds the number of results returned.
	// Service enforces a safe default and a hard maximum.
	Limit int `json:"limit,omitempty"`

	// EnableFTS allows the store to use DuckDB FTS as a fallback.
	// Keep false for MVP autocomplete; enable when you want word-based fuzzy matches.
	EnableFTS bool `json:"enable_fts,omitempty"`
}

// Result is the minimal projection needed to show search results.
// Keep this cheap so it stays fast and cache-friendly.
type Result struct {
	ID         string `json:"id,omitempty"`          // dataset id, for example "enwiki/32552979"
	WikiName   string `json:"wikiname,omitempty"`    // for example "enwiki"
	InLanguage string `json:"in_language,omitempty"` // for example "en"
	Title      string `json:"title,omitempty"`
}

type Store interface {
	Search(ctx context.Context, q Query) ([]Result, error)
}

type API interface {
	Search(ctx context.Context, q Query) ([]Result, error)
}
