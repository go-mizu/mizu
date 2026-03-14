package goodread

import (
	"container/heap"
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
// All queue operations (Pop, MarkFetched, Done, Fail, Enqueue) operate on an
// in-memory priority queue and are persisted to DuckDB by a background
// checkpoint goroutine (every 5 s by default). This eliminates per-operation
// DuckDB write latency during high-throughput crawl/import phases.
type State struct {
	db   *sql.DB
	path string

	// In-memory queue cache — see state_mem.go for types and helpers.
	mem memState
}

// OpenState opens or creates the state DuckDB database and loads the queue
// into memory for zero-latency queue operations.
func OpenState(path string) (*State, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}

	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open state duckdb %s: %w", path, err)
	}

	s := &State{db: db, path: path, mem: newMemState()}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.loadMemQueue(); err != nil {
		db.Close()
		return nil, fmt.Errorf("load mem queue: %w", err)
	}

	go s.checkpointLoop(DefaultCheckpointInterval)

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
		// Migration: add html_path column if it doesn't exist yet.
		`ALTER TABLE queue ADD COLUMN IF NOT EXISTS html_path VARCHAR`,
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

// ── Queue operations (all in-memory, checkpointed to DB in background) ────────

// Enqueue adds a URL to the in-memory queue. Silently ignores duplicates.
// The item is persisted to DB on the next background checkpoint.
func (s *State) Enqueue(url, entityType string, priority int) error {
	s.mem.mu.Lock()
	s.enqueueInMem(url, entityType, priority)
	s.mem.mu.Unlock()
	return nil
}

// EnqueueBatch adds multiple URLs to the in-memory queue in one lock cycle.
func (s *State) EnqueueBatch(items []QueueItem) error {
	if len(items) == 0 {
		return nil
	}
	s.mem.mu.Lock()
	for _, it := range items {
		s.enqueueInMem(it.URL, it.EntityType, it.Priority)
	}
	s.mem.mu.Unlock()
	return nil
}

// Pop atomically claims up to n pending items from the in-memory queue.
// If the memory queue is empty it falls back to DB to pick up any items
// that were seeded via EnqueueBulk (which bypasses in-memory for speed).
func (s *State) Pop(n int) ([]QueueItem, error) {
	s.mem.mu.Lock()

	// Fast path: in-memory heap has items.
	if len(s.mem.pendingHeap) > 0 {
		items := s.popFromHeap(n)
		s.mem.mu.Unlock()
		return items, nil
	}
	s.mem.mu.Unlock()

	// Slow path: heap is empty — load a batch from DB (covers EnqueueBulk items).
	if err := s.refillFromDB(n * 4); err != nil {
		return nil, err
	}

	s.mem.mu.Lock()
	items := s.popFromHeap(n)
	s.mem.mu.Unlock()
	return items, nil
}

// popFromHeap pops up to n items from the in-memory pending heap.
// Must be called with s.mem.mu held.
func (s *State) popFromHeap(n int) []QueueItem {
	var items []QueueItem
	for len(items) < n && len(s.mem.pendingHeap) > 0 {
		it := heap.Pop(&s.mem.pendingHeap).(*memItem)
		it.status = "in_progress"
		it.attempts++
		s.mem.dirty[it.url] = it
		s.mem.pendingN.Add(-1)
		s.mem.inProgressN.Add(1)
		items = append(items, QueueItem{
			ID:         it.id,
			URL:        it.url,
			EntityType: it.entityType,
			Priority:   it.priority,
			HtmlPath:   it.htmlPath,
		})
	}
	return items
}

