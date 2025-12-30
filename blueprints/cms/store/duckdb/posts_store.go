package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/feature/posts"
)

// PostsStore handles post data access.
type PostsStore struct {
	db *sql.DB
}

// NewPostsStore creates a new posts store.
func NewPostsStore(db *sql.DB) *PostsStore {
	return &PostsStore{db: db}
}

func (s *PostsStore) Create(ctx context.Context, p *posts.Post) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO posts (id, author_id, title, slug, excerpt, content, content_format, featured_image_id, status, visibility, password, published_at, scheduled_at, meta, reading_time, word_count, allow_comments, is_featured, is_sticky, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`, p.ID, p.AuthorID, p.Title, p.Slug, p.Excerpt, p.Content, p.ContentFormat, nullString(p.FeaturedImageID), p.Status, p.Visibility, nullString(p.Password), nullTime(p.PublishedAt), nullTime(p.ScheduledAt), nullString(p.Meta), p.ReadingTime, p.WordCount, p.AllowComments, p.IsFeatured, p.IsSticky, p.SortOrder, p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *PostsStore) GetByID(ctx context.Context, id string) (*posts.Post, error) {
	return s.scanPost(s.db.QueryRowContext(ctx, `
		SELECT id, author_id, title, slug, excerpt, content, content_format, featured_image_id, status, visibility, password, published_at, scheduled_at, meta, reading_time, word_count, allow_comments, is_featured, is_sticky, sort_order, created_at, updated_at
		FROM posts WHERE id = $1
	`, id))
}

func (s *PostsStore) GetBySlug(ctx context.Context, slug string) (*posts.Post, error) {
	return s.scanPost(s.db.QueryRowContext(ctx, `
		SELECT id, author_id, title, slug, excerpt, content, content_format, featured_image_id, status, visibility, password, published_at, scheduled_at, meta, reading_time, word_count, allow_comments, is_featured, is_sticky, sort_order, created_at, updated_at
		FROM posts WHERE slug = $1
	`, slug))
}

func (s *PostsStore) List(ctx context.Context, in *posts.ListIn) ([]*posts.Post, int, error) {
	var conditions []string
	var args []any
	argNum := 1

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
	if in.Visibility != "" {
		conditions = append(conditions, fmt.Sprintf("visibility = $%d", argNum))
		args = append(args, in.Visibility)
		argNum++
	}
	if in.IsFeatured != nil {
		conditions = append(conditions, fmt.Sprintf("is_featured = $%d", argNum))
		args = append(args, *in.IsFeatured)
		argNum++
	}
	if in.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR content ILIKE $%d)", argNum, argNum))
		args = append(args, "%"+in.Search+"%")
		argNum++
	}
	if in.CategoryID != "" {
		conditions = append(conditions, fmt.Sprintf("id IN (SELECT post_id FROM post_categories WHERE category_id = $%d)", argNum))
		args = append(args, in.CategoryID)
		argNum++
	}
	if in.TagID != "" {
		conditions = append(conditions, fmt.Sprintf("id IN (SELECT post_id FROM post_tags WHERE tag_id = $%d)", argNum))
		args = append(args, in.TagID)
		argNum++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM posts %s", where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Determine order
	orderBy := in.OrderBy
	if orderBy == "" {
		orderBy = "created_at"
	}
	order := in.Order
	if order == "" {
		order = "DESC"
	}

	// Get items
	args = append(args, in.Limit, in.Offset)
	query := fmt.Sprintf(`
		SELECT id, author_id, title, slug, excerpt, content, content_format, featured_image_id, status, visibility, password, published_at, scheduled_at, meta, reading_time, word_count, allow_comments, is_featured, is_sticky, sort_order, created_at, updated_at
		FROM posts %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, orderBy, order, argNum, argNum+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*posts.Post
	for rows.Next() {
		p, err := s.scanPostRow(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, p)
	}
	return list, total, rows.Err()
}

