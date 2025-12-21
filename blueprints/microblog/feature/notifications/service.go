// Package notifications provides notification delivery and management.
package notifications

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

// NotificationType is the type of notification.
type NotificationType string

const (
	TypeFollow  NotificationType = "follow"
	TypeLike    NotificationType = "like"
	TypeRepost  NotificationType = "repost"
	TypeMention NotificationType = "mention"
	TypeReply   NotificationType = "reply"
	TypePoll    NotificationType = "poll"
	TypeUpdate  NotificationType = "update"
)

// Notification represents a user notification.
type Notification struct {
	ID        string            `json:"id"`
	Type      NotificationType  `json:"type"`
	AccountID string            `json:"account_id"`
	ActorID   string            `json:"actor_id,omitempty"`
	PostID    string            `json:"post_id,omitempty"`
	Read      bool              `json:"read"`
	CreatedAt time.Time         `json:"created_at"`

	// Loaded relations
	Actor *accounts.Account `json:"actor,omitempty"`
}

// Service handles notification operations.
type Service struct {
	store    *duckdb.Store
	accounts *accounts.Service
}

// NewService creates a new notifications service.
func NewService(store *duckdb.Store, accounts *accounts.Service) *Service {
	return &Service{store: store, accounts: accounts}
}

// List returns notifications for an account.
func (s *Service) List(ctx context.Context, accountID string, types []NotificationType, limit int, maxID, sinceID string, excludeTypes []NotificationType) ([]*Notification, error) {
	query := `
		SELECT id, type, account_id, actor_id, post_id, read, created_at
		FROM notifications
		WHERE account_id = $1
	`

	args := []any{accountID}
	argIdx := 2

	// Filter by types
	if len(types) > 0 {
		query += " AND type IN ("
		for i, t := range types {
			if i > 0 {
				query += ","
			}
			query += fmt.Sprintf("$%d", argIdx)
			args = append(args, string(t))
			argIdx++
		}
		query += ")"
	}

	// Exclude types
	if len(excludeTypes) > 0 {
		query += " AND type NOT IN ("
		for i, t := range excludeTypes {
			if i > 0 {
				query += ","
			}
			query += fmt.Sprintf("$%d", argIdx)
			args = append(args, string(t))
			argIdx++
		}
		query += ")"
	}

	if maxID != "" {
		query += fmt.Sprintf(" AND id < $%d", argIdx)
		args = append(args, maxID)
		argIdx++
	}
	if sinceID != "" {
		query += fmt.Sprintf(" AND id > $%d", argIdx)
		args = append(args, sinceID)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.store.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("notifications: list: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		var n Notification
		var actorID, postID sql.NullString

		err := rows.Scan(&n.ID, &n.Type, &n.AccountID, &actorID, &postID, &n.Read, &n.CreatedAt)
		if err != nil {
			continue
		}

		n.ActorID = actorID.String
		n.PostID = postID.String

		// Load actor
		if n.ActorID != "" {
			n.Actor, _ = s.accounts.GetByID(ctx, n.ActorID)
		}

		notifications = append(notifications, &n)
	}

	return notifications, nil
}

// Get returns a single notification.
func (s *Service) Get(ctx context.Context, id, accountID string) (*Notification, error) {
	var n Notification
	var actorID, postID sql.NullString

	err := s.store.QueryRow(ctx, `
		SELECT id, type, account_id, actor_id, post_id, read, created_at
		FROM notifications
		WHERE id = $1 AND account_id = $2
	`, id, accountID).Scan(&n.ID, &n.Type, &n.AccountID, &actorID, &postID, &n.Read, &n.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("notifications: get: %w", err)
	}

	n.ActorID = actorID.String
	n.PostID = postID.String

	if n.ActorID != "" {
		n.Actor, _ = s.accounts.GetByID(ctx, n.ActorID)
	}

	return &n, nil
}

// MarkAsRead marks a notification as read.
func (s *Service) MarkAsRead(ctx context.Context, id, accountID string) error {
	_, err := s.store.Exec(ctx, "UPDATE notifications SET read = TRUE WHERE id = $1 AND account_id = $2", id, accountID)
	return err
}

// MarkAllAsRead marks all notifications as read for an account.
func (s *Service) MarkAllAsRead(ctx context.Context, accountID string) error {
	_, err := s.store.Exec(ctx, "UPDATE notifications SET read = TRUE WHERE account_id = $1 AND read = FALSE", accountID)
	return err
}

// Dismiss removes a notification.
func (s *Service) Dismiss(ctx context.Context, id, accountID string) error {
	_, err := s.store.Exec(ctx, "DELETE FROM notifications WHERE id = $1 AND account_id = $2", id, accountID)
	return err
}

// DismissAll removes all notifications for an account.
func (s *Service) DismissAll(ctx context.Context, accountID string) error {
	_, err := s.store.Exec(ctx, "DELETE FROM notifications WHERE account_id = $1", accountID)
	return err
}

// CountUnread returns the number of unread notifications.
func (s *Service) CountUnread(ctx context.Context, accountID string) (int, error) {
	var count int
	err := s.store.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE account_id = $1 AND read = FALSE", accountID).Scan(&count)
	return count, err
}

// CleanOld removes notifications older than the given duration.
func (s *Service) CleanOld(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.store.Exec(ctx, "DELETE FROM notifications WHERE created_at < $1", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
