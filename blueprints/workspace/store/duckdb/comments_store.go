package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/comments"
)

// CommentsStore implements comments.Store.
type CommentsStore struct {
	db *sql.DB
}

// NewCommentsStore creates a new CommentsStore.
func NewCommentsStore(db *sql.DB) *CommentsStore {
	return &CommentsStore{db: db}
}

func (s *CommentsStore) Create(ctx context.Context, c *comments.Comment) error {
	contentJSON, _ := json.Marshal(c.Content)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO comments (id, page_id, block_id, parent_id, content, author_id, is_resolved, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.PageID, c.BlockID, c.ParentID, string(contentJSON), c.AuthorID, c.IsResolved, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *CommentsStore) GetByID(ctx context.Context, id string) (*comments.Comment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, page_id, block_id, parent_id, CAST(content AS VARCHAR), author_id, is_resolved, created_at, updated_at
		FROM comments WHERE id = ?
	`, id)
	return s.scanComment(row)
}

func (s *CommentsStore) Update(ctx context.Context, id string, content []blocks.RichText) error {
	contentJSON, _ := json.Marshal(content)
	_, err := s.db.ExecContext(ctx, "UPDATE comments SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", string(contentJSON), id)
	return err
}

func (s *CommentsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM comments WHERE id = ?", id)
	return err
}

func (s *CommentsStore) ListByPage(ctx context.Context, pageID string) ([]*comments.Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, page_id, block_id, parent_id, CAST(content AS VARCHAR), author_id, is_resolved, created_at, updated_at
		FROM comments WHERE page_id = ?
		ORDER BY created_at
	`, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanComments(rows)
}

func (s *CommentsStore) ListByBlock(ctx context.Context, blockID string) ([]*comments.Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, page_id, block_id, parent_id, CAST(content AS VARCHAR), author_id, is_resolved, created_at, updated_at
		FROM comments WHERE block_id = ?
		ORDER BY created_at
	`, blockID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanComments(rows)
}

func (s *CommentsStore) SetResolved(ctx context.Context, id string, resolved bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE comments SET is_resolved = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", resolved, id)
	return err
}

func (s *CommentsStore) scanComment(row *sql.Row) (*comments.Comment, error) {
	var c comments.Comment
	var contentJSON string
	err := row.Scan(&c.ID, &c.PageID, &c.BlockID, &c.ParentID, &contentJSON, &c.AuthorID, &c.IsResolved, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(contentJSON), &c.Content)
	return &c, nil
}

func (s *CommentsStore) scanComments(rows *sql.Rows) ([]*comments.Comment, error) {
	var result []*comments.Comment
	for rows.Next() {
		var c comments.Comment
		var contentJSON string
		err := rows.Scan(&c.ID, &c.PageID, &c.BlockID, &c.ParentID, &contentJSON, &c.AuthorID, &c.IsResolved, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(contentJSON), &c.Content)
		result = append(result, &c)
	}
	return result, rows.Err()
}
