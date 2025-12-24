package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/channels"
)

// ChannelsStore implements channels.Store.
type ChannelsStore struct {
	db *sql.DB
}

// NewChannelsStore creates a new ChannelsStore.
func NewChannelsStore(db *sql.DB) *ChannelsStore {
	return &ChannelsStore{db: db}
}

// Insert creates a new channel.
func (s *ChannelsStore) Insert(ctx context.Context, ch *channels.Channel) error {
	query := `
		INSERT INTO channels (id, server_id, category_id, type, name, topic, position, is_private, is_nsfw, slow_mode_delay, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	// Convert empty strings to nil for nullable foreign keys
	var serverID, categoryID, ownerID any
	if ch.ServerID != "" {
		serverID = ch.ServerID
	}
	if ch.CategoryID != "" {
		categoryID = ch.CategoryID
	}
	if ch.OwnerID != "" {
		ownerID = ch.OwnerID
	}
	_, err := s.db.ExecContext(ctx, query,
		ch.ID, serverID, categoryID, ch.Type, ch.Name, ch.Topic,
		ch.Position, ch.IsPrivate, ch.IsNSFW, ch.SlowModeDelay, ownerID,
		ch.CreatedAt, ch.UpdatedAt,
	)
	return err
}

// GetByID retrieves a channel by ID.
func (s *ChannelsStore) GetByID(ctx context.Context, id string) (*channels.Channel, error) {
	query := `
		SELECT id, server_id, category_id, type, name, topic, position, is_private, is_nsfw, slow_mode_delay, bitrate, user_limit, last_message_id, last_message_at, message_count, icon_url, owner_id, created_at, updated_at
		FROM channels WHERE id = ?
	`
	ch := &channels.Channel{}
	var serverID, categoryID, name, topic, lastMsgID, iconURL, ownerID sql.NullString
	var bitrate, userLimit sql.NullInt64
	var lastMsgAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&ch.ID, &serverID, &categoryID, &ch.Type, &name, &topic,
		&ch.Position, &ch.IsPrivate, &ch.IsNSFW, &ch.SlowModeDelay, &bitrate, &userLimit,
		&lastMsgID, &lastMsgAt, &ch.MessageCount, &iconURL, &ownerID,
		&ch.CreatedAt, &ch.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, channels.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	ch.ServerID = serverID.String
	ch.CategoryID = categoryID.String
	ch.Name = name.String
	ch.Topic = topic.String
	ch.LastMessageID = lastMsgID.String
	ch.IconURL = iconURL.String
	ch.OwnerID = ownerID.String
	if bitrate.Valid {
		ch.Bitrate = int(bitrate.Int64)
	}
	if userLimit.Valid {
		ch.UserLimit = int(userLimit.Int64)
	}
	if lastMsgAt.Valid {
		ch.LastMessageAt = &lastMsgAt.Time
	}
	return ch, nil
}

// Update updates a channel.
func (s *ChannelsStore) Update(ctx context.Context, id string, in *channels.UpdateIn) error {
	var sets []string
	var args []any

	if in.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *in.Name)
	}
	if in.Topic != nil {
		sets = append(sets, "topic = ?")
		args = append(args, *in.Topic)
	}
	if in.Position != nil {
		sets = append(sets, "position = ?")
		args = append(args, *in.Position)
	}
	if in.IsPrivate != nil {
		sets = append(sets, "is_private = ?")
		args = append(args, *in.IsPrivate)
	}
	if in.IsNSFW != nil {
		sets = append(sets, "is_nsfw = ?")
		args = append(args, *in.IsNSFW)
	}
	if in.SlowModeDelay != nil {
		sets = append(sets, "slow_mode_delay = ?")
		args = append(args, *in.SlowModeDelay)
	}
	if in.CategoryID != nil {
		sets = append(sets, "category_id = ?")
		args = append(args, *in.CategoryID)
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE channels SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// Delete deletes a channel.
func (s *ChannelsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM channels WHERE id = ?", id)
	return err
}

// ListByServer lists channels in a server.
func (s *ChannelsStore) ListByServer(ctx context.Context, serverID string) ([]*channels.Channel, error) {
	query := `
		SELECT id, server_id, category_id, type, name, topic, position, is_private, is_nsfw, slow_mode_delay, bitrate, user_limit, last_message_id, last_message_at, message_count, icon_url, owner_id, created_at, updated_at
		FROM channels
		WHERE server_id = ?
		ORDER BY position ASC
	`
	rows, err := s.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chs []*channels.Channel
	for rows.Next() {
		ch := &channels.Channel{}
		var srvID, categoryID, name, topic, lastMsgID, iconURL, ownerID sql.NullString
		var bitrate, userLimit sql.NullInt64
		var lastMsgAt sql.NullTime
		if err := rows.Scan(
			&ch.ID, &srvID, &categoryID, &ch.Type, &name, &topic,
			&ch.Position, &ch.IsPrivate, &ch.IsNSFW, &ch.SlowModeDelay, &bitrate, &userLimit,
			&lastMsgID, &lastMsgAt, &ch.MessageCount, &iconURL, &ownerID,
			&ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, err
		}
		ch.ServerID = srvID.String
		ch.CategoryID = categoryID.String
		ch.Name = name.String
		ch.Topic = topic.String
		ch.LastMessageID = lastMsgID.String
		ch.IconURL = iconURL.String
		ch.OwnerID = ownerID.String
		if bitrate.Valid {
			ch.Bitrate = int(bitrate.Int64)
		}
		if userLimit.Valid {
			ch.UserLimit = int(userLimit.Int64)
		}
		if lastMsgAt.Valid {
			ch.LastMessageAt = &lastMsgAt.Time
		}
		chs = append(chs, ch)
	}
	return chs, rows.Err()
}

// ListDMsByUser lists DM channels for a user.
func (s *ChannelsStore) ListDMsByUser(ctx context.Context, userID string) ([]*channels.Channel, error) {
	query := `
		SELECT c.id, c.server_id, c.category_id, c.type, c.name, c.topic, c.position, c.is_private, c.is_nsfw, c.slow_mode_delay, c.bitrate, c.user_limit, c.last_message_id, c.last_message_at, c.message_count, c.icon_url, c.owner_id, c.created_at, c.updated_at
		FROM channels c
		JOIN channel_recipients cr ON c.id = cr.channel_id
		WHERE cr.user_id = ? AND c.type IN ('dm', 'group_dm')
		ORDER BY c.last_message_at DESC NULLS LAST
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chs []*channels.Channel
	for rows.Next() {
		ch := &channels.Channel{}
		var srvID, categoryID, name, topic, lastMsgID, iconURL, ownerID sql.NullString
		var bitrate, userLimit sql.NullInt64
		var lastMsgAt sql.NullTime
		if err := rows.Scan(
			&ch.ID, &srvID, &categoryID, &ch.Type, &name, &topic,
			&ch.Position, &ch.IsPrivate, &ch.IsNSFW, &ch.SlowModeDelay, &bitrate, &userLimit,
			&lastMsgID, &lastMsgAt, &ch.MessageCount, &iconURL, &ownerID,
			&ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, err
		}
		ch.ServerID = srvID.String
		ch.CategoryID = categoryID.String
		ch.Name = name.String
		ch.Topic = topic.String
		ch.LastMessageID = lastMsgID.String
		ch.IconURL = iconURL.String
		ch.OwnerID = ownerID.String
		if bitrate.Valid {
			ch.Bitrate = int(bitrate.Int64)
		}
		if userLimit.Valid {
			ch.UserLimit = int(userLimit.Int64)
		}
		if lastMsgAt.Valid {
			ch.LastMessageAt = &lastMsgAt.Time
		}
		chs = append(chs, ch)
	}
	return chs, rows.Err()
}

