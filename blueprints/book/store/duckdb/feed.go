package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// FeedStore implements store.FeedStore backed by SQLite.
type FeedStore struct {
	db *sql.DB
}

func (s *FeedStore) Add(ctx context.Context, item *types.FeedItem) error {
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO feed (type, book_id, book_title, data, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		item.Type, item.BookID, item.BookTitle, item.Data, item.CreatedAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	item.ID = id
	return nil
}

func (s *FeedStore) GetRecent(ctx context.Context, limit int) ([]types.FeedItem, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, book_id, book_title, data, created_at
		FROM feed ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.FeedItem
	for rows.Next() {
		var f types.FeedItem
		err := rows.Scan(&f.ID, &f.Type, &f.BookID, &f.BookTitle, &f.Data, &f.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan feed item: %w", err)
		}
		items = append(items, f)
	}
	if items == nil {
		items = []types.FeedItem{}
	}
	return items, nil
}
