package facebook

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

type State struct {
	db   *sql.DB
	path string
}

func OpenState(path string) (*State, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb state %s: %w", path, err)
	}
	s := &State{db: db, path: path}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *State) initSchema() error {
	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS queue_id_seq`,
		`CREATE TABLE IF NOT EXISTS queue (
			id           BIGINT DEFAULT nextval('queue_id_seq') PRIMARY KEY,
			url          VARCHAR UNIQUE NOT NULL,
			entity_type  VARCHAR NOT NULL,
			priority     INTEGER DEFAULT 0,
			status       VARCHAR DEFAULT 'pending',
			attempts     INTEGER DEFAULT 0,
			last_attempt TIMESTAMP,
			error        VARCHAR,
			created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_queue_status_priority ON queue(status, priority DESC, created_at)`,
		`CREATE TABLE IF NOT EXISTS jobs (
			job_id       VARCHAR PRIMARY KEY,
			name         VARCHAR,
			type         VARCHAR,
			status       VARCHAR,
			started_at   TIMESTAMP,
			completed_at TIMESTAMP,
			config       VARCHAR,
			stats        VARCHAR
		)`,
		`CREATE TABLE IF NOT EXISTS visited (
			url         VARCHAR PRIMARY KEY,
			fetched_at  TIMESTAMP,
			status_code INTEGER,
			entity_type VARCHAR
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("init state schema: %w", err)
		}
	}
	return nil
}

func (s *State) Enqueue(url, entityType string, priority int) error {
	url = NormalizeURL(url, true)
	_, err := s.db.Exec(`INSERT OR IGNORE INTO queue (url, entity_type, priority) VALUES (?, ?, ?)`, url, entityType, priority)
	return err
}

func (s *State) EnqueueBatch(items []QueueItem) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, it := range items {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO queue (url, entity_type, priority) VALUES (?, ?, ?)`,
			NormalizeURL(it.URL, true), it.EntityType, it.Priority,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *State) Pop(n int) ([]QueueItem, error) {
	rows, err := s.db.Query(`
		UPDATE queue
		SET status = 'in_progress', last_attempt = NOW(), attempts = attempts + 1
		WHERE id IN (
			SELECT id FROM queue
			WHERE status = 'pending'
			ORDER BY priority DESC, created_at
			LIMIT ?
		)
		RETURNING id, url, entity_type, priority`, n)
	if err != nil {
		return nil, fmt.Errorf("pop queue: %w", err)
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		var it QueueItem
		if err := rows.Scan(&it.ID, &it.URL, &it.EntityType, &it.Priority); err != nil {
			return items, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *State) Done(url, entityType string, statusCode int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE queue SET status = 'done' WHERE url = ?`, NormalizeURL(url, true)); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR REPLACE INTO visited (url, fetched_at, status_code, entity_type) VALUES (?, ?, ?, ?)`,
		NormalizeURL(url, true), time.Now(), statusCode, entityType,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *State) Fail(url, errMsg string) error {
	_, err := s.db.Exec(`
		UPDATE queue
		SET error = ?, status = CASE WHEN attempts >= 3 THEN 'failed' ELSE 'pending' END
		WHERE url = ?`, errMsg, NormalizeURL(url, true))
	return err
}

func (s *State) QueueStats() (pending, inProgress, done, failed int64) {
	s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status = 'pending'`).Scan(&pending)
	s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status = 'in_progress'`).Scan(&inProgress)
	s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status = 'done'`).Scan(&done)
	s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status = 'failed'`).Scan(&failed)
	return
}

func (s *State) ListQueue(status string, limit int) ([]QueueItem, error) {
	rows, err := s.db.Query(`SELECT id, url, entity_type, priority FROM queue WHERE status = ? ORDER BY created_at DESC LIMIT ?`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		var it QueueItem
		if err := rows.Scan(&it.ID, &it.URL, &it.EntityType, &it.Priority); err != nil {
			return items, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *State) CreateJob(id, name, jobType string) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO jobs (job_id, name, type, status, started_at) VALUES (?,?,?,?,?)`,
		id, name, jobType, "running", time.Now(),
	)
	return err
}

func (s *State) UpdateJob(id, status, stats string) error {
	_, err := s.db.Exec(`UPDATE jobs SET status = ?, stats = ?, completed_at = ? WHERE job_id = ?`,
		status, stats, time.Now(), id,
	)
	return err
}

func (s *State) ListJobs(limit int) ([]JobRecord, error) {
	rows, err := s.db.Query(`SELECT job_id, name, type, status, started_at, completed_at FROM jobs ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []JobRecord
	for rows.Next() {
		var j JobRecord
		var completed sql.NullTime
		if err := rows.Scan(&j.JobID, &j.Name, &j.Type, &j.Status, &j.StartedAt, &completed); err != nil {
			return jobs, err
		}
		if completed.Valid {
			j.CompletedAt = completed.Time
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (s *State) Close() error {
	return s.db.Close()
}
