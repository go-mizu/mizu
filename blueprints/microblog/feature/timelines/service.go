// Package timelines provides timeline/feed generation.
package timelines

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/feature/posts"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

// Service handles timeline operations.
type Service struct {
	store    *duckdb.Store
	accounts *accounts.Service
	posts    *posts.Service
}

// NewService creates a new timelines service.
func NewService(store *duckdb.Store, accounts *accounts.Service, posts *posts.Service) *Service {
	return &Service{store: store, accounts: accounts, posts: posts}
}

// Home returns the home timeline for an account (posts from followed accounts).
func (s *Service) Home(ctx context.Context, accountID string, limit int, maxID, sinceID string) ([]*posts.Post, error) {
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

	rows, err := s.store.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("timelines: home: %w", err)
	}
	defer rows.Close()

	return s.scanPosts(ctx, rows, accountID)
}

// Local returns the local timeline (all public posts on the instance).
func (s *Service) Local(ctx context.Context, viewerID string, limit int, maxID, sinceID string) ([]*posts.Post, error) {
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

	rows, err := s.store.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("timelines: local: %w", err)
	}
	defer rows.Close()

	return s.scanPosts(ctx, rows, viewerID)
}

// Hashtag returns posts with a specific hashtag.
func (s *Service) Hashtag(ctx context.Context, tag, viewerID string, limit int, maxID, sinceID string) ([]*posts.Post, error) {
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

	rows, err := s.store.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("timelines: hashtag: %w", err)
	}
	defer rows.Close()

	return s.scanPosts(ctx, rows, viewerID)
}

// Account returns posts by a specific account.
func (s *Service) Account(ctx context.Context, accountID, viewerID string, limit int, maxID string, onlyMedia, excludeReplies bool) ([]*posts.Post, error) {
	query := `
		SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility,
		       p.reply_to_id, p.thread_id, p.quote_of_id, p.language, p.sensitive,
		       p.edited_at, p.created_at, p.likes_count, p.reposts_count, p.replies_count
		FROM posts p
		WHERE p.account_id = $1
	`

	args := []any{accountID}
	argIdx := 2

	// Determine visibility based on viewer
	if viewerID == accountID {
		// Own posts - show all
	} else if viewerID != "" {
		// Check if following
		var following bool
		_ = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)", viewerID, accountID).Scan(&following)
		if following {
			query += " AND p.visibility IN ('public', 'unlisted', 'followers')"
		} else {
			query += " AND p.visibility IN ('public', 'unlisted')"
		}
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

	rows, err := s.store.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("timelines: account: %w", err)
	}
	defer rows.Close()

	return s.scanPosts(ctx, rows, viewerID)
}

// List returns posts from accounts in a list.
func (s *Service) List(ctx context.Context, listID, viewerID string, limit int, maxID string) ([]*posts.Post, error) {
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

	rows, err := s.store.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("timelines: list: %w", err)
	}
	defer rows.Close()

	return s.scanPosts(ctx, rows, viewerID)
}

// Bookmarks returns bookmarked posts for an account.
func (s *Service) Bookmarks(ctx context.Context, accountID string, limit int, maxID string) ([]*posts.Post, error) {
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

	rows, err := s.store.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("timelines: bookmarks: %w", err)
	}
	defer rows.Close()

	return s.scanPosts(ctx, rows, accountID)
}

func (s *Service) scanPosts(ctx context.Context, rows *sql.Rows, viewerID string) ([]*posts.Post, error) {
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

		// Load account
		p.Account, _ = s.accounts.GetByID(ctx, p.AccountID)

		// Load viewer state
		if viewerID != "" {
			s.loadViewerState(ctx, &p, viewerID)
		}

		result = append(result, &p)
	}

	return result, nil
}

func (s *Service) loadViewerState(ctx context.Context, post *posts.Post, viewerID string) {
	var exists bool

	_ = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM likes WHERE account_id = $1 AND post_id = $2)", viewerID, post.ID).Scan(&exists)
	post.Liked = exists

	_ = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM reposts WHERE account_id = $1 AND post_id = $2)", viewerID, post.ID).Scan(&exists)
	post.Reposted = exists

	_ = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM bookmarks WHERE account_id = $1 AND post_id = $2)", viewerID, post.ID).Scan(&exists)
	post.Bookmarked = exists
}
