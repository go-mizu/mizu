package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/microblog/pkg/ulid"
)

// RelationshipsStore implements relationships.Store using DuckDB.
type RelationshipsStore struct {
	db *sql.DB
}

// NewRelationshipsStore creates a new relationships store.
func NewRelationshipsStore(db *sql.DB) *RelationshipsStore {
	return &RelationshipsStore{db: db}
}

func (s *RelationshipsStore) Follow(ctx context.Context, followerID, followingID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO follows (id, follower_id, following_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, ulid.New(), followerID, followingID, time.Now())
	return err
}

func (s *RelationshipsStore) Unfollow(ctx context.Context, followerID, followingID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM follows WHERE follower_id = $1 AND following_id = $2", followerID, followingID)
	return err
}

func (s *RelationshipsStore) IsFollowing(ctx context.Context, followerID, followingID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)", followerID, followingID).Scan(&exists)
	return exists, err
}

func (s *RelationshipsStore) GetFollowers(ctx context.Context, targetID string, limit, offset int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT follower_id FROM follows
		WHERE following_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, targetID, limit, offset)
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

func (s *RelationshipsStore) GetFollowing(ctx context.Context, targetID string, limit, offset int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT following_id FROM follows
		WHERE follower_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, targetID, limit, offset)
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

func (s *RelationshipsStore) CountFollowers(ctx context.Context, accountID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM follows WHERE following_id = $1", accountID).Scan(&count)
	return count, err
}

func (s *RelationshipsStore) CountFollowing(ctx context.Context, accountID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM follows WHERE follower_id = $1", accountID).Scan(&count)
	return count, err
}

func (s *RelationshipsStore) Block(ctx context.Context, accountID, targetID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO blocks (id, account_id, target_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, ulid.New(), accountID, targetID, time.Now())
	return err
}

func (s *RelationshipsStore) Unblock(ctx context.Context, accountID, targetID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM blocks WHERE account_id = $1 AND target_id = $2", accountID, targetID)
	return err
}

func (s *RelationshipsStore) IsBlocking(ctx context.Context, accountID, targetID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM blocks WHERE account_id = $1 AND target_id = $2)", accountID, targetID).Scan(&exists)
	return exists, err
}

func (s *RelationshipsStore) GetBlocked(ctx context.Context, accountID string, limit, offset int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
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

func (s *RelationshipsStore) RemoveFollowsBetween(ctx context.Context, accountID, targetID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM follows WHERE (follower_id = $1 AND following_id = $2) OR (follower_id = $2 AND following_id = $1)", accountID, targetID)
	return err
}

func (s *RelationshipsStore) Mute(ctx context.Context, accountID, targetID string, hideNotifs bool, expiresAt *time.Time) error {
	_, _ = s.db.ExecContext(ctx, "DELETE FROM mutes WHERE account_id = $1 AND target_id = $2", accountID, targetID)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mutes (id, account_id, target_id, hide_notifications, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, ulid.New(), accountID, targetID, hideNotifs, expiresAt, time.Now())
	return err
}

func (s *RelationshipsStore) Unmute(ctx context.Context, accountID, targetID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM mutes WHERE account_id = $1 AND target_id = $2", accountID, targetID)
	return err
}

func (s *RelationshipsStore) IsMuting(ctx context.Context, accountID, targetID string) (bool, bool, error) {
	var hideNotifs bool
	err := s.db.QueryRowContext(ctx, `
		SELECT hide_notifications FROM mutes
		WHERE account_id = $1 AND target_id = $2
		AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
	`, accountID, targetID).Scan(&hideNotifs)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, false, nil
		}
		return false, false, err
	}
	return true, hideNotifs, nil
}

func (s *RelationshipsStore) GetMuted(ctx context.Context, accountID string, limit, offset int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
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

func (s *RelationshipsStore) CreateNotification(ctx context.Context, accountID, actorID, notifType string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (id, account_id, type, actor_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, ulid.New(), accountID, notifType, actorID, time.Now())
	return err
}
