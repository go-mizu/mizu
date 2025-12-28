package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/notifications"
)

// NotificationsStore handles notification data access.
type NotificationsStore struct {
	db *sql.DB
}

// NewNotificationsStore creates a new notifications store.
func NewNotificationsStore(db *sql.DB) *NotificationsStore {
	return &NotificationsStore{db: db}
}

func (s *NotificationsStore) Create(ctx context.Context, n *notifications.Notification, userID int64) error {
	if n.ID == "" {
		n.ID = fmt.Sprintf("ntf_%d", time.Now().UnixNano())
	}
	n.UpdatedAt = time.Now()

	repoID := int64(0)
	if n.Repository != nil {
		repoID = n.Repository.ID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, repo_id, unread, reason, subject_type, subject_title,
			subject_url, subject_latest_comment_url, last_read_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, n.ID, userID, repoID, n.Unread, n.Reason, n.Subject.Type, n.Subject.Title,
		n.Subject.URL, n.Subject.LatestCommentURL, nullTime(n.LastReadAt), n.UpdatedAt)
	return err
}

func (s *NotificationsStore) GetByID(ctx context.Context, id string, userID int64) (*notifications.Notification, error) {
	n := &notifications.Notification{Subject: &notifications.Subject{}}
	var repoID int64
	var lastReadAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, unread, reason, subject_type, subject_title, subject_url,
			subject_latest_comment_url, last_read_at, updated_at
		FROM notifications WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&n.ID, &repoID, &n.Unread, &n.Reason, &n.Subject.Type, &n.Subject.Title,
		&n.Subject.URL, &n.Subject.LatestCommentURL, &lastReadAt, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if repoID > 0 {
		n.Repository = &notifications.Repository{ID: repoID}
	}
	if lastReadAt.Valid {
		n.LastReadAt = &lastReadAt.Time
	}
	return n, err
}

func (s *NotificationsStore) MarkAsRead(ctx context.Context, userID int64, lastReadAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET unread = FALSE, last_read_at = $2, updated_at = $3
		WHERE user_id = $1 AND unread = TRUE
	`, userID, lastReadAt, time.Now())
	return err
}

func (s *NotificationsStore) MarkRepoAsRead(ctx context.Context, userID, repoID int64, lastReadAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET unread = FALSE, last_read_at = $3, updated_at = $4
		WHERE user_id = $1 AND repo_id = $2 AND unread = TRUE
	`, userID, repoID, lastReadAt, time.Now())
	return err
}

func (s *NotificationsStore) MarkThreadAsRead(ctx context.Context, id string, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET unread = FALSE, last_read_at = $3, updated_at = $4
		WHERE id = $1 AND user_id = $2
	`, id, userID, time.Now(), time.Now())
	return err
}

func (s *NotificationsStore) Delete(ctx context.Context, id string, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM notifications WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (s *NotificationsStore) List(ctx context.Context, userID int64, opts *notifications.ListOpts) ([]*notifications.Notification, error) {
	page, perPage := 1, 50
	all := false
	participating := false
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
		all = opts.All
		participating = opts.Participating
	}

	query := `
		SELECT id, repo_id, unread, reason, subject_type, subject_title, subject_url,
			subject_latest_comment_url, last_read_at, updated_at
		FROM notifications WHERE user_id = $1`
	args := []any{userID}

	if !all {
		query += ` AND unread = TRUE`
	}
	if participating {
		query += ` AND reason IN ('assign', 'author', 'comment', 'mention', 'review_requested')`
	}

	query += ` ORDER BY updated_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanNotifications(rows)
}

func (s *NotificationsStore) ListForRepo(ctx context.Context, userID, repoID int64, opts *notifications.ListOpts) ([]*notifications.Notification, error) {
	page, perPage := 1, 50
	all := false
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
		all = opts.All
	}

	query := `
		SELECT id, repo_id, unread, reason, subject_type, subject_title, subject_url,
			subject_latest_comment_url, last_read_at, updated_at
		FROM notifications WHERE user_id = $1 AND repo_id = $2`
	args := []any{userID, repoID}

	if !all {
		query += ` AND unread = TRUE`
	}

	query += ` ORDER BY updated_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanNotifications(rows)
}

// Thread subscription methods

func (s *NotificationsStore) GetSubscription(ctx context.Context, id string, userID int64) (*notifications.ThreadSubscription, error) {
	sub := &notifications.ThreadSubscription{}
	err := s.db.QueryRowContext(ctx, `
		SELECT ignored, created_at FROM thread_subscriptions WHERE thread_id = $1 AND user_id = $2
	`, id, userID).Scan(&sub.Ignored, &sub.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	sub.Subscribed = true
	return sub, err
}

func (s *NotificationsStore) SetSubscription(ctx context.Context, id string, userID int64, ignored bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO thread_subscriptions (thread_id, user_id, ignored, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (thread_id, user_id) DO UPDATE SET ignored = $3
	`, id, userID, ignored, time.Now())
	return err
}

func (s *NotificationsStore) DeleteSubscription(ctx context.Context, id string, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM thread_subscriptions WHERE thread_id = $1 AND user_id = $2`, id, userID)
	return err
}

func scanNotifications(rows *sql.Rows) ([]*notifications.Notification, error) {
	var list []*notifications.Notification
	for rows.Next() {
		n := &notifications.Notification{Subject: &notifications.Subject{}}
		var repoID int64
		var lastReadAt sql.NullTime
		if err := rows.Scan(&n.ID, &repoID, &n.Unread, &n.Reason, &n.Subject.Type, &n.Subject.Title,
			&n.Subject.URL, &n.Subject.LatestCommentURL, &lastReadAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		if repoID > 0 {
			n.Repository = &notifications.Repository{ID: repoID}
		}
		if lastReadAt.Valid {
			n.LastReadAt = &lastReadAt.Time
		}
		list = append(list, n)
	}
	return list, rows.Err()
}
