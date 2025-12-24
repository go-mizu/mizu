// Package relationships provides social graph management.
package relationships

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrAlreadyFollowing      = errors.New("already following")
	ErrNotFollowing          = errors.New("not following")
	ErrCannotFollowSelf      = errors.New("cannot follow yourself")
	ErrBlocked               = errors.New("user is blocked")
	ErrAlreadyBlocked        = errors.New("already blocked")
	ErrNotBlocked            = errors.New("not blocked")
	ErrCannotBlockSelf       = errors.New("cannot block yourself")
	ErrAlreadyMuted          = errors.New("already muted")
	ErrNotMuted              = errors.New("not muted")
	ErrCannotMuteSelf        = errors.New("cannot mute yourself")
	ErrFollowRequestNotFound = errors.New("follow request not found")
)

// Relationship represents the relationship between two accounts.
type Relationship struct {
	ID         string `json:"id"`
	Following  bool   `json:"following"`
	FollowedBy bool   `json:"followed_by"`
	Requested  bool   `json:"requested"`
	Blocking   bool   `json:"blocking"`
	BlockedBy  bool   `json:"blocked_by"`
	Muting     bool   `json:"muting"`
}

// Follow represents a follow relationship.
type Follow struct {
	ID          string    `json:"id"`
	FollowerID  string    `json:"follower_id"`
	FollowingID string    `json:"following_id"`
	Pending     bool      `json:"pending"`
	CreatedAt   time.Time `json:"created_at"`
}

// Block represents a block relationship.
type Block struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	TargetID  string    `json:"target_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Mute represents a mute relationship.
type Mute struct {
	ID                string     `json:"id"`
	AccountID         string     `json:"account_id"`
	TargetID          string     `json:"target_id"`
	HideNotifications bool       `json:"hide_notifications"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// MuteIn contains input for muting a user.
type MuteIn struct {
	HideNotifications bool   `json:"hide_notifications"`
	Duration          string `json:"duration,omitempty"` // "1h", "1d", "7d", or empty for forever
}

// FollowersOpts specifies options for listing followers.
type FollowersOpts struct {
	AccountID string
	Limit     int
	Offset    int
}

// API defines the relationships service contract.
type API interface {
	Follow(ctx context.Context, followerID, followingID string) (*Relationship, error)
	Unfollow(ctx context.Context, followerID, followingID string) (*Relationship, error)
	AcceptFollow(ctx context.Context, ownerID, followerID string) error
	RejectFollow(ctx context.Context, ownerID, followerID string) error
	GetFollowers(ctx context.Context, opts FollowersOpts) ([]*Follow, error)
	GetFollowing(ctx context.Context, opts FollowersOpts) ([]*Follow, error)
	GetPendingFollowers(ctx context.Context, accountID string, limit, offset int) ([]*Follow, error)

	Block(ctx context.Context, accountID, targetID string) (*Relationship, error)
	Unblock(ctx context.Context, accountID, targetID string) (*Relationship, error)
	GetBlocked(ctx context.Context, accountID string, limit, offset int) ([]*Block, error)

	Mute(ctx context.Context, accountID, targetID string, in *MuteIn) (*Relationship, error)
	Unmute(ctx context.Context, accountID, targetID string) (*Relationship, error)
	GetMuted(ctx context.Context, accountID string, limit, offset int) ([]*Mute, error)

	GetRelationship(ctx context.Context, accountID, targetID string) (*Relationship, error)
	GetRelationships(ctx context.Context, accountID string, targetIDs []string) ([]*Relationship, error)

	IsFollowing(ctx context.Context, followerID, followingID string) (bool, error)
	IsBlocked(ctx context.Context, accountID, targetID string) (bool, error)
	IsMuted(ctx context.Context, accountID, targetID string) (bool, error)
}

// Store defines the data access contract for relationships.
type Store interface {
	// Follows
	InsertFollow(ctx context.Context, f *Follow) error
	DeleteFollow(ctx context.Context, followerID, followingID string) error
	GetFollow(ctx context.Context, followerID, followingID string) (*Follow, error)
	SetFollowPending(ctx context.Context, followerID, followingID string, pending bool) error
	GetFollowers(ctx context.Context, accountID string, limit, offset int) ([]*Follow, error)
	GetFollowing(ctx context.Context, accountID string, limit, offset int) ([]*Follow, error)
	GetPendingFollowers(ctx context.Context, accountID string, limit, offset int) ([]*Follow, error)
	ExistsFollow(ctx context.Context, followerID, followingID string) (bool, error)

	// Blocks
	InsertBlock(ctx context.Context, b *Block) error
	DeleteBlock(ctx context.Context, accountID, targetID string) error
	GetBlock(ctx context.Context, accountID, targetID string) (*Block, error)
	GetBlocks(ctx context.Context, accountID string, limit, offset int) ([]*Block, error)
	ExistsBlock(ctx context.Context, accountID, targetID string) (bool, error)
	ExistsBlockEither(ctx context.Context, accountID, targetID string) (bool, error)

	// Mutes
	InsertMute(ctx context.Context, m *Mute) error
	DeleteMute(ctx context.Context, accountID, targetID string) error
	GetMute(ctx context.Context, accountID, targetID string) (*Mute, error)
	GetMutes(ctx context.Context, accountID string, limit, offset int) ([]*Mute, error)
	ExistsMute(ctx context.Context, accountID, targetID string) (bool, error)

	// Relationship queries
	GetRelationship(ctx context.Context, accountID, targetID string) (*Relationship, error)
}
