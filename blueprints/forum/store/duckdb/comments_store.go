package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
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
		INSERT INTO comments (
			id, thread_id, parent_id, author_id, content, content_html,
			score, upvote_count, downvote_count, depth, path, child_count,
			is_removed, is_deleted, remove_reason, created_at, updated_at, edited_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, comment.ID, comment.ThreadID, comment.ParentID, comment.AuthorID,
		comment.Content, comment.ContentHTML, comment.Score, comment.UpvoteCount,
		comment.DownvoteCount, comment.Depth, comment.Path, comment.ChildCount,
		comment.IsRemoved, comment.IsDeleted, comment.RemoveReason,
		comment.CreatedAt, comment.UpdatedAt, comment.EditedAt)
	return err
}

// GetByID retrieves a comment by ID.
func (s *CommentsStore) GetByID(ctx context.Context, id string) (*comments.Comment, error) {
	return s.scanComment(s.db.QueryRowContext(ctx, `
		SELECT id, thread_id, parent_id, author_id, content, content_html,
			score, upvote_count, downvote_count, depth, path, child_count,
			is_removed, is_deleted, remove_reason, created_at, updated_at, edited_at
		FROM comments WHERE id = $1
	`, id))
}

// GetByIDs retrieves multiple comments by their IDs.
func (s *CommentsStore) GetByIDs(ctx context.Context, ids []string) (map[string]*comments.Comment, error) {
	if len(ids) == 0 {
		return make(map[string]*comments.Comment), nil
	}

	// Build placeholders
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := `
		SELECT id, thread_id, parent_id, author_id, content, content_html,
			score, upvote_count, downvote_count, depth, path, child_count,
			is_removed, is_deleted, remove_reason, created_at, updated_at, edited_at
		FROM comments WHERE id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	commentList, err := s.scanComments(rows)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*comments.Comment)
	for _, c := range commentList {
		result[c.ID] = c
	}
	return result, nil
}

// Update updates a comment.
func (s *CommentsStore) Update(ctx context.Context, comment *comments.Comment) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE comments SET
			content = $2, content_html = $3, score = $4, upvote_count = $5,
			downvote_count = $6, child_count = $7, is_removed = $8, is_deleted = $9,
			remove_reason = $10, updated_at = $11, edited_at = $12
		WHERE id = $1
	`, comment.ID, comment.Content, comment.ContentHTML, comment.Score,
		comment.UpvoteCount, comment.DownvoteCount, comment.ChildCount,
		comment.IsRemoved, comment.IsDeleted, comment.RemoveReason,
		comment.UpdatedAt, comment.EditedAt)
	return err
}

// Delete deletes a comment.
func (s *CommentsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE id = $1`, id)
	return err
}

// ListByThread lists comments for a thread.
func (s *CommentsStore) ListByThread(ctx context.Context, threadID string, opts comments.ListOpts) ([]*comments.Comment, error) {
	orderBy := s.getOrderBy(opts.SortBy)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, thread_id, parent_id, author_id, content, content_html,
			score, upvote_count, downvote_count, depth, path, child_count,
			is_removed, is_deleted, remove_reason, created_at, updated_at, edited_at
		FROM comments
		WHERE thread_id = $1 AND NOT is_removed
		ORDER BY `+orderBy+`
		LIMIT $2
	`, threadID, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanComments(rows)
}

// ListByParent lists children of a comment.
func (s *CommentsStore) ListByParent(ctx context.Context, parentID string, opts comments.ListOpts) ([]*comments.Comment, error) {
	orderBy := s.getOrderBy(opts.SortBy)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, thread_id, parent_id, author_id, content, content_html,
			score, upvote_count, downvote_count, depth, path, child_count,
			is_removed, is_deleted, remove_reason, created_at, updated_at, edited_at
		FROM comments
		WHERE parent_id = $1 AND NOT is_removed
		ORDER BY `+orderBy+`
		LIMIT $2
	`, parentID, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanComments(rows)
}

// ListByAuthor lists comments by an author.
func (s *CommentsStore) ListByAuthor(ctx context.Context, authorID string, opts comments.ListOpts) ([]*comments.Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, thread_id, parent_id, author_id, content, content_html,
			score, upvote_count, downvote_count, depth, path, child_count,
			is_removed, is_deleted, remove_reason, created_at, updated_at, edited_at
		FROM comments
		WHERE author_id = $1 AND NOT is_removed AND NOT is_deleted
		ORDER BY created_at DESC
		LIMIT $2
	`, authorID, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanComments(rows)
}

