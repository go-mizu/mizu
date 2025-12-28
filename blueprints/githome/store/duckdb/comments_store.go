package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/githome/feature/comments"
)

// CommentsStore implements comments.Store
type CommentsStore struct {
	db *sql.DB
}

// NewCommentsStore creates a new comments store
func NewCommentsStore(db *sql.DB) *CommentsStore {
	return &CommentsStore{db: db}
}

// Create creates a new comment
func (s *CommentsStore) Create(ctx context.Context, c *comments.Comment) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO comments (id, target_type, target_id, user_id, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, c.ID, c.TargetType, c.TargetID, c.UserID, c.Body, c.CreatedAt, c.UpdatedAt)
	return err
}

// GetByID retrieves a comment by ID
func (s *CommentsStore) GetByID(ctx context.Context, id string) (*comments.Comment, error) {
	c := &comments.Comment{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, target_type, target_id, user_id, body, created_at, updated_at
		FROM comments WHERE id = $1
	`, id).Scan(&c.ID, &c.TargetType, &c.TargetID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// Update updates a comment
func (s *CommentsStore) Update(ctx context.Context, c *comments.Comment) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE comments SET body = $2, updated_at = $3
		WHERE id = $1
	`, c.ID, c.Body, c.UpdatedAt)
	return err
}

// Delete deletes a comment
func (s *CommentsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE id = $1`, id)
	return err
}

// List lists comments for a target
func (s *CommentsStore) List(ctx context.Context, targetType, targetID string, limit, offset int) ([]*comments.Comment, int, error) {
	// Get total count
	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM comments WHERE target_type = $1 AND target_id = $2
	`, targetType, targetID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get comments
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, target_type, target_id, user_id, body, created_at, updated_at
		FROM comments WHERE target_type = $1 AND target_id = $2
		ORDER BY created_at ASC
		LIMIT $3 OFFSET $4
	`, targetType, targetID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*comments.Comment
	for rows.Next() {
		c := &comments.Comment{}
		if err := rows.Scan(&c.ID, &c.TargetType, &c.TargetID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, c)
	}
	return list, total, rows.Err()
}

// CountByTarget counts comments for a target
func (s *CommentsStore) CountByTarget(ctx context.Context, targetType, targetID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM comments WHERE target_type = $1 AND target_id = $2
	`, targetType, targetID).Scan(&count)
	return count, err
}
