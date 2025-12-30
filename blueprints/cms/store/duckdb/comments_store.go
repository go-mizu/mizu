package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Comment represents a WordPress comment.
type Comment struct {
	CommentID          string
	CommentPostID      string
	CommentAuthor      string
	CommentAuthorEmail string
	CommentAuthorURL   string
	CommentAuthorIP    string
	CommentDate        time.Time
	CommentDateGmt     time.Time
	CommentContent     string
	CommentKarma       int
	CommentApproved    string
	CommentAgent       string
	CommentType        string
	CommentParent      string
	UserID             string
}

// CommentsStore handles comment persistence.
type CommentsStore struct {
	db *sql.DB
}

// NewCommentsStore creates a new comments store.
func NewCommentsStore(db *sql.DB) *CommentsStore {
	return &CommentsStore{db: db}
}

// Create creates a new comment.
func (s *CommentsStore) Create(ctx context.Context, c *Comment) error {
	query := `
		INSERT INTO wp_comments (comment_ID, comment_post_ID, comment_author, comment_author_email,
			comment_author_url, comment_author_IP, comment_date, comment_date_gmt, comment_content,
			comment_karma, comment_approved, comment_agent, comment_type, comment_parent, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := s.db.ExecContext(ctx, query,
		c.CommentID, c.CommentPostID, c.CommentAuthor, c.CommentAuthorEmail,
		c.CommentAuthorURL, c.CommentAuthorIP, c.CommentDate, c.CommentDateGmt, c.CommentContent,
		c.CommentKarma, c.CommentApproved, c.CommentAgent, c.CommentType, c.CommentParent, c.UserID,
	)
	return err
}

// GetByID retrieves a comment by ID.
func (s *CommentsStore) GetByID(ctx context.Context, id string) (*Comment, error) {
	query := `
		SELECT comment_ID, comment_post_ID, comment_author, comment_author_email,
			comment_author_url, comment_author_IP, comment_date, comment_date_gmt, comment_content,
			comment_karma, comment_approved, comment_agent, comment_type, comment_parent, user_id
		FROM wp_comments WHERE comment_ID = $1
	`
	c := &Comment{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&c.CommentID, &c.CommentPostID, &c.CommentAuthor, &c.CommentAuthorEmail,
		&c.CommentAuthorURL, &c.CommentAuthorIP, &c.CommentDate, &c.CommentDateGmt, &c.CommentContent,
		&c.CommentKarma, &c.CommentApproved, &c.CommentAgent, &c.CommentType, &c.CommentParent, &c.UserID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// Update updates a comment.
func (s *CommentsStore) Update(ctx context.Context, c *Comment) error {
	query := `
		UPDATE wp_comments SET
			comment_post_ID = $2, comment_author = $3, comment_author_email = $4,
			comment_author_url = $5, comment_author_IP = $6, comment_date = $7, comment_date_gmt = $8,
			comment_content = $9, comment_karma = $10, comment_approved = $11, comment_agent = $12,
			comment_type = $13, comment_parent = $14, user_id = $15
		WHERE comment_ID = $1
	`
	_, err := s.db.ExecContext(ctx, query,
		c.CommentID, c.CommentPostID, c.CommentAuthor, c.CommentAuthorEmail,
		c.CommentAuthorURL, c.CommentAuthorIP, c.CommentDate, c.CommentDateGmt, c.CommentContent,
		c.CommentKarma, c.CommentApproved, c.CommentAgent, c.CommentType, c.CommentParent, c.UserID,
	)
	return err
}

// Delete deletes a comment by ID.
func (s *CommentsStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM wp_comments WHERE comment_ID = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// UpdateStatus updates the approval status of a comment.
func (s *CommentsStore) UpdateStatus(ctx context.Context, id, status string) error {
	query := `UPDATE wp_comments SET comment_approved = $2 WHERE comment_ID = $1`
	_, err := s.db.ExecContext(ctx, query, id, status)
	return err
}

// CommentListOpts contains options for listing comments.
type CommentListOpts struct {
	Limit       int
	Offset      int
	Page        int
	PerPage     int
	OrderBy     string
	Order       string
	PostID      string
	Status      string
	Type        string
	Parent      *string
	AuthorEmail string
	Search      string
	Include     []string
	Exclude     []string
	After       *time.Time
	Before      *time.Time
}

// List lists comments with filtering and pagination.
func (s *CommentsStore) List(ctx context.Context, opts CommentListOpts) ([]*Comment, int, error) {
	var args []interface{}
	var where []string
	argNum := 1

	if opts.PostID != "" {
		where = append(where, fmt.Sprintf("comment_post_ID = $%d", argNum))
		args = append(args, opts.PostID)
		argNum++
	}

	if opts.Status != "" {
		where = append(where, fmt.Sprintf("comment_approved = $%d", argNum))
		args = append(args, opts.Status)
		argNum++
	}

	if opts.Type != "" {
		where = append(where, fmt.Sprintf("comment_type = $%d", argNum))
		args = append(args, opts.Type)
		argNum++
	}

	if opts.Parent != nil {
		where = append(where, fmt.Sprintf("comment_parent = $%d", argNum))
		args = append(args, *opts.Parent)
		argNum++
	}

	if opts.AuthorEmail != "" {
		where = append(where, fmt.Sprintf("comment_author_email = $%d", argNum))
		args = append(args, opts.AuthorEmail)
		argNum++
	}

	if opts.Search != "" {
		where = append(where, fmt.Sprintf("LOWER(comment_content) LIKE $%d", argNum))
		args = append(args, "%"+strings.ToLower(opts.Search)+"%")
		argNum++
	}

	if opts.After != nil {
		where = append(where, fmt.Sprintf("comment_date_gmt > $%d", argNum))
		args = append(args, *opts.After)
		argNum++
	}

	if opts.Before != nil {
		where = append(where, fmt.Sprintf("comment_date_gmt < $%d", argNum))
		args = append(args, *opts.Before)
		argNum++
	}

	baseQuery := `SELECT comment_ID, comment_post_ID, comment_author, comment_author_email,
		comment_author_url, comment_author_IP, comment_date, comment_date_gmt, comment_content,
		comment_karma, comment_approved, comment_agent, comment_type, comment_parent, user_id
		FROM wp_comments`
	countQuery := `SELECT COUNT(*) FROM wp_comments`

	if len(where) > 0 {
		whereClause := " WHERE " + strings.Join(where, " AND ")
		baseQuery += whereClause
		countQuery += whereClause
	}

	// Order
	orderBy := "comment_date_gmt"
	if opts.OrderBy != "" {
		orderBy = opts.OrderBy
	}
	order := "DESC"
	if opts.Order != "" {
		order = strings.ToUpper(opts.Order)
	}
	baseQuery += fmt.Sprintf(" ORDER BY %s %s", orderBy, order)

	// Pagination
	limit := opts.Limit
	if opts.PerPage > 0 {
		limit = opts.PerPage
	}
	if limit == 0 {
		limit = 10
	}

	offset := opts.Offset
	if opts.Page > 0 {
		offset = (opts.Page - 1) * limit
	}

	baseQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	// Execute count query
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Execute main query
	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []*Comment
	for rows.Next() {
		c := &Comment{}
		if err := rows.Scan(
			&c.CommentID, &c.CommentPostID, &c.CommentAuthor, &c.CommentAuthorEmail,
			&c.CommentAuthorURL, &c.CommentAuthorIP, &c.CommentDate, &c.CommentDateGmt, &c.CommentContent,
			&c.CommentKarma, &c.CommentApproved, &c.CommentAgent, &c.CommentType, &c.CommentParent, &c.UserID,
		); err != nil {
			return nil, 0, err
		}
		comments = append(comments, c)
	}

	return comments, total, rows.Err()
}

// CountByPost returns the count of approved comments for a post.
func (s *CommentsStore) CountByPost(ctx context.Context, postID string) (int, error) {
	query := `SELECT COUNT(*) FROM wp_comments WHERE comment_post_ID = $1 AND comment_approved = '1'`
	var count int
	err := s.db.QueryRowContext(ctx, query, postID).Scan(&count)
	return count, err
}

// CountByStatus returns counts of comments by status.
func (s *CommentsStore) CountByStatus(ctx context.Context) (map[string]int, error) {
	query := `SELECT comment_approved, COUNT(*) FROM wp_comments GROUP BY comment_approved`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

// GetChildren retrieves all child comments for a parent.
func (s *CommentsStore) GetChildren(ctx context.Context, parentID string) ([]*Comment, error) {
	query := `
		SELECT comment_ID, comment_post_ID, comment_author, comment_author_email,
			comment_author_url, comment_author_IP, comment_date, comment_date_gmt, comment_content,
			comment_karma, comment_approved, comment_agent, comment_type, comment_parent, user_id
		FROM wp_comments WHERE comment_parent = $1
		ORDER BY comment_date_gmt ASC
	`
	rows, err := s.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*Comment
	for rows.Next() {
		c := &Comment{}
		if err := rows.Scan(
			&c.CommentID, &c.CommentPostID, &c.CommentAuthor, &c.CommentAuthorEmail,
			&c.CommentAuthorURL, &c.CommentAuthorIP, &c.CommentDate, &c.CommentDateGmt, &c.CommentContent,
			&c.CommentKarma, &c.CommentApproved, &c.CommentAgent, &c.CommentType, &c.CommentParent, &c.UserID,
		); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}
