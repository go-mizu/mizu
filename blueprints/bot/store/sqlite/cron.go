package sqlite

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/bot/types"
	"github.com/google/uuid"
)

func (s *Store) ListCronJobs(ctx context.Context) ([]types.CronJob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, agent_id, enabled, schedule, session_target, wake_mode, payload, last_run_at, last_status, created_at, updated_at
		 FROM cron_jobs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []types.CronJob
	for rows.Next() {
		var j types.CronJob
		var lastRun sql.NullTime
		if err := rows.Scan(&j.ID, &j.Name, &j.Description, &j.AgentID, &j.Enabled,
			&j.Schedule, &j.SessionTarget, &j.WakeMode, &j.Payload,
			&lastRun, &j.LastStatus, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		if lastRun.Valid {
			j.LastRunAt = lastRun.Time
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (s *Store) GetCronJob(ctx context.Context, id string) (*types.CronJob, error) {
	var j types.CronJob
	var lastRun sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, agent_id, enabled, schedule, session_target, wake_mode, payload, last_run_at, last_status, created_at, updated_at
		 FROM cron_jobs WHERE id = ?`, id).
		Scan(&j.ID, &j.Name, &j.Description, &j.AgentID, &j.Enabled,
			&j.Schedule, &j.SessionTarget, &j.WakeMode, &j.Payload,
			&lastRun, &j.LastStatus, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if lastRun.Valid {
		j.LastRunAt = lastRun.Time
	}
	return &j, nil
}

func (s *Store) CreateCronJob(ctx context.Context, job *types.CronJob) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	if job.Schedule == "" {
		job.Schedule = "{}"
	}
	if job.Payload == "" {
		job.Payload = "{}"
	}
	if job.SessionTarget == "" {
		job.SessionTarget = "main"
	}
	if job.WakeMode == "" {
		job.WakeMode = "next-heartbeat"
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cron_jobs (id, name, description, agent_id, enabled, schedule, session_target, wake_mode, payload)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.Name, job.Description, job.AgentID, job.Enabled,
		job.Schedule, job.SessionTarget, job.WakeMode, job.Payload)
	return err
}

func (s *Store) UpdateCronJob(ctx context.Context, job *types.CronJob) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE cron_jobs SET name=?, description=?, agent_id=?, enabled=?, schedule=?, session_target=?, wake_mode=?, payload=?, last_run_at=?, last_status=?, updated_at=datetime('now')
		 WHERE id=?`,
		job.Name, job.Description, job.AgentID, job.Enabled,
		job.Schedule, job.SessionTarget, job.WakeMode, job.Payload,
		job.LastRunAt, job.LastStatus, job.ID)
	return err
}

func (s *Store) DeleteCronJob(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM cron_jobs WHERE id=?`, id)
	return err
}

func (s *Store) CreateCronRun(ctx context.Context, run *types.CronRun) error {
	if run.ID == "" {
		run.ID = uuid.New().String()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cron_runs (id, job_id, status, started_at)
		 VALUES (?, ?, ?, datetime('now'))`,
		run.ID, run.JobID, run.Status)
	return err
}

func (s *Store) UpdateCronRun(ctx context.Context, run *types.CronRun) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE cron_runs SET status=?, ended_at=datetime('now'), duration_ms=?, summary=?, error=?
		 WHERE id=?`,
		run.Status, run.DurationMs, run.Summary, run.Error, run.ID)
	return err
}

func (s *Store) ListCronRuns(ctx context.Context, jobID string, limit int) ([]types.CronRun, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, job_id, status, started_at, ended_at, duration_ms, summary, error
		 FROM cron_runs WHERE job_id=? ORDER BY started_at DESC LIMIT ?`,
		jobID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []types.CronRun
	for rows.Next() {
		var r types.CronRun
		var ended sql.NullTime
		if err := rows.Scan(&r.ID, &r.JobID, &r.Status, &r.StartedAt, &ended, &r.DurationMs, &r.Summary, &r.Error); err != nil {
			return nil, err
		}
		if ended.Valid {
			r.EndedAt = ended.Time
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}
