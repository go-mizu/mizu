package goodread

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"path/filepath"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

// State manages the crawl queue, jobs, and visited URLs in a separate DuckDB.
type State struct {
	db   *sql.DB
	path string
}

// OpenState opens or creates the state DuckDB database.
func OpenState(path string) (*State, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}

	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open state duckdb %s: %w", path, err)
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

// Enqueue adds a URL to the queue. Silently ignores duplicates.
func (s *State) Enqueue(url, entityType string, priority int) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO queue (url, entity_type, priority) VALUES (?, ?, ?)`,
		url, entityType, priority,
	)
	return err
}

// EnqueueBatch adds multiple URLs to the queue in a transaction.
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
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO queue (url, entity_type, priority) VALUES (?, ?, ?)`,
			it.URL, it.EntityType, it.Priority,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Pop atomically claims up to n pending queue items.
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

// Done marks a queue item as done and records it as visited.
func (s *State) Done(url string, statusCode int, entityType string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`UPDATE queue SET status = 'done' WHERE url = ?`, url,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT OR REPLACE INTO visited (url, fetched_at, status_code, entity_type) VALUES (?, ?, ?, ?)`,
		url, time.Now(), statusCode, entityType,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// Fail increments attempts and marks as failed if >= 3 attempts.
func (s *State) Fail(url, errMsg string) error {
	_, err := s.db.Exec(`
		UPDATE queue
		SET
			error = ?,
			status = CASE WHEN attempts >= 3 THEN 'failed' ELSE 'pending' END
		WHERE url = ?`, errMsg, url)
	return err
}

// PendingCount returns the number of pending queue items.
func (s *State) PendingCount() (int64, error) {
	var n int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status = 'pending'`).Scan(&n)
	return n, err
}

// QueueStats returns counts by status.
func (s *State) QueueStats() (pending, inProgress, done, failed int64) {
	s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status = 'pending'`).Scan(&pending)
	s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status = 'in_progress'`).Scan(&inProgress)
	s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status = 'done'`).Scan(&done)
	s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status = 'failed'`).Scan(&failed)
	return
}

// IsVisited returns true if a URL has already been visited.
func (s *State) IsVisited(url string) bool {
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM visited WHERE url = ?`, url).Scan(&n)
	return n > 0
}

// ListQueue returns queue items filtered by status.
func (s *State) ListQueue(status string, limit int) ([]QueueItem, error) {
	rows, err := s.db.Query(
		`SELECT id, url, entity_type, priority FROM queue WHERE status = ? ORDER BY created_at DESC LIMIT ?`,
		status, limit,
	)
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

// CreateJob records a new crawl job.
func (s *State) CreateJob(id, name, jobType string) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO jobs (job_id, name, type, status, started_at) VALUES (?,?,?,?,?)`,
		id, name, jobType, "running", time.Now(),
	)
	return err
}

// UpdateJob updates job status and stats.
func (s *State) UpdateJob(id, status, stats string) error {
	_, err := s.db.Exec(
		`UPDATE jobs SET status = ?, stats = ?, completed_at = ? WHERE job_id = ?`,
		status, stats, time.Now(), id,
	)
	return err
}

