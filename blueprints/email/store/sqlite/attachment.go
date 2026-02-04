package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/email/types"
	"github.com/google/uuid"
)

// ListAttachments returns all attachments for an email.
func (s *Store) ListAttachments(ctx context.Context, emailID string) ([]types.Attachment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email_id, filename, content_type, size_bytes, created_at
		FROM attachments
		WHERE email_id = ?
		ORDER BY created_at ASC
	`, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments: %w", err)
	}
	defer rows.Close()

	var attachments []types.Attachment
	for rows.Next() {
		var a types.Attachment
		var createdAt string
		if err := rows.Scan(&a.ID, &a.EmailID, &a.Filename, &a.ContentType, &a.SizeBytes, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		attachments = append(attachments, a)
	}

	if attachments == nil {
		attachments = []types.Attachment{}
	}
	return attachments, nil
}

// GetAttachment returns an attachment with its data.
func (s *Store) GetAttachment(ctx context.Context, id string) (*types.Attachment, []byte, error) {
	var a types.Attachment
	var data []byte
	var createdAt string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, email_id, filename, content_type, size_bytes, data, created_at
		FROM attachments
		WHERE id = ?
	`, id).Scan(&a.ID, &a.EmailID, &a.Filename, &a.ContentType, &a.SizeBytes, &data, &createdAt)
	if err != nil {
		return nil, nil, fmt.Errorf("attachment not found: %w", err)
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &a, data, nil
}

// CreateAttachment stores an attachment.
func (s *Store) CreateAttachment(ctx context.Context, attachment *types.Attachment, data []byte) error {
	if attachment.ID == "" {
		attachment.ID = uuid.New().String()
	}
	now := time.Now()
	if attachment.CreatedAt.IsZero() {
		attachment.CreatedAt = now
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO attachments (id, email_id, filename, content_type, size_bytes, data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, attachment.ID, attachment.EmailID, attachment.Filename, attachment.ContentType,
		attachment.SizeBytes, data, attachment.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to create attachment: %w", err)
	}

	// Update has_attachments on the email
	s.db.ExecContext(ctx, "UPDATE emails SET has_attachments = 1 WHERE id = ?", attachment.EmailID)

	return nil
}

// DeleteAttachment removes an attachment.
func (s *Store) DeleteAttachment(ctx context.Context, id string) error {
	// Get email_id before deleting
	var emailID string
	s.db.QueryRowContext(ctx, "SELECT email_id FROM attachments WHERE id = ?", id).Scan(&emailID)

	_, err := s.db.ExecContext(ctx, "DELETE FROM attachments WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	// Check if email still has attachments
	if emailID != "" {
		var count int
		s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM attachments WHERE email_id = ?", emailID).Scan(&count)
		if count == 0 {
			s.db.ExecContext(ctx, "UPDATE emails SET has_attachments = 0 WHERE id = ?", emailID)
		}
	}

	return nil
}
