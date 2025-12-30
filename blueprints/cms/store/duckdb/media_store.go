package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/feature/media"
)

// MediaStore handles media data access.
type MediaStore struct {
	db *sql.DB
}

// NewMediaStore creates a new media store.
func NewMediaStore(db *sql.DB) *MediaStore {
	return &MediaStore{db: db}
}

func (s *MediaStore) Create(ctx context.Context, m *media.Media) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO media (id, uploader_id, filename, original_filename, mime_type, file_size, storage_path, storage_provider, url, alt_text, caption, title, description, width, height, duration, meta, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`, m.ID, m.UploaderID, m.Filename, m.OriginalFilename, m.MimeType, m.FileSize, m.StoragePath, m.StorageProvider, m.URL, nullString(m.AltText), nullString(m.Caption), nullString(m.Title), nullString(m.Description), nullInt(m.Width), nullInt(m.Height), nullInt(m.Duration), nullString(m.Meta), m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *MediaStore) GetByID(ctx context.Context, id string) (*media.Media, error) {
	return s.scanMedia(s.db.QueryRowContext(ctx, `
		SELECT id, uploader_id, filename, original_filename, mime_type, file_size, storage_path, storage_provider, url, alt_text, caption, title, description, width, height, duration, meta, created_at, updated_at
		FROM media WHERE id = $1
	`, id))
}

func (s *MediaStore) GetByFilename(ctx context.Context, filename string) (*media.Media, error) {
	return s.scanMedia(s.db.QueryRowContext(ctx, `
		SELECT id, uploader_id, filename, original_filename, mime_type, file_size, storage_path, storage_provider, url, alt_text, caption, title, description, width, height, duration, meta, created_at, updated_at
		FROM media WHERE filename = $1
	`, filename))
}

func (s *MediaStore) List(ctx context.Context, in *media.ListIn) ([]*media.Media, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	if in.UploaderID != "" {
		conditions = append(conditions, fmt.Sprintf("uploader_id = $%d", argNum))
		args = append(args, in.UploaderID)
		argNum++
	}
	if in.MimeType != "" {
		conditions = append(conditions, fmt.Sprintf("mime_type LIKE $%d", argNum))
		args = append(args, in.MimeType+"%")
		argNum++
	}
	if in.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(filename ILIKE $%d OR original_filename ILIKE $%d OR title ILIKE $%d)", argNum, argNum, argNum))
		args = append(args, "%"+in.Search+"%")
		argNum++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM media %s", where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Determine order
	orderBy := in.OrderBy
	if orderBy == "" {
		orderBy = "created_at"
	}
	order := in.Order
	if order == "" {
		order = "DESC"
	}

	// Get items
	args = append(args, in.Limit, in.Offset)
	query := fmt.Sprintf(`
		SELECT id, uploader_id, filename, original_filename, mime_type, file_size, storage_path, storage_provider, url, alt_text, caption, title, description, width, height, duration, meta, created_at, updated_at
		FROM media %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, orderBy, order, argNum, argNum+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*media.Media
	for rows.Next() {
		m, err := s.scanMediaRow(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, m)
	}
	return list, total, rows.Err()
}

func (s *MediaStore) Update(ctx context.Context, id string, in *media.UpdateIn) error {
	var sets []string
	var args []any
	argNum := 1

	if in.AltText != nil {
		sets = append(sets, fmt.Sprintf("alt_text = $%d", argNum))
		args = append(args, nullString(*in.AltText))
		argNum++
	}
	if in.Caption != nil {
		sets = append(sets, fmt.Sprintf("caption = $%d", argNum))
		args = append(args, nullString(*in.Caption))
		argNum++
	}
	if in.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", argNum))
		args = append(args, nullString(*in.Title))
		argNum++
	}
	if in.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argNum))
		args = append(args, nullString(*in.Description))
		argNum++
	}
	if in.Meta != nil {
		sets = append(sets, fmt.Sprintf("meta = $%d", argNum))
		args = append(args, nullString(*in.Meta))
		argNum++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++

	args = append(args, id)
	query := fmt.Sprintf("UPDATE media SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *MediaStore) Delete(ctx context.Context, id string) error {
	// Remove from post_media relationships
	s.db.ExecContext(ctx, `DELETE FROM post_media WHERE media_id = $1`, id)
	_, err := s.db.ExecContext(ctx, `DELETE FROM media WHERE id = $1`, id)
	return err
}

func (s *MediaStore) scanMedia(row *sql.Row) (*media.Media, error) {
	m := &media.Media{}
	var altText, caption, title, description, meta sql.NullString
	var width, height, duration sql.NullInt64
	err := row.Scan(&m.ID, &m.UploaderID, &m.Filename, &m.OriginalFilename, &m.MimeType, &m.FileSize, &m.StoragePath, &m.StorageProvider, &m.URL, &altText, &caption, &title, &description, &width, &height, &duration, &meta, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.AltText = altText.String
	m.Caption = caption.String
	m.Title = title.String
	m.Description = description.String
	m.Meta = meta.String
	m.Width = int(width.Int64)
	m.Height = int(height.Int64)
	m.Duration = int(duration.Int64)
	return m, nil
}

func (s *MediaStore) scanMediaRow(rows *sql.Rows) (*media.Media, error) {
	m := &media.Media{}
	var altText, caption, title, description, meta sql.NullString
	var width, height, duration sql.NullInt64
	err := rows.Scan(&m.ID, &m.UploaderID, &m.Filename, &m.OriginalFilename, &m.MimeType, &m.FileSize, &m.StoragePath, &m.StorageProvider, &m.URL, &altText, &caption, &title, &description, &width, &height, &duration, &meta, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	m.AltText = altText.String
	m.Caption = caption.String
	m.Title = title.String
	m.Description = description.String
	m.Meta = meta.String
	m.Width = int(width.Int64)
	m.Height = int(height.Int64)
	m.Duration = int(duration.Int64)
	return m, nil
}

func nullInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}
