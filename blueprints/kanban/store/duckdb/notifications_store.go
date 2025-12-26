package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/notifications"
)

// NotificationsStore handles notification data access.
type NotificationsStore struct {
	db *sql.DB
}

// NewNotificationsStore creates a new notifications store.
func NewNotificationsStore(db *sql.DB) *NotificationsStore {
	return &NotificationsStore{db: db}
}

func (s *NotificationsStore) Create(ctx context.Context, n *notifications.Notification) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, type, issue_id, actor_id, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, n.ID, n.UserID, n.Type, n.IssueID, n.ActorID, n.Content, n.CreatedAt)
	return err
}

func (s *NotificationsStore) GetByID(ctx context.Context, id string) (*notifications.Notification, error) {
	n := &notifications.Notification{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, type, issue_id, actor_id, content, read_at, created_at
		FROM notifications WHERE id = $1
	`, id).Scan(&n.ID, &n.UserID, &n.Type, &n.IssueID, &n.ActorID, &n.Content, &n.ReadAt, &n.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return n, err
}

func (s *NotificationsStore) ListByUser(ctx context.Context, userID string, limit int) ([]*notifications.Notification, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, type, issue_id, actor_id, content, read_at, created_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*notifications.Notification
	for rows.Next() {
		n := &notifications.Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.IssueID, &n.ActorID, &n.Content, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, rows.Err()
}

func (s *NotificationsStore) ListUnread(ctx context.Context, userID string) ([]*notifications.Notification, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, type, issue_id, actor_id, content, read_at, created_at
		FROM notifications WHERE user_id = $1 AND read_at IS NULL
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*notifications.Notification
	for rows.Next() {
		n := &notifications.Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.IssueID, &n.ActorID, &n.Content, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, rows.Err()
}

func (s *NotificationsStore) MarkRead(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET read_at = $2 WHERE id = $1
	`, id, time.Now())
	return err
}

func (s *NotificationsStore) MarkAllRead(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET read_at = $2 WHERE user_id = $1 AND read_at IS NULL
	`, userID, time.Now())
	return err
}

func (s *NotificationsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM notifications WHERE id = $1`, id)
	return err
}

func (s *NotificationsStore) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL
	`, userID).Scan(&count)
	return count, err
}
