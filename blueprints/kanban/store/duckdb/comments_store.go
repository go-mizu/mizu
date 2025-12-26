package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/comments"
)

// CommentsStore handles comment data access.
type CommentsStore struct {
	db *sql.DB
}

// NewCommentsStore creates a new comments store.
func NewCommentsStore(db *sql.DB) *CommentsStore {
	return &CommentsStore{db: db}
}

func (s *CommentsStore) Create(ctx context.Context, c *comments.Comment) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO comments (id, issue_id, author_id, content, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, c.ID, c.IssueID, c.AuthorID, c.Content, c.CreatedAt)
	return err
}

func (s *CommentsStore) GetByID(ctx context.Context, id string) (*comments.Comment, error) {
	c := &comments.Comment{}
	var editedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, issue_id, author_id, content, edited_at, created_at
		FROM comments WHERE id = $1
	`, id).Scan(&c.ID, &c.IssueID, &c.AuthorID, &c.Content, &editedAt, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if editedAt.Valid {
		c.EditedAt = &editedAt.Time
	}
	return c, err
}

func (s *CommentsStore) ListByIssue(ctx context.Context, issueID string) ([]*comments.Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, issue_id, author_id, content, edited_at, created_at
		FROM comments WHERE issue_id = $1
		ORDER BY created_at ASC
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*comments.Comment
	for rows.Next() {
		c := &comments.Comment{}
		var editedAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.IssueID, &c.AuthorID, &c.Content, &editedAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		if editedAt.Valid {
			c.EditedAt = &editedAt.Time
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (s *CommentsStore) Update(ctx context.Context, id, content string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE comments SET content = $2, edited_at = $3 WHERE id = $1
	`, id, content, time.Now())
	return err
}

func (s *CommentsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE id = $1`, id)
	return err
}

func (s *CommentsStore) CountByIssue(ctx context.Context, issueID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM comments WHERE issue_id = $1
	`, issueID).Scan(&count)
	return count, err
}
