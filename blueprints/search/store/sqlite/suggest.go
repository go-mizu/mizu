package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// SuggestStore handles autocomplete suggestions.
type SuggestStore struct {
	db *sql.DB
}

// GetSuggestions returns autocomplete suggestions for a prefix.
func (s *SuggestStore) GetSuggestions(ctx context.Context, prefix string, limit int) ([]store.Suggestion, error) {
	if limit <= 0 {
		limit = 10
	}

	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		return []store.Suggestion{}, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT query, frequency
		FROM suggestions
		WHERE query LIKE ? || '%'
		ORDER BY frequency DESC, last_used DESC
		LIMIT ?
	`, prefix, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []store.Suggestion
	for rows.Next() {
		var sg store.Suggestion
		if err := rows.Scan(&sg.Text, &sg.Frequency); err != nil {
			return nil, err
		}
		sg.Type = "query"
		suggestions = append(suggestions, sg)
	}

	return suggestions, nil
}

// RecordQuery records a search query for future suggestions.
func (s *SuggestStore) RecordQuery(ctx context.Context, query string) error {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO suggestions (query, frequency, last_used)
		VALUES (?, 1, ?)
		ON CONFLICT(query) DO UPDATE SET
			frequency = frequency + 1,
			last_used = excluded.last_used
	`, query, time.Now())

	return err
}

// GetTrendingQueries returns the most popular queries.
func (s *SuggestStore) GetTrendingQueries(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT query
		FROM suggestions
		ORDER BY frequency DESC, last_used DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queries []string
	for rows.Next() {
		var q string
		if err := rows.Scan(&q); err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}

	return queries, nil
}
