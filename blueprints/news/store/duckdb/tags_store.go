package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/news/feature/tags"
)

// TagsStore implements tags.Store.
type TagsStore struct {
	db *sql.DB
}

// NewTagsStore creates a new tags store.
func NewTagsStore(db *sql.DB) *TagsStore {
	return &TagsStore{db: db}
}

// GetByID retrieves a tag by ID.
func (s *TagsStore) GetByID(ctx context.Context, id string) (*tags.Tag, error) {
	return s.scanTag(s.db.QueryRowContext(ctx, `
		SELECT id, name, description, color, story_count
		FROM tags WHERE id = $1
	`, id))
}

// GetByName retrieves a tag by name.
func (s *TagsStore) GetByName(ctx context.Context, name string) (*tags.Tag, error) {
	return s.scanTag(s.db.QueryRowContext(ctx, `
		SELECT id, name, description, color, story_count
		FROM tags WHERE LOWER(name) = LOWER($1)
	`, name))
}

// GetByNames retrieves tags by names.
func (s *TagsStore) GetByNames(ctx context.Context, names []string) ([]*tags.Tag, error) {
	if len(names) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(names))
	args := make([]any, len(names))
	for i, name := range names {
		placeholders[i] = fmt.Sprintf("LOWER($%d)", i+1)
		args[i] = strings.ToLower(name)
	}

	query := `
		SELECT id, name, description, color, story_count
		FROM tags WHERE LOWER(name) IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*tags.Tag
	for rows.Next() {
		tag, err := s.scanTagFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

// List lists all tags.
func (s *TagsStore) List(ctx context.Context, limit int) ([]*tags.Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, color, story_count
		FROM tags
		ORDER BY name ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*tags.Tag
	for rows.Next() {
		tag, err := s.scanTagFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

// ListPopular lists tags by story count.
func (s *TagsStore) ListPopular(ctx context.Context, limit int) ([]*tags.Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, color, story_count
		FROM tags
		ORDER BY story_count DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*tags.Tag
	for rows.Next() {
		tag, err := s.scanTagFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

func (s *TagsStore) scanTag(row *sql.Row) (*tags.Tag, error) {
	tag := &tags.Tag{}
	var description, color sql.NullString

	err := row.Scan(&tag.ID, &tag.Name, &description, &color, &tag.StoryCount)
	if err == sql.ErrNoRows {
		return nil, tags.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if description.Valid {
		tag.Description = description.String
	}
	if color.Valid {
		tag.Color = color.String
	}

	return tag, nil
}

func (s *TagsStore) scanTagFromRows(rows *sql.Rows) (*tags.Tag, error) {
	tag := &tags.Tag{}
	var description, color sql.NullString

	err := rows.Scan(&tag.ID, &tag.Name, &description, &color, &tag.StoryCount)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		tag.Description = description.String
	}
	if color.Valid {
		tag.Color = color.String
	}

	return tag, nil
}
