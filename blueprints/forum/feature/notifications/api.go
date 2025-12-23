package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
)

// Errors
var (
	ErrNotFound = errors.New("notification not found")
)

// NotificationType defines the type of notification.
type NotificationType string

const (
	NotifyReply       NotificationType = "reply"
	NotifyMention     NotificationType = "mention"
	NotifyThreadVote  NotificationType = "thread_vote"
	NotifyCommentVote NotificationType = "comment_vote"
	NotifyFollow      NotificationType = "follow"
	NotifyMod         NotificationType = "mod"
	NotifyBoardInvite NotificationType = "board_invite"
)

// Notification represents a user notification.
type Notification struct {
	ID        string           `json:"id"`
	AccountID string           `json:"account_id"`
	Type      NotificationType `json:"type"`
	ActorID   string           `json:"actor_id,omitempty"`
	BoardID   string           `json:"board_id,omitempty"`
	ThreadID  string           `json:"thread_id,omitempty"`
	CommentID string           `json:"comment_id,omitempty"`
	Message   string           `json:"message,omitempty"`
	Read      bool             `json:"read"`
	CreatedAt time.Time        `json:"created_at"`

	// Relationships
	Actor   *accounts.Account  `json:"actor,omitempty"`
	Board   *boards.Board      `json:"board,omitempty"`
	Thread  *threads.Thread    `json:"thread,omitempty"`
	Comment *comments.Comment  `json:"comment,omitempty"`
}

// CreateIn contains input for creating a notification.
type CreateIn struct {
	AccountID string
	Type      NotificationType
	ActorID   string
	BoardID   string
	ThreadID  string
	CommentID string
	Message   string
}

// ListOpts contains options for listing notifications.
type ListOpts struct {
	Limit    int
	Cursor   string
	Unread   bool
}

// API defines the notifications service interface.
type API interface {
	Create(ctx context.Context, in CreateIn) (*Notification, error)
	GetByID(ctx context.Context, id string) (*Notification, error)
	List(ctx context.Context, accountID string, opts ListOpts) ([]*Notification, error)
	MarkRead(ctx context.Context, accountID string, ids []string) error
	MarkAllRead(ctx context.Context, accountID string) error
	Delete(ctx context.Context, id string) error
	DeleteOld(ctx context.Context, olderThan time.Duration) error
	GetUnreadCount(ctx context.Context, accountID string) (int64, error)
}

// Store defines the data storage interface for notifications.
type Store interface {
	Create(ctx context.Context, notification *Notification) error
	GetByID(ctx context.Context, id string) (*Notification, error)
	List(ctx context.Context, accountID string, opts ListOpts) ([]*Notification, error)
	MarkRead(ctx context.Context, ids []string) error
	MarkAllRead(ctx context.Context, accountID string) error
	Delete(ctx context.Context, id string) error
	DeleteBefore(ctx context.Context, before time.Time) error
	CountUnread(ctx context.Context, accountID string) (int64, error)
}
