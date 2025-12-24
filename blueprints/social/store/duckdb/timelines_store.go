package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/social/feature/posts"
)

// TimelinesStore implements timelines.Store.
type TimelinesStore struct {
	db *sql.DB
}

// NewTimelinesStore creates a new timelines store.
func NewTimelinesStore(db *sql.DB) *TimelinesStore {
	return &TimelinesStore{db: db}
}

// GetHomeFeed returns the home timeline for an account.
func (s *TimelinesStore) GetHomeFeed(ctx context.Context, accountID string, limit int, maxID, minID string) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility, p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive, p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count, p.quotes_count
		FROM posts p
		WHERE (
			p.account_id = $1
			OR p.account_id IN (SELECT following_id FROM follows WHERE follower_id = $1 AND pending = FALSE)
		)
		AND p.visibility IN ('public', 'followers')
	`
	args := []interface{}{accountID}
	argNum := 2

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argNum)
		args = append(args, maxID)
		argNum++
	}
	if minID != "" {
		query += fmt.Sprintf(" AND p.id > $%d", argNum)
		args = append(args, minID)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d", argNum)
	args = append(args, limit)

	return s.queryPosts(ctx, query, args...)
}

// GetPublicFeed returns the public timeline.
func (s *TimelinesStore) GetPublicFeed(ctx context.Context, limit int, maxID, minID string, onlyMedia bool) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility, p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive, p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count, p.quotes_count
		FROM posts p
		WHERE p.visibility = 'public' AND p.reply_to_id IS NULL
	`
	args := []interface{}{}
	argNum := 1

	if onlyMedia {
		query += " AND EXISTS (SELECT 1 FROM media m WHERE m.post_id = p.id)"
	}

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argNum)
		args = append(args, maxID)
		argNum++
	}
	if minID != "" {
		query += fmt.Sprintf(" AND p.id > $%d", argNum)
		args = append(args, minID)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d", argNum)
	args = append(args, limit)

	return s.queryPosts(ctx, query, args...)
}

// GetUserFeed returns posts from a specific user.
func (s *TimelinesStore) GetUserFeed(ctx context.Context, userID string, limit int, maxID, minID string, includeReplies, onlyMedia bool) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility, p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive, p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count, p.quotes_count
		FROM posts p
		WHERE p.account_id = $1 AND p.visibility = 'public'
	`
	args := []interface{}{userID}
	argNum := 2

	if !includeReplies {
		query += " AND p.reply_to_id IS NULL"
	}
	if onlyMedia {
		query += " AND EXISTS (SELECT 1 FROM media m WHERE m.post_id = p.id)"
	}

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argNum)
		args = append(args, maxID)
		argNum++
	}
	if minID != "" {
		query += fmt.Sprintf(" AND p.id > $%d", argNum)
		args = append(args, minID)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d", argNum)
	args = append(args, limit)

	return s.queryPosts(ctx, query, args...)
}

// GetHashtagFeed returns posts with a specific hashtag.
func (s *TimelinesStore) GetHashtagFeed(ctx context.Context, tag string, limit int, maxID, minID string) ([]*posts.Post, error) {
	tag = strings.ToLower(tag)
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility, p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive, p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count, p.quotes_count
		FROM posts p
		JOIN post_hashtags ph ON p.id = ph.post_id
		JOIN hashtags h ON ph.hashtag_id = h.id
		WHERE h.name = $1 AND p.visibility = 'public'
	`
	args := []interface{}{tag}
	argNum := 2

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argNum)
		args = append(args, maxID)
		argNum++
	}
	if minID != "" {
		query += fmt.Sprintf(" AND p.id > $%d", argNum)
		args = append(args, minID)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d", argNum)
	args = append(args, limit)

	return s.queryPosts(ctx, query, args...)
}

// GetListFeed returns posts from accounts in a list.
func (s *TimelinesStore) GetListFeed(ctx context.Context, listID string, limit int, maxID, minID string) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility, p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive, p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count, p.quotes_count
		FROM posts p
		WHERE p.account_id IN (SELECT account_id FROM list_members WHERE list_id = $1)
		AND p.visibility IN ('public', 'followers')
	`
	args := []interface{}{listID}
	argNum := 2

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argNum)
		args = append(args, maxID)
		argNum++
	}
	if minID != "" {
		query += fmt.Sprintf(" AND p.id > $%d", argNum)
		args = append(args, minID)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d", argNum)
	args = append(args, limit)

	return s.queryPosts(ctx, query, args...)
}

// GetBookmarksFeed returns bookmarked posts.
func (s *TimelinesStore) GetBookmarksFeed(ctx context.Context, accountID string, limit int, maxID, minID string) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility, p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive, p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count, p.quotes_count
		FROM posts p
		JOIN bookmarks b ON p.id = b.post_id
		WHERE b.account_id = $1
	`
	args := []interface{}{accountID}
	argNum := 2

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argNum)
		args = append(args, maxID)
		argNum++
	}
	if minID != "" {
		query += fmt.Sprintf(" AND p.id > $%d", argNum)
		args = append(args, minID)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY b.created_at DESC LIMIT $%d", argNum)
	args = append(args, limit)

	return s.queryPosts(ctx, query, args...)
}

// GetLikesFeed returns liked posts.
func (s *TimelinesStore) GetLikesFeed(ctx context.Context, accountID string, limit int, maxID, minID string) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility, p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive, p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count, p.quotes_count
		FROM posts p
		JOIN likes l ON p.id = l.post_id
		WHERE l.account_id = $1
	`
	args := []interface{}{accountID}
	argNum := 2

	if maxID != "" {
		query += fmt.Sprintf(" AND p.id < $%d", argNum)
		args = append(args, maxID)
		argNum++
	}
	if minID != "" {
		query += fmt.Sprintf(" AND p.id > $%d", argNum)
		args = append(args, minID)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY l.created_at DESC LIMIT $%d", argNum)
	args = append(args, limit)

	return s.queryPosts(ctx, query, args...)
}

func (s *TimelinesStore) queryPosts(ctx context.Context, query string, args ...interface{}) ([]*posts.Post, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ps []*posts.Post
	for rows.Next() {
		p, err := s.scanPostRow(rows)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, rows.Err()
}

func (s *TimelinesStore) scanPostRow(rows *sql.Rows) (*posts.Post, error) {
	var p posts.Post
	var contentWarning, replyToID, threadID, quoteOfID, language sql.NullString
	var editedAt sql.NullTime

	err := rows.Scan(&p.ID, &p.AccountID, &p.Content, &contentWarning, &p.Visibility, &replyToID, &threadID, &quoteOfID, &language, &p.Sensitive, &editedAt, &p.CreatedAt, &p.LikesCount, &p.RepostsCount, &p.RepliesCount, &p.QuotesCount)
	if err != nil {
		return nil, err
	}

	p.ContentWarning = contentWarning.String
	p.ReplyToID = replyToID.String
	p.ThreadID = threadID.String
	p.QuoteOfID = quoteOfID.String
	p.Language = language.String
	if editedAt.Valid {
		p.EditedAt = &editedAt.Time
	}

	return &p, nil
}
