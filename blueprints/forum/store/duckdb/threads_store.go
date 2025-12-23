package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
)

// ThreadsStore implements threads.Store.
type ThreadsStore struct {
	db *sql.DB
}

// NewThreadsStore creates a new threads store.
func NewThreadsStore(db *sql.DB) *ThreadsStore {
	return &ThreadsStore{db: db}
}

// Create creates a thread.
func (s *ThreadsStore) Create(ctx context.Context, thread *threads.Thread) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO threads (
			id, board_id, author_id, title, content, content_html,
			url, domain, thumbnail_url, type, score, upvote_count, downvote_count,
			comment_count, view_count, hot_score, is_pinned, is_locked, is_removed,
			is_nsfw, is_spoiler, is_oc, remove_reason, created_at, updated_at, edited_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
	`, thread.ID, thread.BoardID, thread.AuthorID, thread.Title, thread.Content, thread.ContentHTML,
		thread.URL, thread.Domain, thread.ThumbnailURL, thread.Type, thread.Score,
		thread.UpvoteCount, thread.DownvoteCount, thread.CommentCount, thread.ViewCount,
		thread.HotScore, thread.IsPinned, thread.IsLocked, thread.IsRemoved,
		thread.IsNSFW, thread.IsSpoiler, thread.IsOC, thread.RemoveReason,
		thread.CreatedAt, thread.UpdatedAt, thread.EditedAt)
	return err
}

// GetByID retrieves a thread by ID.
func (s *ThreadsStore) GetByID(ctx context.Context, id string) (*threads.Thread, error) {
	return s.scanThread(s.db.QueryRowContext(ctx, `
		SELECT id, board_id, author_id, title, content, content_html,
			url, domain, thumbnail_url, type, score, upvote_count, downvote_count,
			comment_count, view_count, hot_score, is_pinned, is_locked, is_removed,
			is_nsfw, is_spoiler, is_oc, remove_reason, created_at, updated_at, edited_at
		FROM threads WHERE id = $1
	`, id))
}

// GetByIDs retrieves multiple threads by their IDs.
func (s *ThreadsStore) GetByIDs(ctx context.Context, ids []string) (map[string]*threads.Thread, error) {
	if len(ids) == 0 {
		return make(map[string]*threads.Thread), nil
	}

	// Build placeholders
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := `
		SELECT id, board_id, author_id, title, content, content_html,
			url, domain, thumbnail_url, type, score, upvote_count, downvote_count,
			comment_count, view_count, hot_score, is_pinned, is_locked, is_removed,
			is_nsfw, is_spoiler, is_oc, remove_reason, created_at, updated_at, edited_at
		FROM threads WHERE id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	threadList, err := s.scanThreads(rows)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*threads.Thread)
	for _, t := range threadList {
		result[t.ID] = t
	}
	return result, nil
}

// Update updates a thread.
func (s *ThreadsStore) Update(ctx context.Context, thread *threads.Thread) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE threads SET
			title = $2, content = $3, content_html = $4, url = $5, domain = $6,
			thumbnail_url = $7, type = $8, score = $9, upvote_count = $10,
			downvote_count = $11, comment_count = $12, view_count = $13, hot_score = $14,
			is_pinned = $15, is_locked = $16, is_removed = $17, is_nsfw = $18,
			is_spoiler = $19, is_oc = $20, remove_reason = $21, updated_at = $22, edited_at = $23
		WHERE id = $1
	`, thread.ID, thread.Title, thread.Content, thread.ContentHTML, thread.URL, thread.Domain,
		thread.ThumbnailURL, thread.Type, thread.Score, thread.UpvoteCount,
		thread.DownvoteCount, thread.CommentCount, thread.ViewCount, thread.HotScore,
		thread.IsPinned, thread.IsLocked, thread.IsRemoved, thread.IsNSFW,
		thread.IsSpoiler, thread.IsOC, thread.RemoveReason, thread.UpdatedAt, thread.EditedAt)
	return err
}

// Delete deletes a thread.
func (s *ThreadsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM threads WHERE id = $1`, id)
	return err
}

