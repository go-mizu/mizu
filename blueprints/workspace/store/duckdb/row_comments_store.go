package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/rowcomments"
)

// RowCommentsStore implements rowcomments.Store using DuckDB.
type RowCommentsStore struct {
	db *sql.DB
}

// NewRowCommentsStore creates a new RowCommentsStore.
func NewRowCommentsStore(db *sql.DB) *RowCommentsStore {
	return &RowCommentsStore{db: db}
}

func (s *RowCommentsStore) Create(ctx context.Context, c *rowcomments.Comment) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO row_comments (id, row_id, user_id, content, resolved, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.RowID, c.UserID, c.Content, c.Resolved, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *RowCommentsStore) GetByID(ctx context.Context, id string) (*rowcomments.Comment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, row_id, user_id, content, resolved, created_at, updated_at
		FROM row_comments WHERE id = ?
	`, id)
	return s.scanComment(row)
}

func (s *RowCommentsStore) Update(ctx context.Context, id string, content string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE row_comments SET content = ?, updated_at = ? WHERE id = ?
	`, content, time.Now(), id)
	return err
}

func (s *RowCommentsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM row_comments WHERE id = ?", id)
	return err
}

func (s *RowCommentsStore) ListByRow(ctx context.Context, rowID string) ([]*rowcomments.Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, row_id, user_id, content, resolved, created_at, updated_at
		FROM row_comments WHERE row_id = ?
		ORDER BY created_at ASC
	`, rowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanComments(rows)
}

func (s *RowCommentsStore) SetResolved(ctx context.Context, id string, resolved bool) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE row_comments SET resolved = ?, updated_at = ? WHERE id = ?
	`, resolved, time.Now(), id)
	return err
}

func (s *RowCommentsStore) scanComment(row *sql.Row) (*rowcomments.Comment, error) {
	var c rowcomments.Comment
	err := row.Scan(&c.ID, &c.RowID, &c.UserID, &c.Content, &c.Resolved, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *RowCommentsStore) scanComments(rows *sql.Rows) ([]*rowcomments.Comment, error) {
	var result []*rowcomments.Comment
	for rows.Next() {
		var c rowcomments.Comment
		err := rows.Scan(&c.ID, &c.RowID, &c.UserID, &c.Content, &c.Resolved, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, &c)
	}
	return result, rows.Err()
}
