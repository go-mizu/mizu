package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/notifications"
)

// NotificationsStore implements notifications.Store
type NotificationsStore struct {
	db *sql.DB
}

// NewNotificationsStore creates a new notifications store
func NewNotificationsStore(db *sql.DB) *NotificationsStore {
	return &NotificationsStore{db: db}
}

// Create creates a new notification
func (s *NotificationsStore) Create(ctx context.Context, n *notifications.Notification) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, repo_id, type, actor_id, target_type, target_id, title, reason, unread, created_at, updated_at, last_read_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, n.ID, n.UserID, nullString(n.RepoID), n.Type, nullString(n.ActorID), n.TargetType, n.TargetID, n.Title, n.Reason, n.Unread, n.CreatedAt, n.UpdatedAt, n.LastReadAt)
	return err
}

// GetByID retrieves a notification by ID
func (s *NotificationsStore) GetByID(ctx context.Context, id string) (*notifications.Notification, error) {
	n := &notifications.Notification{}
	var repoID, actorID sql.NullString
	var lastReadAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, repo_id, type, actor_id, target_type, target_id, title, reason, unread, created_at, updated_at, last_read_at
		FROM notifications WHERE id = $1
	`, id).Scan(&n.ID, &n.UserID, &repoID, &n.Type, &actorID, &n.TargetType, &n.TargetID, &n.Title, &n.Reason, &n.Unread, &n.CreatedAt, &n.UpdatedAt, &lastReadAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if repoID.Valid {
		n.RepoID = repoID.String
	}
	if actorID.Valid {
		n.ActorID = actorID.String
	}
	if lastReadAt.Valid {
		n.LastReadAt = &lastReadAt.Time
	}
	return n, nil
}

// List lists notifications for a user
func (s *NotificationsStore) List(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]*notifications.Notification, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_id = $1`
	args := []interface{}{userID}
	if unreadOnly {
		countQuery += ` AND unread = TRUE`
	}
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get notifications
	query := `
		SELECT id, user_id, repo_id, type, actor_id, target_type, target_id, title, reason, unread, created_at, updated_at, last_read_at
		FROM notifications WHERE user_id = $1`
	if unreadOnly {
		query += ` AND unread = TRUE`
	}
	query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*notifications.Notification
	for rows.Next() {
		n := &notifications.Notification{}
		var repoID, actorID sql.NullString
		var lastReadAt sql.NullTime
		if err := rows.Scan(&n.ID, &n.UserID, &repoID, &n.Type, &actorID, &n.TargetType, &n.TargetID, &n.Title, &n.Reason, &n.Unread, &n.CreatedAt, &n.UpdatedAt, &lastReadAt); err != nil {
			return nil, 0, err
		}
		if repoID.Valid {
			n.RepoID = repoID.String
		}
		if actorID.Valid {
			n.ActorID = actorID.String
		}
		if lastReadAt.Valid {
			n.LastReadAt = &lastReadAt.Time
		}
		list = append(list, n)
	}
	return list, total, rows.Err()
}

// MarkAsRead marks a notification as read
func (s *NotificationsStore) MarkAsRead(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET unread = FALSE, last_read_at = $2, updated_at = $2
		WHERE id = $1
	`, id, now)
	return err
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationsStore) MarkAllAsRead(ctx context.Context, userID string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET unread = FALSE, last_read_at = $2, updated_at = $2
		WHERE user_id = $1 AND unread = TRUE
	`, userID, now)
	return err
}

// MarkRepoAsRead marks all notifications for a repository as read
func (s *NotificationsStore) MarkRepoAsRead(ctx context.Context, userID, repoID string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET unread = FALSE, last_read_at = $3, updated_at = $3
		WHERE user_id = $1 AND repo_id = $2 AND unread = TRUE
	`, userID, repoID, now)
	return err
}

// Delete deletes a notification
func (s *NotificationsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM notifications WHERE id = $1`, id)
	return err
}

// CountUnread counts unread notifications for a user
func (s *NotificationsStore) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND unread = TRUE
	`, userID).Scan(&count)
	return count, err
}
