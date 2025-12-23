package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/news/feature/comments"
)

// CommentsStore implements comments.Store.
type CommentsStore struct {
	db *sql.DB
}

// NewCommentsStore creates a new comments store.
func NewCommentsStore(db *sql.DB) *CommentsStore {
	return &CommentsStore{db: db}
}

// Create creates a comment.
func (s *CommentsStore) Create(ctx context.Context, comment *comments.Comment) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO comments (id, story_id, parent_id, author_id, text, text_html, score, depth, path, child_count, is_removed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, comment.ID, comment.StoryID, sql.NullString{String: comment.ParentID, Valid: comment.ParentID != ""},
		comment.AuthorID, comment.Text, comment.TextHTML, comment.Score,
		comment.Depth, comment.Path, comment.ChildCount, comment.IsRemoved, comment.CreatedAt)
	return err
}

// GetByID retrieves a comment by ID.
func (s *CommentsStore) GetByID(ctx context.Context, id string) (*comments.Comment, error) {
	return s.scanComment(s.db.QueryRowContext(ctx, `
		SELECT id, story_id, parent_id, author_id, text, text_html, score, depth, path, child_count, is_removed, created_at
		FROM comments WHERE id = $1 AND is_removed = FALSE
	`, id))
}

// Update updates a comment.
func (s *CommentsStore) Update(ctx context.Context, comment *comments.Comment) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE comments SET
			text = $2, text_html = $3, score = $4, child_count = $5, is_removed = $6
		WHERE id = $1
	`, comment.ID, comment.Text, comment.TextHTML, comment.Score, comment.ChildCount, comment.IsRemoved)
	return err
}

// Delete marks a comment as removed.
func (s *CommentsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE comments SET is_removed = TRUE WHERE id = $1`, id)
	return err
}

// ListByStory lists all comments for a story.
func (s *CommentsStore) ListByStory(ctx context.Context, storyID string) ([]*comments.Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, story_id, parent_id, author_id, text, text_html, score, depth, path, child_count, is_removed, created_at
		FROM comments
		WHERE story_id = $1 AND is_removed = FALSE
		ORDER BY path ASC
	`, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*comments.Comment
	for rows.Next() {
		comment, err := s.scanCommentFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, comment)
	}
	return result, rows.Err()
}

// ListByParent lists comments by parent.
func (s *CommentsStore) ListByParent(ctx context.Context, parentID string) ([]*comments.Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, story_id, parent_id, author_id, text, text_html, score, depth, path, child_count, is_removed, created_at
		FROM comments
		WHERE parent_id = $1 AND is_removed = FALSE
		ORDER BY score DESC, created_at ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*comments.Comment
	for rows.Next() {
		comment, err := s.scanCommentFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, comment)
	}
	return result, rows.Err()
}

// ListByAuthor lists comments by author.
func (s *CommentsStore) ListByAuthor(ctx context.Context, authorID string, limit, offset int) ([]*comments.Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, story_id, parent_id, author_id, text, text_html, score, depth, path, child_count, is_removed, created_at
		FROM comments
		WHERE author_id = $1 AND is_removed = FALSE
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, authorID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*comments.Comment
	for rows.Next() {
		comment, err := s.scanCommentFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, comment)
	}
	return result, rows.Err()
}

// UpdateScore updates a comment's score.
func (s *CommentsStore) UpdateScore(ctx context.Context, id string, delta int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE comments SET score = score + $2 WHERE id = $1
	`, id, delta)
	return err
}

// IncrementChildCount increments a comment's child count.
func (s *CommentsStore) IncrementChildCount(ctx context.Context, id string, delta int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE comments SET child_count = child_count + $2 WHERE id = $1
	`, id, delta)
	return err
}

func (s *CommentsStore) scanComment(row *sql.Row) (*comments.Comment, error) {
	comment := &comments.Comment{}
	var parentID sql.NullString
	var textHTML sql.NullString

	err := row.Scan(
		&comment.ID, &comment.StoryID, &parentID, &comment.AuthorID,
		&comment.Text, &textHTML, &comment.Score, &comment.Depth,
		&comment.Path, &comment.ChildCount, &comment.IsRemoved, &comment.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, comments.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		comment.ParentID = parentID.String
	}
	if textHTML.Valid {
		comment.TextHTML = textHTML.String
	}

	return comment, nil
}

func (s *CommentsStore) scanCommentFromRows(rows *sql.Rows) (*comments.Comment, error) {
	comment := &comments.Comment{}
	var parentID sql.NullString
	var textHTML sql.NullString

	err := rows.Scan(
		&comment.ID, &comment.StoryID, &parentID, &comment.AuthorID,
		&comment.Text, &textHTML, &comment.Score, &comment.Depth,
		&comment.Path, &comment.ChildCount, &comment.IsRemoved, &comment.CreatedAt)

	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		comment.ParentID = parentID.String
	}
	if textHTML.Valid {
		comment.TextHTML = textHTML.String
	}

	return comment, nil
}
