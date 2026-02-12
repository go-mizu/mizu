package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// ProgressStore implements store.ProgressStore backed by SQLite.
type ProgressStore struct {
	db *sql.DB
}

func (s *ProgressStore) Create(ctx context.Context, p *types.ReadingProgress) error {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO reading_progress (book_id, page, percent, note, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		p.BookID, p.Page, p.Percent, p.Note, p.CreatedAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	return nil
}

func (s *ProgressStore) GetByBook(ctx context.Context, bookID int64) ([]types.ReadingProgress, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, book_id, page, percent, note, created_at
		FROM reading_progress WHERE book_id = ? ORDER BY created_at DESC`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.ReadingProgress
	for rows.Next() {
		var p types.ReadingProgress
		err := rows.Scan(&p.ID, &p.BookID, &p.Page, &p.Percent, &p.Note, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan progress: %w", err)
		}
		items = append(items, p)
	}
	if items == nil {
		items = []types.ReadingProgress{}
	}
	return items, nil
}

func (s *ProgressStore) GetLatest(ctx context.Context, bookID int64) (*types.ReadingProgress, error) {
	var p types.ReadingProgress
	err := s.db.QueryRowContext(ctx, `
		SELECT id, book_id, page, percent, note, created_at
		FROM reading_progress WHERE book_id = ? ORDER BY created_at DESC LIMIT 1`, bookID).
		Scan(&p.ID, &p.BookID, &p.Page, &p.Percent, &p.Note, &p.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
