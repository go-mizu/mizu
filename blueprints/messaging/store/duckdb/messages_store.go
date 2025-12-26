package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/messaging/feature/messages"
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
func (s *MessagesStore) Insert(ctx context.Context, m *messages.Message) error {
	query := `
		INSERT INTO messages (id, chat_id, sender_id, type, content, content_html, reply_to_id,
			forward_from_id, forward_from_chat_id, forward_from_sender_name, is_forwarded,
			mention_everyone, expires_at,
			media_id, media_url, media_type, media_content_type, media_filename,
			media_size, media_width, media_height, media_duration, media_thumbnail_url, media_waveform,
			sticker_pack_id, sticker_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		m.ID, m.ChatID, m.SenderID, m.Type, m.Content, m.ContentHTML,
		nullString(m.ReplyToID), nullString(m.ForwardFromID), nullString(m.ForwardFromChatID),
		nullString(m.ForwardFromSenderName), m.IsForwarded, m.MentionEveryone, m.ExpiresAt,
		nullString(m.MediaID), nullString(m.MediaURL), nullString(m.MediaType), nullString(m.MediaContentType),
		nullString(m.MediaFilename), m.MediaSize, m.MediaWidth, m.MediaHeight, m.MediaDuration,
		nullString(m.MediaThumbnailURL), nullString(m.MediaWaveform),
		nullString(m.StickerPackID), nullString(m.StickerID), m.CreatedAt,
	)
	return err
}

// GetByID retrieves a message by ID.
func (s *MessagesStore) GetByID(ctx context.Context, id string) (*messages.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, type, content, content_html, reply_to_id,
			forward_from_id, forward_from_chat_id, forward_from_sender_name, is_forwarded,
			is_edited, edited_at, is_deleted, deleted_at, deleted_for_everyone,
			expires_at, mention_everyone,
			media_id, media_url, media_type, media_content_type, media_filename,
			media_size, media_width, media_height, media_duration, media_thumbnail_url, media_waveform,
			sticker_pack_id, sticker_id, created_at
		FROM messages WHERE id = ? AND (is_deleted = FALSE OR deleted_for_everyone = FALSE)
	`
	m, err := scanMessage(s.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, messages.ErrNotFound
	}
	return m, err
}

