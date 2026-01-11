package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// FieldsStore provides PostgreSQL-based field storage.
type FieldsStore struct {
	db *sql.DB
}

// NewFieldsStore creates a new fields store.
func NewFieldsStore(db *sql.DB) *FieldsStore {
	return &FieldsStore{db: db}
}

// Create creates a new field.
func (s *FieldsStore) Create(ctx context.Context, field *fields.Field) error {
	now := time.Now()
	field.CreatedAt = now
	field.UpdatedAt = now
	if field.Width == 0 {
		field.Width = 200
	}

	// Get max position
	var maxPos sql.NullInt64
	s.db.QueryRowContext(ctx, `SELECT MAX(position) FROM fields WHERE table_id = $1`, field.TableID).Scan(&maxPos)
	if maxPos.Valid {
		field.Position = int(maxPos.Int64) + 1
	}

	var optionsJSON []byte
	if field.Options != nil {
		optionsJSON = field.Options
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO fields (id, table_id, name, type, description, options, position, is_primary, is_computed, is_hidden, width, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, field.ID, field.TableID, field.Name, field.Type, nullString(field.Description), optionsJSON, field.Position, field.IsPrimary, field.IsComputed, field.IsHidden, field.Width, field.CreatedBy, field.CreatedAt, field.UpdatedAt)
	return err
}

// GetByID retrieves a field by ID.
func (s *FieldsStore) GetByID(ctx context.Context, id string) (*fields.Field, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, table_id, name, type, description, options, position, is_primary, is_computed, is_hidden, width, created_by, created_at, updated_at
		FROM fields WHERE id = $1
	`, id)
	return s.scanField(row)
}

// Update updates a field.
func (s *FieldsStore) Update(ctx context.Context, field *fields.Field) error {
	field.UpdatedAt = time.Now()

	var optionsJSON []byte
	if field.Options != nil {
		optionsJSON = field.Options
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE fields SET
			name = $1, type = $2, description = $3, options = $4, position = $5, is_primary = $6, is_computed = $7, is_hidden = $8, width = $9, updated_at = $10
		WHERE id = $11
	`, field.Name, field.Type, nullString(field.Description), optionsJSON, field.Position, field.IsPrimary, field.IsComputed, field.IsHidden, field.Width, field.UpdatedAt, field.ID)
	return err
}

// Delete deletes a field.
func (s *FieldsStore) Delete(ctx context.Context, id string) error {
	// Delete select choices
	_, _ = s.db.ExecContext(ctx, `DELETE FROM select_choices WHERE field_id = $1`, id)
	// Delete attachments and record links tied to the field
	_, _ = s.db.ExecContext(ctx, `DELETE FROM attachments WHERE field_id = $1`, id)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM record_links WHERE source_field_id = $1`, id)

	_, err := s.db.ExecContext(ctx, `DELETE FROM fields WHERE id = $1`, id)
	return err
}

// ListByTable lists all fields in a table.
func (s *FieldsStore) ListByTable(ctx context.Context, tableID string) ([]*fields.Field, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, table_id, name, type, description, options, position, is_primary, is_computed, is_hidden, width, created_by, created_at, updated_at
		FROM fields WHERE table_id = $1
		ORDER BY position ASC
	`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fieldList []*fields.Field
	for rows.Next() {
		field, err := s.scanFieldRows(rows)
		if err != nil {
			return nil, err
		}
		fieldList = append(fieldList, field)
	}
	return fieldList, rows.Err()
}

