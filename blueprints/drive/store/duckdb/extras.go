package duckdb

import (
	"context"
	"database/sql"
	"time"
)

// Tag represents a tag for organizing files.
type Tag struct {
	ID        string
	OwnerID   string
	Name      string
	Color     string
	CreatedAt time.Time
}

// TagsStore handles tag storage.
type TagsStore struct {
	db *sql.DB
}

// Create creates a new tag.
func (s *TagsStore) Create(ctx context.Context, tag *Tag) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tags (id, owner_id, name, color, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, tag.ID, tag.OwnerID, tag.Name, tag.Color, tag.CreatedAt)
	return err
}

// GetByID retrieves a tag by ID.
func (s *TagsStore) GetByID(ctx context.Context, id string) (*Tag, error) {
	tag := &Tag{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, owner_id, name, color, created_at
		FROM tags WHERE id = ?
	`, id).Scan(&tag.ID, &tag.OwnerID, &tag.Name, &tag.Color, &tag.CreatedAt)
	if err != nil {
		return nil, err
	}
	return tag, nil
}

// ListByOwner lists tags for an owner.
func (s *TagsStore) ListByOwner(ctx context.Context, ownerID string) ([]*Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, owner_id, name, color, created_at
		FROM tags WHERE owner_id = ? ORDER BY name
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*Tag
	for rows.Next() {
		t := &Tag{}
		if err := rows.Scan(&t.ID, &t.OwnerID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// Delete deletes a tag.
func (s *TagsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, id)
	return err
}

// Comment represents a comment on a file.
type Comment struct {
	ID        string
	FileID    string
	AuthorID  string
	ParentID  string
	Content   string
	Resolved  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CommentsStore handles comment storage.
type CommentsStore struct {
	db *sql.DB
}

// Create creates a new comment.
func (s *CommentsStore) Create(ctx context.Context, c *Comment) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO comments (id, file_id, author_id, parent_id, content, resolved, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.FileID, c.AuthorID, c.ParentID, c.Content, c.Resolved, c.CreatedAt, c.UpdatedAt)
	return err
}

// GetByID retrieves a comment by ID.
func (s *CommentsStore) GetByID(ctx context.Context, id string) (*Comment, error) {
	c := &Comment{}
	var parentID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, file_id, author_id, parent_id, content, resolved, created_at, updated_at
		FROM comments WHERE id = ?
	`, id).Scan(&c.ID, &c.FileID, &c.AuthorID, &parentID, &c.Content, &c.Resolved, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	c.ParentID = parentID.String
	return c, nil
}

// ListByFile lists comments for a file.
func (s *CommentsStore) ListByFile(ctx context.Context, fileID string) ([]*Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_id, author_id, parent_id, content, resolved, created_at, updated_at
		FROM comments WHERE file_id = ? ORDER BY created_at
	`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*Comment
	for rows.Next() {
		c := &Comment{}
		var parentID sql.NullString
		if err := rows.Scan(&c.ID, &c.FileID, &c.AuthorID, &parentID, &c.Content, &c.Resolved, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.ParentID = parentID.String
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// Delete deletes a comment.
func (s *CommentsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE id = ?`, id)
	return err
}

// Activity represents an activity log entry.
type Activity struct {
	ID        string
	AccountID string
	Action    string
	ItemID    string
	ItemType  string
	ItemName  string
	Details   string
	CreatedAt time.Time
}

// ActivitiesStore handles activity storage.
type ActivitiesStore struct {
	db *sql.DB
}

// Create creates a new activity.
func (s *ActivitiesStore) Create(ctx context.Context, a *Activity) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO activities (id, account_id, action, item_id, item_type, item_name, details, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, a.ID, a.AccountID, a.Action, a.ItemID, a.ItemType, a.ItemName, a.Details, a.CreatedAt)
	return err
}

// ListByAccount lists activities for an account.
func (s *ActivitiesStore) ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*Activity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, action, item_id, item_type, item_name, details, created_at
		FROM activities WHERE account_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*Activity
	for rows.Next() {
		a := &Activity{}
		var details sql.NullString
		if err := rows.Scan(&a.ID, &a.AccountID, &a.Action, &a.ItemID, &a.ItemType, &a.ItemName, &details, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.Details = details.String
		activities = append(activities, a)
	}
	return activities, rows.Err()
}

// Notification represents a notification.
type Notification struct {
	ID        string
	AccountID string
	Type      string
	ActorID   string
	ItemID    string
	ItemType  string
	Message   string
	Read      bool
	CreatedAt time.Time
}

// NotificationsStore handles notification storage.
type NotificationsStore struct {
	db *sql.DB
}

// Create creates a new notification.
func (s *NotificationsStore) Create(ctx context.Context, n *Notification) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (id, account_id, type, actor_id, item_id, item_type, message, read, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, n.ID, n.AccountID, n.Type, n.ActorID, n.ItemID, n.ItemType, n.Message, n.Read, n.CreatedAt)
	return err
}

// ListByAccount lists notifications for an account.
func (s *NotificationsStore) ListByAccount(ctx context.Context, accountID string, unreadOnly bool) ([]*Notification, error) {
	query := `
		SELECT id, account_id, type, actor_id, item_id, item_type, message, read, created_at
		FROM notifications WHERE account_id = ?`
	if unreadOnly {
		query += ` AND read = false`
	}
	query += ` ORDER BY created_at DESC LIMIT 100`

	rows, err := s.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		n := &Notification{}
		var actorID, itemID, itemType sql.NullString
		if err := rows.Scan(&n.ID, &n.AccountID, &n.Type, &actorID, &itemID, &itemType, &n.Message, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		n.ActorID = actorID.String
		n.ItemID = itemID.String
		n.ItemType = itemType.String
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

// MarkAsRead marks a notification as read.
func (s *NotificationsStore) MarkAsRead(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE notifications SET read = true WHERE id = ?`, id)
	return err
}

// MarkAllAsRead marks all notifications for an account as read.
func (s *NotificationsStore) MarkAllAsRead(ctx context.Context, accountID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE notifications SET read = true WHERE account_id = ?`, accountID)
	return err
}
