// Package notifications provides notification functionality.
package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/posts"
)

// Notification types
const (
	TypeFollow        = "follow"
	TypeFollowRequest = "follow_request"
	TypeMention       = "mention"
	TypeReply         = "reply"
	TypeLike          = "like"
	TypeRepost        = "repost"
	TypeQuote         = "quote"
)

// Errors
var (
	ErrNotFound = errors.New("notification not found")
)

// Notification represents a notification.
type Notification struct {
	ID        string            `json:"id"`
	AccountID string            `json:"account_id"`
	Type      string            `json:"type"`
	ActorID   string            `json:"actor_id,omitempty"`
	PostID    string            `json:"post_id,omitempty"`
	Read      bool              `json:"read"`
	CreatedAt time.Time         `json:"created_at"`

	// Enriched fields
	Actor *accounts.Account `json:"actor,omitempty"`
	Post  *posts.Post       `json:"post,omitempty"`
}

// ListOpts specifies options for listing notifications.
type ListOpts struct {
	Limit       int
	MaxID       string
	SinceID     string
	Types       []string
	ExcludeTypes []string
}

// API defines the notifications service contract.
type API interface {
	Create(ctx context.Context, n *Notification) error
	List(ctx context.Context, accountID string, opts ListOpts) ([]*Notification, error)
	Get(ctx context.Context, id string) (*Notification, error)
	MarkRead(ctx context.Context, accountID, id string) error
	MarkAllRead(ctx context.Context, accountID string) error
	Dismiss(ctx context.Context, accountID, id string) error
	Clear(ctx context.Context, accountID string) error
	UnreadCount(ctx context.Context, accountID string) (int, error)

	// Helper methods to create specific notification types
	NotifyFollow(ctx context.Context, followerID, followedID string) error
	NotifyFollowRequest(ctx context.Context, followerID, targetID string) error
	NotifyMention(ctx context.Context, authorID, mentionedID, postID string) error
	NotifyReply(ctx context.Context, replierID, parentAuthorID, postID string) error
	NotifyLike(ctx context.Context, likerID, postAuthorID, postID string) error
	NotifyRepost(ctx context.Context, reposterID, postAuthorID, postID string) error
	NotifyQuote(ctx context.Context, quoterID, quotedAuthorID, postID string) error
}

// Store defines the data access contract for notifications.
type Store interface {
	Insert(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, id string) (*Notification, error)
	List(ctx context.Context, accountID string, limit int, maxID, sinceID string, types, excludeTypes []string) ([]*Notification, error)
	MarkRead(ctx context.Context, id string) error
	MarkAllRead(ctx context.Context, accountID string) error
	Delete(ctx context.Context, id string) error
	DeleteAll(ctx context.Context, accountID string) error
	UnreadCount(ctx context.Context, accountID string) (int, error)

	// Duplicate check
	Exists(ctx context.Context, accountID, notifType, actorID, postID string) (bool, error)
}