func scanMessage(row interface{ Scan(...any) error }) (*messages.Message, error) {
	m := &messages.Message{}
	var content, contentHTML, replyToID, forwardFromID, forwardFromChatID, forwardFromSenderName sql.NullString
	var mediaID, mediaURL, mediaType, mediaContentType, mediaFilename, mediaThumbnailURL, mediaWaveform sql.NullString
	var stickerPackID, stickerID sql.NullString
	var mediaSize sql.NullInt64
	var mediaWidth, mediaHeight, mediaDuration sql.NullInt64
	var editedAt, deletedAt, expiresAt sql.NullTime
	err := row.Scan(
		&m.ID, &m.ChatID, &m.SenderID, &m.Type, &content, &contentHTML, &replyToID,
		&forwardFromID, &forwardFromChatID, &forwardFromSenderName, &m.IsForwarded,
		&m.IsEdited, &editedAt, &m.IsDeleted, &deletedAt, &m.DeletedForEveryone,
		&expiresAt, &m.MentionEveryone,
		&mediaID, &mediaURL, &mediaType, &mediaContentType, &mediaFilename,
		&mediaSize, &mediaWidth, &mediaHeight, &mediaDuration, &mediaThumbnailURL, &mediaWaveform,
		&stickerPackID, &stickerID, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	m.Content = content.String
	m.ContentHTML = contentHTML.String
	m.ReplyToID = replyToID.String
	m.ForwardFromID = forwardFromID.String
	m.ForwardFromChatID = forwardFromChatID.String
	m.ForwardFromSenderName = forwardFromSenderName.String
	m.MediaID = mediaID.String
	m.MediaURL = mediaURL.String
	m.MediaType = mediaType.String
	m.MediaContentType = mediaContentType.String
	m.MediaFilename = mediaFilename.String
	m.MediaSize = mediaSize.Int64
	m.MediaWidth = int(mediaWidth.Int64)
	m.MediaHeight = int(mediaHeight.Int64)
	m.MediaDuration = int(mediaDuration.Int64)
	m.MediaThumbnailURL = mediaThumbnailURL.String
	m.MediaWaveform = mediaWaveform.String
	m.StickerPackID = stickerPackID.String
	m.StickerID = stickerID.String
	if editedAt.Valid {
		m.EditedAt = &editedAt.Time
	}
	if deletedAt.Valid {
		m.DeletedAt = &deletedAt.Time
	}
	if expiresAt.Valid {
		m.ExpiresAt = &expiresAt.Time
	}
	return m, nil
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

	sets = append(sets, "is_edited = TRUE", "edited_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE messages SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// Delete deletes a message.
func (s *MessagesStore) Delete(ctx context.Context, id string, forEveryone bool) error {
	now := time.Now()
	query := "UPDATE messages SET is_deleted = TRUE, deleted_at = ?, deleted_for_everyone = ? WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, now, forEveryone, id)
	return err
}

// List lists messages in a chat.
func (s *MessagesStore) List(ctx context.Context, chatID string, opts messages.ListOpts) ([]*messages.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, type, content, content_html, reply_to_id,
			forward_from_id, forward_from_chat_id, forward_from_sender_name, is_forwarded,
			is_edited, edited_at, is_deleted, deleted_at, deleted_for_everyone,
			expires_at, mention_everyone,
			media_id, media_url, media_type, media_content_type, media_filename,
			media_size, media_width, media_height, media_duration, media_thumbnail_url, media_waveform,
			sticker_pack_id, sticker_id, created_at
		FROM messages
		WHERE chat_id = ? AND (is_deleted = FALSE OR deleted_for_everyone = FALSE)
	`

	var args []any
	args = append(args, chatID)

	if opts.Before != "" {
		query += " AND id < ?"
		args = append(args, opts.Before)
	}
	if opts.After != "" {
		query += " AND id > ?"
		args = append(args, opts.After)
	}

	query += " ORDER BY created_at ASC LIMIT ?"
	args = append(args, opts.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgList []*messages.Message
	for rows.Next() {
		m := &messages.Message{}
		var content, contentHTML, replyToID, forwardFromID, forwardFromChatID, forwardFromSenderName sql.NullString
		var mediaID, mediaURL, mediaType, mediaContentType, mediaFilename, mediaThumbnailURL, mediaWaveform sql.NullString
		var stickerPackID, stickerID sql.NullString
		var mediaSize sql.NullInt64
		var mediaWidth, mediaHeight, mediaDuration sql.NullInt64
		var editedAt, deletedAt, expiresAt sql.NullTime
		if err := rows.Scan(
			&m.ID, &m.ChatID, &m.SenderID, &m.Type, &content, &contentHTML, &replyToID,
			&forwardFromID, &forwardFromChatID, &forwardFromSenderName, &m.IsForwarded,
			&m.IsEdited, &editedAt, &m.IsDeleted, &deletedAt, &m.DeletedForEveryone,
			&expiresAt, &m.MentionEveryone,
			&mediaID, &mediaURL, &mediaType, &mediaContentType, &mediaFilename,
			&mediaSize, &mediaWidth, &mediaHeight, &mediaDuration, &mediaThumbnailURL, &mediaWaveform,
			&stickerPackID, &stickerID, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.Content = content.String
		m.ContentHTML = contentHTML.String
		m.ReplyToID = replyToID.String
		m.ForwardFromID = forwardFromID.String
		m.ForwardFromChatID = forwardFromChatID.String
		m.ForwardFromSenderName = forwardFromSenderName.String
		m.MediaID = mediaID.String
		m.MediaURL = mediaURL.String
		m.MediaType = mediaType.String
		m.MediaContentType = mediaContentType.String
		m.MediaFilename = mediaFilename.String
		m.MediaSize = mediaSize.Int64
		m.MediaWidth = int(mediaWidth.Int64)
		m.MediaHeight = int(mediaHeight.Int64)
		m.MediaDuration = int(mediaDuration.Int64)
		m.MediaThumbnailURL = mediaThumbnailURL.String
		m.MediaWaveform = mediaWaveform.String
		m.StickerPackID = stickerPackID.String
		m.StickerID = stickerID.String
		if editedAt.Valid {
			m.EditedAt = &editedAt.Time
		}
		if deletedAt.Valid {
			m.DeletedAt = &deletedAt.Time
		}
		if expiresAt.Valid {
			m.ExpiresAt = &expiresAt.Time
		}
		msgList = append(msgList, m)
	}
	return msgList, rows.Err()
}

// Search searches messages.
func (s *MessagesStore) Search(ctx context.Context, opts messages.SearchOpts) ([]*messages.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, type, content, content_html, reply_to_id,
			forward_from_id, forward_from_chat_id, forward_from_sender_name, is_forwarded,
			is_edited, edited_at, is_deleted, deleted_at, deleted_for_everyone,
			expires_at, mention_everyone,
			media_id, media_url, media_type, media_content_type, media_filename,
			media_size, media_width, media_height, media_duration, media_thumbnail_url, media_waveform,
			sticker_pack_id, sticker_id, created_at
		FROM messages
		WHERE (is_deleted = FALSE OR deleted_for_everyone = FALSE)
			AND content ILIKE ?
	`

	var args []any
	args = append(args, "%"+opts.Query+"%")

	if opts.ChatID != "" {
		query += " AND chat_id = ?"
		args = append(args, opts.ChatID)
	}
	if opts.SenderID != "" {
		query += " AND sender_id = ?"
		args = append(args, opts.SenderID)
	}
	if opts.Type != "" {
		query += " AND type = ?"
		args = append(args, opts.Type)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, opts.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgList []*messages.Message
	for rows.Next() {
		m := &messages.Message{}
		var content, contentHTML, replyToID, forwardFromID, forwardFromChatID, forwardFromSenderName sql.NullString
		var mediaID, mediaURL, mediaType, mediaContentType, mediaFilename, mediaThumbnailURL, mediaWaveform sql.NullString
		var stickerPackID, stickerID sql.NullString
		var mediaSize sql.NullInt64
		var mediaWidth, mediaHeight, mediaDuration sql.NullInt64
		var editedAt, deletedAt, expiresAt sql.NullTime
		if err := rows.Scan(
			&m.ID, &m.ChatID, &m.SenderID, &m.Type, &content, &contentHTML, &replyToID,
			&forwardFromID, &forwardFromChatID, &forwardFromSenderName, &m.IsForwarded,
			&m.IsEdited, &editedAt, &m.IsDeleted, &deletedAt, &m.DeletedForEveryone,
			&expiresAt, &m.MentionEveryone,
			&mediaID, &mediaURL, &mediaType, &mediaContentType, &mediaFilename,
			&mediaSize, &mediaWidth, &mediaHeight, &mediaDuration, &mediaThumbnailURL, &mediaWaveform,
			&stickerPackID, &stickerID, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.Content = content.String
		m.ContentHTML = contentHTML.String
		m.ReplyToID = replyToID.String
		m.ForwardFromID = forwardFromID.String
		m.ForwardFromChatID = forwardFromChatID.String
		m.ForwardFromSenderName = forwardFromSenderName.String
		m.MediaID = mediaID.String
		m.MediaURL = mediaURL.String
		m.MediaType = mediaType.String
		m.MediaContentType = mediaContentType.String
		m.MediaFilename = mediaFilename.String
		m.MediaSize = mediaSize.Int64
		m.MediaWidth = int(mediaWidth.Int64)
		m.MediaHeight = int(mediaHeight.Int64)
		m.MediaDuration = int(mediaDuration.Int64)
		m.MediaThumbnailURL = mediaThumbnailURL.String
		m.MediaWaveform = mediaWaveform.String
		m.StickerPackID = stickerPackID.String
		m.StickerID = stickerID.String
		if editedAt.Valid {
			m.EditedAt = &editedAt.Time
		}
		if deletedAt.Valid {
			m.DeletedAt = &deletedAt.Time
		}
		if expiresAt.Valid {
			m.ExpiresAt = &expiresAt.Time
		}
		msgList = append(msgList, m)
	}
	return msgList, rows.Err()
}

// AddReaction adds a reaction.
func (s *MessagesStore) AddReaction(ctx context.Context, messageID, userID, emoji string) error {
	query := `INSERT INTO message_reactions (message_id, user_id, emoji, created_at) VALUES (?, ?, ?, ?) ON CONFLICT DO UPDATE SET emoji = ?`
	_, err := s.db.ExecContext(ctx, query, messageID, userID, emoji, time.Now(), emoji)
	return err
}

// RemoveReaction removes a reaction.
func (s *MessagesStore) RemoveReaction(ctx context.Context, messageID, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM message_reactions WHERE message_id = ? AND user_id = ?", messageID, userID)
	return err
}

// GetReactions gets reactions for a message.
func (s *MessagesStore) GetReactions(ctx context.Context, messageID string) ([]messages.Reaction, error) {
	query := `
		SELECT emoji, COUNT(*) as count, ARRAY_AGG(user_id) as users
		FROM message_reactions WHERE message_id = ?
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
		var usersJSON string
		if err := rows.Scan(&r.Emoji, &r.Count, &usersJSON); err != nil {
			return nil, err
		}
		// Parse users array - simplified
		reactions = append(reactions, r)
	}
	return reactions, rows.Err()
}

// Star stars a message.
func (s *MessagesStore) Star(ctx context.Context, messageID, userID string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO starred_messages (user_id, message_id, created_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING", userID, messageID, time.Now())
	return err
}

// Unstar unstars a message.
func (s *MessagesStore) Unstar(ctx context.Context, messageID, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM starred_messages WHERE user_id = ? AND message_id = ?", userID, messageID)
	return err
}

// ListStarred lists starred messages.
func (s *MessagesStore) ListStarred(ctx context.Context, userID string, limit int) ([]*messages.Message, error) {
	query := `
		SELECT m.id, m.chat_id, m.sender_id, m.type, m.content, m.content_html, m.reply_to_id,
			m.forward_from_id, m.forward_from_chat_id, m.forward_from_sender_name, m.is_forwarded,
			m.is_edited, m.edited_at, m.is_deleted, m.deleted_at, m.deleted_for_everyone,
			m.expires_at, m.mention_everyone,
			m.media_id, m.media_url, m.media_type, m.media_content_type, m.media_filename,
			m.media_size, m.media_width, m.media_height, m.media_duration, m.media_thumbnail_url, m.media_waveform,
			m.sticker_pack_id, m.sticker_id, m.created_at
		FROM messages m
		JOIN starred_messages sm ON m.id = sm.message_id AND sm.user_id = ?
		WHERE m.is_deleted = FALSE
		ORDER BY sm.created_at DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgList []*messages.Message
	for rows.Next() {
		m := &messages.Message{}
		var content, contentHTML, replyToID, forwardFromID, forwardFromChatID, forwardFromSenderName sql.NullString
		var mediaID, mediaURL, mediaType, mediaContentType, mediaFilename, mediaThumbnailURL, mediaWaveform sql.NullString
		var stickerPackID, stickerID sql.NullString
		var mediaSize sql.NullInt64
		var mediaWidth, mediaHeight, mediaDuration sql.NullInt64
		var editedAt, deletedAt, expiresAt sql.NullTime
		if err := rows.Scan(
			&m.ID, &m.ChatID, &m.SenderID, &m.Type, &content, &contentHTML, &replyToID,
			&forwardFromID, &forwardFromChatID, &forwardFromSenderName, &m.IsForwarded,
			&m.IsEdited, &editedAt, &m.IsDeleted, &deletedAt, &m.DeletedForEveryone,
			&expiresAt, &m.MentionEveryone,
			&mediaID, &mediaURL, &mediaType, &mediaContentType, &mediaFilename,
			&mediaSize, &mediaWidth, &mediaHeight, &mediaDuration, &mediaThumbnailURL, &mediaWaveform,
			&stickerPackID, &stickerID, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.Content = content.String
		m.ContentHTML = contentHTML.String
		m.ReplyToID = replyToID.String
		m.ForwardFromID = forwardFromID.String
		m.ForwardFromChatID = forwardFromChatID.String
		m.ForwardFromSenderName = forwardFromSenderName.String
		m.MediaID = mediaID.String
		m.MediaURL = mediaURL.String
		m.MediaType = mediaType.String
		m.MediaContentType = mediaContentType.String
		m.MediaFilename = mediaFilename.String
		m.MediaSize = mediaSize.Int64
		m.MediaWidth = int(mediaWidth.Int64)
		m.MediaHeight = int(mediaHeight.Int64)
		m.MediaDuration = int(mediaDuration.Int64)
		m.MediaThumbnailURL = mediaThumbnailURL.String
		m.MediaWaveform = mediaWaveform.String
		m.StickerPackID = stickerPackID.String
		m.StickerID = stickerID.String
		if editedAt.Valid {
			m.EditedAt = &editedAt.Time
		}
		if deletedAt.Valid {
			m.DeletedAt = &deletedAt.Time
		}
		if expiresAt.Valid {
			m.ExpiresAt = &expiresAt.Time
		}
		msgList = append(msgList, m)
	}
	return msgList, rows.Err()
}

// InsertMedia inserts media.
func (s *MessagesStore) InsertMedia(ctx context.Context, media *messages.Media) error {
	query := `
		INSERT INTO message_media (id, message_id, type, filename, content_type, size, url, thumbnail_url,
			duration, width, height, waveform, is_voice_note, is_view_once, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		media.ID, media.MessageID, media.Type, media.Filename, media.ContentType, media.Size, media.URL, media.ThumbnailURL,
		media.Duration, media.Width, media.Height, media.Waveform, media.IsVoiceNote, media.IsViewOnce, media.CreatedAt,
	)
	return err
}

// GetMedia gets media for a message.
func (s *MessagesStore) GetMedia(ctx context.Context, messageID string) ([]*messages.Media, error) {
	query := `
		SELECT id, message_id, type, filename, content_type, size, url, thumbnail_url,
			duration, width, height, waveform, is_voice_note, is_view_once, view_count, created_at
		FROM message_media WHERE message_id = ?
	`
	rows, err := s.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mediaList []*messages.Media
	for rows.Next() {
		m := &messages.Media{}
		var filename, contentType, thumbnailURL, waveform sql.NullString
		var duration, width, height sql.NullInt64
		if err := rows.Scan(
			&m.ID, &m.MessageID, &m.Type, &filename, &contentType, &m.Size, &m.URL, &thumbnailURL,
			&duration, &width, &height, &waveform, &m.IsVoiceNote, &m.IsViewOnce, &m.ViewCount, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.Filename = filename.String
		m.ContentType = contentType.String
		m.ThumbnailURL = thumbnailURL.String
		m.Waveform = waveform.String
		m.Duration = int(duration.Int64)
		m.Width = int(width.Int64)
		m.Height = int(height.Int64)
		mediaList = append(mediaList, m)
	}
	return mediaList, rows.Err()
}

// IncrementViewCount increments view count.
func (s *MessagesStore) IncrementViewCount(ctx context.Context, mediaID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE message_media SET view_count = view_count + 1 WHERE id = ?", mediaID)
	return err
}

// InsertMention inserts a mention.
func (s *MessagesStore) InsertMention(ctx context.Context, messageID, userID string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO message_mentions (message_id, user_id) VALUES (?, ?) ON CONFLICT DO NOTHING", messageID, userID)
	return err
}

// GetMentions gets mentions for a message.
func (s *MessagesStore) GetMentions(ctx context.Context, messageID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT user_id FROM message_mentions WHERE message_id = ?", messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mentions []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		mentions = append(mentions, userID)
	}
	return mentions, rows.Err()
}

// InsertRecipient inserts a recipient.
func (s *MessagesStore) InsertRecipient(ctx context.Context, r *messages.Recipient) error {
	query := `INSERT INTO message_recipients (message_id, user_id, status) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, r.MessageID, r.UserID, r.Status)
	return err
}

// UpdateRecipientStatus updates recipient status.
func (s *MessagesStore) UpdateRecipientStatus(ctx context.Context, messageID, userID string, status messages.MessageStatus) error {
	now := time.Now()
	var query string
	var args []any
	switch status {
	case messages.StatusDelivered:
		query = "UPDATE message_recipients SET status = ?, delivered_at = ? WHERE message_id = ? AND user_id = ?"
		args = []any{status, now, messageID, userID}
	case messages.StatusRead:
		query = "UPDATE message_recipients SET status = ?, read_at = ? WHERE message_id = ? AND user_id = ?"
		args = []any{status, now, messageID, userID}
	default:
		query = "UPDATE message_recipients SET status = ? WHERE message_id = ? AND user_id = ?"
		args = []any{status, messageID, userID}
	}
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// GetRecipients gets recipients for a message.
func (s *MessagesStore) GetRecipients(ctx context.Context, messageID string) ([]*messages.Recipient, error) {
	query := `SELECT message_id, user_id, status, delivered_at, read_at FROM message_recipients WHERE message_id = ?`
	rows, err := s.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipients []*messages.Recipient
	for rows.Next() {
		r := &messages.Recipient{}
		var deliveredAt, readAt sql.NullTime
		if err := rows.Scan(&r.MessageID, &r.UserID, &r.Status, &deliveredAt, &readAt); err != nil {
			return nil, err
		}
		if deliveredAt.Valid {
			r.DeliveredAt = &deliveredAt.Time
		}
		if readAt.Valid {
			r.ReadAt = &readAt.Time
		}
		recipients = append(recipients, r)
	}
	return recipients, rows.Err()
}

// Pin pins a message.
func (s *MessagesStore) Pin(ctx context.Context, chatID, messageID, userID string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO pinned_messages (chat_id, message_id, pinned_by, pinned_at) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING", chatID, messageID, userID, time.Now())
	return err
}

// Unpin unpins a message.
func (s *MessagesStore) Unpin(ctx context.Context, chatID, messageID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM pinned_messages WHERE chat_id = ? AND message_id = ?", chatID, messageID)
	return err
}

// ListPinned lists pinned messages.
func (s *MessagesStore) ListPinned(ctx context.Context, chatID string) ([]*messages.Message, error) {
	query := `
		SELECT m.id, m.chat_id, m.sender_id, m.type, m.content, m.content_html, m.reply_to_id,
			m.forward_from_id, m.forward_from_chat_id, m.forward_from_sender_name, m.is_forwarded,
			m.is_edited, m.edited_at, m.is_deleted, m.deleted_at, m.deleted_for_everyone,
			m.expires_at, m.mention_everyone,
			m.media_id, m.media_url, m.media_type, m.media_content_type, m.media_filename,
			m.media_size, m.media_width, m.media_height, m.media_duration, m.media_thumbnail_url, m.media_waveform,
			m.sticker_pack_id, m.sticker_id, m.created_at
		FROM messages m
		JOIN pinned_messages pm ON m.id = pm.message_id AND pm.chat_id = ?
		WHERE m.is_deleted = FALSE
		ORDER BY pm.pinned_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgList []*messages.Message
	for rows.Next() {
		m := &messages.Message{}
		var content, contentHTML, replyToID, forwardFromID, forwardFromChatID, forwardFromSenderName sql.NullString
		var mediaID, mediaURL, mediaType, mediaContentType, mediaFilename, mediaThumbnailURL, mediaWaveform sql.NullString
		var stickerPackID, stickerID sql.NullString
		var mediaSize sql.NullInt64
		var mediaWidth, mediaHeight, mediaDuration sql.NullInt64
		var editedAt, deletedAt, expiresAt sql.NullTime
		if err := rows.Scan(
			&m.ID, &m.ChatID, &m.SenderID, &m.Type, &content, &contentHTML, &replyToID,
			&forwardFromID, &forwardFromChatID, &forwardFromSenderName, &m.IsForwarded,
			&m.IsEdited, &editedAt, &m.IsDeleted, &deletedAt, &m.DeletedForEveryone,
			&expiresAt, &m.MentionEveryone,
			&mediaID, &mediaURL, &mediaType, &mediaContentType, &mediaFilename,
			&mediaSize, &mediaWidth, &mediaHeight, &mediaDuration, &mediaThumbnailURL, &mediaWaveform,
			&stickerPackID, &stickerID, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.Content = content.String
		m.ContentHTML = contentHTML.String
		m.ReplyToID = replyToID.String
		m.ForwardFromID = forwardFromID.String
		m.ForwardFromChatID = forwardFromChatID.String
		m.ForwardFromSenderName = forwardFromSenderName.String
		m.MediaID = mediaID.String
		m.MediaURL = mediaURL.String
		m.MediaType = mediaType.String
		m.MediaContentType = mediaContentType.String
		m.MediaFilename = mediaFilename.String
		m.MediaSize = mediaSize.Int64
		m.MediaWidth = int(mediaWidth.Int64)
		m.MediaHeight = int(mediaHeight.Int64)
		m.MediaDuration = int(mediaDuration.Int64)
		m.MediaThumbnailURL = mediaThumbnailURL.String
		m.MediaWaveform = mediaWaveform.String
		m.StickerPackID = stickerPackID.String
		m.StickerID = stickerID.String
		if editedAt.Valid {
			m.EditedAt = &editedAt.Time
		}
		if deletedAt.Valid {
			m.DeletedAt = &deletedAt.Time
		}
		if expiresAt.Valid {
			m.ExpiresAt = &expiresAt.Time
		}
		msgList = append(msgList, m)
	}
	return msgList, rows.Err()
}
