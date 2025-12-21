// Package interactions provides like, repost, and bookmark functionality.
package interactions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/microblog/pkg/ulid"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

// Service handles interaction operations.
type Service struct {
	store *duckdb.Store
}

// NewService creates a new interactions service.
func NewService(store *duckdb.Store) *Service {
	return &Service{store: store}
}

// Like creates a like on a post.
func (s *Service) Like(ctx context.Context, accountID, postID string) error {
	// Check if already liked
	var exists bool
	err := s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM likes WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("interactions: check like: %w", err)
	}
	if exists {
		return nil // Idempotent
	}

	// Create like
	_, err = s.store.Exec(ctx, `
		INSERT INTO likes (id, account_id, post_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, ulid.New(), accountID, postID, time.Now())
	if err != nil {
		return fmt.Errorf("interactions: create like: %w", err)
	}

	// Update post counter
	_, _ = s.store.Exec(ctx, "UPDATE posts SET likes_count = likes_count + 1 WHERE id = $1", postID)

	// Create notification
	s.createNotification(ctx, postID, accountID, "like")

	return nil
}

// Unlike removes a like from a post.
func (s *Service) Unlike(ctx context.Context, accountID, postID string) error {
	result, err := s.store.Exec(ctx, "DELETE FROM likes WHERE account_id = $1 AND post_id = $2", accountID, postID)
	if err != nil {
		return fmt.Errorf("interactions: delete like: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		_, _ = s.store.Exec(ctx, "UPDATE posts SET likes_count = likes_count - 1 WHERE id = $1 AND likes_count > 0", postID)
	}

	return nil
}

// Repost creates a repost (boost) of a post.
func (s *Service) Repost(ctx context.Context, accountID, postID string) error {
	// Check if already reposted
	var exists bool
	err := s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM reposts WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("interactions: check repost: %w", err)
	}
	if exists {
		return nil // Idempotent
	}

	// Create repost
	_, err = s.store.Exec(ctx, `
		INSERT INTO reposts (id, account_id, post_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, ulid.New(), accountID, postID, time.Now())
	if err != nil {
		return fmt.Errorf("interactions: create repost: %w", err)
	}

	// Update post counter
	_, _ = s.store.Exec(ctx, "UPDATE posts SET reposts_count = reposts_count + 1 WHERE id = $1", postID)

	// Create notification
	s.createNotification(ctx, postID, accountID, "repost")

	return nil
}

// Unrepost removes a repost.
func (s *Service) Unrepost(ctx context.Context, accountID, postID string) error {
	result, err := s.store.Exec(ctx, "DELETE FROM reposts WHERE account_id = $1 AND post_id = $2", accountID, postID)
	if err != nil {
		return fmt.Errorf("interactions: delete repost: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		_, _ = s.store.Exec(ctx, "UPDATE posts SET reposts_count = reposts_count - 1 WHERE id = $1 AND reposts_count > 0", postID)
	}

	return nil
}

// Bookmark saves a post privately.
func (s *Service) Bookmark(ctx context.Context, accountID, postID string) error {
	// Check if already bookmarked
	var exists bool
	err := s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM bookmarks WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("interactions: check bookmark: %w", err)
	}
	if exists {
		return nil // Idempotent
	}

	// Create bookmark
	_, err = s.store.Exec(ctx, `
		INSERT INTO bookmarks (id, account_id, post_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, ulid.New(), accountID, postID, time.Now())
	if err != nil {
		return fmt.Errorf("interactions: create bookmark: %w", err)
	}

	return nil
}

// Unbookmark removes a bookmark.
func (s *Service) Unbookmark(ctx context.Context, accountID, postID string) error {
	_, err := s.store.Exec(ctx, "DELETE FROM bookmarks WHERE account_id = $1 AND post_id = $2", accountID, postID)
	return err
}

// VotePoll casts a vote on a poll.
func (s *Service) VotePoll(ctx context.Context, accountID, pollID string, choices []int) error {
	// Check if poll exists and not expired
	var expiresAt sql.NullTime
	var multiple bool
	err := s.store.QueryRow(ctx, "SELECT expires_at, multiple FROM polls WHERE id = $1", pollID).Scan(&expiresAt, &multiple)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("interactions: get poll: %w", err)
	}

	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		return errors.New("poll has expired")
	}

	// Check if already voted
	var voted bool
	err = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM poll_votes WHERE poll_id = $1 AND account_id = $2)", pollID, accountID).Scan(&voted)
	if err != nil {
		return fmt.Errorf("interactions: check vote: %w", err)
	}
	if voted {
		return errors.New("already voted")
	}

	// Validate choices
	if !multiple && len(choices) > 1 {
		return errors.New("multiple choices not allowed")
	}

	// Insert votes
	for _, choice := range choices {
		_, err = s.store.Exec(ctx, `
			INSERT INTO poll_votes (id, poll_id, account_id, choice, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, ulid.New(), pollID, accountID, choice, time.Now())
		if err != nil {
			return fmt.Errorf("interactions: insert vote: %w", err)
		}
	}

	// Update voters count
	_, _ = s.store.Exec(ctx, "UPDATE polls SET voters_count = voters_count + 1 WHERE id = $1", pollID)

	// Update options vote counts (this is a simplified approach)
	// In production, you'd use a proper JSON update or separate table
	for _, choice := range choices {
		_, _ = s.store.Exec(ctx, `
			UPDATE polls
			SET options = json_set(options, '$[' || $1 || '].votes_count',
				COALESCE(json_extract(options, '$[' || $1 || '].votes_count')::INT, 0) + 1)
			WHERE id = $2
		`, choice, pollID)
	}

	return nil
}

// GetLikedBy returns accounts that liked a post.
func (s *Service) GetLikedBy(ctx context.Context, postID string, limit, offset int) ([]string, error) {
	rows, err := s.store.Query(ctx, `
		SELECT account_id FROM likes
		WHERE post_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, postID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("interactions: get liked by: %w", err)
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

// GetRepostedBy returns accounts that reposted a post.
func (s *Service) GetRepostedBy(ctx context.Context, postID string, limit, offset int) ([]string, error) {
	rows, err := s.store.Query(ctx, `
		SELECT account_id FROM reposts
		WHERE post_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, postID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("interactions: get reposted by: %w", err)
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

func (s *Service) createNotification(ctx context.Context, postID, actorID, notifType string) {
	// Get post owner
	var ownerID string
	err := s.store.QueryRow(ctx, "SELECT account_id FROM posts WHERE id = $1", postID).Scan(&ownerID)
	if err != nil || ownerID == actorID {
		return // Don't notify yourself
	}

	_, _ = s.store.Exec(ctx, `
		INSERT INTO notifications (id, account_id, type, actor_id, post_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, ulid.New(), ownerID, notifType, actorID, postID, time.Now())
}
