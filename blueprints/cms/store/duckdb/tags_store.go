package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/feature/tags"
)

// TagsStore handles tag data access.
type TagsStore struct {
	db *sql.DB
}

// NewTagsStore creates a new tags store.
func NewTagsStore(db *sql.DB) *TagsStore {
	return &TagsStore{db: db}
}

func (s *TagsStore) Create(ctx context.Context, t *tags.Tag) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tags (id, name, slug, description, featured_image_id, meta, post_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, t.ID, t.Name, t.Slug, t.Description, nullString(t.FeaturedImageID), nullString(t.Meta), t.PostCount, t.CreatedAt, t.UpdatedAt)
	return err
}

func (s *TagsStore) GetByID(ctx context.Context, id string) (*tags.Tag, error) {
	return s.scanTag(s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, description, featured_image_id, meta, post_count, created_at, updated_at
		FROM tags WHERE id = $1
	`, id))
}

func (s *TagsStore) GetBySlug(ctx context.Context, slug string) (*tags.Tag, error) {
	return s.scanTag(s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, description, featured_image_id, meta, post_count, created_at, updated_at
		FROM tags WHERE slug = $1
	`, slug))
}

func (s *TagsStore) GetByIDs(ctx context.Context, ids []string) ([]*tags.Tag, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, name, slug, description, featured_image_id, meta, post_count, created_at, updated_at
		FROM tags WHERE id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*tags.Tag
	for rows.Next() {
		t, err := s.scanTagRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (s *TagsStore) List(ctx context.Context, in *tags.ListIn) ([]*tags.Tag, int, error) {
	var conditions []string
	var args []any
	argNum := 1

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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tags %s", where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Determine order
	orderBy := in.OrderBy
	if orderBy == "" {
		orderBy = "name"
	}
	order := in.Order
	if order == "" {
		order = "ASC"
	}

	// Get items
	args = append(args, in.Limit, in.Offset)
	query := fmt.Sprintf(`
		SELECT id, name, slug, description, featured_image_id, meta, post_count, created_at, updated_at
		FROM tags %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, orderBy, order, argNum, argNum+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*tags.Tag
	for rows.Next() {
		t, err := s.scanTagRow(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, t)
	}
	return list, total, rows.Err()
}

func (s *TagsStore) Update(ctx context.Context, id string, in *tags.UpdateIn) error {
	var sets []string
	var args []any
	argNum := 1

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

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++

	args = append(args, id)
	query := fmt.Sprintf("UPDATE tags SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *TagsStore) Delete(ctx context.Context, id string) error {
	// Delete post_tags relationships
	s.db.ExecContext(ctx, `DELETE FROM post_tags WHERE tag_id = $1`, id)
	_, err := s.db.ExecContext(ctx, `DELETE FROM tags WHERE id = $1`, id)
	return err
}

func (s *TagsStore) IncrementPostCount(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tags SET post_count = post_count + 1 WHERE id = $1`, id)
	return err
}

func (s *TagsStore) DecrementPostCount(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tags SET post_count = GREATEST(post_count - 1, 0) WHERE id = $1`, id)
	return err
}

func (s *TagsStore) scanTag(row *sql.Row) (*tags.Tag, error) {
	t := &tags.Tag{}
	var description, featuredImageID, meta sql.NullString
	err := row.Scan(&t.ID, &t.Name, &t.Slug, &description, &featuredImageID, &meta, &t.PostCount, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Description = description.String
	t.FeaturedImageID = featuredImageID.String
	t.Meta = meta.String
	return t, nil
}

func (s *TagsStore) scanTagRow(rows *sql.Rows) (*tags.Tag, error) {
	t := &tags.Tag{}
	var description, featuredImageID, meta sql.NullString
	err := rows.Scan(&t.ID, &t.Name, &t.Slug, &description, &featuredImageID, &meta, &t.PostCount, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.Description = description.String
	t.FeaturedImageID = featuredImageID.String
	t.Meta = meta.String
	return t, nil
}
