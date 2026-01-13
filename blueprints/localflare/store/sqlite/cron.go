package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

// CronStoreImpl implements store.CronStore.
type CronStoreImpl struct {
	db *sql.DB
}

func (s *CronStoreImpl) CreateTrigger(ctx context.Context, trigger *store.CronTrigger) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cron_triggers (id, script_name, cron, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		trigger.ID, trigger.ScriptName, trigger.Cron, trigger.Enabled, trigger.CreatedAt, trigger.UpdatedAt)
	return err
}

func (s *CronStoreImpl) GetTrigger(ctx context.Context, id string) (*store.CronTrigger, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, script_name, cron, enabled, created_at, updated_at
		FROM cron_triggers WHERE id = ?`, id)
	var t store.CronTrigger
	if err := row.Scan(&t.ID, &t.ScriptName, &t.Cron, &t.Enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *CronStoreImpl) ListTriggers(ctx context.Context) ([]*store.CronTrigger, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, script_name, cron, enabled, created_at, updated_at
		FROM cron_triggers ORDER BY script_name, cron`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []*store.CronTrigger
	for rows.Next() {
		var t store.CronTrigger
		if err := rows.Scan(&t.ID, &t.ScriptName, &t.Cron, &t.Enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		triggers = append(triggers, &t)
	}
	return triggers, rows.Err()
}

func (s *CronStoreImpl) ListTriggersByScript(ctx context.Context, scriptName string) ([]*store.CronTrigger, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, script_name, cron, enabled, created_at, updated_at
		FROM cron_triggers WHERE script_name = ? ORDER BY cron`, scriptName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []*store.CronTrigger
	for rows.Next() {
		var t store.CronTrigger
		if err := rows.Scan(&t.ID, &t.ScriptName, &t.Cron, &t.Enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		triggers = append(triggers, &t)
	}
	return triggers, rows.Err()
}

func (s *CronStoreImpl) UpdateTrigger(ctx context.Context, trigger *store.CronTrigger) error {
	trigger.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE cron_triggers SET script_name = ?, cron = ?, enabled = ?, updated_at = ?
		WHERE id = ?`,
		trigger.ScriptName, trigger.Cron, trigger.Enabled, trigger.UpdatedAt, trigger.ID)
	return err
}

func (s *CronStoreImpl) DeleteTrigger(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM cron_triggers WHERE id = ?`, id)
	return err
}

func (s *CronStoreImpl) RecordExecution(ctx context.Context, exec *store.CronExecution) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cron_executions (id, trigger_id, scheduled_at, started_at, finished_at, status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		exec.ID, exec.TriggerID, exec.ScheduledAt, exec.StartedAt, exec.FinishedAt, exec.Status, exec.Error)
	return err
}

func (s *CronStoreImpl) UpdateExecution(ctx context.Context, exec *store.CronExecution) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE cron_executions SET finished_at = ?, status = ?, error = ?
		WHERE id = ?`,
		exec.FinishedAt, exec.Status, exec.Error, exec.ID)
	return err
}

func (s *CronStoreImpl) GetRecentExecutions(ctx context.Context, triggerID string, limit int) ([]*store.CronExecution, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, trigger_id, scheduled_at, started_at, finished_at, status, error
		FROM cron_executions WHERE trigger_id = ? ORDER BY started_at DESC LIMIT ?`,
		triggerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []*store.CronExecution
	for rows.Next() {
		var e store.CronExecution
		var finishedAt sql.NullTime
		var errMsg sql.NullString
		if err := rows.Scan(&e.ID, &e.TriggerID, &e.ScheduledAt, &e.StartedAt,
			&finishedAt, &e.Status, &errMsg); err != nil {
			return nil, err
		}
		if finishedAt.Valid {
			e.FinishedAt = &finishedAt.Time
		}
		e.Error = errMsg.String
		executions = append(executions, &e)
	}
	return executions, rows.Err()
}

func (s *CronStoreImpl) GetDueTriggers(ctx context.Context, before time.Time) ([]*store.CronTrigger, error) {
	// This is a simplified implementation
	// In production, you'd want to parse cron expressions and find triggers that should run
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, script_name, cron, enabled, created_at, updated_at
		FROM cron_triggers WHERE enabled = 1 ORDER BY script_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []*store.CronTrigger
	for rows.Next() {
		var t store.CronTrigger
		if err := rows.Scan(&t.ID, &t.ScriptName, &t.Cron, &t.Enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		triggers = append(triggers, &t)
	}
	return triggers, rows.Err()
}

// Schema for Cron
const cronSchema = `
	-- Cron Triggers
	CREATE TABLE IF NOT EXISTS cron_triggers (
		id TEXT PRIMARY KEY,
		script_name TEXT NOT NULL,
		cron TEXT NOT NULL,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_cron_triggers_script ON cron_triggers(script_name);
	CREATE INDEX IF NOT EXISTS idx_cron_triggers_enabled ON cron_triggers(enabled);

	-- Cron Executions
	CREATE TABLE IF NOT EXISTS cron_executions (
		id TEXT PRIMARY KEY,
		trigger_id TEXT NOT NULL,
		scheduled_at DATETIME NOT NULL,
		started_at DATETIME NOT NULL,
		finished_at DATETIME,
		status TEXT NOT NULL,
		error TEXT,
		FOREIGN KEY (trigger_id) REFERENCES cron_triggers(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_cron_executions_trigger ON cron_executions(trigger_id);
	CREATE INDEX IF NOT EXISTS idx_cron_executions_started ON cron_executions(trigger_id, started_at);
`