// List lists all threads.
func (s *ThreadsStore) List(ctx context.Context, opts threads.ListOpts) ([]*threads.Thread, error) {
	query := s.buildListQuery("", opts)
	rows, err := s.db.QueryContext(ctx, query, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanThreads(rows)
}

// ListByBoard lists threads in a board.
func (s *ThreadsStore) ListByBoard(ctx context.Context, boardID string, opts threads.ListOpts) ([]*threads.Thread, error) {
	query := s.buildListQuery(fmt.Sprintf("board_id = '%s'", boardID), opts)
	rows, err := s.db.QueryContext(ctx, query, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanThreads(rows)
}

// ListByAuthor lists threads by an author.
func (s *ThreadsStore) ListByAuthor(ctx context.Context, authorID string, opts threads.ListOpts) ([]*threads.Thread, error) {
	query := s.buildListQuery(fmt.Sprintf("author_id = '%s'", authorID), opts)
	rows, err := s.db.QueryContext(ctx, query, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanThreads(rows)
}

// UpdateHotScores recalculates hot scores for all threads.
func (s *ThreadsStore) UpdateHotScores(ctx context.Context) error {
	// This is a simplified version - in production, you'd use a background job
	_, err := s.db.ExecContext(ctx, `
		UPDATE threads SET hot_score = (
			CASE WHEN score > 0 THEN 1
				 WHEN score < 0 THEN -1
				 ELSE 0
			END * LOG10(GREATEST(ABS(score), 1))
			+ (EXTRACT(EPOCH FROM created_at) - 1134028003) / 45000
		)
	`)
	return err
}

func (s *ThreadsStore) buildListQuery(where string, opts threads.ListOpts) string {
	query := `
		SELECT id, board_id, author_id, title, content, content_html,
			url, domain, thumbnail_url, type, score, upvote_count, downvote_count,
			comment_count, view_count, hot_score, is_pinned, is_locked, is_removed,
			is_nsfw, is_spoiler, is_oc, remove_reason, created_at, updated_at, edited_at
		FROM threads
	`

	conditions := []string{"NOT is_removed"}
	if where != "" {
		conditions = append(conditions, where)
	}

	// Time range filter for "top" sorting
	if opts.SortBy == threads.SortTop && opts.TimeRange != threads.TimeAll {
		var since time.Time
		now := time.Now()
		switch opts.TimeRange {
		case threads.TimeHour:
			since = now.Add(-time.Hour)
		case threads.TimeDay:
			since = now.Add(-24 * time.Hour)
		case threads.TimeWeek:
			since = now.Add(-7 * 24 * time.Hour)
		case threads.TimeMonth:
			since = now.Add(-30 * 24 * time.Hour)
		case threads.TimeYear:
			since = now.Add(-365 * 24 * time.Hour)
		}
		conditions = append(conditions, fmt.Sprintf("created_at >= '%s'", since.Format(time.RFC3339)))
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			query += " AND " + c
		}
	}

	// Sorting
	switch opts.SortBy {
	case threads.SortHot:
		query += " ORDER BY is_pinned DESC, hot_score DESC"
	case threads.SortNew:
		query += " ORDER BY is_pinned DESC, created_at DESC"
	case threads.SortTop:
		query += " ORDER BY is_pinned DESC, score DESC"
	case threads.SortRising:
		query += " ORDER BY is_pinned DESC, (score / GREATEST(EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - created_at)) / 3600, 1)) DESC"
	case threads.SortControversial:
		query += " ORDER BY is_pinned DESC, (upvote_count + downvote_count) * LEAST(upvote_count, downvote_count)::float / GREATEST(upvote_count, downvote_count, 1) DESC"
	default:
		query += " ORDER BY is_pinned DESC, hot_score DESC"
	}

	query += " LIMIT $1"
	return query
}

func (s *ThreadsStore) scanThread(row *sql.Row) (*threads.Thread, error) {
	thread := &threads.Thread{}
	var editedAt sql.NullTime
	var removeReason sql.NullString

	err := row.Scan(
		&thread.ID, &thread.BoardID, &thread.AuthorID, &thread.Title,
		&thread.Content, &thread.ContentHTML, &thread.URL, &thread.Domain,
		&thread.ThumbnailURL, &thread.Type, &thread.Score, &thread.UpvoteCount,
		&thread.DownvoteCount, &thread.CommentCount, &thread.ViewCount, &thread.HotScore,
		&thread.IsPinned, &thread.IsLocked, &thread.IsRemoved, &thread.IsNSFW,
		&thread.IsSpoiler, &thread.IsOC, &removeReason, &thread.CreatedAt,
		&thread.UpdatedAt, &editedAt)

	if err == sql.ErrNoRows {
		return nil, threads.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if editedAt.Valid {
		thread.EditedAt = &editedAt.Time
	}
	if removeReason.Valid {
		thread.RemoveReason = removeReason.String
	}

	return thread, nil
}

func (s *ThreadsStore) scanThreads(rows *sql.Rows) ([]*threads.Thread, error) {
	var result []*threads.Thread
	for rows.Next() {
		thread := &threads.Thread{}
		var editedAt sql.NullTime
		var removeReason sql.NullString

		err := rows.Scan(
			&thread.ID, &thread.BoardID, &thread.AuthorID, &thread.Title,
			&thread.Content, &thread.ContentHTML, &thread.URL, &thread.Domain,
			&thread.ThumbnailURL, &thread.Type, &thread.Score, &thread.UpvoteCount,
			&thread.DownvoteCount, &thread.CommentCount, &thread.ViewCount, &thread.HotScore,
			&thread.IsPinned, &thread.IsLocked, &thread.IsRemoved, &thread.IsNSFW,
			&thread.IsSpoiler, &thread.IsOC, &removeReason, &thread.CreatedAt,
			&thread.UpdatedAt, &editedAt)

		if err != nil {
			return nil, err
		}

		if editedAt.Valid {
			thread.EditedAt = &editedAt.Time
		}
		if removeReason.Valid {
			thread.RemoveReason = removeReason.String
		}

		result = append(result, thread)
	}
	return result, rows.Err()
}
