package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/qa/feature/notifications"
)

// NotificationsStore implements notifications.Store.
type NotificationsStore struct {
	db *sql.DB
}

// NewNotificationsStore creates a new notifications store.
func NewNotificationsStore(db *sql.DB) *NotificationsStore {
	return &NotificationsStore{db: db}
}

// Create creates a notification.
func (s *NotificationsStore) Create(ctx context.Context, notification *notifications.Notification) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (id, account_id, type, title, body, url, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, notification.ID, notification.AccountID, notification.Type,
		notification.Title, notification.Body, notification.URL, notification.IsRead, notification.CreatedAt)
	return err
}

// ListByAccount lists notifications.
func (s *NotificationsStore) ListByAccount(ctx context.Context, accountID string, limit int) ([]*notifications.Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, type, title, body, url, is_read, created_at
		FROM notifications
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*notifications.Notification
	for rows.Next() {
		n := &notifications.Notification{}
		if err := rows.Scan(&n.ID, &n.AccountID, &n.Type, &n.Title, &n.Body, &n.URL, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

// GetUnreadCount counts unread notifications.
func (s *NotificationsStore) GetUnreadCount(ctx context.Context, accountID string) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications WHERE account_id = $1 AND is_read = FALSE
	`, accountID).Scan(&count)
	return count, err
}

// MarkRead marks a notification as read.
func (s *NotificationsStore) MarkRead(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE notifications SET is_read = TRUE WHERE id = $1`, id)
	return err
}

// MarkAllRead marks all notifications as read.
func (s *NotificationsStore) MarkAllRead(ctx context.Context, accountID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE notifications SET is_read = TRUE WHERE account_id = $1`, accountID)
	return err
}