// refillFromDB loads pending items from DuckDB into the in-memory queue.
// Used when the in-memory heap runs dry (covers bulk-seeded items).
func (s *State) refillFromDB(n int) error {
	rows, err := s.db.Query(`
		SELECT id, url, entity_type, priority, COALESCE(html_path,''), COALESCE(error,''), attempts
		FROM queue
		WHERE status = 'pending'
		ORDER BY priority DESC, id ASC
		LIMIT ?
	`, n)
	if err != nil {
		return fmt.Errorf("refill from db: %w", err)
	}
	defer rows.Close()

	s.mem.mu.Lock()
	defer s.mem.mu.Unlock()

	for rows.Next() {
		it := &memItem{status: "pending"}
		if err := rows.Scan(&it.id, &it.url, &it.entityType, &it.priority,
			&it.htmlPath, &it.errMsg, &it.attempts); err != nil {
			return err
		}
		if _, exists := s.mem.items[it.url]; !exists {
			s.mem.items[it.url] = it
			heap.Push(&s.mem.pendingHeap, it)
			s.mem.pendingN.Add(1)
		}
	}
	return rows.Err()
}

// MarkFetched records that the HTML for a queue item has been downloaded to disk.
func (s *State) MarkFetched(url, htmlPath string) error {
	s.mem.mu.Lock()
	defer s.mem.mu.Unlock()

	if it, ok := s.mem.items[url]; ok {
		it.status = "fetched"
		it.htmlPath = htmlPath
		s.mem.fetchedItems = append(s.mem.fetchedItems, url)
		s.mem.dirty[url] = it
		s.mem.inProgressN.Add(-1)
		s.mem.fetchedN.Add(1)
	}
	return nil
}

// PopFetched atomically claims up to n items that are ready to import (status='fetched').
func (s *State) PopFetched(n int) ([]QueueItem, error) {
	s.mem.mu.Lock()
	defer s.mem.mu.Unlock()

	var items []QueueItem
	remaining := s.mem.fetchedItems[:0]
	for _, url := range s.mem.fetchedItems {
		if len(items) >= n {
			remaining = append(remaining, url)
			continue
		}
		it, ok := s.mem.items[url]
		if !ok || it.status != "fetched" {
			continue // already processed or missing
		}
		it.status = "in_progress"
		s.mem.dirty[url] = it
		s.mem.fetchedN.Add(-1)
		s.mem.inProgressN.Add(1)
		items = append(items, QueueItem{
			ID:         it.id,
			URL:        it.url,
			EntityType: it.entityType,
			Priority:   it.priority,
			HtmlPath:   it.htmlPath,
		})
		remaining = append(remaining, url) // keep in list until Done removes it
	}
	// Rebuild fetchedItems: keep remaining non-processed items + newly fetched
	// that were appended after we took the slice. We use a fresh copy.
	newFetched := make([]string, 0, len(remaining))
	for _, url := range remaining {
		if it, ok := s.mem.items[url]; ok && (it.status == "fetched" || it.status == "in_progress") {
			newFetched = append(newFetched, url)
		}
	}
	s.mem.fetchedItems = newFetched
	return items, nil
}

// Done marks a queue item as done and records it as visited.
func (s *State) Done(url string, statusCode int, entityType string) error {
	s.mem.mu.Lock()
	defer s.mem.mu.Unlock()

	it, ok := s.mem.items[url]
	if !ok {
		return nil
	}
	s.markItemDone(it, statusCode)
	return nil
}

// DoneAndEnqueue marks a URL as done and enqueues discovered links in one operation.
func (s *State) DoneAndEnqueue(url string, statusCode int, entityType string, links []QueueItem) error {
	s.mem.mu.Lock()
	defer s.mem.mu.Unlock()

	if it, ok := s.mem.items[url]; ok {
		s.markItemDone(it, statusCode)
	}
	for _, link := range links {
		s.enqueueInMem(link.URL, link.EntityType, link.Priority)
	}
	return nil
}

