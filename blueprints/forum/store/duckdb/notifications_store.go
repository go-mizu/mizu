package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/notifications"
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
		INSERT INTO notifications (id, account_id, type, actor_id, board_id, thread_id, comment_id, message, read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, notification.ID, notification.AccountID, notification.Type, notification.ActorID,
		notification.BoardID, notification.ThreadID, notification.CommentID,
		notification.Message, notification.Read, notification.CreatedAt)
	return err
}

// GetByID retrieves a notification by ID.
func (s *NotificationsStore) GetByID(ctx context.Context, id string) (*notifications.Notification, error) {
	notification := &notifications.Notification{}
	var actorID, boardID, threadID, commentID, message sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, type, actor_id, board_id, thread_id, comment_id, message, read, created_at
		FROM notifications WHERE id = $1
	`, id).Scan(
		&notification.ID, &notification.AccountID, &notification.Type, &actorID,
		&boardID, &threadID, &commentID, &message, &notification.Read, &notification.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, notifications.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if actorID.Valid {
		notification.ActorID = actorID.String
	}
	if boardID.Valid {
		notification.BoardID = boardID.String
	}
	if threadID.Valid {
		notification.ThreadID = threadID.String
	}
	if commentID.Valid {
		notification.CommentID = commentID.String
	}
	if message.Valid {
		notification.Message = message.String
	}

	return notification, nil
}

// List lists notifications.
func (s *NotificationsStore) List(ctx context.Context, accountID string, opts notifications.ListOpts) ([]*notifications.Notification, error) {
	query := `
		SELECT id, account_id, type, actor_id, board_id, thread_id, comment_id, message, read, created_at
		FROM notifications WHERE account_id = $1
	`
	if opts.Unread {
		query += " AND NOT read"
	}
	query += " ORDER BY created_at DESC LIMIT $2"

	rows, err := s.db.QueryContext(ctx, query, accountID, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*notifications.Notification
	for rows.Next() {
		notification := &notifications.Notification{}
		var actorID, boardID, threadID, commentID, message sql.NullString
		err := rows.Scan(
			&notification.ID, &notification.AccountID, &notification.Type, &actorID,
			&boardID, &threadID, &commentID, &message, &notification.Read, &notification.CreatedAt)
		if err != nil {
			return nil, err
		}
		if actorID.Valid {
			notification.ActorID = actorID.String
		}
		if boardID.Valid {
			notification.BoardID = boardID.String
		}
		if threadID.Valid {
			notification.ThreadID = threadID.String
		}
		if commentID.Valid {
			notification.CommentID = commentID.String
		}
		if message.Valid {
			notification.Message = message.String
		}
		result = append(result, notification)
	}
	return result, rows.Err()
}

// MarkRead marks notifications as read.
func (s *NotificationsStore) MarkRead(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	query := "UPDATE notifications SET read = TRUE WHERE id IN ("
	args := make([]any, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += ", "
		}
		query += "?"
		args[i] = id
	}
	query += ")"

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// MarkAllRead marks all notifications as read.
func (s *NotificationsStore) MarkAllRead(ctx context.Context, accountID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET read = TRUE WHERE account_id = $1
	`, accountID)
	return err
}

// Delete deletes a notification.
func (s *NotificationsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM notifications WHERE id = $1`, id)
	return err
}

// DeleteBefore deletes notifications before a time.
func (s *NotificationsStore) DeleteBefore(ctx context.Context, before time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM notifications WHERE created_at < $1`, before)
	return err
}

// CountUnread counts unread notifications.
func (s *NotificationsStore) CountUnread(ctx context.Context, accountID string) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications WHERE account_id = $1 AND NOT read
	`, accountID).Scan(&count)
	return count, err
}
