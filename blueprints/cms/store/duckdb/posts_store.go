package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Post represents a WordPress post (posts, pages, attachments, revisions, custom post types).
type Post struct {
	ID                  string
	PostAuthor          string
	PostDate            time.Time
	PostDateGmt         time.Time
	PostContent         string
	PostTitle           string
	PostExcerpt         string
	PostStatus          string
	CommentStatus       string
	PingStatus          string
	PostPassword        string
	PostName            string
	ToPing              string
	Pinged              string
	PostModified        time.Time
	PostModifiedGmt     time.Time
	PostContentFiltered string
	PostParent          string
	Guid                string
	MenuOrder           int
	PostType            string
	PostMimeType        string
	CommentCount        int64
}

// PostsStore handles post persistence.
type PostsStore struct {
	db *sql.DB
}

// NewPostsStore creates a new posts store.
func NewPostsStore(db *sql.DB) *PostsStore {
	return &PostsStore{db: db}
}

// Create creates a new post.
func (s *PostsStore) Create(ctx context.Context, p *Post) error {
	query := `
		INSERT INTO wp_posts (ID, post_author, post_date, post_date_gmt, post_content, post_title,
			post_excerpt, post_status, comment_status, ping_status, post_password, post_name,
			to_ping, pinged, post_modified, post_modified_gmt, post_content_filtered, post_parent,
			guid, menu_order, post_type, post_mime_type, comment_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
	`
	_, err := s.db.ExecContext(ctx, query,
		p.ID, p.PostAuthor, p.PostDate, p.PostDateGmt, p.PostContent, p.PostTitle,
		p.PostExcerpt, p.PostStatus, p.CommentStatus, p.PingStatus, p.PostPassword, p.PostName,
		p.ToPing, p.Pinged, p.PostModified, p.PostModifiedGmt, p.PostContentFiltered, p.PostParent,
		p.Guid, p.MenuOrder, p.PostType, p.PostMimeType, p.CommentCount,
	)
	return err
}

// GetByID retrieves a post by ID.
func (s *PostsStore) GetByID(ctx context.Context, id string) (*Post, error) {
	query := `
		SELECT ID, post_author, post_date, post_date_gmt, post_content, post_title,
			post_excerpt, post_status, comment_status, ping_status, post_password, post_name,
			to_ping, pinged, post_modified, post_modified_gmt, post_content_filtered, post_parent,
			guid, menu_order, post_type, post_mime_type, comment_count
		FROM wp_posts WHERE ID = $1
	`
	return s.scanPost(s.db.QueryRowContext(ctx, query, id))
}

// GetBySlug retrieves a post by slug (post_name) and post type.
func (s *PostsStore) GetBySlug(ctx context.Context, slug, postType string) (*Post, error) {
	query := `
		SELECT ID, post_author, post_date, post_date_gmt, post_content, post_title,
			post_excerpt, post_status, comment_status, ping_status, post_password, post_name,
			to_ping, pinged, post_modified, post_modified_gmt, post_content_filtered, post_parent,
			guid, menu_order, post_type, post_mime_type, comment_count
		FROM wp_posts WHERE post_name = $1 AND post_type = $2
	`
	return s.scanPost(s.db.QueryRowContext(ctx, query, slug, postType))
}

// GetByGUID retrieves a post by GUID.
func (s *PostsStore) GetByGUID(ctx context.Context, guid string) (*Post, error) {
	query := `
		SELECT ID, post_author, post_date, post_date_gmt, post_content, post_title,
			post_excerpt, post_status, comment_status, ping_status, post_password, post_name,
			to_ping, pinged, post_modified, post_modified_gmt, post_content_filtered, post_parent,
			guid, menu_order, post_type, post_mime_type, comment_count
		FROM wp_posts WHERE guid = $1
	`
	return s.scanPost(s.db.QueryRowContext(ctx, query, guid))
}

