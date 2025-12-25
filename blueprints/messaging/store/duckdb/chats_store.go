package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/messaging/feature/chats"
)

// ChatsStore implements chats.Store.
type ChatsStore struct {
	db *sql.DB
}

// NewChatsStore creates a new ChatsStore.
func NewChatsStore(db *sql.DB) *ChatsStore {
	return &ChatsStore{db: db}
}

// Insert creates a new chat.
func (s *ChatsStore) Insert(ctx context.Context, c *chats.Chat) error {
	query := `
		INSERT INTO chats (id, type, name, description, icon_url, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		c.ID, c.Type, c.Name, c.Description, c.IconURL, nullString(c.OwnerID), c.CreatedAt, c.UpdatedAt,
	)
	return err
}

// GetByID retrieves a chat by ID.
func (s *ChatsStore) GetByID(ctx context.Context, id string) (*chats.Chat, error) {
	query := `
		SELECT id, type, name, description, icon_url, owner_id, last_message_id, last_message_at, message_count, created_at, updated_at
		FROM chats WHERE id = ?
	`
	c, err := scanChat(s.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, chats.ErrNotFound
	}
	return c, err
}

func scanChat(row interface{ Scan(...any) error }) (*chats.Chat, error) {
	c := &chats.Chat{}
	var name, description, iconURL, ownerID, lastMessageID sql.NullString
	var lastMessageAt sql.NullTime
	err := row.Scan(
		&c.ID, &c.Type, &name, &description, &iconURL, &ownerID,
		&lastMessageID, &lastMessageAt, &c.MessageCount, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.Name = name.String
	c.Description = description.String
	c.IconURL = iconURL.String
	c.OwnerID = ownerID.String
	c.LastMessageID = lastMessageID.String
	if lastMessageAt.Valid {
		c.LastMessageAt = lastMessageAt.Time
	}
	return c, nil
}

// GetByIDForUser retrieves a chat with user-specific data.
func (s *ChatsStore) GetByIDForUser(ctx context.Context, id, userID string) (*chats.Chat, error) {
	query := `
		SELECT c.id, c.type, c.name, c.description, c.icon_url, c.owner_id,
			c.last_message_id, c.last_message_at, c.message_count, c.created_at, c.updated_at,
			cp.unread_count, cp.is_muted, cp.last_read_message_id,
			CASE WHEN ac.chat_id IS NOT NULL THEN TRUE ELSE FALSE END as is_archived,
			CASE WHEN pc.chat_id IS NOT NULL THEN TRUE ELSE FALSE END as is_pinned
		FROM chats c
		JOIN chat_participants cp ON c.id = cp.chat_id AND cp.user_id = ?
		LEFT JOIN archived_chats ac ON c.id = ac.chat_id AND ac.user_id = ?
		LEFT JOIN pinned_chats pc ON c.id = pc.chat_id AND pc.user_id = ?
		WHERE c.id = ?
	`
	c := &chats.Chat{}
	var name, description, iconURL, ownerID, lastMessageID, lastReadMessageID sql.NullString
	var lastMessageAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, userID, userID, userID, id).Scan(
		&c.ID, &c.Type, &name, &description, &iconURL, &ownerID,
		&lastMessageID, &lastMessageAt, &c.MessageCount, &c.CreatedAt, &c.UpdatedAt,
		&c.UnreadCount, &c.IsMuted, &lastReadMessageID, &c.IsArchived, &c.IsPinned,
	)
	if err == sql.ErrNoRows {
		return nil, chats.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Name = name.String
	c.Description = description.String
	c.IconURL = iconURL.String
	c.OwnerID = ownerID.String
	c.LastMessageID = lastMessageID.String
	c.LastReadMessageID = lastReadMessageID.String
	if lastMessageAt.Valid {
		c.LastMessageAt = lastMessageAt.Time
	}
	return c, nil
}

// GetDirectChat finds an existing direct chat between two users.
func (s *ChatsStore) GetDirectChat(ctx context.Context, userID1, userID2 string) (*chats.Chat, error) {
	query := `
		SELECT c.id, c.type, c.name, c.description, c.icon_url, c.owner_id,
			c.last_message_id, c.last_message_at, c.message_count, c.created_at, c.updated_at
		FROM chats c
		WHERE c.type = 'direct'
		AND EXISTS (SELECT 1 FROM chat_participants WHERE chat_id = c.id AND user_id = ?)
		AND EXISTS (SELECT 1 FROM chat_participants WHERE chat_id = c.id AND user_id = ?)
	`
	c, err := scanChat(s.db.QueryRowContext(ctx, query, userID1, userID2))
	if err == sql.ErrNoRows {
		return nil, chats.ErrNotFound
	}
	return c, err
}

// List lists chats for a user.
func (s *ChatsStore) List(ctx context.Context, userID string, opts chats.ListOpts) ([]*chats.Chat, error) {
	query := `
		SELECT c.id, c.type, c.name, c.description, c.icon_url, c.owner_id,
			c.last_message_id, c.last_message_at, c.message_count, c.created_at, c.updated_at,
			cp.unread_count, cp.is_muted, cp.last_read_message_id,
			CASE WHEN ac.chat_id IS NOT NULL THEN TRUE ELSE FALSE END as is_archived,
			CASE WHEN pc.chat_id IS NOT NULL THEN TRUE ELSE FALSE END as is_pinned
		FROM chats c
		JOIN chat_participants cp ON c.id = cp.chat_id AND cp.user_id = ?
		LEFT JOIN archived_chats ac ON c.id = ac.chat_id AND ac.user_id = ?
		LEFT JOIN pinned_chats pc ON c.id = pc.chat_id AND pc.user_id = ?
	`

	if !opts.IncludeArchived {
		query += " WHERE ac.chat_id IS NULL"
	}

	query += " ORDER BY pc.position DESC NULLS LAST, c.last_message_at DESC NULLS LAST LIMIT ? OFFSET ?"

	rows, err := s.db.QueryContext(ctx, query, userID, userID, userID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chatList []*chats.Chat
	for rows.Next() {
		c := &chats.Chat{}
		var name, description, iconURL, ownerID, lastMessageID, lastReadMessageID sql.NullString
		var lastMessageAt sql.NullTime
		if err := rows.Scan(
			&c.ID, &c.Type, &name, &description, &iconURL, &ownerID,
			&lastMessageID, &lastMessageAt, &c.MessageCount, &c.CreatedAt, &c.UpdatedAt,
			&c.UnreadCount, &c.IsMuted, &lastReadMessageID, &c.IsArchived, &c.IsPinned,
		); err != nil {
			return nil, err
		}
		c.Name = name.String
		c.Description = description.String
		c.IconURL = iconURL.String
		c.OwnerID = ownerID.String
		c.LastMessageID = lastMessageID.String
		c.LastReadMessageID = lastReadMessageID.String
		if lastMessageAt.Valid {
			c.LastMessageAt = lastMessageAt.Time
		}
		chatList = append(chatList, c)
	}
	return chatList, rows.Err()
}

// Update updates a chat.
func (s *ChatsStore) Update(ctx context.Context, id string, in *chats.UpdateIn) error {
	var sets []string
	var args []any

	if in.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *in.Name)
	}
	if in.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *in.Description)
	}
	if in.IconURL != nil {
		sets = append(sets, "icon_url = ?")
		args = append(args, *in.IconURL)
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE chats SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// Delete deletes a chat.
func (s *ChatsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM chats WHERE id = ?", id)
	return err
}

// InsertParticipant adds a participant.
func (s *ChatsStore) InsertParticipant(ctx context.Context, p *chats.Participant) error {
	query := `
		INSERT INTO chat_participants (chat_id, user_id, role, joined_at, is_muted, unread_count, notification_level)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query, p.ChatID, p.UserID, p.Role, p.JoinedAt, p.IsMuted, p.UnreadCount, "all")
	return err
}

