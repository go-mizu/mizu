package duckdb

import (
	"context"
	"database/sql"
	"time"
)

// Comment represents a comment record.
type Comment struct {
	ID         string
	FileID     string
	UserID     string
	ParentID   sql.NullString
	Content    string
	IsResolved bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CreateComment inserts a new comment.
func (s *Store) CreateComment(ctx context.Context, c *Comment) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO comments (id, file_id, user_id, parent_id, content, is_resolved, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.FileID, c.UserID, c.ParentID, c.Content, c.IsResolved, c.CreatedAt, c.UpdatedAt)
	return err
}

// GetCommentByID retrieves a comment by ID.
func (s *Store) GetCommentByID(ctx context.Context, id string) (*Comment, error) {
	c := &Comment{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, file_id, user_id, parent_id, content, is_resolved, created_at, updated_at
		FROM comments WHERE id = ?
	`, id).Scan(&c.ID, &c.FileID, &c.UserID, &c.ParentID, &c.Content, &c.IsResolved, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// UpdateComment updates a comment.
func (s *Store) UpdateComment(ctx context.Context, c *Comment) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE comments SET content = ?, is_resolved = ?, updated_at = ?
		WHERE id = ?
	`, c.Content, c.IsResolved, c.UpdatedAt, c.ID)
	return err
}

// DeleteComment deletes a comment by ID.
func (s *Store) DeleteComment(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE id = ?`, id)
	return err
}

// ListCommentsByFile lists all comments for a file.
func (s *Store) ListCommentsByFile(ctx context.Context, fileID string) ([]*Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_id, user_id, parent_id, content, is_resolved, created_at, updated_at
		FROM comments WHERE file_id = ? ORDER BY created_at ASC
	`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanComments(rows)
}

// ListTopLevelCommentsByFile lists top-level comments for a file.
func (s *Store) ListTopLevelCommentsByFile(ctx context.Context, fileID string) ([]*Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_id, user_id, parent_id, content, is_resolved, created_at, updated_at
		FROM comments WHERE file_id = ? AND parent_id IS NULL ORDER BY created_at ASC
	`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanComments(rows)
}

// ListReplies lists replies to a comment.
func (s *Store) ListReplies(ctx context.Context, parentID string) ([]*Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_id, user_id, parent_id, content, is_resolved, created_at, updated_at
		FROM comments WHERE parent_id = ? ORDER BY created_at ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanComments(rows)
}

// ResolveComment marks a comment as resolved.
func (s *Store) ResolveComment(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE comments SET is_resolved = TRUE, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// UnresolveComment marks a comment as unresolved.
func (s *Store) UnresolveComment(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE comments SET is_resolved = FALSE, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// DeleteCommentsByFile deletes all comments for a file.
func (s *Store) DeleteCommentsByFile(ctx context.Context, fileID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE file_id = ?`, fileID)
	return err
}

// CountCommentsByFile counts comments for a file.
func (s *Store) CountCommentsByFile(ctx context.Context, fileID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comments WHERE file_id = ?`, fileID).Scan(&count)
	return count, err
}

func scanComments(rows *sql.Rows) ([]*Comment, error) {
	var comments []*Comment
	for rows.Next() {
		c := &Comment{}
		if err := rows.Scan(&c.ID, &c.FileID, &c.UserID, &c.ParentID, &c.Content, &c.IsResolved, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}
