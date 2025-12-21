package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/microblog/feature/trending"
)

// TrendingStore implements trending.Store using DuckDB.
type TrendingStore struct {
	db *sql.DB
}

// NewTrendingStore creates a new trending store.
func NewTrendingStore(db *sql.DB) *TrendingStore {
	return &TrendingStore{db: db}
}

func (s *TrendingStore) Tags(ctx context.Context, limit int) ([]*trending.TrendingTag, error) {
	rows, err := s.db.QueryContext(ctx, `
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
		return nil, err
	}
	defer rows.Close()

	var tags []*trending.TrendingTag
	for rows.Next() {
		var t trending.TrendingTag
		var postsPrev int
		var velocity float64

		if err := rows.Scan(&t.Name, &t.PostsCount, &t.Accounts, &postsPrev, &velocity); err != nil {
			continue
		}
		tags = append(tags, &t)
	}

	return tags, nil
}

func (s *TrendingStore) Posts(ctx context.Context, limit int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
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
		return nil, err
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

func (s *TrendingStore) SuggestedAccounts(ctx context.Context, accountID string, limit int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
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
		return nil, err
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
