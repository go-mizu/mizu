package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// SuggestStore implements store.SuggestStore using PostgreSQL.
type SuggestStore struct {
	db *sql.DB
}

// GetSuggestions returns autocomplete suggestions for a prefix.
func (s *SuggestStore) GetSuggestions(ctx context.Context, prefix string, limit int) ([]store.Suggestion, error) {
	if limit <= 0 {
		limit = 10
	}

	prefix = strings.TrimSpace(strings.ToLower(prefix))
	if prefix == "" {
		return nil, nil
	}

	// Use trigram similarity for fuzzy matching
	rows, err := s.db.QueryContext(ctx, `
		SELECT query, frequency,
			similarity(query, $1) as sim
		FROM search.suggestions
		WHERE query % $1 OR query ILIKE $2
		ORDER BY sim DESC, frequency DESC
		LIMIT $3
	`, prefix, prefix+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get suggestions: %w", err)
	}
	defer rows.Close()

	var suggestions []store.Suggestion
	for rows.Next() {
		var s store.Suggestion
		var sim float64
		if err := rows.Scan(&s.Text, &s.Frequency, &sim); err != nil {
			return nil, fmt.Errorf("failed to scan suggestion: %w", err)
		}
		s.Type = "query"
		suggestions = append(suggestions, s)
	}

	return suggestions, nil
}

// RecordQuery records a search query for suggestions.
func (s *SuggestStore) RecordQuery(ctx context.Context, query string) error {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO search.suggestions (query, frequency, last_searched)
		VALUES ($1, 1, NOW())
		ON CONFLICT (query) DO UPDATE SET
			frequency = search.suggestions.frequency + 1,
			last_searched = NOW()
	`, query)
	if err != nil {
		return fmt.Errorf("failed to record query: %w", err)
	}

	return nil
}

// GetTrendingQueries returns trending search queries.
func (s *SuggestStore) GetTrendingQueries(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT query FROM search.suggestions
		WHERE last_searched > NOW() - INTERVAL '24 hours'
		ORDER BY frequency DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending queries: %w", err)
	}
	defer rows.Close()

	var queries []string
	for rows.Next() {
		var q string
		if err := rows.Scan(&q); err != nil {
			return nil, fmt.Errorf("failed to scan query: %w", err)
		}
		queries = append(queries, q)
	}

	return queries, nil
}
