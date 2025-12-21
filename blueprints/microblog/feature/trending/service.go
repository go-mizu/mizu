// Package trending provides trending topics and posts calculation.
package trending

import (
	"context"
	"fmt"

	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

// TrendingTag represents a trending hashtag.
type TrendingTag struct {
	Name       string `json:"name"`
	PostsCount int    `json:"posts_count"`
	Accounts   int    `json:"accounts,omitempty"` // Unique accounts using this tag
}

// Service handles trending calculations.
type Service struct {
	store *duckdb.Store
}

// NewService creates a new trending service.
func NewService(store *duckdb.Store) *Service {
	return &Service{store: store}
}

// Tags returns trending hashtags.
func (s *Service) Tags(ctx context.Context, limit int) ([]*TrendingTag, error) {
	// Calculate trending based on:
	// 1. Usage in last 24 hours
	// 2. Velocity (growth compared to previous period)
	// 3. Unique accounts using it

	rows, err := s.store.Query(ctx, `
		WITH recent_usage AS (
			SELECT
				h.id,
				h.name,
				COUNT(DISTINCT p.id) as posts_24h,
				COUNT(DISTINCT p.account_id) as accounts_24h
			FROM hashtags h
			JOIN post_hashtags ph ON ph.hashtag_id = h.id
			JOIN posts p ON p.id = ph.post_id
			WHERE p.created_at > CURRENT_TIMESTAMP - INTERVAL '24 hours'
			AND p.visibility = 'public'
			GROUP BY h.id, h.name
			HAVING COUNT(DISTINCT p.id) >= 2
		),
		previous_usage AS (
			SELECT
				h.id,
				COUNT(DISTINCT p.id) as posts_prev
			FROM hashtags h
			JOIN post_hashtags ph ON ph.hashtag_id = h.id
			JOIN posts p ON p.id = ph.post_id
			WHERE p.created_at BETWEEN CURRENT_TIMESTAMP - INTERVAL '48 hours'
			                       AND CURRENT_TIMESTAMP - INTERVAL '24 hours'
			GROUP BY h.id
		)
		SELECT
			r.name,
			r.posts_24h,
			r.accounts_24h,
			COALESCE(p.posts_prev, 0) as posts_prev,
			CASE
				WHEN COALESCE(p.posts_prev, 0) = 0 THEN r.posts_24h::FLOAT
				ELSE r.posts_24h::FLOAT / p.posts_prev
			END as velocity
		FROM recent_usage r
		LEFT JOIN previous_usage p ON p.id = r.id
		ORDER BY velocity DESC, r.posts_24h DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("trending: tags: %w", err)
	}
	defer rows.Close()

	var tags []*TrendingTag
	for rows.Next() {
		var t TrendingTag
		var postsPrev int
		var velocity float64

		if err := rows.Scan(&t.Name, &t.PostsCount, &t.Accounts, &postsPrev, &velocity); err != nil {
			continue
		}
		tags = append(tags, &t)
	}

	return tags, nil
}

// Posts returns trending posts.
func (s *Service) Posts(ctx context.Context, limit int) ([]string, error) {
	// Calculate trending based on weighted engagement score with time decay

	rows, err := s.store.Query(ctx, `
		SELECT p.id
		FROM posts p
		JOIN accounts a ON a.id = p.account_id
		WHERE p.visibility = 'public'
		AND a.suspended = FALSE
		AND p.created_at > CURRENT_TIMESTAMP - INTERVAL '24 hours'
		ORDER BY (
			(p.likes_count * 1.0 + p.reposts_count * 2.0 + p.replies_count * 1.5) /
			POWER(EXTRACT(EPOCH FROM CURRENT_TIMESTAMP - p.created_at) / 3600 + 2, 1.5)
		) DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("trending: posts: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

// SuggestedAccounts returns suggested accounts to follow.
func (s *Service) SuggestedAccounts(ctx context.Context, accountID string, limit int) ([]string, error) {
	// Suggest accounts based on:
	// 1. Popular accounts not yet followed
	// 2. Accounts followed by people you follow
	// 3. Recently active accounts

	rows, err := s.store.Query(ctx, `
		WITH followed_ids AS (
			SELECT following_id FROM follows WHERE follower_id = $1
		),
		blocked_ids AS (
			SELECT target_id FROM blocks WHERE account_id = $1
			UNION
			SELECT account_id FROM blocks WHERE target_id = $1
		)
		SELECT a.id
		FROM accounts a
		LEFT JOIN (
			SELECT following_id, COUNT(*) as follower_count
			FROM follows
			GROUP BY following_id
		) f ON f.following_id = a.id
		WHERE a.id != $1
		AND a.id NOT IN (SELECT following_id FROM followed_ids)
		AND a.id NOT IN (SELECT target_id FROM blocked_ids)
		AND a.suspended = FALSE
		ORDER BY COALESCE(f.follower_count, 0) DESC
		LIMIT $2
	`, accountID, limit)
	if err != nil {
		return nil, fmt.Errorf("trending: suggested accounts: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}

	return ids, nil
}
