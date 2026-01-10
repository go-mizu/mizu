package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/table/feature/attachments"
)

// AttachmentsStore provides DuckDB-based attachment storage.
type AttachmentsStore struct {
	db *sql.DB
}

// NewAttachmentsStore creates a new attachments store.
func NewAttachmentsStore(db *sql.DB) *AttachmentsStore {
	return &AttachmentsStore{db: db}
}

// Create creates a new attachment.
func (s *AttachmentsStore) Create(ctx context.Context, att *attachments.Attachment) error {
	att.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO attachments (id, record_id, field_id, filename, size, mime_type, url, thumbnail_url, width, height, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, att.ID, att.RecordID, att.FieldID, att.Filename, att.Size, att.MimeType, att.URL, att.ThumbnailURL, att.Width, att.Height, att.CreatedAt)
	return err
}

// GetByID retrieves an attachment by ID.
func (s *AttachmentsStore) GetByID(ctx context.Context, id string) (*attachments.Attachment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, record_id, field_id, filename, size, mime_type, url, thumbnail_url, width, height, created_at
		FROM attachments WHERE id = $1
	`, id)
	return s.scanAttachment(row)
}

// Delete deletes an attachment.
func (s *AttachmentsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM attachments WHERE id = $1`, id)
	return err
}

// ListByRecord lists all attachments for a record/field.
func (s *AttachmentsStore) ListByRecord(ctx context.Context, recordID, fieldID string) ([]*attachments.Attachment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, record_id, field_id, filename, size, mime_type, url, thumbnail_url, width, height, created_at
		FROM attachments WHERE record_id = $1 AND field_id = $2
		ORDER BY created_at ASC
	`, recordID, fieldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var atts []*attachments.Attachment
	for rows.Next() {
		att, err := s.scanAttachmentRows(rows)
		if err != nil {
			return nil, err
		}
		atts = append(atts, att)
	}
	return atts, rows.Err()
}

// DeleteByRecord deletes all attachments for a record.
func (s *AttachmentsStore) DeleteByRecord(ctx context.Context, recordID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM attachments WHERE record_id = $1`, recordID)
	return err
}

func (s *AttachmentsStore) scanAttachment(row *sql.Row) (*attachments.Attachment, error) {
	att := &attachments.Attachment{}
	var thumbnailURL sql.NullString
	var width, height sql.NullInt64

	err := row.Scan(&att.ID, &att.RecordID, &att.FieldID, &att.Filename, &att.Size, &att.MimeType, &att.URL, &thumbnailURL, &width, &height, &att.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, attachments.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if thumbnailURL.Valid {
		att.ThumbnailURL = thumbnailURL.String
	}
	if width.Valid {
		att.Width = int(width.Int64)
	}
	if height.Valid {
		att.Height = int(height.Int64)
	}

	return att, nil
}

func (s *AttachmentsStore) scanAttachmentRows(rows *sql.Rows) (*attachments.Attachment, error) {
	att := &attachments.Attachment{}
	var thumbnailURL sql.NullString
	var width, height sql.NullInt64

	err := rows.Scan(&att.ID, &att.RecordID, &att.FieldID, &att.Filename, &att.Size, &att.MimeType, &att.URL, &thumbnailURL, &width, &height, &att.CreatedAt)
	if err != nil {
		return nil, err
	}

	if thumbnailURL.Valid {
		att.ThumbnailURL = thumbnailURL.String
	}
	if width.Valid {
		att.Width = int(width.Int64)
	}
	if height.Valid {
		att.Height = int(height.Int64)
	}

	return att, nil
}
