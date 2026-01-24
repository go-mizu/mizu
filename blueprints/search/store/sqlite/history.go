package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// HistoryStore handles search history.
type HistoryStore struct {
	db *sql.DB
}

// RecordSearch records a search in history.
func (s *HistoryStore) RecordSearch(ctx context.Context, history *store.SearchHistory) error {
	if history.ID == "" {
		history.ID = generateID()
	}
	history.SearchedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO history (id, query, results, clicked_url, searched_at)
		VALUES (?, ?, ?, ?, ?)
	`, history.ID, history.Query, history.Results, history.ClickedURL, history.SearchedAt)

	return err
}

// GetHistory retrieves search history with pagination.
func (s *HistoryStore) GetHistory(ctx context.Context, limit, offset int) ([]*store.SearchHistory, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, query, results, clicked_url, searched_at
		FROM history
		ORDER BY searched_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*store.SearchHistory
	for rows.Next() {
		var h store.SearchHistory
		var clickedURL sql.NullString

		if err := rows.Scan(&h.ID, &h.Query, &h.Results, &clickedURL, &h.SearchedAt); err != nil {
			return nil, err
		}

		if clickedURL.Valid {
			h.ClickedURL = clickedURL.String
		}

		entries = append(entries, &h)
	}

	return entries, nil
}

// ClearHistory clears all search history.
func (s *HistoryStore) ClearHistory(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM history")
	return err
}

// DeleteHistoryEntry deletes a single history entry.
func (s *HistoryStore) DeleteHistoryEntry(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM history WHERE id = ?", id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("history entry not found")
	}

	return nil
}