func (s *PostsStore) Update(ctx context.Context, id string, in *posts.UpdateIn) error {
	var sets []string
	var args []any
	argNum := 1

	if in.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", argNum))
		args = append(args, *in.Title)
		argNum++
	}
	if in.Slug != nil {
		sets = append(sets, fmt.Sprintf("slug = $%d", argNum))
		args = append(args, *in.Slug)
		argNum++
	}
	if in.Excerpt != nil {
		sets = append(sets, fmt.Sprintf("excerpt = $%d", argNum))
		args = append(args, *in.Excerpt)
		argNum++
	}
	if in.Content != nil {
		sets = append(sets, fmt.Sprintf("content = $%d", argNum))
		args = append(args, *in.Content)
		argNum++
	}
	if in.ContentFormat != nil {
		sets = append(sets, fmt.Sprintf("content_format = $%d", argNum))
		args = append(args, *in.ContentFormat)
		argNum++
	}
	if in.FeaturedImageID != nil {
		sets = append(sets, fmt.Sprintf("featured_image_id = $%d", argNum))
		args = append(args, nullString(*in.FeaturedImageID))
		argNum++
	}
	if in.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *in.Status)
		argNum++
	}
	if in.Visibility != nil {
		sets = append(sets, fmt.Sprintf("visibility = $%d", argNum))
		args = append(args, *in.Visibility)
		argNum++
	}
	if in.Password != nil {
		sets = append(sets, fmt.Sprintf("password = $%d", argNum))
		args = append(args, nullString(*in.Password))
		argNum++
	}
	if in.PublishedAt != nil {
		sets = append(sets, fmt.Sprintf("published_at = $%d", argNum))
		args = append(args, *in.PublishedAt)
		argNum++
	}
	if in.ScheduledAt != nil {
		sets = append(sets, fmt.Sprintf("scheduled_at = $%d", argNum))
		args = append(args, *in.ScheduledAt)
		argNum++
	}
	if in.Meta != nil {
		sets = append(sets, fmt.Sprintf("meta = $%d", argNum))
		args = append(args, nullString(*in.Meta))
		argNum++
	}
	if in.AllowComments != nil {
		sets = append(sets, fmt.Sprintf("allow_comments = $%d", argNum))
		args = append(args, *in.AllowComments)
		argNum++
	}
	if in.IsFeatured != nil {
		sets = append(sets, fmt.Sprintf("is_featured = $%d", argNum))
		args = append(args, *in.IsFeatured)
		argNum++
	}
	if in.IsSticky != nil {
		sets = append(sets, fmt.Sprintf("is_sticky = $%d", argNum))
		args = append(args, *in.IsSticky)
		argNum++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++

	args = append(args, id)
	query := fmt.Sprintf("UPDATE posts SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *PostsStore) Delete(ctx context.Context, id string) error {
	// Delete relationships first
	s.db.ExecContext(ctx, `DELETE FROM post_categories WHERE post_id = $1`, id)
	s.db.ExecContext(ctx, `DELETE FROM post_tags WHERE post_id = $1`, id)
	s.db.ExecContext(ctx, `DELETE FROM post_media WHERE post_id = $1`, id)
	s.db.ExecContext(ctx, `DELETE FROM comments WHERE post_id = $1`, id)

	_, err := s.db.ExecContext(ctx, `DELETE FROM posts WHERE id = $1`, id)
	return err
}

func (s *PostsStore) GetCategoryIDs(ctx context.Context, postID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT category_id FROM post_categories WHERE post_id = $1 ORDER BY sort_order
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *PostsStore) GetTagIDs(ctx context.Context, postID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT tag_id FROM post_tags WHERE post_id = $1
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *PostsStore) SetCategories(ctx context.Context, postID string, categoryIDs []string) error {
	// Delete existing
	if _, err := s.db.ExecContext(ctx, `DELETE FROM post_categories WHERE post_id = $1`, postID); err != nil {
		return err
	}

	// Insert new
	for i, catID := range categoryIDs {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO post_categories (post_id, category_id, sort_order) VALUES ($1, $2, $3)
		`, postID, catID, i); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostsStore) SetTags(ctx context.Context, postID string, tagIDs []string) error {
	// Delete existing
	if _, err := s.db.ExecContext(ctx, `DELETE FROM post_tags WHERE post_id = $1`, postID); err != nil {
		return err
	}

	// Insert new
	for _, tagID := range tagIDs {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO post_tags (post_id, tag_id) VALUES ($1, $2)
		`, postID, tagID); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostsStore) scanPost(row *sql.Row) (*posts.Post, error) {
	p := &posts.Post{}
	var excerpt, content, featuredImageID, password, meta sql.NullString
	var publishedAt, scheduledAt sql.NullTime
	err := row.Scan(&p.ID, &p.AuthorID, &p.Title, &p.Slug, &excerpt, &content, &p.ContentFormat, &featuredImageID, &p.Status, &p.Visibility, &password, &publishedAt, &scheduledAt, &meta, &p.ReadingTime, &p.WordCount, &p.AllowComments, &p.IsFeatured, &p.IsSticky, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Excerpt = excerpt.String
	p.Content = content.String
	p.FeaturedImageID = featuredImageID.String
	p.Password = password.String
	p.Meta = meta.String
	if publishedAt.Valid {
		p.PublishedAt = &publishedAt.Time
	}
	if scheduledAt.Valid {
		p.ScheduledAt = &scheduledAt.Time
	}
	return p, nil
}

func (s *PostsStore) scanPostRow(rows *sql.Rows) (*posts.Post, error) {
	p := &posts.Post{}
	var excerpt, content, featuredImageID, password, meta sql.NullString
	var publishedAt, scheduledAt sql.NullTime
	err := rows.Scan(&p.ID, &p.AuthorID, &p.Title, &p.Slug, &excerpt, &content, &p.ContentFormat, &featuredImageID, &p.Status, &p.Visibility, &password, &publishedAt, &scheduledAt, &meta, &p.ReadingTime, &p.WordCount, &p.AllowComments, &p.IsFeatured, &p.IsSticky, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.Excerpt = excerpt.String
	p.Content = content.String
	p.FeaturedImageID = featuredImageID.String
	p.Password = password.String
	p.Meta = meta.String
	if publishedAt.Valid {
		p.PublishedAt = &publishedAt.Time
	}
	if scheduledAt.Valid {
		p.ScheduledAt = &scheduledAt.Time
	}
	return p, nil
}

// Helper functions
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
