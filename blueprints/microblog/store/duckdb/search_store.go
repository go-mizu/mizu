package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-mizu/blueprints/microblog/feature/search"
)

// SearchStore implements search.Store using DuckDB.
type SearchStore struct {
	db *sql.DB
}

// NewSearchStore creates a new search store.
func NewSearchStore(db *sql.DB) *SearchStore {
	return &SearchStore{db: db}
}

func (s *SearchStore) SearchAccounts(ctx context.Context, query string, limit int) ([]*search.Result, error) {
	rows, err := s.db.QueryContext(ctx, `
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

	var results []*search.Result
	for rows.Next() {
		var id, username string
		var displayName sql.NullString

		if rows.Scan(&id, &username, &displayName) == nil {
			text := username
			if displayName.Valid {
				text = displayName.String
			}
			results = append(results, &search.Result{
				Type:     search.ResultTypeAccount,
				ID:       id,
				Text:     text,
				Username: username,
			})
		}
	}

	return results, nil
}

func (s *SearchStore) SearchHashtags(ctx context.Context, query string, limit int) ([]*search.Result, error) {
	rows, err := s.db.QueryContext(ctx, `
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

	var results []*search.Result
	for rows.Next() {
		var id, name string
		var count int

		if rows.Scan(&id, &name, &count) == nil {
			results = append(results, &search.Result{
				Type: search.ResultTypeHashtag,
				ID:   id,
				Text: fmt.Sprintf("#%s (%d posts)", name, count),
			})
		}
	}

	return results, nil
}

func (s *SearchStore) SearchPosts(ctx context.Context, query string, limit int) ([]*search.Result, error) {
	rows, err := s.db.QueryContext(ctx, `
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

	var results []*search.Result
	for rows.Next() {
		var id, content string

		if rows.Scan(&id, &content) == nil {
			text := content
			if len(text) > 100 {
				text = text[:100] + "..."
			}
			results = append(results, &search.Result{
				Type: search.ResultTypePost,
				ID:   id,
				Text: text,
			})
		}
	}

	return results, nil
}

func (s *SearchStore) SearchPostIDs(ctx context.Context, query string, limit int, maxID, sinceID string) ([]string, error) {
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

	rows, err := s.db.QueryContext(ctx, q, args...)
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

func (s *SearchStore) SearchAccountIDs(ctx context.Context, query string, limit int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
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