// Fail increments attempts and marks as failed if >= 3 attempts, else re-queues.
func (s *State) Fail(url, errMsg string) error {
	s.mem.mu.Lock()
	defer s.mem.mu.Unlock()

	it, ok := s.mem.items[url]
	if !ok {
		return nil
	}

	it.attempts++
	it.errMsg = errMsg
	s.mem.dirty[url] = it

	if it.attempts >= 3 {
		it.status = "failed"
		s.mem.inProgressN.Add(-1)
		s.mem.failedN.Add(1)
		delete(s.mem.items, url)
	} else {
		it.status = "pending"
		it.htmlPath = "" // discard any partially-fetched HTML
		s.mem.inProgressN.Add(-1)
		s.mem.pendingN.Add(1)
		heap.Push(&s.mem.pendingHeap, it)
	}
	return nil
}

// ResetInProgress resets any items stuck in 'in_progress' back to the
// appropriate status based on whether they have html_path set.
// Call this at startup to recover from a previously killed run.
func (s *State) ResetInProgress() error {
	s.mem.mu.Lock()
	defer s.mem.mu.Unlock()

	for url, it := range s.mem.items {
		if it.status != "in_progress" {
			continue
		}
		s.mem.dirty[url] = it
		s.mem.inProgressN.Add(-1)
		if it.htmlPath != "" {
			it.status = "fetched"
			s.mem.fetchedItems = append(s.mem.fetchedItems, url)
			s.mem.fetchedN.Add(1)
		} else {
			it.status = "pending"
			heap.Push(&s.mem.pendingHeap, it)
			s.mem.pendingN.Add(1)
		}
	}
	return nil
}

// FetchedCount returns the number of items waiting to be imported (status='fetched').
func (s *State) FetchedCount() (int64, error) {
	return s.mem.fetchedN.Load(), nil
}

// PendingCount returns the number of pending queue items.
func (s *State) PendingCount() (int64, error) {
	return s.mem.pendingN.Load(), nil
}

// QueueStats returns counts by status (all from in-memory counters, O(1)).
func (s *State) QueueStats() (pending, inProgress, done, failed int64) {
	return s.mem.pendingN.Load(), s.mem.inProgressN.Load(),
		s.mem.doneN.Load(), s.mem.failedN.Load()
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

// ── Bulk seed operations (write directly to DB; call LoadPendingFromDB after) ─

// EnqueueBulk imports items in bulk using DuckDB Appender + staging table.
// ~785× faster than EnqueueBatch for large slices. Call LoadPendingFromDB()
// afterwards to make the new items visible to the in-memory queue.
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

	_, err = conn.ExecContext(ctx, `
		INSERT INTO queue (url, entity_type, priority)
		SELECT s.url, s.entity_type, s.priority
		FROM _stage s
		LEFT JOIN queue q ON s.url = q.url
		WHERE q.url IS NULL
	`)
	return err
}

// ── Bulk-seed session helpers ─────────────────────────────────────────────────

// CreateSeedStage drops any leftover staging table and creates a fresh one.
func (s *State) CreateSeedStage() error {
	_, err := s.db.Exec(`
		DROP TABLE IF EXISTS _seed_stage;
		CREATE TABLE _seed_stage (url VARCHAR, entity_type VARCHAR, priority INTEGER)
	`)
	return err
}

// AppendSeedBatch streams items into _seed_stage via DuckDB Appender.
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

// FlushSeedToQueue inserts all rows from _seed_stage into the queue via a
// single hash anti-join. Returns the number of newly inserted rows.
// Call LoadPendingFromDB() afterwards to make the new items available in memory.
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

// ── Job tracking ──────────────────────────────────────────────────────────────

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

// ── Lifecycle ─────────────────────────────────────────────────────────────────

// Close flushes the in-memory queue to DuckDB and closes the database.
func (s *State) Close() error {
	close(s.mem.stopCh)
	<-s.mem.stopped // wait for final checkpoint
	return s.db.Close()
}

// DB returns the underlying *sql.DB for advanced use (e.g. DuckDB Appender).
func (s *State) DB() *sql.DB {
	return s.db
}

// ── Types ─────────────────────────────────────────────────────────────────────

// JobRecord holds a job record for display.
type JobRecord struct {
	JobID       string
	Name        string
	Type        string
	Status      string
	StartedAt   time.Time
	CompletedAt time.Time
}
