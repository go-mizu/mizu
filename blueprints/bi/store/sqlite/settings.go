package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// SettingsStore implements store.SettingsStore.
type SettingsStore struct {
	db *sql.DB
}

func (s *SettingsStore) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *SettingsStore) Set(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}

func (s *SettingsStore) List(ctx context.Context) ([]*store.Settings, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Settings
	for rows.Next() {
		var setting store.Settings
		if err := rows.Scan(&setting.Key, &setting.Value); err != nil {
			return nil, err
		}
		result = append(result, &setting)
	}
	return result, rows.Err()
}

func (s *SettingsStore) WriteAuditLog(ctx context.Context, log *store.AuditLog) error {
	if log.ID == "" {
		log.ID = generateID()
	}
	log.Timestamp = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, actor_id, actor_email, action, resource_type, resource_id, metadata, ip_address, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, log.ID, log.ActorID, log.ActorEmail, log.Action, log.ResourceType, log.ResourceID, toJSON(log.Metadata), log.IPAddress, log.Timestamp)
	return err
}

func (s *SettingsStore) ListAuditLogs(ctx context.Context, limit, offset int) ([]*store.AuditLog, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, actor_id, actor_email, action, resource_type, resource_id, metadata, ip_address, timestamp
		FROM audit_logs ORDER BY timestamp DESC LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.AuditLog
	for rows.Next() {
		var log store.AuditLog
		var metadata string
		if err := rows.Scan(&log.ID, &log.ActorID, &log.ActorEmail, &log.Action, &log.ResourceType, &log.ResourceID, &metadata, &log.IPAddress, &log.Timestamp); err != nil {
			return nil, err
		}
		fromJSON(metadata, &log.Metadata)
		result = append(result, &log)
	}
	return result, rows.Err()
}
