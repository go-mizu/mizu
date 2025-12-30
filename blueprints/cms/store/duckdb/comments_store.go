package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/feature/comments"
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
		INSERT INTO comments (id, post_id, parent_id, author_id, author_name, author_email, author_url, content, status, ip_address, user_agent, likes_count, meta, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, c.ID, c.PostID, nullString(c.ParentID), nullString(c.AuthorID), nullString(c.AuthorName), nullString(c.AuthorEmail), nullString(c.AuthorURL), c.Content, c.Status, nullString(c.IPAddress), nullString(c.UserAgent), c.LikesCount, nullString(c.Meta), c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *CommentsStore) GetByID(ctx context.Context, id string) (*comments.Comment, error) {
	return s.scanComment(s.db.QueryRowContext(ctx, `
		SELECT id, post_id, parent_id, author_id, author_name, author_email, author_url, content, status, ip_address, user_agent, likes_count, meta, created_at, updated_at
		FROM comments WHERE id = $1
	`, id))
}

func (s *CommentsStore) ListByPost(ctx context.Context, postID string, in *comments.ListIn) ([]*comments.Comment, int, error) {
	in.PostID = postID
	return s.List(ctx, in)
}

func (s *CommentsStore) List(ctx context.Context, in *comments.ListIn) ([]*comments.Comment, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	if in.PostID != "" {
		conditions = append(conditions, fmt.Sprintf("post_id = $%d", argNum))
		args = append(args, in.PostID)
		argNum++
	}
	if in.ParentID != "" {
		conditions = append(conditions, fmt.Sprintf("parent_id = $%d", argNum))
		args = append(args, in.ParentID)
		argNum++
	}
	if in.AuthorID != "" {
		conditions = append(conditions, fmt.Sprintf("author_id = $%d", argNum))
		args = append(args, in.AuthorID)
		argNum++
	}
	if in.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, in.Status)
		argNum++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM comments %s", where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get items
	args = append(args, in.Limit, in.Offset)
	query := fmt.Sprintf(`
		SELECT id, post_id, parent_id, author_id, author_name, author_email, author_url, content, status, ip_address, user_agent, likes_count, meta, created_at, updated_at
		FROM comments %s
		ORDER BY created_at ASC
		LIMIT $%d OFFSET $%d
	`, where, argNum, argNum+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*comments.Comment
	for rows.Next() {
		c, err := s.scanCommentRow(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, c)
	}
	return list, total, rows.Err()
}

func (s *CommentsStore) Update(ctx context.Context, id string, in *comments.UpdateIn) error {
	var sets []string
	var args []any
	argNum := 1

	if in.Content != nil {
		sets = append(sets, fmt.Sprintf("content = $%d", argNum))
		args = append(args, *in.Content)
		argNum++
	}
	if in.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *in.Status)
		argNum++
	}
	if in.Meta != nil {
		sets = append(sets, fmt.Sprintf("meta = $%d", argNum))
		args = append(args, nullString(*in.Meta))
		argNum++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++

	args = append(args, id)
	query := fmt.Sprintf("UPDATE comments SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *CommentsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE id = $1`, id)
	return err
}

func (s *CommentsStore) CountByPost(ctx context.Context, postID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comments WHERE post_id = $1 AND status = 'approved'`, postID).Scan(&count)
	return count, err
}

func (s *CommentsStore) scanComment(row *sql.Row) (*comments.Comment, error) {
	c := &comments.Comment{}
	var parentID, authorID, authorName, authorEmail, authorURL, ipAddress, userAgent, meta sql.NullString
	err := row.Scan(&c.ID, &c.PostID, &parentID, &authorID, &authorName, &authorEmail, &authorURL, &c.Content, &c.Status, &ipAddress, &userAgent, &c.LikesCount, &meta, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.ParentID = parentID.String
	c.AuthorID = authorID.String
	c.AuthorName = authorName.String
	c.AuthorEmail = authorEmail.String
	c.AuthorURL = authorURL.String
	c.IPAddress = ipAddress.String
	c.UserAgent = userAgent.String
	c.Meta = meta.String
	return c, nil
}

func (s *CommentsStore) scanCommentRow(rows *sql.Rows) (*comments.Comment, error) {
	c := &comments.Comment{}
	var parentID, authorID, authorName, authorEmail, authorURL, ipAddress, userAgent, meta sql.NullString
	err := rows.Scan(&c.ID, &c.PostID, &parentID, &authorID, &authorName, &authorEmail, &authorURL, &c.Content, &c.Status, &ipAddress, &userAgent, &c.LikesCount, &meta, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	c.ParentID = parentID.String
	c.AuthorID = authorID.String
	c.AuthorName = authorName.String
	c.AuthorEmail = authorEmail.String
	c.AuthorURL = authorURL.String
	c.IPAddress = ipAddress.String
	c.UserAgent = userAgent.String
	c.Meta = meta.String
	return c, nil
}
