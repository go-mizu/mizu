package notifications

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/ulid"
)

// Service implements the notifications API.
type Service struct {
	store    Store
	accounts accounts.API
	boards   boards.API
	threads  threads.API
	comments comments.API
}

// NewService creates a new notifications service.
func NewService(store Store, accounts accounts.API, boards boards.API, threads threads.API, comments comments.API) *Service {
	return &Service{
		store:    store,
		accounts: accounts,
		boards:   boards,
		threads:  threads,
		comments: comments,
	}
}

// Create creates a notification.
func (s *Service) Create(ctx context.Context, in CreateIn) (*Notification, error) {
	notification := &Notification{
		ID:        ulid.New(),
		AccountID: in.AccountID,
		Type:      in.Type,
		ActorID:   in.ActorID,
		BoardID:   in.BoardID,
		ThreadID:  in.ThreadID,
		CommentID: in.CommentID,
		Message:   in.Message,
		Read:      false,
		CreatedAt: time.Now(),
	}

	if err := s.store.Create(ctx, notification); err != nil {
		return nil, err
	}

	return notification, nil
}

// GetByID retrieves a notification by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Notification, error) {
	notification, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load relationships
	s.loadRelationships(ctx, notification)

	return notification, nil
}

// List lists notifications.
func (s *Service) List(ctx context.Context, accountID string, opts ListOpts) ([]*Notification, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 25
	}

	notifications, err := s.store.List(ctx, accountID, opts)
	if err != nil {
		return nil, err
	}

	// Load relationships
	for _, n := range notifications {
		s.loadRelationships(ctx, n)
	}

	return notifications, nil
}

// MarkRead marks notifications as read.
func (s *Service) MarkRead(ctx context.Context, accountID string, ids []string) error {
	// Verify ownership (simplified - in production, filter by account)
	return s.store.MarkRead(ctx, ids)
}

// MarkAllRead marks all notifications as read.
func (s *Service) MarkAllRead(ctx context.Context, accountID string) error {
	return s.store.MarkAllRead(ctx, accountID)
}

// Delete deletes a notification.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// DeleteOld deletes old notifications.
func (s *Service) DeleteOld(ctx context.Context, olderThan time.Duration) error {
	before := time.Now().Add(-olderThan)
	return s.store.DeleteBefore(ctx, before)
}

// GetUnreadCount gets the unread notification count.
func (s *Service) GetUnreadCount(ctx context.Context, accountID string) (int64, error) {
	return s.store.CountUnread(ctx, accountID)
}

// loadRelationships loads related entities.
func (s *Service) loadRelationships(ctx context.Context, n *Notification) {
	if n.ActorID != "" && s.accounts != nil {
		n.Actor, _ = s.accounts.GetByID(ctx, n.ActorID)
	}
	if n.BoardID != "" && s.boards != nil {
		n.Board, _ = s.boards.GetByID(ctx, n.BoardID)
	}
	if n.ThreadID != "" && s.threads != nil {
		n.Thread, _ = s.threads.GetByID(ctx, n.ThreadID)
	}
	if n.CommentID != "" && s.comments != nil {
		n.Comment, _ = s.comments.GetByID(ctx, n.CommentID)
	}
}

// NotifyReply creates a reply notification.
func (s *Service) NotifyReply(ctx context.Context, recipientID, actorID, threadID, commentID string) error {
	if recipientID == actorID {
		return nil // Don't notify self
	}
	_, err := s.Create(ctx, CreateIn{
		AccountID: recipientID,
		Type:      NotifyReply,
		ActorID:   actorID,
		ThreadID:  threadID,
		CommentID: commentID,
	})
	return err
}

// NotifyMention creates a mention notification.
func (s *Service) NotifyMention(ctx context.Context, recipientID, actorID, threadID, commentID string) error {
	if recipientID == actorID {
		return nil
	}
	_, err := s.Create(ctx, CreateIn{
		AccountID: recipientID,
		Type:      NotifyMention,
		ActorID:   actorID,
		ThreadID:  threadID,
		CommentID: commentID,
	})
	return err
}

// NotifyVote creates a vote notification (for milestones).
func (s *Service) NotifyVote(ctx context.Context, recipientID, actorID, threadID, commentID string, isThread bool) error {
	if recipientID == actorID {
		return nil
	}
	notifyType := NotifyCommentVote
	if isThread {
		notifyType = NotifyThreadVote
	}
	_, err := s.Create(ctx, CreateIn{
		AccountID: recipientID,
		Type:      notifyType,
		ActorID:   actorID,
		ThreadID:  threadID,
		CommentID: commentID,
	})
	return err
}