// GetDMChannel gets or creates a DM channel between two users.
func (s *ChannelsStore) GetDMChannel(ctx context.Context, userID1, userID2 string) (*channels.Channel, error) {
	// Find existing DM
	query := `
		SELECT c.id
		FROM channels c
		JOIN channel_recipients cr1 ON c.id = cr1.channel_id AND cr1.user_id = ?
		JOIN channel_recipients cr2 ON c.id = cr2.channel_id AND cr2.user_id = ?
		WHERE c.type = 'dm'
		LIMIT 1
	`
	var id string
	err := s.db.QueryRowContext(ctx, query, userID1, userID2).Scan(&id)
	if err == nil {
		return s.GetByID(ctx, id)
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	return nil, channels.ErrNotFound
}

// AddRecipient adds a recipient to a channel.
func (s *ChannelsStore) AddRecipient(ctx context.Context, channelID, userID string) error {
	query := `INSERT INTO channel_recipients (channel_id, user_id, created_at) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, channelID, userID, time.Now())
	return err
}

// RemoveRecipient removes a recipient from a channel.
func (s *ChannelsStore) RemoveRecipient(ctx context.Context, channelID, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM channel_recipients WHERE channel_id = ? AND user_id = ?", channelID, userID)
	return err
}

// GetRecipients gets all recipients of a channel.
func (s *ChannelsStore) GetRecipients(ctx context.Context, channelID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT user_id FROM channel_recipients WHERE channel_id = ?", channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// UpdateLastMessage updates the last message info.
func (s *ChannelsStore) UpdateLastMessage(ctx context.Context, channelID, messageID string, at time.Time) error {
	query := `UPDATE channels SET last_message_id = ?, last_message_at = ?, message_count = message_count + 1, updated_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, messageID, at, at, channelID)
	return err
}

// InsertCategory creates a new category.
func (s *ChannelsStore) InsertCategory(ctx context.Context, cat *channels.Category) error {
	query := `INSERT INTO categories (id, server_id, name, position, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, cat.ID, cat.ServerID, cat.Name, cat.Position, cat.CreatedAt)
	return err
}

// GetCategory retrieves a category by ID.
func (s *ChannelsStore) GetCategory(ctx context.Context, id string) (*channels.Category, error) {
	query := `SELECT id, server_id, name, position, created_at FROM categories WHERE id = ?`
	cat := &channels.Category{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(&cat.ID, &cat.ServerID, &cat.Name, &cat.Position, &cat.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, channels.ErrNotFound
	}
	return cat, err
}

// ListCategories lists categories in a server.
func (s *ChannelsStore) ListCategories(ctx context.Context, serverID string) ([]*channels.Category, error) {
	query := `SELECT id, server_id, name, position, created_at FROM categories WHERE server_id = ? ORDER BY position ASC`
	rows, err := s.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []*channels.Category
	for rows.Next() {
		cat := &channels.Category{}
		if err := rows.Scan(&cat.ID, &cat.ServerID, &cat.Name, &cat.Position, &cat.CreatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, cat)
	}
	return cats, rows.Err()
}

// DeleteCategory deletes a category.
func (s *ChannelsStore) DeleteCategory(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM categories WHERE id = ?", id)
	return err
}
