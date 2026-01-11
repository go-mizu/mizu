package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/table/feature/comments"
)

// CommentsStore provides PostgreSQL-based comment storage.
type CommentsStore struct {
	db *sql.DB
}

// NewCommentsStore creates a new comments store.
func NewCommentsStore(db *sql.DB) *CommentsStore {
	return &CommentsStore{db: db}
}

// Create creates a new comment.
func (s *CommentsStore) Create(ctx context.Context, comment *comments.Comment) error {
	now := time.Now()
	comment.CreatedAt = now
	comment.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO comments (id, record_id, parent_id, user_id, content, is_resolved, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, comment.ID, comment.RecordID, nullString(comment.ParentID), comment.UserID, comment.Content, comment.IsResolved, comment.CreatedAt, comment.UpdatedAt)
	return err
}

// GetByID retrieves a comment by ID.
func (s *CommentsStore) GetByID(ctx context.Context, id string) (*comments.Comment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, record_id, parent_id, user_id, content, is_resolved, created_at, updated_at
		FROM comments WHERE id = $1
	`, id)
	return s.scanComment(row)
}

// Update updates a comment.
func (s *CommentsStore) Update(ctx context.Context, comment *comments.Comment) error {
	comment.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		UPDATE comments SET
			content = $1, is_resolved = $2, updated_at = $3
		WHERE id = $4
	`, comment.Content, comment.IsResolved, comment.UpdatedAt, comment.ID)
	return err
}

// Delete deletes a comment.
func (s *CommentsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE id = $1`, id)
	return err
}

// ListByRecord lists all comments for a record.
func (s *CommentsStore) ListByRecord(ctx context.Context, recordID string) ([]*comments.Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, record_id, parent_id, user_id, content, is_resolved, created_at, updated_at
		FROM comments WHERE record_id = $1
		ORDER BY created_at ASC
	`, recordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commentList []*comments.Comment
	for rows.Next() {
		comment, err := s.scanCommentRows(rows)
		if err != nil {
			return nil, err
		}
		commentList = append(commentList, comment)
	}
	return commentList, rows.Err()
}

// DeleteByRecord deletes all comments for a record.
func (s *CommentsStore) DeleteByRecord(ctx context.Context, recordID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE record_id = $1`, recordID)
	return err
}

func (s *CommentsStore) scanComment(row *sql.Row) (*comments.Comment, error) {
	comment := &comments.Comment{}
	var parentID sql.NullString

	err := row.Scan(&comment.ID, &comment.RecordID, &parentID, &comment.UserID, &comment.Content, &comment.IsResolved, &comment.CreatedAt, &comment.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, comments.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		comment.ParentID = parentID.String
	}

	return comment, nil
}

func (s *CommentsStore) scanCommentRows(rows *sql.Rows) (*comments.Comment, error) {
	comment := &comments.Comment{}
	var parentID sql.NullString

	err := rows.Scan(&comment.ID, &comment.RecordID, &parentID, &comment.UserID, &comment.Content, &comment.IsResolved, &comment.CreatedAt, &comment.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		comment.ParentID = parentID.String
	}

	return comment, nil
}
