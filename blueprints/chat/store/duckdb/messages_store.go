package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/messages"
)

// MessagesStore implements messages.Store.
type MessagesStore struct {
	db *sql.DB
}

// NewMessagesStore creates a new MessagesStore.
func NewMessagesStore(db *sql.DB) *MessagesStore {
	return &MessagesStore{db: db}
}

// Insert creates a new message.
func (s *MessagesStore) Insert(ctx context.Context, msg *messages.Message) error {
	query := `
		INSERT INTO messages (id, channel_id, author_id, content, content_html, type, reply_to_id, thread_id, flags, mention_everyone, is_pinned, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		msg.ID, msg.ChannelID, msg.AuthorID, msg.Content, msg.ContentHTML,
		msg.Type, msg.ReplyToID, msg.ThreadID, msg.Flags, msg.MentionEveryone,
		msg.IsPinned, msg.CreatedAt,
	)
	if err != nil {
		return err
	}

	// Insert mentions
	for _, userID := range msg.Mentions {
		if _, err := s.db.ExecContext(ctx,
			"INSERT INTO message_mentions (message_id, user_id) VALUES (?, ?)",
			msg.ID, userID,
		); err != nil {
			return err
		}
	}

	return nil
}

// GetByID retrieves a message by ID.
func (s *MessagesStore) GetByID(ctx context.Context, id string) (*messages.Message, error) {
	query := `
		SELECT id, channel_id, author_id, content, content_html, type, reply_to_id, thread_id, flags, mention_everyone, is_pinned, is_edited, edited_at, created_at
		FROM messages WHERE id = ?
	`
	msg := &messages.Message{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&msg.ID, &msg.ChannelID, &msg.AuthorID, &msg.Content, &msg.ContentHTML,
		&msg.Type, &msg.ReplyToID, &msg.ThreadID, &msg.Flags, &msg.MentionEveryone,
		&msg.IsPinned, &msg.IsEdited, &msg.EditedAt, &msg.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, messages.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Load mentions
	msg.Mentions, _ = s.getMentions(ctx, id)
	// Load reactions
	msg.Reactions, _ = s.getReactions(ctx, id)
	// Load attachments
	msg.Attachments, _ = s.getAttachments(ctx, id)

	return msg, nil
}

// Update updates a message.
func (s *MessagesStore) Update(ctx context.Context, id string, in *messages.UpdateIn) error {
	var sets []string
	var args []any

	if in.Content != nil {
		sets = append(sets, "content = ?")
		args = append(args, *in.Content)
	}
	if in.ContentHTML != nil {
		sets = append(sets, "content_html = ?")
		args = append(args, *in.ContentHTML)
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "is_edited = TRUE", "edited_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE messages SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// Delete deletes a message.
func (s *MessagesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM messages WHERE id = ?", id)
	return err
}

// List lists messages in a channel with pagination.
func (s *MessagesStore) List(ctx context.Context, channelID string, opts messages.ListOpts) ([]*messages.Message, error) {
	var query string
	var args []any

	if opts.Before != "" {
		query = `
			SELECT id, channel_id, author_id, content, content_html, type, reply_to_id, thread_id, flags, mention_everyone, is_pinned, is_edited, edited_at, created_at
			FROM messages
			WHERE channel_id = ? AND id < ?
			ORDER BY created_at DESC
			LIMIT ?
		`
		args = []any{channelID, opts.Before, opts.Limit}
	} else if opts.After != "" {
		query = `
			SELECT id, channel_id, author_id, content, content_html, type, reply_to_id, thread_id, flags, mention_everyone, is_pinned, is_edited, edited_at, created_at
			FROM messages
			WHERE channel_id = ? AND id > ?
			ORDER BY created_at ASC
			LIMIT ?
		`
		args = []any{channelID, opts.After, opts.Limit}
	} else if opts.Around != "" {
		// Get messages around a specific message
		query = `
			(SELECT id, channel_id, author_id, content, content_html, type, reply_to_id, thread_id, flags, mention_everyone, is_pinned, is_edited, edited_at, created_at
			FROM messages WHERE channel_id = ? AND id <= ? ORDER BY created_at DESC LIMIT ?)
			UNION ALL
			(SELECT id, channel_id, author_id, content, content_html, type, reply_to_id, thread_id, flags, mention_everyone, is_pinned, is_edited, edited_at, created_at
			FROM messages WHERE channel_id = ? AND id > ? ORDER BY created_at ASC LIMIT ?)
			ORDER BY created_at DESC
		`
		half := opts.Limit / 2
		args = []any{channelID, opts.Around, half, channelID, opts.Around, half}
	} else {
		query = `
			SELECT id, channel_id, author_id, content, content_html, type, reply_to_id, thread_id, flags, mention_everyone, is_pinned, is_edited, edited_at, created_at
			FROM messages
			WHERE channel_id = ?
			ORDER BY created_at DESC
			LIMIT ?
		`
		args = []any{channelID, opts.Limit}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*messages.Message
	for rows.Next() {
		msg := &messages.Message{}
		if err := rows.Scan(
			&msg.ID, &msg.ChannelID, &msg.AuthorID, &msg.Content, &msg.ContentHTML,
			&msg.Type, &msg.ReplyToID, &msg.ThreadID, &msg.Flags, &msg.MentionEveryone,
			&msg.IsPinned, &msg.IsEdited, &msg.EditedAt, &msg.CreatedAt,
		); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}

	// Load related data for all messages
	for _, msg := range msgs {
		msg.Mentions, _ = s.getMentions(ctx, msg.ID)
		msg.Reactions, _ = s.getReactions(ctx, msg.ID)
		msg.Attachments, _ = s.getAttachments(ctx, msg.ID)
	}

	return msgs, rows.Err()
}

// Search searches messages.
func (s *MessagesStore) Search(ctx context.Context, opts messages.SearchOpts) ([]*messages.Message, error) {
	var conditions []string
	var args []any

	if opts.ChannelID != "" {
		conditions = append(conditions, "channel_id = ?")
		args = append(args, opts.ChannelID)
	}
	if opts.AuthorID != "" {
		conditions = append(conditions, "author_id = ?")
		args = append(args, opts.AuthorID)
	}
	if opts.Query != "" {
		conditions = append(conditions, "content ILIKE ?")
		args = append(args, "%"+opts.Query+"%")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, channel_id, author_id, content, content_html, type, reply_to_id, thread_id, flags, mention_everyone, is_pinned, is_edited, edited_at, created_at
		FROM messages
		%s
		ORDER BY created_at DESC
		LIMIT ?
	`, where)
	args = append(args, opts.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*messages.Message
	for rows.Next() {
		msg := &messages.Message{}
		if err := rows.Scan(
			&msg.ID, &msg.ChannelID, &msg.AuthorID, &msg.Content, &msg.ContentHTML,
			&msg.Type, &msg.ReplyToID, &msg.ThreadID, &msg.Flags, &msg.MentionEveryone,
			&msg.IsPinned, &msg.IsEdited, &msg.EditedAt, &msg.CreatedAt,
		); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}

// Pin pins a message.
func (s *MessagesStore) Pin(ctx context.Context, channelID, messageID, userID string) error {
	query := `INSERT INTO pins (channel_id, message_id, pinned_by, pinned_at) VALUES (?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, channelID, messageID, userID, time.Now())
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "UPDATE messages SET is_pinned = TRUE WHERE id = ?", messageID)
	return err
}

// Unpin unpins a message.
func (s *MessagesStore) Unpin(ctx context.Context, channelID, messageID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM pins WHERE channel_id = ? AND message_id = ?", channelID, messageID)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "UPDATE messages SET is_pinned = FALSE WHERE id = ?", messageID)
	return err
}

// ListPinned lists pinned messages in a channel.
func (s *MessagesStore) ListPinned(ctx context.Context, channelID string) ([]*messages.Message, error) {
	query := `
		SELECT m.id, m.channel_id, m.author_id, m.content, m.content_html, m.type, m.reply_to_id, m.thread_id, m.flags, m.mention_everyone, m.is_pinned, m.is_edited, m.edited_at, m.created_at
		FROM messages m
		JOIN pins p ON m.id = p.message_id
		WHERE p.channel_id = ?
		ORDER BY p.pinned_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*messages.Message
	for rows.Next() {
		msg := &messages.Message{}
		if err := rows.Scan(
			&msg.ID, &msg.ChannelID, &msg.AuthorID, &msg.Content, &msg.ContentHTML,
			&msg.Type, &msg.ReplyToID, &msg.ThreadID, &msg.Flags, &msg.MentionEveryone,
			&msg.IsPinned, &msg.IsEdited, &msg.EditedAt, &msg.CreatedAt,
		); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}

// AddReaction adds a reaction to a message.
func (s *MessagesStore) AddReaction(ctx context.Context, messageID, userID, emoji string) error {
	query := `INSERT INTO reactions (message_id, user_id, emoji, created_at) VALUES (?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, messageID, userID, emoji, time.Now())
	return err
}

// RemoveReaction removes a reaction from a message.
func (s *MessagesStore) RemoveReaction(ctx context.Context, messageID, userID, emoji string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM reactions WHERE message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji)
	return err
}

// GetReactionUsers gets users who reacted with a specific emoji.
func (s *MessagesStore) GetReactionUsers(ctx context.Context, messageID, emoji string, limit int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT user_id FROM reactions WHERE message_id = ? AND emoji = ? LIMIT ?",
		messageID, emoji, limit,
	)
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

// InsertAttachment creates an attachment.
func (s *MessagesStore) InsertAttachment(ctx context.Context, att *messages.Attachment) error {
	query := `
		INSERT INTO attachments (id, message_id, filename, content_type, size, url, proxy_url, width, height, is_spoiler, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		att.ID, att.MessageID, att.Filename, att.ContentType, att.Size,
		att.URL, att.ProxyURL, att.Width, att.Height, att.IsSpoiler, att.CreatedAt,
	)
	return err
}

// Helper functions

func (s *MessagesStore) getMentions(ctx context.Context, messageID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT user_id FROM message_mentions WHERE message_id = ?", messageID)
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

func (s *MessagesStore) getReactions(ctx context.Context, messageID string) ([]messages.Reaction, error) {
	query := `
		SELECT emoji, COUNT(*) as count
		FROM reactions
		WHERE message_id = ?
		GROUP BY emoji
	`
	rows, err := s.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []messages.Reaction
	for rows.Next() {
		var r messages.Reaction
		if err := rows.Scan(&r.Emoji, &r.Count); err != nil {
			return nil, err
		}
		reactions = append(reactions, r)
	}
	return reactions, rows.Err()
}

func (s *MessagesStore) getAttachments(ctx context.Context, messageID string) ([]messages.Attachment, error) {
	query := `
		SELECT id, message_id, filename, content_type, size, url, proxy_url, width, height, is_spoiler, created_at
		FROM attachments WHERE message_id = ?
	`
	rows, err := s.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []messages.Attachment
	for rows.Next() {
		var a messages.Attachment
		if err := rows.Scan(
			&a.ID, &a.MessageID, &a.Filename, &a.ContentType, &a.Size,
			&a.URL, &a.ProxyURL, &a.Width, &a.Height, &a.IsSpoiler, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	return attachments, rows.Err()
}

// InsertEmbed creates an embed.
func (s *MessagesStore) InsertEmbed(ctx context.Context, messageID string, embed *messages.Embed) error {
	fieldsJSON, _ := json.Marshal(embed.Fields)
	query := `
		INSERT INTO embeds (id, message_id, type, title, description, url, color, image_url, video_url, thumbnail, footer, author_name, author_url, fields, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		embed.ID, messageID, embed.Type, embed.Title, embed.Description, embed.URL,
		embed.Color, embed.ImageURL, embed.VideoURL, embed.Thumbnail, embed.Footer,
		embed.AuthorName, embed.AuthorURL, fieldsJSON, time.Now(),
	)
	return err
}
