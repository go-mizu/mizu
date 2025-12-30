package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/cms/feature/settings"
)

// SettingsStore handles settings data access.
type SettingsStore struct {
	db *sql.DB
}

// NewSettingsStore creates a new settings store.
func NewSettingsStore(db *sql.DB) *SettingsStore {
	return &SettingsStore{db: db}
}

func (s *SettingsStore) Get(ctx context.Context, key string) (*settings.Setting, error) {
	return s.scanSetting(s.db.QueryRowContext(ctx, `
		SELECT id, key, value, value_type, group_name, description, is_public, created_at, updated_at
		FROM settings WHERE key = $1
	`, key))
}

func (s *SettingsStore) GetByGroup(ctx context.Context, group string) ([]*settings.Setting, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, key, value, value_type, group_name, description, is_public, created_at, updated_at
		FROM settings WHERE group_name = $1
		ORDER BY key
	`, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*settings.Setting
	for rows.Next() {
		setting, err := s.scanSettingRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, setting)
	}
	return list, rows.Err()
}

func (s *SettingsStore) GetAll(ctx context.Context) ([]*settings.Setting, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, key, value, value_type, group_name, description, is_public, created_at, updated_at
		FROM settings
		ORDER BY group_name, key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*settings.Setting
	for rows.Next() {
		setting, err := s.scanSettingRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, setting)
	}
	return list, rows.Err()
}

func (s *SettingsStore) GetPublic(ctx context.Context) ([]*settings.Setting, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, key, value, value_type, group_name, description, is_public, created_at, updated_at
		FROM settings WHERE is_public = true
		ORDER BY key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*settings.Setting
	for rows.Next() {
		setting, err := s.scanSettingRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, setting)
	}
	return list, rows.Err()
}

func (s *SettingsStore) Set(ctx context.Context, setting *settings.Setting) error {
	// Upsert
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (id, key, value, value_type, group_name, description, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			value_type = EXCLUDED.value_type,
			group_name = EXCLUDED.group_name,
			description = EXCLUDED.description,
			is_public = EXCLUDED.is_public,
			updated_at = EXCLUDED.updated_at
	`, setting.ID, setting.Key, setting.Value, setting.ValueType, nullString(setting.GroupName), nullString(setting.Description), setting.IsPublic, setting.CreatedAt, setting.UpdatedAt)
	return err
}

func (s *SettingsStore) Delete(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM settings WHERE key = $1`, key)
	return err
}

func (s *SettingsStore) scanSetting(row *sql.Row) (*settings.Setting, error) {
	setting := &settings.Setting{}
	var value, groupName, description sql.NullString
	err := row.Scan(&setting.ID, &setting.Key, &value, &setting.ValueType, &groupName, &description, &setting.IsPublic, &setting.CreatedAt, &setting.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	setting.Value = value.String
	setting.GroupName = groupName.String
	setting.Description = description.String
	return setting, nil
}

func (s *SettingsStore) scanSettingRow(rows *sql.Rows) (*settings.Setting, error) {
	setting := &settings.Setting{}
	var value, groupName, description sql.NullString
	err := rows.Scan(&setting.ID, &setting.Key, &value, &setting.ValueType, &groupName, &description, &setting.IsPublic, &setting.CreatedAt, &setting.UpdatedAt)
	if err != nil {
		return nil, err
	}
	setting.Value = value.String
	setting.GroupName = groupName.String
	setting.Description = description.String
	return setting, nil
}
