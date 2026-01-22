package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// QueryHistoryStore implements store.QueryHistoryStore.
type QueryHistoryStore struct {
	db *sql.DB
}

func (s *QueryHistoryStore) Create(ctx context.Context, qh *store.QueryHistory) error {
	if qh.ID == "" {
		qh.ID = generateID()
	}
	qh.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO query_history (id, user_id, datasource_id, query, duration, row_count, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, qh.ID, qh.UserID, qh.DataSourceID, qh.Query, qh.Duration, qh.RowCount, qh.Error, qh.CreatedAt)
	return err
}

func (s *QueryHistoryStore) List(ctx context.Context, userID string, limit int) ([]*store.QueryHistory, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, datasource_id, query, duration, row_count, error, created_at
		FROM query_history WHERE user_id = ? ORDER BY created_at DESC LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.QueryHistory
	for rows.Next() {
		var qh store.QueryHistory
		if err := rows.Scan(&qh.ID, &qh.UserID, &qh.DataSourceID, &qh.Query, &qh.Duration, &qh.RowCount, &qh.Error, &qh.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, &qh)
	}
	return result, rows.Err()
}
