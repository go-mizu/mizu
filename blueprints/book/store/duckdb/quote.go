package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// QuoteStore implements store.QuoteStore backed by SQLite.
type QuoteStore struct {
	db *sql.DB
}

func (s *QuoteStore) Create(ctx context.Context, q *types.Quote) error {
	if q.CreatedAt.IsZero() {
		q.CreatedAt = time.Now()
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO quotes (book_id, author_name, text, likes_count, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		q.BookID, q.AuthorName, q.Text, q.LikesCount, q.CreatedAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	q.ID = id
	return nil
}

func (s *QuoteStore) GetByBook(ctx context.Context, bookID int64, limit int) ([]types.Quote, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, book_id, author_name, text, likes_count, created_at
		FROM quotes WHERE book_id = ? ORDER BY likes_count DESC, created_at DESC LIMIT ?`,
		bookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanQuotes(rows)
}

func (s *QuoteStore) GetAll(ctx context.Context, page, limit int) ([]types.Quote, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM quotes`).Scan(&total)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, book_id, author_name, text, likes_count, created_at
		FROM quotes ORDER BY likes_count DESC, created_at DESC LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	quotes, err := s.scanQuotes(rows)
	if err != nil {
		return nil, 0, err
	}
	return quotes, total, nil
}

func (s *QuoteStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM quotes WHERE id = ?`, id)
	return err
}

func (s *QuoteStore) scanQuotes(rows *sql.Rows) ([]types.Quote, error) {
	var quotes []types.Quote
	for rows.Next() {
		var q types.Quote
		err := rows.Scan(&q.ID, &q.BookID, &q.AuthorName, &q.Text, &q.LikesCount, &q.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan quote: %w", err)
		}
		quotes = append(quotes, q)
	}
	if quotes == nil {
		quotes = []types.Quote{}
	}
	return quotes, nil
}