// DeleteParticipant removes a participant.
func (s *ChatsStore) DeleteParticipant(ctx context.Context, chatID, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM chat_participants WHERE chat_id = ? AND user_id = ?", chatID, userID)
	return err
}

// UpdateParticipantRole updates a participant's role.
func (s *ChatsStore) UpdateParticipantRole(ctx context.Context, chatID, userID, role string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE chat_participants SET role = ? WHERE chat_id = ? AND user_id = ?", role, chatID, userID)
	return err
}

// GetParticipants gets all participants.
func (s *ChatsStore) GetParticipants(ctx context.Context, chatID string) ([]*chats.Participant, error) {
	query := `
		SELECT chat_id, user_id, role, joined_at, is_muted, mute_until, unread_count, last_read_message_id, last_read_at, notification_level
		FROM chat_participants WHERE chat_id = ?
	`
	rows, err := s.db.QueryContext(ctx, query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []*chats.Participant
	for rows.Next() {
		p := &chats.Participant{}
		var muteUntil, lastReadAt sql.NullTime
		var lastReadMessageID sql.NullString
		if err := rows.Scan(
			&p.ChatID, &p.UserID, &p.Role, &p.JoinedAt, &p.IsMuted, &muteUntil,
			&p.UnreadCount, &lastReadMessageID, &lastReadAt, &p.NotificationLevel,
		); err != nil {
			return nil, err
		}
		if muteUntil.Valid {
			p.MuteUntil = muteUntil.Time
		}
		if lastReadAt.Valid {
			p.LastReadAt = lastReadAt.Time
		}
		p.LastReadMessageID = lastReadMessageID.String
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

// GetParticipant gets a participant.
func (s *ChatsStore) GetParticipant(ctx context.Context, chatID, userID string) (*chats.Participant, error) {
	query := `
		SELECT chat_id, user_id, role, joined_at, is_muted, mute_until, unread_count, last_read_message_id, last_read_at, notification_level
		FROM chat_participants WHERE chat_id = ? AND user_id = ?
	`
	p := &chats.Participant{}
	var muteUntil, lastReadAt sql.NullTime
	var lastReadMessageID sql.NullString
	err := s.db.QueryRowContext(ctx, query, chatID, userID).Scan(
		&p.ChatID, &p.UserID, &p.Role, &p.JoinedAt, &p.IsMuted, &muteUntil,
		&p.UnreadCount, &lastReadMessageID, &lastReadAt, &p.NotificationLevel,
	)
	if err == sql.ErrNoRows {
		return nil, chats.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if muteUntil.Valid {
		p.MuteUntil = muteUntil.Time
	}
	if lastReadAt.Valid {
		p.LastReadAt = lastReadAt.Time
	}
	p.LastReadMessageID = lastReadMessageID.String
	return p, nil
}

// IsParticipant checks if a user is a participant.
func (s *ChatsStore) IsParticipant(ctx context.Context, chatID, userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM chat_participants WHERE chat_id = ? AND user_id = ?)", chatID, userID).Scan(&exists)
	return exists, err
}

// Mute mutes a chat.
func (s *ChatsStore) Mute(ctx context.Context, chatID, userID string, until *time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE chat_participants SET is_muted = TRUE, mute_until = ? WHERE chat_id = ? AND user_id = ?", until, chatID, userID)
	return err
}

// Unmute unmutes a chat.
func (s *ChatsStore) Unmute(ctx context.Context, chatID, userID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE chat_participants SET is_muted = FALSE, mute_until = NULL WHERE chat_id = ? AND user_id = ?", chatID, userID)
	return err
}

// Archive archives a chat.
func (s *ChatsStore) Archive(ctx context.Context, chatID, userID string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO archived_chats (user_id, chat_id, archived_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING", userID, chatID, time.Now())
	return err
}

// Unarchive unarchives a chat.
func (s *ChatsStore) Unarchive(ctx context.Context, chatID, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM archived_chats WHERE user_id = ? AND chat_id = ?", userID, chatID)
	return err
}

// Pin pins a chat.
func (s *ChatsStore) Pin(ctx context.Context, chatID, userID string) error {
	// Get max position
	var maxPos sql.NullInt64
	s.db.QueryRowContext(ctx, "SELECT MAX(position) FROM pinned_chats WHERE user_id = ?", userID).Scan(&maxPos)
	pos := 0
	if maxPos.Valid {
		pos = int(maxPos.Int64) + 1
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO pinned_chats (user_id, chat_id, position, pinned_at) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING", userID, chatID, pos, time.Now())
	return err
}

// Unpin unpins a chat.
func (s *ChatsStore) Unpin(ctx context.Context, chatID, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM pinned_chats WHERE user_id = ? AND chat_id = ?", userID, chatID)
	return err
}

// MarkAsRead marks a chat as read.
func (s *ChatsStore) MarkAsRead(ctx context.Context, chatID, userID, messageID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE chat_participants SET last_read_message_id = ?, last_read_at = ? WHERE chat_id = ? AND user_id = ?", messageID, time.Now(), chatID, userID)
	return err
}

// IncrementUnreadCount increments unread count for all participants except one.
func (s *ChatsStore) IncrementUnreadCount(ctx context.Context, chatID string, excludeUserID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE chat_participants SET unread_count = unread_count + 1 WHERE chat_id = ? AND user_id != ?", chatID, excludeUserID)
	return err
}

// ResetUnreadCount resets unread count.
func (s *ChatsStore) ResetUnreadCount(ctx context.Context, chatID, userID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE chat_participants SET unread_count = 0 WHERE chat_id = ? AND user_id = ?", chatID, userID)
	return err
}

// IncrementMessageCount increments message count.
func (s *ChatsStore) IncrementMessageCount(ctx context.Context, chatID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE chats SET message_count = message_count + 1 WHERE id = ?", chatID)
	return err
}

// UpdateLastMessage updates the last message.
func (s *ChatsStore) UpdateLastMessage(ctx context.Context, chatID, messageID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE chats SET last_message_id = ?, last_message_at = ?, updated_at = ? WHERE id = ?", messageID, time.Now(), time.Now(), chatID)
	return err
}
