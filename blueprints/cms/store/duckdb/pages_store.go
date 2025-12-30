package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/feature/pages"
)

// PagesStore handles page data access.
type PagesStore struct {
	db *sql.DB
}

// NewPagesStore creates a new pages store.
func NewPagesStore(db *sql.DB) *PagesStore {
	return &PagesStore{db: db}
}

func (s *PagesStore) Create(ctx context.Context, p *pages.Page) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pages (id, author_id, parent_id, title, slug, content, content_format, featured_image_id, template, status, visibility, meta, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, p.ID, p.AuthorID, nullString(p.ParentID), p.Title, p.Slug, p.Content, p.ContentFormat, nullString(p.FeaturedImageID), nullString(p.Template), p.Status, p.Visibility, nullString(p.Meta), p.SortOrder, p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *PagesStore) GetByID(ctx context.Context, id string) (*pages.Page, error) {
	return s.scanPage(s.db.QueryRowContext(ctx, `
		SELECT id, author_id, parent_id, title, slug, content, content_format, featured_image_id, template, status, visibility, meta, sort_order, created_at, updated_at
		FROM pages WHERE id = $1
	`, id))
}

func (s *PagesStore) GetBySlug(ctx context.Context, slug string) (*pages.Page, error) {
	return s.scanPage(s.db.QueryRowContext(ctx, `
		SELECT id, author_id, parent_id, title, slug, content, content_format, featured_image_id, template, status, visibility, meta, sort_order, created_at, updated_at
		FROM pages WHERE slug = $1
	`, slug))
}

func (s *PagesStore) GetByParentAndSlug(ctx context.Context, parentID, slug string) (*pages.Page, error) {
	var query string
	var args []any

	if parentID == "" {
		query = `
			SELECT id, author_id, parent_id, title, slug, content, content_format, featured_image_id, template, status, visibility, meta, sort_order, created_at, updated_at
			FROM pages WHERE parent_id IS NULL AND slug = $1
		`
		args = []any{slug}
	} else {
		query = `
			SELECT id, author_id, parent_id, title, slug, content, content_format, featured_image_id, template, status, visibility, meta, sort_order, created_at, updated_at
			FROM pages WHERE parent_id = $1 AND slug = $2
		`
		args = []any{parentID, slug}
	}

	return s.scanPage(s.db.QueryRowContext(ctx, query, args...))
}

func (s *PagesStore) List(ctx context.Context, in *pages.ListIn) ([]*pages.Page, int, error) {
	var conditions []string
	var args []any
	argNum := 1

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
	if in.Visibility != "" {
		conditions = append(conditions, fmt.Sprintf("visibility = $%d", argNum))
		args = append(args, in.Visibility)
		argNum++
	}
	if in.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR content ILIKE $%d)", argNum, argNum))
		args = append(args, "%"+in.Search+"%")
		argNum++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM pages %s", where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get items
	args = append(args, in.Limit, in.Offset)
	query := fmt.Sprintf(`
		SELECT id, author_id, parent_id, title, slug, content, content_format, featured_image_id, template, status, visibility, meta, sort_order, created_at, updated_at
		FROM pages %s
		ORDER BY sort_order, title
		LIMIT $%d OFFSET $%d
	`, where, argNum, argNum+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*pages.Page
	for rows.Next() {
		p, err := s.scanPageRow(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, p)
	}
	return list, total, rows.Err()
}

func (s *PagesStore) GetTree(ctx context.Context) ([]*pages.Page, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, author_id, parent_id, title, slug, content, content_format, featured_image_id, template, status, visibility, meta, sort_order, created_at, updated_at
		FROM pages
		ORDER BY sort_order, title
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*pages.Page
	for rows.Next() {
		p, err := s.scanPageRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (s *PagesStore) Update(ctx context.Context, id string, in *pages.UpdateIn) error {
	var sets []string
	var args []any
	argNum := 1

	if in.ParentID != nil {
		sets = append(sets, fmt.Sprintf("parent_id = $%d", argNum))
		args = append(args, nullString(*in.ParentID))
		argNum++
	}
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
	if in.Template != nil {
		sets = append(sets, fmt.Sprintf("template = $%d", argNum))
		args = append(args, nullString(*in.Template))
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
	if in.Meta != nil {
		sets = append(sets, fmt.Sprintf("meta = $%d", argNum))
		args = append(args, nullString(*in.Meta))
		argNum++
	}
	if in.SortOrder != nil {
		sets = append(sets, fmt.Sprintf("sort_order = $%d", argNum))
		args = append(args, *in.SortOrder)
		argNum++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++

	args = append(args, id)
	query := fmt.Sprintf("UPDATE pages SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *PagesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pages WHERE id = $1`, id)
	return err
}

func (s *PagesStore) scanPage(row *sql.Row) (*pages.Page, error) {
	p := &pages.Page{}
	var parentID, content, featuredImageID, template, meta sql.NullString
	err := row.Scan(&p.ID, &p.AuthorID, &parentID, &p.Title, &p.Slug, &content, &p.ContentFormat, &featuredImageID, &template, &p.Status, &p.Visibility, &meta, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.ParentID = parentID.String
	p.Content = content.String
	p.FeaturedImageID = featuredImageID.String
	p.Template = template.String
	p.Meta = meta.String
	return p, nil
}

func (s *PagesStore) scanPageRow(rows *sql.Rows) (*pages.Page, error) {
	p := &pages.Page{}
	var parentID, content, featuredImageID, template, meta sql.NullString
	err := rows.Scan(&p.ID, &p.AuthorID, &parentID, &p.Title, &p.Slug, &content, &p.ContentFormat, &featuredImageID, &template, &p.Status, &p.Visibility, &meta, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.ParentID = parentID.String
	p.Content = content.String
	p.FeaturedImageID = featuredImageID.String
	p.Template = template.String
	p.Meta = meta.String
	return p, nil
}
