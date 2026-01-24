package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// HistoryStore implements store.HistoryStore using PostgreSQL.
type HistoryStore struct {
	db *sql.DB
}

// RecordSearch records a search query in history.
func (s *HistoryStore) RecordSearch(ctx context.Context, history *store.SearchHistory) error {
	if history.SearchedAt.IsZero() {
		history.SearchedAt = time.Now()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO search.history (query, results, clicked_url, searched_at)
		VALUES ($1, $2, $3, $4)
	`, history.Query, history.Results, history.ClickedURL, history.SearchedAt)
	if err != nil {
		return fmt.Errorf("failed to record search: %w", err)
	}

	return nil
}

// GetHistory retrieves search history with pagination.
func (s *HistoryStore) GetHistory(ctx context.Context, limit, offset int) ([]*store.SearchHistory, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, query, results, clicked_url, searched_at
		FROM search.history
		ORDER BY searched_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}
	defer rows.Close()

	var history []*store.SearchHistory
	for rows.Next() {
		var h store.SearchHistory
		var clickedURL sql.NullString
		if err := rows.Scan(&h.ID, &h.Query, &h.Results, &clickedURL, &h.SearchedAt); err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		if clickedURL.Valid {
			h.ClickedURL = clickedURL.String
		}
		history = append(history, &h)
	}

	return history, nil
}

// ClearHistory clears all search history.
func (s *HistoryStore) ClearHistory(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM search.history")
	if err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}
	return nil
}

// DeleteHistoryEntry deletes a single history entry.
func (s *HistoryStore) DeleteHistoryEntry(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM search.history WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete history entry: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("history entry not found")
	}

	return nil
}
