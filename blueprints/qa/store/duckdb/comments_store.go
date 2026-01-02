package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/qa/feature/comments"
)

// CommentsStore implements comments.Store.
type CommentsStore struct {
	db *sql.DB
}

// NewCommentsStore creates a new comments store.
func NewCommentsStore(db *sql.DB) *CommentsStore {
	return &CommentsStore{db: db}
}

// Create creates a comment.
func (s *CommentsStore) Create(ctx context.Context, comment *comments.Comment) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO comments (id, target_type, target_id, author_id, body, score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, comment.ID, comment.TargetType, comment.TargetID, comment.AuthorID,
		comment.Body, comment.Score, comment.CreatedAt, comment.UpdatedAt)
	return err
}

// ListByTarget lists comments.
func (s *CommentsStore) ListByTarget(ctx context.Context, targetType comments.TargetType, targetID string, opts comments.ListOpts) ([]*comments.Comment, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, target_type, target_id, author_id, body, score, created_at, updated_at
		FROM comments
		WHERE target_type = $1 AND target_id = $2
		ORDER BY created_at ASC
		LIMIT $3
	`, targetType, targetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*comments.Comment
	for rows.Next() {
		comment := &comments.Comment{}
		if err := rows.Scan(
			&comment.ID, &comment.TargetType, &comment.TargetID, &comment.AuthorID,
			&comment.Body, &comment.Score, &comment.CreatedAt, &comment.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, comment)
	}
	return result, rows.Err()
}

// Delete deletes a comment.
func (s *CommentsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE id = $1`, id)
	return err
}

// UpdateScore updates score.
func (s *CommentsStore) UpdateScore(ctx context.Context, id string, delta int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE comments SET score = score + $2 WHERE id = $1`, id, delta)
	return err
}
