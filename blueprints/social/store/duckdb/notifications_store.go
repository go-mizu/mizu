package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/social/feature/notifications"
)

// NotificationsStore implements notifications.Store.
type NotificationsStore struct {
	db *sql.DB
}

// NewNotificationsStore creates a new notifications store.
func NewNotificationsStore(db *sql.DB) *NotificationsStore {
	return &NotificationsStore{db: db}
}

// Insert inserts a notification.
func (s *NotificationsStore) Insert(ctx context.Context, n *notifications.Notification) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (id, account_id, type, actor_id, post_id, read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, n.ID, n.AccountID, n.Type, nullString(n.ActorID), nullString(n.PostID), n.Read, n.CreatedAt)
	return err
}

// GetByID retrieves a notification by ID.
func (s *NotificationsStore) GetByID(ctx context.Context, id string) (*notifications.Notification, error) {
	var n notifications.Notification
	var actorID, postID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, type, actor_id, post_id, read, created_at
		FROM notifications WHERE id = $1
	`, id).Scan(&n.ID, &n.AccountID, &n.Type, &actorID, &postID, &n.Read, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	n.ActorID = actorID.String
	n.PostID = postID.String
	return &n, nil
}

// List lists notifications for an account.
func (s *NotificationsStore) List(ctx context.Context, accountID string, limit int, maxID, sinceID string, types, excludeTypes []string) ([]*notifications.Notification, error) {
	query := `
		SELECT id, account_id, type, actor_id, post_id, read, created_at
		FROM notifications WHERE account_id = $1
	`
	args := []interface{}{accountID}
	argNum := 2

	if maxID != "" {
		query += fmt.Sprintf(" AND id < $%d", argNum)
		args = append(args, maxID)
		argNum++
	}
	if sinceID != "" {
		query += fmt.Sprintf(" AND id > $%d", argNum)
		args = append(args, sinceID)
		argNum++
	}
	if len(types) > 0 {
		placeholders := make([]string, len(types))
		for i, t := range types {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, t)
			argNum++
		}
		query += fmt.Sprintf(" AND type IN (%s)", strings.Join(placeholders, ", "))
	}
	if len(excludeTypes) > 0 {
		placeholders := make([]string, len(excludeTypes))
		for i, t := range excludeTypes {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, t)
			argNum++
		}
		query += fmt.Sprintf(" AND type NOT IN (%s)", strings.Join(placeholders, ", "))
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argNum)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ns []*notifications.Notification
	for rows.Next() {
		var n notifications.Notification
		var actorID, postID sql.NullString
		err := rows.Scan(&n.ID, &n.AccountID, &n.Type, &actorID, &postID, &n.Read, &n.CreatedAt)
		if err != nil {
			return nil, err
		}
		n.ActorID = actorID.String
		n.PostID = postID.String
		ns = append(ns, &n)
	}
	return ns, rows.Err()
}

// MarkRead marks a notification as read.
func (s *NotificationsStore) MarkRead(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE notifications SET read = TRUE WHERE id = $1", id)
	return err
}

// MarkAllRead marks all notifications as read.
func (s *NotificationsStore) MarkAllRead(ctx context.Context, accountID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE notifications SET read = TRUE WHERE account_id = $1", accountID)
	return err
}

// Delete deletes a notification.
func (s *NotificationsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM notifications WHERE id = $1", id)
	return err
}

// DeleteAll deletes all notifications for an account.
func (s *NotificationsStore) DeleteAll(ctx context.Context, accountID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM notifications WHERE account_id = $1", accountID)
	return err
}

// UnreadCount returns the unread notification count.
func (s *NotificationsStore) UnreadCount(ctx context.Context, accountID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM notifications WHERE account_id = $1 AND read = FALSE", accountID).Scan(&count)
	return count, err
}

// Exists checks if a notification already exists.
func (s *NotificationsStore) Exists(ctx context.Context, accountID, notifType, actorID, postID string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM notifications WHERE account_id = $1 AND type = $2"
	args := []interface{}{accountID, notifType}
	argNum := 3

	if actorID != "" {
		query += fmt.Sprintf(" AND actor_id = $%d", argNum)
		args = append(args, actorID)
		argNum++
	}
	if postID != "" {
		query += fmt.Sprintf(" AND post_id = $%d", argNum)
		args = append(args, postID)
	}
	query += ")"

	var exists bool
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&exists)
	return exists, err
}
