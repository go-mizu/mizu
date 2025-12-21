// Package interactions provides like, repost, and bookmark functionality.
package interactions

import (
	"context"
)

// API defines the interactions service contract.
type API interface {
	Like(ctx context.Context, accountID, postID string) error
	Unlike(ctx context.Context, accountID, postID string) error
	Repost(ctx context.Context, accountID, postID string) error
	Unrepost(ctx context.Context, accountID, postID string) error
	Bookmark(ctx context.Context, accountID, postID string) error
	Unbookmark(ctx context.Context, accountID, postID string) error
	VotePoll(ctx context.Context, accountID, pollID string, choices []int) error
	GetLikedBy(ctx context.Context, postID string, limit, offset int) ([]string, error)
	GetRepostedBy(ctx context.Context, postID string, limit, offset int) ([]string, error)
}

// Store defines the data access contract for interactions.
type Store interface {
	// Like operations
	Like(ctx context.Context, accountID, postID string) (created bool, err error)
	Unlike(ctx context.Context, accountID, postID string) (deleted bool, err error)
	CheckLiked(ctx context.Context, accountID, postID string) (bool, error)
	GetLikedBy(ctx context.Context, postID string, limit, offset int) ([]string, error)

	// Repost operations
	Repost(ctx context.Context, accountID, postID string) (created bool, err error)
	Unrepost(ctx context.Context, accountID, postID string) (deleted bool, err error)
	CheckReposted(ctx context.Context, accountID, postID string) (bool, error)
	GetRepostedBy(ctx context.Context, postID string, limit, offset int) ([]string, error)

	// Bookmark operations
	Bookmark(ctx context.Context, accountID, postID string) error
	Unbookmark(ctx context.Context, accountID, postID string) error

	// Poll voting
	VotePoll(ctx context.Context, accountID, pollID string, choices []int) error
	GetPollInfo(ctx context.Context, pollID string) (expiresAt *string, multiple bool, err error)
	CheckVoted(ctx context.Context, pollID, accountID string) (bool, error)

	// Post owner for notifications
	GetPostOwner(ctx context.Context, postID string) (string, error)

	// Counters
	IncrementLikes(ctx context.Context, postID string) error
	DecrementLikes(ctx context.Context, postID string) error
	IncrementReposts(ctx context.Context, postID string) error
	DecrementReposts(ctx context.Context, postID string) error

	// Notifications
	CreateNotification(ctx context.Context, accountID, actorID, notifType, postID string) error
}