// Reorder reorders fields efficiently using a single UPDATE with CASE.
func (s *FieldsStore) Reorder(ctx context.Context, tableID string, fieldIDs []string) error {
	if len(fieldIDs) == 0 {
		return nil
	}

	// Build CASE statement for batch update
	// UPDATE fields SET position = CASE id WHEN 'id1' THEN 0 WHEN 'id2' THEN 1 ... END WHERE table_id = ? AND id IN (...)
	var caseBuilder strings.Builder
	caseBuilder.WriteString("UPDATE fields SET position = CASE id ")

	args := make([]any, 0, len(fieldIDs)+1)
	placeholders := make([]string, len(fieldIDs))

	for i, id := range fieldIDs {
		caseBuilder.WriteString(fmt.Sprintf("WHEN $%d THEN %d ", i+1, i))
		args = append(args, id)
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	caseBuilder.WriteString(fmt.Sprintf("END WHERE table_id = $%d AND id IN (%s)",
		len(fieldIDs)+1, strings.Join(placeholders, ", ")))
	args = append(args, tableID)

	_, err := s.db.ExecContext(ctx, caseBuilder.String(), args...)
	return err
}

// AddSelectChoice adds a choice to a select field.
func (s *FieldsStore) AddSelectChoice(ctx context.Context, choice *fields.SelectChoice) error {
	if choice.ID == "" {
		choice.ID = ulid.New()
	}
	if choice.Color == "" {
		choice.Color = "#6B7280"
	}

	// Get max position
	var maxPos sql.NullInt64
	s.db.QueryRowContext(ctx, `SELECT MAX(position) FROM select_choices WHERE field_id = $1`, choice.FieldID).Scan(&maxPos)
	if maxPos.Valid {
		choice.Position = int(maxPos.Int64) + 1
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO select_choices (id, field_id, name, color, position)
		VALUES ($1, $2, $3, $4, $5)
	`, choice.ID, choice.FieldID, choice.Name, choice.Color, choice.Position)
	return err
}

// UpdateSelectChoice updates a select choice.
func (s *FieldsStore) UpdateSelectChoice(ctx context.Context, choiceID string, in fields.UpdateChoiceIn) error {
	if in.Name != "" {
		_, err := s.db.ExecContext(ctx, `UPDATE select_choices SET name = $1 WHERE id = $2`, in.Name, choiceID)
		if err != nil {
			return err
		}
	}
	if in.Color != "" {
		_, err := s.db.ExecContext(ctx, `UPDATE select_choices SET color = $1 WHERE id = $2`, in.Color, choiceID)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteSelectChoice deletes a select choice.
func (s *FieldsStore) DeleteSelectChoice(ctx context.Context, choiceID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM select_choices WHERE id = $1`, choiceID)
	return err
}

// ListSelectChoices lists all choices for a select field.
func (s *FieldsStore) ListSelectChoices(ctx context.Context, fieldID string) ([]*fields.SelectChoice, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, field_id, name, color, position
		FROM select_choices WHERE field_id = $1
		ORDER BY position ASC
	`, fieldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var choices []*fields.SelectChoice
	for rows.Next() {
		choice := &fields.SelectChoice{}
		if err := rows.Scan(&choice.ID, &choice.FieldID, &choice.Name, &choice.Color, &choice.Position); err != nil {
			return nil, err
		}
		choices = append(choices, choice)
	}
	return choices, rows.Err()
}

// GetSelectChoice retrieves a select choice by ID.
func (s *FieldsStore) GetSelectChoice(ctx context.Context, choiceID string) (*fields.SelectChoice, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, field_id, name, color, position
		FROM select_choices WHERE id = $1
	`, choiceID)

	choice := &fields.SelectChoice{}
	err := row.Scan(&choice.ID, &choice.FieldID, &choice.Name, &choice.Color, &choice.Position)
	if err == sql.ErrNoRows {
		return nil, fields.ErrChoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	return choice, nil
}

func (s *FieldsStore) scanField(row *sql.Row) (*fields.Field, error) {
	field := &fields.Field{}
	var description sql.NullString
	var optionsJSON []byte

	err := row.Scan(&field.ID, &field.TableID, &field.Name, &field.Type, &description, &optionsJSON, &field.Position, &field.IsPrimary, &field.IsComputed, &field.IsHidden, &field.Width, &field.CreatedBy, &field.CreatedAt, &field.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fields.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if description.Valid {
		field.Description = description.String
	}
	if len(optionsJSON) > 0 {
		field.Options = json.RawMessage(optionsJSON)
	}
	return field, nil
}

func (s *FieldsStore) scanFieldRows(rows *sql.Rows) (*fields.Field, error) {
	field := &fields.Field{}
	var description sql.NullString
	var optionsJSON []byte

	err := rows.Scan(&field.ID, &field.TableID, &field.Name, &field.Type, &description, &optionsJSON, &field.Position, &field.IsPrimary, &field.IsComputed, &field.IsHidden, &field.Width, &field.CreatedBy, &field.CreatedAt, &field.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		field.Description = description.String
	}
	if len(optionsJSON) > 0 {
		field.Options = json.RawMessage(optionsJSON)
	}
	return field, nil
}
