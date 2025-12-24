package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/social/feature/relationships"
)

// RelationshipsStore implements relationships.Store.
type RelationshipsStore struct {
	db *sql.DB
}

// NewRelationshipsStore creates a new relationships store.
func NewRelationshipsStore(db *sql.DB) *RelationshipsStore {
	return &RelationshipsStore{db: db}
}

// InsertFollow inserts a follow relationship.
func (s *RelationshipsStore) InsertFollow(ctx context.Context, f *relationships.Follow) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO follows (id, follower_id, following_id, pending, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, f.ID, f.FollowerID, f.FollowingID, f.Pending, f.CreatedAt)
	return err
}

// DeleteFollow deletes a follow relationship.
func (s *RelationshipsStore) DeleteFollow(ctx context.Context, followerID, followingID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM follows WHERE follower_id = $1 AND following_id = $2", followerID, followingID)
	return err
}

// GetFollow gets a follow relationship.
func (s *RelationshipsStore) GetFollow(ctx context.Context, followerID, followingID string) (*relationships.Follow, error) {
	var f relationships.Follow
	err := s.db.QueryRowContext(ctx, `
		SELECT id, follower_id, following_id, pending, created_at
		FROM follows WHERE follower_id = $1 AND following_id = $2
	`, followerID, followingID).Scan(&f.ID, &f.FollowerID, &f.FollowingID, &f.Pending, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// SetFollowPending sets the pending status of a follow.
func (s *RelationshipsStore) SetFollowPending(ctx context.Context, followerID, followingID string, pending bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE follows SET pending = $1 WHERE follower_id = $2 AND following_id = $3", pending, followerID, followingID)
	return err
}

// GetFollowers gets followers of an account.
func (s *RelationshipsStore) GetFollowers(ctx context.Context, accountID string, limit, offset int) ([]*relationships.Follow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, follower_id, following_id, pending, created_at
		FROM follows WHERE following_id = $1 AND pending = FALSE
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var follows []*relationships.Follow
	for rows.Next() {
		var f relationships.Follow
		err := rows.Scan(&f.ID, &f.FollowerID, &f.FollowingID, &f.Pending, &f.CreatedAt)
		if err != nil {
			return nil, err
		}
		follows = append(follows, &f)
	}
	return follows, rows.Err()
}

// GetFollowing gets accounts an account follows.
func (s *RelationshipsStore) GetFollowing(ctx context.Context, accountID string, limit, offset int) ([]*relationships.Follow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, follower_id, following_id, pending, created_at
		FROM follows WHERE follower_id = $1 AND pending = FALSE
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var follows []*relationships.Follow
	for rows.Next() {
		var f relationships.Follow
		err := rows.Scan(&f.ID, &f.FollowerID, &f.FollowingID, &f.Pending, &f.CreatedAt)
		if err != nil {
			return nil, err
		}
		follows = append(follows, &f)
	}
	return follows, rows.Err()
}

// GetPendingFollowers gets pending follow requests.
func (s *RelationshipsStore) GetPendingFollowers(ctx context.Context, accountID string, limit, offset int) ([]*relationships.Follow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, follower_id, following_id, pending, created_at
		FROM follows WHERE following_id = $1 AND pending = TRUE
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var follows []*relationships.Follow
	for rows.Next() {
		var f relationships.Follow
		err := rows.Scan(&f.ID, &f.FollowerID, &f.FollowingID, &f.Pending, &f.CreatedAt)
		if err != nil {
			return nil, err
		}
		follows = append(follows, &f)
	}
	return follows, rows.Err()
}

// ExistsFollow checks if a follow exists.
func (s *RelationshipsStore) ExistsFollow(ctx context.Context, followerID, followingID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)", followerID, followingID).Scan(&exists)
	return exists, err
}

// InsertBlock inserts a block.
func (s *RelationshipsStore) InsertBlock(ctx context.Context, b *relationships.Block) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO blocks (id, account_id, target_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, b.ID, b.AccountID, b.TargetID, b.CreatedAt)
	return err
}

// DeleteBlock deletes a block.
func (s *RelationshipsStore) DeleteBlock(ctx context.Context, accountID, targetID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM blocks WHERE account_id = $1 AND target_id = $2", accountID, targetID)
	return err
}

