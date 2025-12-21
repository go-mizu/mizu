// Package search provides search functionality for posts, accounts, and hashtags.
package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/microblog/store/duckdb"
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

// Service handles search operations.
type Service struct {
	store *duckdb.Store
}

// NewService creates a new search service.
func NewService(store *duckdb.Store) *Service {
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
		accountResults, err := s.searchAccounts(ctx, query, limit)
		if err == nil {
			results = append(results, accountResults...)
		}
	}

	// Search hashtags
	if searchHashtags {
		hashtagResults, err := s.searchHashtags(ctx, query, limit)
		if err == nil {
			results = append(results, hashtagResults...)
		}
	}

	// Search posts
	if searchPosts {
		postResults, err := s.searchPosts(ctx, query, limit, viewerID)
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

func (s *Service) searchAccounts(ctx context.Context, query string, limit int) ([]*Result, error) {
	rows, err := s.store.Query(ctx, `
		SELECT id, username, display_name
		FROM accounts
		WHERE suspended = FALSE AND (
			LOWER(username) LIKE LOWER($1) || '%'
			OR LOWER(display_name) LIKE '%' || LOWER($1) || '%'
		)
		ORDER BY
			CASE WHEN LOWER(username) = LOWER($1) THEN 0 ELSE 1 END,
			CASE WHEN LOWER(username) LIKE LOWER($1) || '%' THEN 0 ELSE 1 END,
			LENGTH(username)
		LIMIT $2
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*Result
	for rows.Next() {
		var id, username string
		var displayName sql.NullString

		if rows.Scan(&id, &username, &displayName) == nil {
			text := username
			if displayName.Valid {
				text = displayName.String
			}
			results = append(results, &Result{
				Type:     ResultTypeAccount,
				ID:       id,
				Text:     text,
				Username: username,
			})
		}
	}

	return results, nil
}

func (s *Service) searchHashtags(ctx context.Context, query string, limit int) ([]*Result, error) {
	rows, err := s.store.Query(ctx, `
		SELECT id, name, posts_count
		FROM hashtags
		WHERE LOWER(name) LIKE LOWER($1) || '%'
		ORDER BY
			CASE WHEN LOWER(name) = LOWER($1) THEN 0 ELSE 1 END,
			posts_count DESC,
			LENGTH(name)
		LIMIT $2
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*Result
	for rows.Next() {
		var id, name string
		var count int

		if rows.Scan(&id, &name, &count) == nil {
			results = append(results, &Result{
				Type: ResultTypeHashtag,
				ID:   id,
				Text: fmt.Sprintf("#%s (%d posts)", name, count),
			})
		}
	}

	return results, nil
}

func (s *Service) searchPosts(ctx context.Context, query string, limit int, viewerID string) ([]*Result, error) {
	// Simple content search
	rows, err := s.store.Query(ctx, `
		SELECT p.id, p.content
		FROM posts p
		JOIN accounts a ON a.id = p.account_id
		WHERE p.visibility = 'public'
		AND a.suspended = FALSE
		AND LOWER(p.content) LIKE '%' || LOWER($1) || '%'
		ORDER BY p.created_at DESC
		LIMIT $2
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*Result
	for rows.Next() {
		var id, content string

		if rows.Scan(&id, &content) == nil {
			// Truncate content for preview
			text := content
			if len(text) > 100 {
				text = text[:100] + "..."
			}
			results = append(results, &Result{
				Type: ResultTypePost,
				ID:   id,
				Text: text,
			})
		}
	}

	return results, nil
}

// SearchPosts searches only posts.
func (s *Service) SearchPosts(ctx context.Context, query string, limit int, maxID, sinceID, viewerID string) ([]string, error) {
	q := `
		SELECT p.id
		FROM posts p
		JOIN accounts a ON a.id = p.account_id
		WHERE p.visibility = 'public'
		AND a.suspended = FALSE
		AND LOWER(p.content) LIKE '%' || LOWER($1) || '%'
	`

	args := []any{query}
	argIdx := 2

	if maxID != "" {
		q += fmt.Sprintf(" AND p.id < $%d", argIdx)
		args = append(args, maxID)
		argIdx++
	}
	if sinceID != "" {
		q += fmt.Sprintf(" AND p.id > $%d", argIdx)
		args = append(args, sinceID)
		argIdx++
	}

	q += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.store.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

// SearchAccounts searches only accounts.
func (s *Service) SearchAccounts(ctx context.Context, query string, limit int) ([]string, error) {
	rows, err := s.store.Query(ctx, `
		SELECT id FROM accounts
		WHERE suspended = FALSE AND (
			LOWER(username) LIKE LOWER($1) || '%'
			OR LOWER(display_name) LIKE '%' || LOWER($1) || '%'
		)
		ORDER BY
			CASE WHEN LOWER(username) = LOWER($1) THEN 0 ELSE 1 END,
			CASE WHEN LOWER(username) LIKE LOWER($1) || '%' THEN 0 ELSE 1 END
		LIMIT $2
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

func contains(slice []ResultType, item ResultType) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
