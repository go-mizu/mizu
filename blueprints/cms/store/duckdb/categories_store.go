package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/feature/categories"
)

// CategoriesStore handles category data access.
type CategoriesStore struct {
	db *sql.DB
}

// NewCategoriesStore creates a new categories store.
func NewCategoriesStore(db *sql.DB) *CategoriesStore {
	return &CategoriesStore{db: db}
}

func (s *CategoriesStore) Create(ctx context.Context, c *categories.Category) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO categories (id, parent_id, name, slug, description, featured_image_id, meta, sort_order, post_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, c.ID, nullString(c.ParentID), c.Name, c.Slug, c.Description, nullString(c.FeaturedImageID), nullString(c.Meta), c.SortOrder, c.PostCount, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *CategoriesStore) GetByID(ctx context.Context, id string) (*categories.Category, error) {
	return s.scanCategory(s.db.QueryRowContext(ctx, `
		SELECT id, parent_id, name, slug, description, featured_image_id, meta, sort_order, post_count, created_at, updated_at
		FROM categories WHERE id = $1
	`, id))
}

func (s *CategoriesStore) GetBySlug(ctx context.Context, slug string) (*categories.Category, error) {
	return s.scanCategory(s.db.QueryRowContext(ctx, `
		SELECT id, parent_id, name, slug, description, featured_image_id, meta, sort_order, post_count, created_at, updated_at
		FROM categories WHERE slug = $1
	`, slug))
}

func (s *CategoriesStore) List(ctx context.Context, in *categories.ListIn) ([]*categories.Category, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	if in.ParentID != "" {
		conditions = append(conditions, fmt.Sprintf("parent_id = $%d", argNum))
		args = append(args, in.ParentID)
		argNum++
	}
	if in.Search != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argNum))
		args = append(args, "%"+in.Search+"%")
		argNum++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM categories %s", where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get items
	args = append(args, in.Limit, in.Offset)
	query := fmt.Sprintf(`
		SELECT id, parent_id, name, slug, description, featured_image_id, meta, sort_order, post_count, created_at, updated_at
		FROM categories %s
		ORDER BY sort_order, name
		LIMIT $%d OFFSET $%d
	`, where, argNum, argNum+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*categories.Category
	for rows.Next() {
		c, err := s.scanCategoryRow(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, c)
	}
	return list, total, rows.Err()
}

func (s *CategoriesStore) GetTree(ctx context.Context) ([]*categories.Category, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, parent_id, name, slug, description, featured_image_id, meta, sort_order, post_count, created_at, updated_at
		FROM categories
		ORDER BY sort_order, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*categories.Category
	for rows.Next() {
		c, err := s.scanCategoryRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (s *CategoriesStore) Update(ctx context.Context, id string, in *categories.UpdateIn) error {
	var sets []string
	var args []any
	argNum := 1

	if in.ParentID != nil {
		sets = append(sets, fmt.Sprintf("parent_id = $%d", argNum))
		args = append(args, nullString(*in.ParentID))
		argNum++
	}
	if in.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *in.Name)
		argNum++
	}
	if in.Slug != nil {
		sets = append(sets, fmt.Sprintf("slug = $%d", argNum))
		args = append(args, *in.Slug)
		argNum++
	}
	if in.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argNum))
		args = append(args, *in.Description)
		argNum++
	}
	if in.FeaturedImageID != nil {
		sets = append(sets, fmt.Sprintf("featured_image_id = $%d", argNum))
		args = append(args, nullString(*in.FeaturedImageID))
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
	query := fmt.Sprintf("UPDATE categories SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *CategoriesStore) Delete(ctx context.Context, id string) error {
	// Delete post_categories relationships
	s.db.ExecContext(ctx, `DELETE FROM post_categories WHERE category_id = $1`, id)
	_, err := s.db.ExecContext(ctx, `DELETE FROM categories WHERE id = $1`, id)
	return err
}

func (s *CategoriesStore) IncrementPostCount(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE categories SET post_count = post_count + 1 WHERE id = $1`, id)
	return err
}

func (s *CategoriesStore) DecrementPostCount(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE categories SET post_count = GREATEST(post_count - 1, 0) WHERE id = $1`, id)
	return err
}

func (s *CategoriesStore) scanCategory(row *sql.Row) (*categories.Category, error) {
	c := &categories.Category{}
	var parentID, description, featuredImageID, meta sql.NullString
	err := row.Scan(&c.ID, &parentID, &c.Name, &c.Slug, &description, &featuredImageID, &meta, &c.SortOrder, &c.PostCount, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.ParentID = parentID.String
	c.Description = description.String
	c.FeaturedImageID = featuredImageID.String
	c.Meta = meta.String
	return c, nil
}

func (s *CategoriesStore) scanCategoryRow(rows *sql.Rows) (*categories.Category, error) {
	c := &categories.Category{}
	var parentID, description, featuredImageID, meta sql.NullString
	err := rows.Scan(&c.ID, &parentID, &c.Name, &c.Slug, &description, &featuredImageID, &meta, &c.SortOrder, &c.PostCount, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	c.ParentID = parentID.String
	c.Description = description.String
	c.FeaturedImageID = featuredImageID.String
	c.Meta = meta.String
	return c, nil
}