func (s *PostsStore) scanPost(row *sql.Row) (*Post, error) {
	p := &Post{}
	err := row.Scan(
		&p.ID, &p.PostAuthor, &p.PostDate, &p.PostDateGmt, &p.PostContent, &p.PostTitle,
		&p.PostExcerpt, &p.PostStatus, &p.CommentStatus, &p.PingStatus, &p.PostPassword, &p.PostName,
		&p.ToPing, &p.Pinged, &p.PostModified, &p.PostModifiedGmt, &p.PostContentFiltered, &p.PostParent,
		&p.Guid, &p.MenuOrder, &p.PostType, &p.PostMimeType, &p.CommentCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// Update updates a post.
func (s *PostsStore) Update(ctx context.Context, p *Post) error {
	query := `
		UPDATE wp_posts SET
			post_author = $2, post_date = $3, post_date_gmt = $4, post_content = $5, post_title = $6,
			post_excerpt = $7, post_status = $8, comment_status = $9, ping_status = $10, post_password = $11,
			post_name = $12, to_ping = $13, pinged = $14, post_modified = $15, post_modified_gmt = $16,
			post_content_filtered = $17, post_parent = $18, guid = $19, menu_order = $20, post_type = $21,
			post_mime_type = $22, comment_count = $23
		WHERE ID = $1
	`
	_, err := s.db.ExecContext(ctx, query,
		p.ID, p.PostAuthor, p.PostDate, p.PostDateGmt, p.PostContent, p.PostTitle,
		p.PostExcerpt, p.PostStatus, p.CommentStatus, p.PingStatus, p.PostPassword, p.PostName,
		p.ToPing, p.Pinged, p.PostModified, p.PostModifiedGmt, p.PostContentFiltered, p.PostParent,
		p.Guid, p.MenuOrder, p.PostType, p.PostMimeType, p.CommentCount,
	)
	return err
}

// Delete deletes a post by ID.
func (s *PostsStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM wp_posts WHERE ID = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// PostListOpts contains options for listing posts.
type PostListOpts struct {
	Limit         int
	Offset        int
	Page          int
	PerPage       int
	OrderBy       string
	Order         string
	Status        []string
	PostType      []string
	Author        []string
	AuthorExclude []string
	Include       []string
	Exclude       []string
	Parent        []string
	ParentExclude []string
	Search        string
	Slug          []string
	After         *time.Time
	Before        *time.Time
	ModifiedAfter  *time.Time
	ModifiedBefore *time.Time
	Sticky        *bool
	Categories    []string
	CategoriesExclude []string
	Tags          []string
	TagsExclude   []string
}

// List lists posts with filtering and pagination.
func (s *PostsStore) List(ctx context.Context, opts PostListOpts) ([]*Post, int, error) {
	var args []interface{}
	var where []string
	argNum := 1

	// Build WHERE clauses
	if len(opts.Status) > 0 {
		placeholders := make([]string, len(opts.Status))
		for i, status := range opts.Status {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, status)
			argNum++
		}
		where = append(where, "post_status IN ("+strings.Join(placeholders, ",")+")")
	}

	if len(opts.PostType) > 0 {
		placeholders := make([]string, len(opts.PostType))
		for i, pt := range opts.PostType {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, pt)
			argNum++
		}
		where = append(where, "post_type IN ("+strings.Join(placeholders, ",")+")")
	}

	if len(opts.Author) > 0 {
		placeholders := make([]string, len(opts.Author))
		for i, a := range opts.Author {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, a)
			argNum++
		}
		where = append(where, "post_author IN ("+strings.Join(placeholders, ",")+")")
	}

	if opts.Search != "" {
		where = append(where, fmt.Sprintf("(LOWER(post_title) LIKE $%d OR LOWER(post_content) LIKE $%d)", argNum, argNum))
		args = append(args, "%"+strings.ToLower(opts.Search)+"%")
		argNum++
	}

	if opts.After != nil {
		where = append(where, fmt.Sprintf("post_date_gmt > $%d", argNum))
		args = append(args, *opts.After)
		argNum++
	}

	if opts.Before != nil {
		where = append(where, fmt.Sprintf("post_date_gmt < $%d", argNum))
		args = append(args, *opts.Before)
		argNum++
	}

	if len(opts.Include) > 0 {
		placeholders := make([]string, len(opts.Include))
		for i, id := range opts.Include {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, id)
			argNum++
		}
		where = append(where, "ID IN ("+strings.Join(placeholders, ",")+")")
	}

	if len(opts.Exclude) > 0 {
		placeholders := make([]string, len(opts.Exclude))
		for i, id := range opts.Exclude {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, id)
			argNum++
		}
		where = append(where, "ID NOT IN ("+strings.Join(placeholders, ",")+")")
	}

	if len(opts.Slug) > 0 {
		placeholders := make([]string, len(opts.Slug))
		for i, slug := range opts.Slug {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, slug)
			argNum++
		}
		where = append(where, "post_name IN ("+strings.Join(placeholders, ",")+")")
	}

	// Build queries
	baseQuery := `SELECT ID, post_author, post_date, post_date_gmt, post_content, post_title,
		post_excerpt, post_status, comment_status, ping_status, post_password, post_name,
		to_ping, pinged, post_modified, post_modified_gmt, post_content_filtered, post_parent,
		guid, menu_order, post_type, post_mime_type, comment_count FROM wp_posts`
	countQuery := `SELECT COUNT(*) FROM wp_posts`

	if len(where) > 0 {
		whereClause := " WHERE " + strings.Join(where, " AND ")
		baseQuery += whereClause
		countQuery += whereClause
	}

	// Order
	orderBy := "post_date"
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

	var posts []*Post
	for rows.Next() {
		p := &Post{}
		if err := rows.Scan(
			&p.ID, &p.PostAuthor, &p.PostDate, &p.PostDateGmt, &p.PostContent, &p.PostTitle,
			&p.PostExcerpt, &p.PostStatus, &p.CommentStatus, &p.PingStatus, &p.PostPassword, &p.PostName,
			&p.ToPing, &p.Pinged, &p.PostModified, &p.PostModifiedGmt, &p.PostContentFiltered, &p.PostParent,
			&p.Guid, &p.MenuOrder, &p.PostType, &p.PostMimeType, &p.CommentCount,
		); err != nil {
			return nil, 0, err
		}
		posts = append(posts, p)
	}

	return posts, total, rows.Err()
}

