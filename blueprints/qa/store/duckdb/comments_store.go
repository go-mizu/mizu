package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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

// ListByTargets lists comments for multiple targets.
func (s *CommentsStore) ListByTargets(ctx context.Context, targetType comments.TargetType, targetIDs []string, opts comments.ListOpts) (map[string][]*comments.Comment, error) {
	if len(targetIDs) == 0 {
		return make(map[string][]*comments.Comment), nil
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	placeholders := make([]string, len(targetIDs))
	args := make([]any, len(targetIDs)+2)
	args[0] = targetType
	for i, id := range targetIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args[i+2] = id
	}
	args[1] = limit * len(targetIDs) // Total limit

	query := `
		SELECT id, target_type, target_id, author_id, body, score, created_at, updated_at
		FROM comments
		WHERE target_type = $1 AND target_id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]*comments.Comment)
	for rows.Next() {
		comment := &comments.Comment{}
		if err := rows.Scan(
			&comment.ID, &comment.TargetType, &comment.TargetID, &comment.AuthorID,
			&comment.Body, &comment.Score, &comment.CreatedAt, &comment.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result[comment.TargetID] = append(result[comment.TargetID], comment)
	}
	return result, rows.Err()
}
