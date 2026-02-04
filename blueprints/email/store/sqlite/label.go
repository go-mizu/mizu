package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/email/types"
	"github.com/google/uuid"
)

// ListLabels returns all labels with unread and total counts.
func (s *Store) ListLabels(ctx context.Context) ([]types.Label, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			l.id, l.name, l.color, l.type, l.visible, l.position, l.created_at,
			COALESCE(counts.total_count, 0) AS total_count,
			COALESCE(counts.unread_count, 0) AS unread_count
		FROM labels l
		LEFT JOIN (
			SELECT
				el.label_id,
				COUNT(*) AS total_count,
				SUM(CASE WHEN e.is_read = 0 THEN 1 ELSE 0 END) AS unread_count
			FROM email_labels el
			JOIN emails e ON e.id = el.email_id
			GROUP BY el.label_id
		) counts ON counts.label_id = l.id
		ORDER BY l.position ASC, l.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}
	defer rows.Close()

	var labels []types.Label
	for rows.Next() {
		var label types.Label
		var createdAt string
		var visible int

		if err := rows.Scan(
			&label.ID, &label.Name, &label.Color, &label.Type,
			&visible, &label.Position, &createdAt,
			&label.TotalCount, &label.UnreadCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan label: %w", err)
		}

		label.Visible = visible == 1
		label.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		labels = append(labels, label)
	}

	if labels == nil {
		labels = []types.Label{}
	}

	return labels, nil
}

// CreateLabel creates a new label.
func (s *Store) CreateLabel(ctx context.Context, label *types.Label) error {
	if label.ID == "" {
		label.ID = uuid.New().String()
	}

	now := time.Now()
	if label.CreatedAt.IsZero() {
		label.CreatedAt = now
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO labels (id, name, color, type, visible, position, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`,
		label.ID, label.Name, label.Color, string(label.Type),
		boolToInt(label.Visible), label.Position, label.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create label: %w", err)
	}

	return nil
}

// UpdateLabel updates specific fields of a label.
func (s *Store) UpdateLabel(ctx context.Context, id string, updates map[string]any) error {
	var setClauses []string
	var args []any

	allowedFields := map[string]string{
		"name":     "name",
		"color":    "color",
		"visible":  "visible",
		"position": "position",
	}

	for key, col := range allowedFields {
		if val, ok := updates[key]; ok {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
			switch v := val.(type) {
			case bool:
				args = append(args, boolToInt(v))
			default:
				args = append(args, v)
			}
		}
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE labels SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update label: %w", err)
	}

	return nil
}

// DeleteLabel deletes a label and removes it from all emails.
func (s *Store) DeleteLabel(ctx context.Context, id string) error {
	// Check if it is a system label
	var labelType string
	err := s.db.QueryRowContext(ctx, "SELECT type FROM labels WHERE id = ?", id).Scan(&labelType)
	if err != nil {
		return fmt.Errorf("label not found: %w", err)
	}

	if labelType == string(types.LabelTypeSystem) {
		return fmt.Errorf("cannot delete system label")
	}

	// Delete label associations and the label itself
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	tx.ExecContext(ctx, "DELETE FROM email_labels WHERE label_id = ?", id)
	tx.ExecContext(ctx, "DELETE FROM labels WHERE id = ?", id)

	return tx.Commit()
}

// AddEmailLabel adds a label to an email.
func (s *Store) AddEmailLabel(ctx context.Context, emailID, labelID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO email_labels (email_id, label_id) VALUES (?, ?)
	`, emailID, labelID)
	if err != nil {
		return fmt.Errorf("failed to add label to email: %w", err)
	}
	return nil
}

// RemoveEmailLabel removes a label from an email.
func (s *Store) RemoveEmailLabel(ctx context.Context, emailID, labelID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM email_labels WHERE email_id = ? AND label_id = ?
	`, emailID, labelID)
	if err != nil {
		return fmt.Errorf("failed to remove label from email: %w", err)
	}
	return nil
}
