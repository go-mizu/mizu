// Package relationships provides follow, block, and mute functionality.
package relationships

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/microblog/pkg/ulid"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrSelfAction   = errors.New("cannot perform action on yourself")
	ErrBlocked      = errors.New("user is blocked")
)

// Relationship represents the relationship between two accounts.
type Relationship struct {
	ID                  string `json:"id"`
	Following           bool   `json:"following"`
	FollowedBy          bool   `json:"followed_by"`
	Blocking            bool   `json:"blocking"`
	BlockedBy           bool   `json:"blocked_by"`
	Muting              bool   `json:"muting"`
	MutingNotifications bool   `json:"muting_notifications"`
}

// Service handles relationship operations.
type Service struct {
	store *duckdb.Store
}

// NewService creates a new relationships service.
func NewService(store *duckdb.Store) *Service {
	return &Service{store: store}
}

// Get returns the relationship between two accounts.
func (s *Service) Get(ctx context.Context, accountID, targetID string) (*Relationship, error) {
	rel := &Relationship{ID: targetID}

	// Check following
	_ = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)", accountID, targetID).Scan(&rel.Following)

	// Check followed by
	_ = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)", targetID, accountID).Scan(&rel.FollowedBy)

	// Check blocking
	_ = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM blocks WHERE account_id = $1 AND target_id = $2)", accountID, targetID).Scan(&rel.Blocking)

	// Check blocked by
	_ = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM blocks WHERE account_id = $1 AND target_id = $2)", targetID, accountID).Scan(&rel.BlockedBy)

	// Check muting
	var hideNotifs bool
	err := s.store.QueryRow(ctx, `
		SELECT hide_notifications FROM mutes
		WHERE account_id = $1 AND target_id = $2
		AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
	`, accountID, targetID).Scan(&hideNotifs)
	if err == nil {
		rel.Muting = true
		rel.MutingNotifications = hideNotifs
	}

	return rel, nil
}

// Follow creates a follow relationship.
func (s *Service) Follow(ctx context.Context, accountID, targetID string) error {
	if accountID == targetID {
		return ErrSelfAction
	}

	// Check if blocked
	var blocked bool
	_ = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM blocks WHERE account_id = $1 AND target_id = $2)", targetID, accountID).Scan(&blocked)
	if blocked {
		return ErrBlocked
	}

	// Check if already following
	var exists bool
	err := s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)", accountID, targetID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("relationships: check follow: %w", err)
	}
	if exists {
		return nil // Idempotent
	}

	// Create follow
	_, err = s.store.Exec(ctx, `
		INSERT INTO follows (id, follower_id, following_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, ulid.New(), accountID, targetID, time.Now())
	if err != nil {
		return fmt.Errorf("relationships: create follow: %w", err)
	}

	// Create notification
	_, _ = s.store.Exec(ctx, `
		INSERT INTO notifications (id, account_id, type, actor_id, created_at)
		VALUES ($1, $2, 'follow', $3, $4)
	`, ulid.New(), targetID, accountID, time.Now())

	return nil
}

// Unfollow removes a follow relationship.
func (s *Service) Unfollow(ctx context.Context, accountID, targetID string) error {
	if accountID == targetID {
		return ErrSelfAction
	}

	_, err := s.store.Exec(ctx, "DELETE FROM follows WHERE follower_id = $1 AND following_id = $2", accountID, targetID)
	return err
}

// Block creates a block relationship.
func (s *Service) Block(ctx context.Context, accountID, targetID string) error {
	if accountID == targetID {
		return ErrSelfAction
	}

	// Check if already blocking
	var exists bool
	err := s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM blocks WHERE account_id = $1 AND target_id = $2)", accountID, targetID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("relationships: check block: %w", err)
	}
	if exists {
		return nil // Idempotent
	}

	// Create block
	_, err = s.store.Exec(ctx, `
		INSERT INTO blocks (id, account_id, target_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, ulid.New(), accountID, targetID, time.Now())
	if err != nil {
		return fmt.Errorf("relationships: create block: %w", err)
	}

	// Remove any existing follow relationships (both directions)
	_, _ = s.store.Exec(ctx, "DELETE FROM follows WHERE (follower_id = $1 AND following_id = $2) OR (follower_id = $2 AND following_id = $1)", accountID, targetID)

	return nil
}

