package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/microblog/feature/notifications"
)

// NotificationsStore implements notifications.Store using DuckDB.
type NotificationsStore struct {
	db *sql.DB
}

// NewNotificationsStore creates a new notifications store.
func NewNotificationsStore(db *sql.DB) *NotificationsStore {
	return &NotificationsStore{db: db}
}

func (s *NotificationsStore) List(ctx context.Context, accountID string, types, excludeTypes []notifications.NotificationType, limit int, maxID, sinceID string) ([]*notifications.Notification, error) {
	query := `
		SELECT id, type, account_id, actor_id, post_id, read, created_at
		FROM notifications
		WHERE account_id = $1
	`

	args := []any{accountID}
	argIdx := 2

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

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*notifications.Notification
	for rows.Next() {
		var n notifications.Notification
		var actorID, postID sql.NullString

		err := rows.Scan(&n.ID, &n.Type, &n.AccountID, &actorID, &postID, &n.Read, &n.CreatedAt)
		if err != nil {
			continue
		}

		n.ActorID = actorID.String
		n.PostID = postID.String
		result = append(result, &n)
	}

	return result, nil
}

func (s *NotificationsStore) Get(ctx context.Context, id, accountID string) (*notifications.Notification, error) {
	var n notifications.Notification
	var actorID, postID sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, type, account_id, actor_id, post_id, read, created_at
		FROM notifications
		WHERE id = $1 AND account_id = $2
	`, id, accountID).Scan(&n.ID, &n.Type, &n.AccountID, &actorID, &postID, &n.Read, &n.CreatedAt)
	if err != nil {
		return nil, err
	}

	n.ActorID = actorID.String
	n.PostID = postID.String
	return &n, nil
}

func (s *NotificationsStore) MarkAsRead(ctx context.Context, id, accountID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE notifications SET read = TRUE WHERE id = $1 AND account_id = $2", id, accountID)
	return err
}

func (s *NotificationsStore) MarkAllAsRead(ctx context.Context, accountID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE notifications SET read = TRUE WHERE account_id = $1 AND read = FALSE", accountID)
	return err
}

func (s *NotificationsStore) Dismiss(ctx context.Context, id, accountID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM notifications WHERE id = $1 AND account_id = $2", id, accountID)
	return err
}

func (s *NotificationsStore) DismissAll(ctx context.Context, accountID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM notifications WHERE account_id = $1", accountID)
	return err
}

func (s *NotificationsStore) CountUnread(ctx context.Context, accountID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM notifications WHERE account_id = $1 AND read = FALSE", accountID).Scan(&count)
	return count, err
}

func (s *NotificationsStore) CleanOld(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.db.ExecContext(ctx, "DELETE FROM notifications WHERE created_at < $1", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
