package notifications

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/posts"
	"github.com/go-mizu/blueprints/social/pkg/ulid"
)

const defaultLimit = 20

// Service implements the notifications API.
type Service struct {
	store    Store
	accounts accounts.API
	posts    posts.API
}

// NewService creates a new notifications service.
func NewService(store Store, accountsSvc accounts.API, postsSvc posts.API) *Service {
	return &Service{
		store:    store,
		accounts: accountsSvc,
		posts:    postsSvc,
	}
}

// Create creates a new notification.
func (s *Service) Create(ctx context.Context, n *Notification) error {
	// Don't notify yourself
	if n.AccountID == n.ActorID {
		return nil
	}

	// Check for duplicate
	exists, err := s.store.Exists(ctx, n.AccountID, n.Type, n.ActorID, n.PostID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if n.ID == "" {
		n.ID = ulid.New()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}

	return s.store.Insert(ctx, n)
}

// List returns notifications for an account.
func (s *Service) List(ctx context.Context, accountID string, opts ListOpts) ([]*Notification, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	notifications, err := s.store.List(ctx, accountID, limit, opts.MaxID, opts.SinceID, opts.Types, opts.ExcludeTypes)
	if err != nil {
		return nil, err
	}

	// Populate actors and posts
	for _, n := range notifications {
		if n.ActorID != "" && s.accounts != nil {
			actor, err := s.accounts.GetByID(ctx, n.ActorID)
			if err == nil {
				n.Actor = actor
			}
		}
		if n.PostID != "" && s.posts != nil {
			post, err := s.posts.GetByID(ctx, n.PostID)
			if err == nil {
				n.Post = post
			}
		}
	}

	return notifications, nil
}

// Get retrieves a notification by ID.
func (s *Service) Get(ctx context.Context, id string) (*Notification, error) {
	return s.store.GetByID(ctx, id)
}

// MarkRead marks a notification as read.
func (s *Service) MarkRead(ctx context.Context, accountID, id string) error {
	n, err := s.store.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if n.AccountID != accountID {
		return ErrNotFound
	}
	return s.store.MarkRead(ctx, id)
}

// MarkAllRead marks all notifications as read.
func (s *Service) MarkAllRead(ctx context.Context, accountID string) error {
	return s.store.MarkAllRead(ctx, accountID)
}

// Dismiss deletes a notification.
func (s *Service) Dismiss(ctx context.Context, accountID, id string) error {
	n, err := s.store.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if n.AccountID != accountID {
		return ErrNotFound
	}
	return s.store.Delete(ctx, id)
}

// Clear deletes all notifications.
func (s *Service) Clear(ctx context.Context, accountID string) error {
	return s.store.DeleteAll(ctx, accountID)
}

// UnreadCount returns the unread notification count.
func (s *Service) UnreadCount(ctx context.Context, accountID string) (int, error) {
	return s.store.UnreadCount(ctx, accountID)
}

// NotifyFollow creates a follow notification.
func (s *Service) NotifyFollow(ctx context.Context, followerID, followedID string) error {
	return s.Create(ctx, &Notification{
		AccountID: followedID,
		Type:      TypeFollow,
		ActorID:   followerID,
	})
}

// NotifyFollowRequest creates a follow request notification.
func (s *Service) NotifyFollowRequest(ctx context.Context, followerID, targetID string) error {
	return s.Create(ctx, &Notification{
		AccountID: targetID,
		Type:      TypeFollowRequest,
		ActorID:   followerID,
	})
}

// NotifyMention creates a mention notification.
func (s *Service) NotifyMention(ctx context.Context, authorID, mentionedID, postID string) error {
	return s.Create(ctx, &Notification{
		AccountID: mentionedID,
		Type:      TypeMention,
		ActorID:   authorID,
		PostID:    postID,
	})
}

// NotifyReply creates a reply notification.
func (s *Service) NotifyReply(ctx context.Context, replierID, parentAuthorID, postID string) error {
	return s.Create(ctx, &Notification{
		AccountID: parentAuthorID,
		Type:      TypeReply,
		ActorID:   replierID,
		PostID:    postID,
	})
}

// NotifyLike creates a like notification.
func (s *Service) NotifyLike(ctx context.Context, likerID, postAuthorID, postID string) error {
	return s.Create(ctx, &Notification{
		AccountID: postAuthorID,
		Type:      TypeLike,
		ActorID:   likerID,
		PostID:    postID,
	})
}

// NotifyRepost creates a repost notification.
func (s *Service) NotifyRepost(ctx context.Context, reposterID, postAuthorID, postID string) error {
	return s.Create(ctx, &Notification{
		AccountID: postAuthorID,
		Type:      TypeRepost,
		ActorID:   reposterID,
		PostID:    postID,
	})
}

// NotifyQuote creates a quote notification.
func (s *Service) NotifyQuote(ctx context.Context, quoterID, quotedAuthorID, postID string) error {
	return s.Create(ctx, &Notification{
		AccountID: quotedAuthorID,
		Type:      TypeQuote,
		ActorID:   quoterID,
		PostID:    postID,
	})
}
