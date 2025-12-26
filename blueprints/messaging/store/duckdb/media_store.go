package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/messaging/feature/media"
)

// MediaStore implements media.Store.
type MediaStore struct {
	db *sql.DB
}

// NewMediaStore creates a new MediaStore.
func NewMediaStore(db *sql.DB) *MediaStore {
	return &MediaStore{db: db}
}

// Insert creates a new media record.
func (s *MediaStore) Insert(ctx context.Context, m *media.Media) error {
	query := `
		INSERT INTO media (id, user_id, message_id, type, filename, original_filename, content_type, size,
			url, thumbnail_url, width, height, duration, waveform, blurhash, is_view_once, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		m.ID, m.UserID, nullString(m.MessageID), m.Type, m.Filename, m.OriginalFilename, m.ContentType, m.Size,
		m.URL, nullString(m.ThumbnailURL), nullInt(m.Width), nullInt(m.Height), nullInt(m.Duration),
		nullString(m.Waveform), nullString(m.Blurhash), m.IsViewOnce, m.CreatedAt,
	)
	return err
}

// GetByID retrieves media by ID.
func (s *MediaStore) GetByID(ctx context.Context, id string) (*media.Media, error) {
	query := `
		SELECT id, user_id, message_id, type, filename, original_filename, content_type, size,
			url, thumbnail_url, width, height, duration, waveform, blurhash, is_view_once,
			view_count, viewed_at, created_at, updated_at, deleted_at
		FROM media WHERE id = ? AND deleted_at IS NULL
	`
	m, err := scanMedia(s.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, media.ErrNotFound
	}
	return m, err
}

func scanMedia(row interface{ Scan(...any) error }) (*media.Media, error) {
	m := &media.Media{}
	var messageID, thumbnailURL, waveform, blurhash sql.NullString
	var width, height, duration sql.NullInt64
	var viewedAt, updatedAt, deletedAt sql.NullTime
	err := row.Scan(
		&m.ID, &m.UserID, &messageID, &m.Type, &m.Filename, &m.OriginalFilename, &m.ContentType, &m.Size,
		&m.URL, &thumbnailURL, &width, &height, &duration, &waveform, &blurhash, &m.IsViewOnce,
		&m.ViewCount, &viewedAt, &m.CreatedAt, &updatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}
	m.MessageID = messageID.String
	m.ThumbnailURL = thumbnailURL.String
	m.Width = int(width.Int64)
	m.Height = int(height.Int64)
	m.Duration = int(duration.Int64)
	m.Waveform = waveform.String
	m.Blurhash = blurhash.String
	if viewedAt.Valid {
		m.ViewedAt = &viewedAt.Time
	}
	if updatedAt.Valid {
		m.UpdatedAt = &updatedAt.Time
	}
	if deletedAt.Valid {
		m.DeletedAt = &deletedAt.Time
	}
	return m, nil
}

// Delete soft deletes media.
func (s *MediaStore) Delete(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, "UPDATE media SET deleted_at = ? WHERE id = ?", now, id)
	return err
}

// Update updates media fields.
func (s *MediaStore) Update(ctx context.Context, id string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	setClauses := make([]string, 0, len(updates)+1)
	args := make([]any, 0, len(updates)+2)

	for k, v := range updates {
		setClauses = append(setClauses, k+" = ?")
		args = append(args, v)
	}

	// Always update updated_at
	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())

	// Add the WHERE clause parameter
	args = append(args, id)

	query := "UPDATE media SET "
	for i, clause := range setClauses {
		if i > 0 {
			query += ", "
		}
		query += clause
	}
	query += " WHERE id = ?"

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// ListByUser lists media uploaded by a user.
func (s *MediaStore) ListByUser(ctx context.Context, userID string, mediaType string, limit, offset int) ([]*media.Media, error) {
	query := `
		SELECT id, user_id, message_id, type, filename, original_filename, content_type, size,
			url, thumbnail_url, width, height, duration, waveform, blurhash, is_view_once,
			view_count, viewed_at, created_at, updated_at, deleted_at
		FROM media WHERE user_id = ? AND deleted_at IS NULL
	`
	args := []any{userID}

	if mediaType != "" {
		query += " AND type = ?"
		args = append(args, mediaType)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	return s.queryMedia(ctx, query, args...)
}

// ListByChat lists media in a chat.
func (s *MediaStore) ListByChat(ctx context.Context, chatID string, mediaType string, limit, offset int) ([]*media.Media, error) {
	query := `
		SELECT m.id, m.user_id, m.message_id, m.type, m.filename, m.original_filename, m.content_type, m.size,
			m.url, m.thumbnail_url, m.width, m.height, m.duration, m.waveform, m.blurhash, m.is_view_once,
			m.view_count, m.viewed_at, m.created_at, m.updated_at, m.deleted_at
		FROM media m
		JOIN messages msg ON m.message_id = msg.id
		WHERE msg.chat_id = ? AND m.deleted_at IS NULL AND msg.is_deleted = FALSE
	`
	args := []any{chatID}

	if mediaType != "" {
		query += " AND m.type = ?"
		args = append(args, mediaType)
	}

	query += " ORDER BY m.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	return s.queryMedia(ctx, query, args...)
}

// ListByMessage lists media attached to a message.
func (s *MediaStore) ListByMessage(ctx context.Context, messageID string) ([]*media.Media, error) {
	query := `
		SELECT id, user_id, message_id, type, filename, original_filename, content_type, size,
			url, thumbnail_url, width, height, duration, waveform, blurhash, is_view_once,
			view_count, viewed_at, created_at, updated_at, deleted_at
		FROM media WHERE message_id = ? AND deleted_at IS NULL
		ORDER BY created_at ASC
	`
	return s.queryMedia(ctx, query, messageID)
}

// AttachToMessage attaches media to a message.
func (s *MediaStore) AttachToMessage(ctx context.Context, mediaID, messageID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE media SET message_id = ?, updated_at = ? WHERE id = ?",
		messageID, time.Now(), mediaID)
	return err
}

// IncrementViewCount increments view count for view-once media.
func (s *MediaStore) IncrementViewCount(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		"UPDATE media SET view_count = view_count + 1, viewed_at = ? WHERE id = ?",
		now, id)
	return err
}

func (s *MediaStore) queryMedia(ctx context.Context, query string, args ...any) ([]*media.Media, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*media.Media
	for rows.Next() {
		m := &media.Media{}
		var messageID, thumbnailURL, waveform, blurhash sql.NullString
		var width, height, duration sql.NullInt64
		var viewedAt, updatedAt, deletedAt sql.NullTime
		if err := rows.Scan(
			&m.ID, &m.UserID, &messageID, &m.Type, &m.Filename, &m.OriginalFilename, &m.ContentType, &m.Size,
			&m.URL, &thumbnailURL, &width, &height, &duration, &waveform, &blurhash, &m.IsViewOnce,
			&m.ViewCount, &viewedAt, &m.CreatedAt, &updatedAt, &deletedAt,
		); err != nil {
			return nil, err
		}
		m.MessageID = messageID.String
		m.ThumbnailURL = thumbnailURL.String
		m.Width = int(width.Int64)
		m.Height = int(height.Int64)
		m.Duration = int(duration.Int64)
		m.Waveform = waveform.String
		m.Blurhash = blurhash.String
		if viewedAt.Valid {
			m.ViewedAt = &viewedAt.Time
		}
		if updatedAt.Valid {
			m.UpdatedAt = &updatedAt.Time
		}
		if deletedAt.Valid {
			m.DeletedAt = &deletedAt.Time
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// Helper functions
func nullInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}