// GetBlock gets a block.
func (s *RelationshipsStore) GetBlock(ctx context.Context, accountID, targetID string) (*relationships.Block, error) {
	var b relationships.Block
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, target_id, created_at
		FROM blocks WHERE account_id = $1 AND target_id = $2
	`, accountID, targetID).Scan(&b.ID, &b.AccountID, &b.TargetID, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// GetBlocks gets blocks for an account.
func (s *RelationshipsStore) GetBlocks(ctx context.Context, accountID string, limit, offset int) ([]*relationships.Block, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, target_id, created_at
		FROM blocks WHERE account_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []*relationships.Block
	for rows.Next() {
		var b relationships.Block
		err := rows.Scan(&b.ID, &b.AccountID, &b.TargetID, &b.CreatedAt)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, &b)
	}
	return blocks, rows.Err()
}

// ExistsBlock checks if a block exists.
func (s *RelationshipsStore) ExistsBlock(ctx context.Context, accountID, targetID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM blocks WHERE account_id = $1 AND target_id = $2)", accountID, targetID).Scan(&exists)
	return exists, err
}

// ExistsBlockEither checks if a block exists in either direction.
func (s *RelationshipsStore) ExistsBlockEither(ctx context.Context, accountID, targetID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM blocks WHERE (account_id = $1 AND target_id = $2) OR (account_id = $2 AND target_id = $1)
		)
	`, accountID, targetID).Scan(&exists)
	return exists, err
}

// InsertMute inserts a mute.
func (s *RelationshipsStore) InsertMute(ctx context.Context, m *relationships.Mute) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mutes (id, account_id, target_id, hide_notifications, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, m.ID, m.AccountID, m.TargetID, m.HideNotifications, m.ExpiresAt, m.CreatedAt)
	return err
}

// DeleteMute deletes a mute.
func (s *RelationshipsStore) DeleteMute(ctx context.Context, accountID, targetID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM mutes WHERE account_id = $1 AND target_id = $2", accountID, targetID)
	return err
}

// GetMute gets a mute.
func (s *RelationshipsStore) GetMute(ctx context.Context, accountID, targetID string) (*relationships.Mute, error) {
	var m relationships.Mute
	var expiresAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, target_id, hide_notifications, expires_at, created_at
		FROM mutes WHERE account_id = $1 AND target_id = $2
	`, accountID, targetID).Scan(&m.ID, &m.AccountID, &m.TargetID, &m.HideNotifications, &expiresAt, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		m.ExpiresAt = &expiresAt.Time
	}
	return &m, nil
}

// GetMutes gets mutes for an account.
func (s *RelationshipsStore) GetMutes(ctx context.Context, accountID string, limit, offset int) ([]*relationships.Mute, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, target_id, hide_notifications, expires_at, created_at
		FROM mutes WHERE account_id = $1 AND (expires_at IS NULL OR expires_at > $4)
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, accountID, limit, offset, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mutes []*relationships.Mute
	for rows.Next() {
		var m relationships.Mute
		var expiresAt sql.NullTime
		err := rows.Scan(&m.ID, &m.AccountID, &m.TargetID, &m.HideNotifications, &expiresAt, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			m.ExpiresAt = &expiresAt.Time
		}
		mutes = append(mutes, &m)
	}
	return mutes, rows.Err()
}

// ExistsMute checks if a mute exists.
func (s *RelationshipsStore) ExistsMute(ctx context.Context, accountID, targetID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM mutes WHERE account_id = $1 AND target_id = $2 AND (expires_at IS NULL OR expires_at > $3))
	`, accountID, targetID, time.Now()).Scan(&exists)
	return exists, err
}

// GetRelationship gets the relationship between two accounts.
func (s *RelationshipsStore) GetRelationship(ctx context.Context, accountID, targetID string) (*relationships.Relationship, error) {
	rel := &relationships.Relationship{ID: targetID}

	// Check following
	var pending bool
	err := s.db.QueryRowContext(ctx, "SELECT pending FROM follows WHERE follower_id = $1 AND following_id = $2", accountID, targetID).Scan(&pending)
	if err == nil {
		if pending {
			rel.Requested = true
		} else {
			rel.Following = true
		}
	}

	// Check followed by
	err = s.db.QueryRowContext(ctx, "SELECT pending FROM follows WHERE follower_id = $1 AND following_id = $2", targetID, accountID).Scan(&pending)
	if err == nil && !pending {
		rel.FollowedBy = true
	}

	// Check blocking
	var exists bool
	s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM blocks WHERE account_id = $1 AND target_id = $2)", accountID, targetID).Scan(&exists)
	rel.Blocking = exists

	// Check blocked by
	s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM blocks WHERE account_id = $1 AND target_id = $2)", targetID, accountID).Scan(&exists)
	rel.BlockedBy = exists

	// Check muting
	s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM mutes WHERE account_id = $1 AND target_id = $2 AND (expires_at IS NULL OR expires_at > $3))", accountID, targetID, time.Now()).Scan(&exists)
	rel.Muting = exists

	return rel, nil
}
