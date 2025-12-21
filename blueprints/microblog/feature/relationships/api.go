// Package relationships provides follow, block, and mute functionality.
package relationships

import (
	"context"
	"time"
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

// API defines the relationships service contract.
type API interface {
	Get(ctx context.Context, accountID, targetID string) (*Relationship, error)
	Follow(ctx context.Context, accountID, targetID string) error
	Unfollow(ctx context.Context, accountID, targetID string) error
	Block(ctx context.Context, accountID, targetID string) error
	Unblock(ctx context.Context, accountID, targetID string) error
	Mute(ctx context.Context, accountID, targetID string, hideNotifications bool, duration *time.Duration) error
	Unmute(ctx context.Context, accountID, targetID string) error
	GetFollowers(ctx context.Context, targetID string, limit, offset int) ([]string, error)
	GetFollowing(ctx context.Context, targetID string, limit, offset int) ([]string, error)
	CountFollowers(ctx context.Context, accountID string) (int, error)
	CountFollowing(ctx context.Context, accountID string) (int, error)
	GetBlocked(ctx context.Context, accountID string, limit, offset int) ([]string, error)
	GetMuted(ctx context.Context, accountID string, limit, offset int) ([]string, error)
	IsBlocked(ctx context.Context, accountID, targetID string) (bool, error)
	IsMuted(ctx context.Context, accountID, targetID string) (bool, error)
}

// Store defines the data access contract for relationships.
type Store interface {
	// Follow operations
	Follow(ctx context.Context, followerID, followingID string) error
	Unfollow(ctx context.Context, followerID, followingID string) error
	IsFollowing(ctx context.Context, followerID, followingID string) (bool, error)
	GetFollowers(ctx context.Context, targetID string, limit, offset int) ([]string, error)
	GetFollowing(ctx context.Context, targetID string, limit, offset int) ([]string, error)
	CountFollowers(ctx context.Context, accountID string) (int, error)
	CountFollowing(ctx context.Context, accountID string) (int, error)

	// Block operations
	Block(ctx context.Context, accountID, targetID string) error
	Unblock(ctx context.Context, accountID, targetID string) error
	IsBlocking(ctx context.Context, accountID, targetID string) (bool, error)
	GetBlocked(ctx context.Context, accountID string, limit, offset int) ([]string, error)
	RemoveFollowsBetween(ctx context.Context, accountID, targetID string) error

	// Mute operations
	Mute(ctx context.Context, accountID, targetID string, hideNotifs bool, expiresAt *time.Time) error
	Unmute(ctx context.Context, accountID, targetID string) error
	IsMuting(ctx context.Context, accountID, targetID string) (muting bool, hideNotifs bool, err error)
	GetMuted(ctx context.Context, accountID string, limit, offset int) ([]string, error)

	// Notifications
	CreateNotification(ctx context.Context, accountID, actorID, notifType string) error
}
