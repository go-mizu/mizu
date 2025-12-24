package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/social/feature/posts"
	"github.com/go-mizu/blueprints/social/feature/trending"
)

// TrendingStore implements trending.Store.
type TrendingStore struct {
	db *sql.DB
}

// NewTrendingStore creates a new trending store.
func NewTrendingStore(db *sql.DB) *TrendingStore {
	return &TrendingStore{db: db}
}

// GetTrendingTags returns trending hashtags.
func (s *TrendingStore) GetTrendingTags(ctx context.Context, limit, offset int) ([]*trending.TrendingTag, error) {
	cutoff := time.Now().Add(-24 * time.Hour)
	rows, err := s.db.QueryContext(ctx, `
		SELECT h.name, COUNT(DISTINCT ph.post_id) as recent_count
		FROM hashtags h
		JOIN post_hashtags ph ON h.id = ph.hashtag_id
		JOIN posts p ON ph.post_id = p.id
		WHERE p.created_at > $1 AND p.visibility = 'public'
		GROUP BY h.id, h.name
		ORDER BY recent_count DESC
		LIMIT $2 OFFSET $3
	`, cutoff, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*trending.TrendingTag
	for rows.Next() {
		var t trending.TrendingTag
		if err := rows.Scan(&t.Name, &t.PostsCount); err != nil {
			return nil, err
		}
		tags = append(tags, &t)
	}
	return tags, rows.Err()
}

// GetTrendingPosts returns trending posts.
func (s *TrendingStore) GetTrendingPosts(ctx context.Context, limit, offset int) ([]*posts.Post, error) {
	cutoff := time.Now().Add(-24 * time.Hour)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, content, content_warning, visibility, reply_to_id, thread_id, quote_of_id, language, sensitive, edited_at, created_at, likes_count, reposts_count, replies_count, quotes_count
		FROM posts
		WHERE created_at > $1 AND visibility = 'public' AND reply_to_id IS NULL
		ORDER BY (likes_count + reposts_count * 2) DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`, cutoff, limit, offset)
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

// ComputeTrendingTags computes trending tags.
func (s *TrendingStore) ComputeTrendingTags(ctx context.Context, window time.Duration, limit int) ([]*trending.TrendingTag, error) {
	return s.GetTrendingTags(ctx, limit, 0)
}

// ComputeTrendingPosts computes trending posts.
func (s *TrendingStore) ComputeTrendingPosts(ctx context.Context, window time.Duration, limit int) ([]*posts.Post, error) {
	return s.GetTrendingPosts(ctx, limit, 0)
}

func (s *TrendingStore) scanPostRow(rows *sql.Rows) (*posts.Post, error) {
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
