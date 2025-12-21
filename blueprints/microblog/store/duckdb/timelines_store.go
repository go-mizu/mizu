package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-mizu/blueprints/microblog/feature/posts"
)

// TimelinesStore implements timelines.Store using DuckDB.
type TimelinesStore struct {
	db *sql.DB
}

// NewTimelinesStore creates a new timelines store.
func NewTimelinesStore(db *sql.DB) *TimelinesStore {
	return &TimelinesStore{db: db}
}

func (s *TimelinesStore) Home(ctx context.Context, accountID string, limit int, maxID, sinceID string) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility,
		       p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive,
		       p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count
		FROM posts p
		WHERE (
			p.account_id IN (SELECT following_id FROM follows WHERE follower_id = $1)
			OR p.account_id = $1
		)
		AND p.account_id NOT IN (SELECT target_id FROM blocks WHERE account_id = $1)
		AND p.account_id NOT IN (
			SELECT target_id FROM mutes WHERE account_id = $1
			AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		)
		AND p.visibility IN ('public', 'unlisted', 'followers')
	`

	args := []any{accountID}
	argIdx := 2

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argIdx)
		args = append(args, maxID)
		argIdx++
	}
	if sinceID != "" {
		query += fmt.Sprintf(" AND p.id > $%d", argIdx)
		args = append(args, sinceID)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTimelinePosts(rows)
}

func (s *TimelinesStore) Local(ctx context.Context, limit int, maxID, sinceID string) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility,
		       p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive,
		       p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count
		FROM posts p
		JOIN accounts a ON a.id = p.account_id
		WHERE p.visibility = 'public'
		AND a.suspended = FALSE
	`

	args := []any{}
	argIdx := 1

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argIdx)
		args = append(args, maxID)
		argIdx++
	}
	if sinceID != "" {
		query += fmt.Sprintf(" AND p.id > $%d", argIdx)
		args = append(args, sinceID)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTimelinePosts(rows)
}

func (s *TimelinesStore) Hashtag(ctx context.Context, tag string, limit int, maxID, sinceID string) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility,
		       p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive,
		       p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count
		FROM posts p
		JOIN post_hashtags ph ON ph.post_id = p.id
		JOIN hashtags h ON h.id = ph.hashtag_id
		JOIN accounts a ON a.id = p.account_id
		WHERE LOWER(h.name) = LOWER($1)
		AND p.visibility IN ('public', 'unlisted')
		AND a.suspended = FALSE
	`

	args := []any{tag}
	argIdx := 2

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argIdx)
		args = append(args, maxID)
		argIdx++
	}
	if sinceID != "" {
		query += fmt.Sprintf(" AND p.id > $%d", argIdx)
		args = append(args, sinceID)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTimelinePosts(rows)
}

func (s *TimelinesStore) Account(ctx context.Context, accountID, viewerID string, limit int, maxID string, onlyMedia, excludeReplies bool, isFollowing bool) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility,
		       p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive,
		       p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count
		FROM posts p
		WHERE p.account_id = $1
	`

	args := []any{accountID}
	argIdx := 2

	if viewerID == accountID {
		// Own posts - show all
	} else if isFollowing {
		query += " AND p.visibility IN ('public', 'unlisted', 'followers')"
	} else {
		query += " AND p.visibility IN ('public', 'unlisted')"
	}

	if excludeReplies {
		query += " AND p.reply_to_id IS NULL"
	}

	if onlyMedia {
		query += " AND EXISTS(SELECT 1 FROM media WHERE post_id = p.id)"
	}

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argIdx)
		args = append(args, maxID)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTimelinePosts(rows)
}

func (s *TimelinesStore) List(ctx context.Context, listID string, limit int, maxID string) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility,
		       p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive,
		       p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count
		FROM posts p
		JOIN list_members lm ON lm.account_id = p.account_id
		WHERE lm.list_id = $1
		AND p.visibility IN ('public', 'unlisted', 'followers')
	`

	args := []any{listID}
	argIdx := 2

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argIdx)
		args = append(args, maxID)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTimelinePosts(rows)
}

func (s *TimelinesStore) Bookmarks(ctx context.Context, accountID string, limit int, maxID string) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility,
		       p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive,
		       p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count
		FROM posts p
		JOIN bookmarks b ON b.post_id = p.id
		WHERE b.account_id = $1
	`

	args := []any{accountID}
	argIdx := 2

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argIdx)
		args = append(args, maxID)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY b.created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTimelinePosts(rows)
}

func (s *TimelinesStore) IsFollowing(ctx context.Context, followerID, followingID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)", followerID, followingID).Scan(&exists)
	return exists, err
}

func (s *TimelinesStore) CheckLiked(ctx context.Context, accountID, postID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM likes WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	return exists, err
}

func (s *TimelinesStore) CheckReposted(ctx context.Context, accountID, postID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM reposts WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	return exists, err
}

func (s *TimelinesStore) CheckBookmarked(ctx context.Context, accountID, postID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM bookmarks WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	return exists, err
}

func scanTimelinePosts(rows *sql.Rows) ([]*posts.Post, error) {
	var result []*posts.Post
	for rows.Next() {
		var p posts.Post
		var cw, replyToID, threadID, quoteOfID, language sql.NullString
		var editedAt sql.NullTime

		err := rows.Scan(
			&p.ID, &p.AccountID, &p.Content, &cw, &p.Visibility, &replyToID, &threadID,
			&quoteOfID, &language, &p.Sensitive, &editedAt, &p.CreatedAt,
			&p.LikesCount, &p.RepostsCount, &p.RepliesCount,
		)
		if err != nil {
			continue
		}

		p.ContentWarning = cw.String
		p.ReplyToID = replyToID.String
		p.ThreadID = threadID.String
		p.QuoteOfID = quoteOfID.String
		p.Language = language.String
		if editedAt.Valid {
			p.EditedAt = &editedAt.Time
		}

		result = append(result, &p)
	}

	return result, nil
}