// ListJobs returns recent jobs.
func (s *State) ListJobs(limit int) ([]JobRecord, error) {
	rows, err := s.db.Query(
		`SELECT job_id, name, type, status, started_at, completed_at FROM jobs ORDER BY started_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []JobRecord
	for rows.Next() {
		var j JobRecord
		var completedAt sql.NullTime
		if err := rows.Scan(&j.JobID, &j.Name, &j.Type, &j.Status, &j.StartedAt, &completedAt); err != nil {
			return jobs, err
		}
		if completedAt.Valid {
			j.CompletedAt = completedAt.Time
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// ResetInProgress resets any items stuck in 'in_progress' back to 'pending'.
// Call this at startup to recover from a previously killed run.
func (s *State) ResetInProgress() error {
	_, err := s.db.Exec(`UPDATE queue SET status = 'pending' WHERE status = 'in_progress'`)
	return err
}

// Close closes the state database.
func (s *State) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB for advanced use (e.g. DuckDB Appender).
func (s *State) DB() *sql.DB {
	return s.db
}

// EnqueueBulk imports items in bulk using DuckDB Appender + staging table.
// This is ~785× faster than EnqueueBatch for large slices because it uses
// DuckDB's binary row protocol for staging, then a single vectorized INSERT OR IGNORE.
func (s *State) EnqueueBulk(items []QueueItem) error {
	if len(items) == 0 {
		return nil
	}
	ctx := context.Background()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("db conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx,
		`CREATE TEMP TABLE IF NOT EXISTS _stage (url VARCHAR, entity_type VARCHAR, priority INTEGER)`,
	); err != nil {
		return fmt.Errorf("create staging: %w", err)
	}
	defer conn.ExecContext(ctx, `DROP TABLE IF EXISTS _stage`) //nolint:errcheck

	if err := conn.Raw(func(driverConn any) error {
		dc, ok := driverConn.(driver.Conn)
		if !ok {
			return fmt.Errorf("unexpected driver type %T", driverConn)
		}
		app, err := duckdb.NewAppenderFromConn(dc, "", "_stage")
		if err != nil {
			return fmt.Errorf("appender: %w", err)
		}
		for _, it := range items {
			if err := app.AppendRow(it.URL, it.EntityType, int32(it.Priority)); err != nil {
				app.Close()
				return err
			}
		}
		return app.Close()
	}); err != nil {
		return fmt.Errorf("fill staging: %w", err)
	}

	// Use LEFT JOIN anti-join instead of INSERT OR IGNORE so DuckDB can do a
	// single vectorized hash join (O(n+m)) rather than per-row ART index
	// lookups (O(m × log n)) that degrade as the queue grows to millions of rows.
	_, err = conn.ExecContext(ctx, `
		INSERT INTO queue (url, entity_type, priority)
		SELECT s.url, s.entity_type, s.priority
		FROM _stage s
		LEFT JOIN queue q ON s.url = q.url
		WHERE q.url IS NULL
	`)
	return err
}

// ── Bulk-seed session ─────────────────────────────────────────────────────────
// Usage: CreateSeedStage → N×AppendSeedBatch → FlushSeedToQueue.
// All sitemap URLs accumulate in a persistent staging table; a single hash
// anti-join INSERT touches the queue UNIQUE index only once, regardless of how
// many batches were appended. This avoids rebuilding the hash table of existing
// queue rows once per sitemap file.

// CreateSeedStage drops any leftover staging table and creates a fresh one.
func (s *State) CreateSeedStage() error {
	_, err := s.db.Exec(`
		DROP TABLE IF EXISTS _seed_stage;
		CREATE TABLE _seed_stage (url VARCHAR, entity_type VARCHAR, priority INTEGER)
	`)
	return err
}

// AppendSeedBatch streams items into _seed_stage via DuckDB Appender (binary
// row protocol — no SQL parser overhead, no index checks).
func (s *State) AppendSeedBatch(items []QueueItem) error {
	if len(items) == 0 {
		return nil
	}
	ctx := context.Background()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("db conn: %w", err)
	}
	defer conn.Close()

	return conn.Raw(func(driverConn any) error {
		dc, ok := driverConn.(driver.Conn)
		if !ok {
			return fmt.Errorf("unexpected driver type %T", driverConn)
		}
		app, err := duckdb.NewAppenderFromConn(dc, "", "_seed_stage")
		if err != nil {
			return fmt.Errorf("appender: %w", err)
		}
		for _, it := range items {
			if err := app.AppendRow(it.URL, it.EntityType, int32(it.Priority)); err != nil {
				app.Close()
				return err
			}
		}
		return app.Close()
	})
}

// FlushSeedToQueue inserts all rows from _seed_stage that are not already in
// the queue via a single hash anti-join (O(n+m)), then drops the stage table.
// Returns the number of newly inserted rows.
func (s *State) FlushSeedToQueue() (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO queue (url, entity_type, priority)
		SELECT s.url, s.entity_type, s.priority
		FROM _seed_stage s
		LEFT JOIN queue q ON s.url = q.url
		WHERE q.url IS NULL
	`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	s.db.Exec(`DROP TABLE IF EXISTS _seed_stage`) //nolint:errcheck
	return n, nil
}

// JobRecord holds a job record for display.
type JobRecord struct {
	JobID       string
	Name        string
	Type        string
	Status      string
	StartedAt   time.Time
	CompletedAt time.Time
}