// ListByPath lists comments by path prefix.
func (s *CommentsStore) ListByPath(ctx context.Context, pathPrefix string, opts comments.ListOpts) ([]*comments.Comment, error) {
	orderBy := s.getOrderBy(opts.SortBy)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, thread_id, parent_id, author_id, content, content_html,
			score, upvote_count, downvote_count, depth, path, child_count,
			is_removed, is_deleted, remove_reason, created_at, updated_at, edited_at
		FROM comments
		WHERE path LIKE $1 || '%' AND NOT is_removed
		ORDER BY `+orderBy+`
		LIMIT $2
	`, pathPrefix, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanComments(rows)
}

// IncrementChildCount updates the child count.
func (s *CommentsStore) IncrementChildCount(ctx context.Context, id string, delta int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE comments SET child_count = child_count + $2 WHERE id = $1
	`, id, delta)
	return err
}

func (s *CommentsStore) getOrderBy(sortBy comments.CommentSort) string {
	switch sortBy {
	case comments.CommentSortBest:
		return "((upvote_count + 1.9208) / (upvote_count + downvote_count + 1) - 1.96 * SQRT((upvote_count * downvote_count) / ((upvote_count + downvote_count + 1) * (upvote_count + downvote_count + 1)))) DESC"
	case comments.CommentSortTop:
		return "score DESC"
	case comments.CommentSortNew:
		return "created_at DESC"
	case comments.CommentSortOld:
		return "created_at ASC"
	case comments.CommentSortControversial:
		return "(upvote_count + downvote_count) * LEAST(upvote_count, downvote_count)::float / GREATEST(upvote_count, downvote_count, 1) DESC"
	default:
		return "path, score DESC"
	}
}

func (s *CommentsStore) scanComment(row *sql.Row) (*comments.Comment, error) {
	comment := &comments.Comment{}
	var parentID sql.NullString
	var editedAt sql.NullTime
	var removeReason sql.NullString

	err := row.Scan(
		&comment.ID, &comment.ThreadID, &parentID, &comment.AuthorID,
		&comment.Content, &comment.ContentHTML, &comment.Score, &comment.UpvoteCount,
		&comment.DownvoteCount, &comment.Depth, &comment.Path, &comment.ChildCount,
		&comment.IsRemoved, &comment.IsDeleted, &removeReason,
		&comment.CreatedAt, &comment.UpdatedAt, &editedAt)

	if err == sql.ErrNoRows {
		return nil, comments.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		comment.ParentID = parentID.String
	}
	if editedAt.Valid {
		comment.EditedAt = &editedAt.Time
	}
	if removeReason.Valid {
		comment.RemoveReason = removeReason.String
	}

	return comment, nil
}

func (s *CommentsStore) scanComments(rows *sql.Rows) ([]*comments.Comment, error) {
	var result []*comments.Comment
	for rows.Next() {
		comment := &comments.Comment{}
		var parentID sql.NullString
		var editedAt sql.NullTime
		var removeReason sql.NullString

		err := rows.Scan(
			&comment.ID, &comment.ThreadID, &parentID, &comment.AuthorID,
			&comment.Content, &comment.ContentHTML, &comment.Score, &comment.UpvoteCount,
			&comment.DownvoteCount, &comment.Depth, &comment.Path, &comment.ChildCount,
			&comment.IsRemoved, &comment.IsDeleted, &removeReason,
			&comment.CreatedAt, &comment.UpdatedAt, &editedAt)

		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			comment.ParentID = parentID.String
		}
		if editedAt.Valid {
			comment.EditedAt = &editedAt.Time
		}
		if removeReason.Valid {
			comment.RemoveReason = removeReason.String
		}

		result = append(result, comment)
	}
	return result, rows.Err()
}
