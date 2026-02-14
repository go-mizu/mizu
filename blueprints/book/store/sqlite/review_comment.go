package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// ReviewCommentStore implements store.ReviewCommentStore backed by SQLite.
type ReviewCommentStore struct {
	db *sql.DB
}

func (s *ReviewCommentStore) Create(ctx context.Context, comment *types.ReviewComment) error {
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = time.Now()
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO review_comments (review_id, author_name, text, created_at)
		VALUES (?, ?, ?, ?)`,
		comment.ReviewID, comment.AuthorName, comment.Text, comment.CreatedAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	comment.ID = id
	_, err = s.db.ExecContext(ctx, `
		UPDATE reviews SET comments_count = comments_count + 1 WHERE id = ?`, comment.ReviewID)
	return err
}

func (s *ReviewCommentStore) GetByReview(ctx context.Context, reviewID int64, page, limit int) ([]types.ReviewComment, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_comments WHERE review_id = ?`, reviewID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, review_id, author_name, text, created_at
		FROM review_comments
		WHERE review_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, reviewID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []types.ReviewComment
	for rows.Next() {
		var c types.ReviewComment
		if err := rows.Scan(&c.ID, &c.ReviewID, &c.AuthorName, &c.Text, &c.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan review comment: %w", err)
		}
		comments = append(comments, c)
	}
	if comments == nil {
		comments = []types.ReviewComment{}
	}
	return comments, total, nil
}

func (s *ReviewCommentStore) Delete(ctx context.Context, id int64) error {
	var reviewID int64
	if err := s.db.QueryRowContext(ctx, `SELECT review_id FROM review_comments WHERE id = ?`, id).Scan(&reviewID); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM review_comments WHERE id = ?`, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE reviews
		SET comments_count = CASE WHEN comments_count > 0 THEN comments_count - 1 ELSE 0 END
		WHERE id = ?`, reviewID)
	return err
}