// Unblock removes a block relationship.
func (s *Service) Unblock(ctx context.Context, accountID, targetID string) error {
	if accountID == targetID {
		return ErrSelfAction
	}

	_, err := s.store.Exec(ctx, "DELETE FROM blocks WHERE account_id = $1 AND target_id = $2", accountID, targetID)
	return err
}

// Mute creates a mute relationship.
func (s *Service) Mute(ctx context.Context, accountID, targetID string, hideNotifications bool, duration *time.Duration) error {
	if accountID == targetID {
		return ErrSelfAction
	}

	// Delete existing mute first
	_, _ = s.store.Exec(ctx, "DELETE FROM mutes WHERE account_id = $1 AND target_id = $2", accountID, targetID)

	var expiresAt *time.Time
	if duration != nil {
		t := time.Now().Add(*duration)
		expiresAt = &t
	}

	_, err := s.store.Exec(ctx, `
		INSERT INTO mutes (id, account_id, target_id, hide_notifications, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, ulid.New(), accountID, targetID, hideNotifications, expiresAt, time.Now())
	if err != nil {
		return fmt.Errorf("relationships: create mute: %w", err)
	}

	return nil
}

// Unmute removes a mute relationship.
func (s *Service) Unmute(ctx context.Context, accountID, targetID string) error {
	if accountID == targetID {
		return ErrSelfAction
	}

	_, err := s.store.Exec(ctx, "DELETE FROM mutes WHERE account_id = $1 AND target_id = $2", accountID, targetID)
	return err
}

// GetFollowers returns accounts following the target.
func (s *Service) GetFollowers(ctx context.Context, targetID string, limit, offset int) ([]string, error) {
	rows, err := s.store.Query(ctx, `
		SELECT follower_id FROM follows
		WHERE following_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, targetID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("relationships: get followers: %w", err)
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

// GetFollowing returns accounts the target is following.
func (s *Service) GetFollowing(ctx context.Context, targetID string, limit, offset int) ([]string, error) {
	rows, err := s.store.Query(ctx, `
		SELECT following_id FROM follows
		WHERE follower_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, targetID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("relationships: get following: %w", err)
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

// CountFollowers returns the number of followers for an account.
func (s *Service) CountFollowers(ctx context.Context, accountID string) (int, error) {
	var count int
	err := s.store.QueryRow(ctx, "SELECT count(*) FROM follows WHERE following_id = $1", accountID).Scan(&count)
	return count, err
}

// CountFollowing returns the number of accounts an account is following.
func (s *Service) CountFollowing(ctx context.Context, accountID string) (int, error) {
	var count int
	err := s.store.QueryRow(ctx, "SELECT count(*) FROM follows WHERE follower_id = $1", accountID).Scan(&count)
	return count, err
}

// GetBlocked returns blocked account IDs.
func (s *Service) GetBlocked(ctx context.Context, accountID string, limit, offset int) ([]string, error) {
	rows, err := s.store.Query(ctx, `
		SELECT target_id FROM blocks
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, accountID, limit, offset)
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

// GetMuted returns muted account IDs.
func (s *Service) GetMuted(ctx context.Context, accountID string, limit, offset int) ([]string, error) {
	rows, err := s.store.Query(ctx, `
		SELECT target_id FROM mutes
		WHERE account_id = $1
		AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, accountID, limit, offset)
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

// IsBlocked checks if targetID is blocked by accountID.
func (s *Service) IsBlocked(ctx context.Context, accountID, targetID string) (bool, error) {
	var blocked bool
	err := s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM blocks WHERE account_id = $1 AND target_id = $2)", accountID, targetID).Scan(&blocked)
	return blocked, err
}

// IsMuted checks if targetID is muted by accountID.
func (s *Service) IsMuted(ctx context.Context, accountID, targetID string) (bool, error) {
	var muted bool
	err := s.store.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM mutes WHERE account_id = $1 AND target_id = $2
		AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP))
	`, accountID, targetID).Scan(&muted)
	return muted, err
}
