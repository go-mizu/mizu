package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/qa/feature/tags"
)

// TagsStore implements tags.Store.
type TagsStore struct {
	db *sql.DB
}

// NewTagsStore creates a new tags store.
func NewTagsStore(db *sql.DB) *TagsStore {
	return &TagsStore{db: db}
}

// Create creates a tag.
func (s *TagsStore) Create(ctx context.Context, tag *tags.Tag) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tags (id, name, excerpt, wiki, question_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tag.ID, tag.Name, tag.Excerpt, tag.Wiki, tag.QuestionCount, tag.CreatedAt)
	return err
}

// GetByName retrieves a tag by name.
func (s *TagsStore) GetByName(ctx context.Context, name string) (*tags.Tag, error) {
	name = strings.ToLower(name)
	tag := &tags.Tag{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, excerpt, wiki, question_count, created_at
		FROM tags WHERE name = $1
	`, name).Scan(&tag.ID, &tag.Name, &tag.Excerpt, &tag.Wiki, &tag.QuestionCount, &tag.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, tags.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return tag, nil
}

// List lists tags.
func (s *TagsStore) List(ctx context.Context, opts tags.ListOpts) ([]*tags.Tag, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, name, excerpt, wiki, question_count, created_at
		FROM tags
	`
	args := []any{}
	if opts.Query != "" {
		query += " WHERE name LIKE $1"
		args = append(args, "%"+opts.Query+"%")
	}
	if len(args) == 0 {
		query += " ORDER BY question_count DESC LIMIT $1"
		args = append(args, limit)
	} else {
		query += " ORDER BY question_count DESC LIMIT $2"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*tags.Tag
	for rows.Next() {
		tag := &tags.Tag{}
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Excerpt, &tag.Wiki, &tag.QuestionCount, &tag.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

// IncrementQuestionCount updates question count.
func (s *TagsStore) IncrementQuestionCount(ctx context.Context, name string, delta int64) error {
	name = strings.ToLower(name)
	_, err := s.db.ExecContext(ctx, `
		UPDATE tags SET question_count = question_count + $2 WHERE name = $1
	`, name, delta)
	return err
}

// IncrementQuestionCountBatch updates question counts for multiple tags.
func (s *TagsStore) IncrementQuestionCountBatch(ctx context.Context, names []string, delta int64) error {
	if len(names) == 0 {
		return nil
	}

	placeholders := make([]string, len(names))
	args := make([]any, len(names)+1)
	args[0] = delta
	for i, name := range names {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = strings.ToLower(name)
	}

	query := `UPDATE tags SET question_count = question_count + $1 WHERE name IN (` + strings.Join(placeholders, ",") + `)`
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// GetByNames retrieves tags by names.
func (s *TagsStore) GetByNames(ctx context.Context, names []string) (map[string]*tags.Tag, error) {
	if len(names) == 0 {
		return make(map[string]*tags.Tag), nil
	}

	placeholders := make([]string, len(names))
	args := make([]any, len(names))
	for i, name := range names {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = strings.ToLower(name)
	}

	query := `
		SELECT id, name, excerpt, wiki, question_count, created_at
		FROM tags WHERE name IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*tags.Tag)
	for rows.Next() {
		tag := &tags.Tag{}
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Excerpt, &tag.Wiki, &tag.QuestionCount, &tag.CreatedAt); err != nil {
			return nil, err
		}
		result[tag.Name] = tag
	}
	return result, rows.Err()
}
