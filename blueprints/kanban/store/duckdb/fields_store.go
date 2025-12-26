package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/kanban/feature/fields"
)

// FieldsStore handles field data access.
type FieldsStore struct {
	db *sql.DB
}

// NewFieldsStore creates a new fields store.
func NewFieldsStore(db *sql.DB) *FieldsStore {
	return &FieldsStore{db: db}
}

func (s *FieldsStore) Create(ctx context.Context, f *fields.Field) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO fields (id, project_id, key, name, kind, position, is_required, is_archived, settings_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, f.ID, f.ProjectID, f.Key, f.Name, f.Kind, f.Position, f.IsRequired, f.IsArchived, nullString(f.SettingsJSON))
	return err
}

func (s *FieldsStore) GetByID(ctx context.Context, id string) (*fields.Field, error) {
	f := &fields.Field{}
	var settingsJSON sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, key, name, kind, position, is_required, is_archived, settings_json
		FROM fields WHERE id = $1
	`, id).Scan(&f.ID, &f.ProjectID, &f.Key, &f.Name, &f.Kind, &f.Position, &f.IsRequired, &f.IsArchived, &settingsJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if settingsJSON.Valid {
		f.SettingsJSON = settingsJSON.String
	}
	return f, err
}

func (s *FieldsStore) GetByKey(ctx context.Context, projectID, key string) (*fields.Field, error) {
	f := &fields.Field{}
	var settingsJSON sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, key, name, kind, position, is_required, is_archived, settings_json
		FROM fields WHERE project_id = $1 AND key = $2
	`, projectID, key).Scan(&f.ID, &f.ProjectID, &f.Key, &f.Name, &f.Kind, &f.Position, &f.IsRequired, &f.IsArchived, &settingsJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if settingsJSON.Valid {
		f.SettingsJSON = settingsJSON.String
	}
	return f, err
}

func (s *FieldsStore) ListByProject(ctx context.Context, projectID string) ([]*fields.Field, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, key, name, kind, position, is_required, is_archived, settings_json
		FROM fields WHERE project_id = $1 AND is_archived = FALSE
		ORDER BY position
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*fields.Field
	for rows.Next() {
		f := &fields.Field{}
		var settingsJSON sql.NullString
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.Key, &f.Name, &f.Kind, &f.Position, &f.IsRequired, &f.IsArchived, &settingsJSON); err != nil {
			return nil, err
		}
		if settingsJSON.Valid {
			f.SettingsJSON = settingsJSON.String
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func (s *FieldsStore) Update(ctx context.Context, id string, in *fields.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE fields SET
			name = COALESCE($2, name),
			is_required = COALESCE($3, is_required),
			settings_json = COALESCE($4, settings_json)
		WHERE id = $1
	`, id, in.Name, in.IsRequired, in.SettingsJSON)
	return err
}

func (s *FieldsStore) UpdatePosition(ctx context.Context, id string, position int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE fields SET position = $2 WHERE id = $1
	`, id, position)
	return err
}

func (s *FieldsStore) Archive(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE fields SET is_archived = TRUE WHERE id = $1
	`, id)
	return err
}

func (s *FieldsStore) Unarchive(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE fields SET is_archived = FALSE WHERE id = $1
	`, id)
	return err
}

func (s *FieldsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM fields WHERE id = $1`, id)
	return err
}
