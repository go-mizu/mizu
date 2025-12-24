package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
)

// StoriesStore implements stories.Store.
type StoriesStore struct {
	db *sql.DB
}

// NewStoriesStore creates a new stories store.
func NewStoriesStore(db *sql.DB) *StoriesStore {
	return &StoriesStore{db: db}
}

// GetByID retrieves a story by ID.
func (s *StoriesStore) GetByID(ctx context.Context, id string) (*stories.Story, error) {
	return s.scanStory(s.db.QueryRowContext(ctx, `
		SELECT id, author_id, title, url, domain, text, text_html, score, comment_count, hot_score, is_removed, created_at
		FROM stories WHERE id = $1 AND is_removed = FALSE
	`, id))
}

// GetByURL retrieves a story by URL.
func (s *StoriesStore) GetByURL(ctx context.Context, url string) (*stories.Story, error) {
	return s.scanStory(s.db.QueryRowContext(ctx, `
		SELECT id, author_id, title, url, domain, text, text_html, score, comment_count, hot_score, is_removed, created_at
		FROM stories WHERE url = $1 AND is_removed = FALSE
	`, url))
}

// List lists stories.
func (s *StoriesStore) List(ctx context.Context, in stories.ListIn) ([]*stories.Story, error) {
	var query strings.Builder
	var args []any
	argIndex := 1

	query.WriteString(`
		SELECT s.id, s.author_id, s.title, s.url, s.domain, s.text, s.text_html, s.score, s.comment_count, s.hot_score, s.is_removed, s.created_at
		FROM stories s
	`)

	// Join with story_tags if filtering by tag
	if in.Tag != "" {
		query.WriteString(`
			JOIN story_tags st ON s.id = st.story_id
			JOIN tags t ON st.tag_id = t.id
		`)
	}

	query.WriteString(" WHERE s.is_removed = FALSE")

	if in.Tag != "" {
		query.WriteString(fmt.Sprintf(" AND LOWER(t.name) = LOWER($%d)", argIndex))
		args = append(args, in.Tag)
		argIndex++
	}

	if in.Domain != "" {
		query.WriteString(fmt.Sprintf(" AND s.domain = $%d", argIndex))
		args = append(args, in.Domain)
		argIndex++
	}

	// Order by
	switch in.Sort {
	case "new":
		query.WriteString(" ORDER BY s.created_at DESC")
	case "top":
		query.WriteString(" ORDER BY s.score DESC")
	default: // "hot"
		query.WriteString(" ORDER BY s.hot_score DESC")
	}

	query.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1))
	args = append(args, in.Limit, in.Offset)

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*stories.Story
	for rows.Next() {
		story, err := s.scanStoryFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, story)
	}
	return result, rows.Err()
}

// ListByAuthor lists stories by author.
func (s *StoriesStore) ListByAuthor(ctx context.Context, authorID string, limit, offset int) ([]*stories.Story, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, author_id, title, url, domain, text, text_html, score, comment_count, hot_score, is_removed, created_at
		FROM stories
		WHERE author_id = $1 AND is_removed = FALSE
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, authorID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*stories.Story
	for rows.Next() {
		story, err := s.scanStoryFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, story)
	}
	return result, rows.Err()
}

// ListByTag lists stories by tag.
func (s *StoriesStore) ListByTag(ctx context.Context, tagID string, limit, offset int) ([]*stories.Story, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.author_id, s.title, s.url, s.domain, s.text, s.text_html, s.score, s.comment_count, s.hot_score, s.is_removed, s.created_at
		FROM stories s
		JOIN story_tags st ON s.id = st.story_id
		WHERE st.tag_id = $1 AND s.is_removed = FALSE
		ORDER BY s.hot_score DESC
		LIMIT $2 OFFSET $3
	`, tagID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*stories.Story
	for rows.Next() {
		story, err := s.scanStoryFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, story)
	}
	return result, rows.Err()
}

// GetTagsForStory gets tags for a story.
func (s *StoriesStore) GetTagsForStory(ctx context.Context, storyID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.name
		FROM tags t
		JOIN story_tags st ON t.id = st.tag_id
		WHERE st.story_id = $1
	`, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	return tags, rows.Err()
}

// GetTagsForStories gets tags for multiple stories.
func (s *StoriesStore) GetTagsForStories(ctx context.Context, storyIDs []string) (map[string][]string, error) {
	if len(storyIDs) == 0 {
		return make(map[string][]string), nil
	}

	placeholders := make([]string, len(storyIDs))
	args := make([]any, len(storyIDs))
	for i, id := range storyIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := `
		SELECT st.story_id, t.name
		FROM tags t
		JOIN story_tags st ON t.id = st.tag_id
		WHERE st.story_id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var storyID, tagName string
		if err := rows.Scan(&storyID, &tagName); err != nil {
			return nil, err
		}
		result[storyID] = append(result[storyID], tagName)
	}
	return result, rows.Err()
}

func (s *StoriesStore) scanStory(row *sql.Row) (*stories.Story, error) {
	story := &stories.Story{}
	var url, domain, text, textHTML sql.NullString

	err := row.Scan(
		&story.ID, &story.AuthorID, &story.Title, &url, &domain,
		&text, &textHTML, &story.Score, &story.CommentCount,
		&story.HotScore, &story.IsRemoved, &story.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, stories.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if url.Valid {
		story.URL = url.String
	}
	if domain.Valid {
		story.Domain = domain.String
	}
	if text.Valid {
		story.Text = text.String
	}
	if textHTML.Valid {
		story.TextHTML = textHTML.String
	}

	return story, nil
}

func (s *StoriesStore) scanStoryFromRows(rows *sql.Rows) (*stories.Story, error) {
	story := &stories.Story{}
	var url, domain, text, textHTML sql.NullString

	err := rows.Scan(
		&story.ID, &story.AuthorID, &story.Title, &url, &domain,
		&text, &textHTML, &story.Score, &story.CommentCount,
		&story.HotScore, &story.IsRemoved, &story.CreatedAt)

	if err != nil {
		return nil, err
	}

	if url.Valid {
		story.URL = url.String
	}
	if domain.Valid {
		story.Domain = domain.String
	}
	if text.Valid {
		story.Text = text.String
	}
	if textHTML.Valid {
		story.TextHTML = textHTML.String
	}

	return story, nil
}

// Create creates a new story with optional tags.
func (s *StoriesStore) Create(ctx context.Context, story *stories.Story, tagIDs []string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO stories (id, author_id, title, url, domain, text, text_html, score, comment_count, hot_score, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, story.ID, story.AuthorID, story.Title, story.URL, story.Domain,
		story.Text, story.TextHTML, story.Score, story.CommentCount, story.HotScore, story.CreatedAt)
	if err != nil {
		return err
	}

	// Add tags
	for _, tagID := range tagIDs {
		_, _ = s.db.ExecContext(ctx, `
			INSERT INTO story_tags (story_id, tag_id) VALUES ($1, $2)
		`, story.ID, tagID)
	}

	return nil
}

// IncrementCommentCount increments the comment count for a story.
func (s *StoriesStore) IncrementCommentCount(ctx context.Context, storyID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE stories SET comment_count = comment_count + 1 WHERE id = $1
	`, storyID)
	return err
}

// UpdateScore updates the score for a story.
func (s *StoriesStore) UpdateScore(ctx context.Context, storyID string, delta int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE stories SET score = score + $1 WHERE id = $2
	`, delta, storyID)
	return err
}