// Count returns the total number of posts by type and status.
func (s *PostsStore) Count(ctx context.Context, postType, status string) (int, error) {
	query := `SELECT COUNT(*) FROM wp_posts WHERE post_type = $1`
	args := []interface{}{postType}
	if status != "" {
		query += " AND post_status = $2"
		args = append(args, status)
	}
	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// GetRevisions retrieves all revisions for a post.
func (s *PostsStore) GetRevisions(ctx context.Context, postID string) ([]*Post, error) {
	query := `
		SELECT ID, post_author, post_date, post_date_gmt, post_content, post_title,
			post_excerpt, post_status, comment_status, ping_status, post_password, post_name,
			to_ping, pinged, post_modified, post_modified_gmt, post_content_filtered, post_parent,
			guid, menu_order, post_type, post_mime_type, comment_count
		FROM wp_posts WHERE post_parent = $1 AND post_type = 'revision'
		ORDER BY post_date DESC
	`
	rows, err := s.db.QueryContext(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []*Post
	for rows.Next() {
		p := &Post{}
		if err := rows.Scan(
			&p.ID, &p.PostAuthor, &p.PostDate, &p.PostDateGmt, &p.PostContent, &p.PostTitle,
			&p.PostExcerpt, &p.PostStatus, &p.CommentStatus, &p.PingStatus, &p.PostPassword, &p.PostName,
			&p.ToPing, &p.Pinged, &p.PostModified, &p.PostModifiedGmt, &p.PostContentFiltered, &p.PostParent,
			&p.Guid, &p.MenuOrder, &p.PostType, &p.PostMimeType, &p.CommentCount,
		); err != nil {
			return nil, err
		}
		revisions = append(revisions, p)
	}
	return revisions, rows.Err()
}

// IncrementCommentCount increments the comment count for a post.
func (s *PostsStore) IncrementCommentCount(ctx context.Context, postID string, delta int) error {
	query := `UPDATE wp_posts SET comment_count = comment_count + $2 WHERE ID = $1`
	_, err := s.db.ExecContext(ctx, query, postID, delta)
	return err
}
