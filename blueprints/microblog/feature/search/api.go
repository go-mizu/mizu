// Package search provides search functionality for posts, accounts, and hashtags.
package search

import (
	"context"
)

// ResultType is the type of search result.
type ResultType string

const (
	ResultTypePost    ResultType = "post"
	ResultTypeAccount ResultType = "account"
	ResultTypeHashtag ResultType = "hashtag"
)

// Result represents a search result.
type Result struct {
	Type     ResultType `json:"type"`
	ID       string     `json:"id"`
	Text     string     `json:"text,omitempty"`
	Username string     `json:"username,omitempty"`
}

// API defines the search service contract.
type API interface {
	Search(ctx context.Context, query string, types []ResultType, limit int, viewerID string) ([]*Result, error)
	SearchPosts(ctx context.Context, query string, limit int, maxID, sinceID, viewerID string) ([]string, error)
	SearchAccounts(ctx context.Context, query string, limit int) ([]string, error)
}

// Store defines the data access contract for search.
type Store interface {
	SearchAccounts(ctx context.Context, query string, limit int) ([]*Result, error)
	SearchHashtags(ctx context.Context, query string, limit int) ([]*Result, error)
	SearchPosts(ctx context.Context, query string, limit int) ([]*Result, error)
	SearchPostIDs(ctx context.Context, query string, limit int, maxID, sinceID string) ([]string, error)
	SearchAccountIDs(ctx context.Context, query string, limit int) ([]string, error)
}
