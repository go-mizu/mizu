package goodread

import (
	"container/heap"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ── In-memory queue types ─────────────────────────────────────────────────────

// memItem is an in-memory representation of a queue row.
type memItem struct {
	id         int64
	url        string
	entityType string
	priority   int
	status     string // pending | in_progress | fetched | done | failed
	attempts   int
	htmlPath   string
	errMsg     string
	isNew      bool // true if not yet persisted to DB

	heapIdx int // index in pendingHeap (maintained by heap.Interface)
}

// memHeap is a max-heap of *memItem ordered by (priority DESC, id ASC).
type memHeap []*memItem

func (h memHeap) Len() int { return len(h) }
func (h memHeap) Less(i, j int) bool {
	if h[i].priority != h[j].priority {
		return h[i].priority > h[j].priority
	}
	return h[i].id < h[j].id
}
func (h memHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].heapIdx = i
	h[j].heapIdx = j
}
func (h *memHeap) Push(x any) {
	it := x.(*memItem)
	it.heapIdx = len(*h)
	*h = append(*h, it)
}
func (h *memHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	it.heapIdx = -1
	return it
}

// visitedRow buffers a visited-table insert until the next checkpoint.
type visitedRow struct {
	url        string
	statusCode int
	entityType string
	fetchedAt  time.Time
}

// ── memState holds all in-memory queue state ──────────────────────────────────

// memState is embedded in State and holds all in-memory queue data.
type memState struct {
	mu sync.Mutex

	// All live items: pending, in_progress, fetched.
	// Done/failed items are removed after being checkpointed.
	items map[string]*memItem

	// pendingHeap holds only items with status="pending", ordered by priority DESC.
	pendingHeap memHeap

	// fetchedItems is a FIFO list of fetched item URLs (for PopFetched).
	fetchedItems []string

	// dirty tracks items modified since last checkpoint.
	// Includes done/failed items that need their final status written.
	dirty map[string]*memItem

	// visitedBuf holds visited rows to write on next checkpoint.
	visitedBuf []visitedRow

	// Atomic counters — updated under mu but readable without it.
	pendingN    atomic.Int64
	inProgressN atomic.Int64
	fetchedN    atomic.Int64
	doneN       atomic.Int64
	failedN     atomic.Int64

	stopCh  chan struct{}
	stopped chan struct{}
}

func newMemState() memState {
	return memState{
		items:   make(map[string]*memItem),
		dirty:   make(map[string]*memItem),
		stopCh:  make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

// ── Load from DB ──────────────────────────────────────────────────────────────

// memQueueCap is the max number of items loaded into the in-memory heap at once.
// With millions of queue items we can't load them all into RAM; refillFromDB
// tops up the heap on demand when it runs low.
const memQueueCap = 50_000

// loadMemQueue loads a capped working set of live queue items into memory.
// pendingN / doneN / failedN are seeded from DB COUNT so progress stats
// reflect the full queue depth even when the heap holds only memQueueCap items.
func (s *State) loadMemQueue() error {
	// Load a capped batch of live items (pending takes priority, then others).
	rows, err := s.db.Query(`
		SELECT id, url, entity_type, priority, status, attempts,
		       COALESCE(html_path, ''), COALESCE(error, '')
		FROM queue
		WHERE status NOT IN ('done', 'failed')
		ORDER BY priority DESC, id ASC
		LIMIT ?
	`, memQueueCap)
	if err != nil {
		return fmt.Errorf("load mem queue: %w", err)
	}

	var pending []*memItem
	s.mem.mu.Lock()
	defer s.mem.mu.Unlock()

	for rows.Next() {
		it := &memItem{}
		if err := rows.Scan(&it.id, &it.url, &it.entityType, &it.priority,
			&it.status, &it.attempts, &it.htmlPath, &it.errMsg); err != nil {
			rows.Close()
			return err
		}
		s.mem.items[it.url] = it
		switch it.status {
		case "pending":
			pending = append(pending, it)
			// pendingN is seeded from DB count below; don't increment here.
		case "in_progress":
			s.mem.inProgressN.Add(1)
		case "fetched":
			s.mem.fetchedItems = append(s.mem.fetchedItems, it.url)
			s.mem.fetchedN.Add(1)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	// Build heap in O(n) rather than n × O(log n) pushes.
	s.mem.pendingHeap = memHeap(pending)
	heap.Init(&s.mem.pendingHeap)
	for i, it := range s.mem.pendingHeap {
		it.heapIdx = i
	}

	// Seed counters from DB so stats reflect full queue depth.
	var dbPending, done, failed int64
	s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status='pending'`).Scan(&dbPending)
	s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status='done'`).Scan(&done)
	s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status='failed'`).Scan(&failed)
	s.mem.pendingN.Store(dbPending)
	s.mem.doneN.Store(done)
	s.mem.failedN.Store(failed)

	return nil
}

// LoadPendingFromDB reloads pending items from DB into the in-memory heap.
// Call this after EnqueueBulk / FlushSeedToQueue to pick up bulk-seeded items.
// Loads a capped batch (memQueueCap) and refreshes pendingN from DB count.
func (s *State) LoadPendingFromDB() error {
	s.mem.mu.Lock()
	// Clear existing pending items from heap and map.
	for _, it := range s.mem.pendingHeap {
		delete(s.mem.items, it.url)
	}
	s.mem.pendingHeap = s.mem.pendingHeap[:0]
	s.mem.mu.Unlock()

	// Re-load a capped batch from DB (query can be slow for large tables).
	rows, err := s.db.Query(`
		SELECT id, url, entity_type, priority, status, attempts,
		       COALESCE(html_path, ''), COALESCE(error, '')
		FROM queue
		WHERE status = 'pending'
		ORDER BY priority DESC, id ASC
		LIMIT ?
	`, memQueueCap)
	if err != nil {
		return fmt.Errorf("reload pending: %w", err)
	}

	var pending []*memItem
	for rows.Next() {
		it := &memItem{}
		if err := rows.Scan(&it.id, &it.url, &it.entityType, &it.priority,
			&it.status, &it.attempts, &it.htmlPath, &it.errMsg); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	s.mem.mu.Lock()
	defer s.mem.mu.Unlock()

	for _, it := range pending {
		if _, exists := s.mem.items[it.url]; !exists {
			s.mem.items[it.url] = it
		}
	}
	// Build heap from new pending items.
	all := make([]*memItem, 0, len(pending))
	for _, it := range pending {
		if s.mem.items[it.url] == it {
			all = append(all, it)
		}
	}
	s.mem.pendingHeap = memHeap(all)
	heap.Init(&s.mem.pendingHeap)
	for i, it := range s.mem.pendingHeap {
		it.heapIdx = i
	}

	// Refresh pendingN from DB count (accounts for bulk-seeded items).
	var dbPending int64
	s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status='pending'`).Scan(&dbPending)
	s.mem.pendingN.Store(dbPending)

	return nil
}

// ── Background checkpoint ─────────────────────────────────────────────────────

func (s *State) checkpointLoop(interval time.Duration) {
	defer close(s.mem.stopped)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.mem.stopCh:
			s.checkpoint() // final flush before exit
			return
		case <-ticker.C:
			s.checkpoint()
		}
	}
}

// checkpoint flushes dirty items and buffered visited rows to DuckDB.
// It collects the buffers under the mutex, then writes to DB without holding it.
func (s *State) checkpoint() {
	s.mem.mu.Lock()
	if len(s.mem.dirty) == 0 && len(s.mem.visitedBuf) == 0 {
		s.mem.mu.Unlock()
		return
	}
	dirty := s.mem.dirty
	s.mem.dirty = make(map[string]*memItem)
	visited := s.mem.visitedBuf
	s.mem.visitedBuf = nil
	s.mem.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		// Re-add to dirty for next attempt.
		s.mem.mu.Lock()
		for k, v := range dirty {
			if _, exists := s.mem.dirty[k]; !exists {
				s.mem.dirty[k] = v
			}
		}
		s.mem.visitedBuf = append(visited, s.mem.visitedBuf...)
		s.mem.mu.Unlock()
		return
	}

	for _, it := range dirty {
		// Use UPSERT so new and existing items both work correctly.
		tx.Exec(`
			INSERT INTO queue (url, entity_type, priority, status, attempts, html_path, error)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (url) DO UPDATE SET
				status       = excluded.status,
				attempts     = excluded.attempts,
				html_path    = excluded.html_path,
				error        = excluded.error,
				last_attempt = CURRENT_TIMESTAMP
		`, it.url, it.entityType, it.priority, it.status,
			it.attempts, nullStr(it.htmlPath), nullStr(it.errMsg),
		) //nolint:errcheck
	}

	for _, v := range visited {
		tx.Exec(`
			INSERT OR REPLACE INTO visited (url, fetched_at, status_code, entity_type)
			VALUES (?, ?, ?, ?)
		`, v.url, v.fetchedAt, v.statusCode, v.entityType) //nolint:errcheck
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback() //nolint:errcheck
		// Re-add to dirty.
		s.mem.mu.Lock()
		for k, v := range dirty {
			if _, exists := s.mem.dirty[k]; !exists {
				s.mem.dirty[k] = v
			}
		}
		s.mem.visitedBuf = append(visited, s.mem.visitedBuf...)
		s.mem.mu.Unlock()
	}
}

// ── In-memory stats ───────────────────────────────────────────────────────────

// MemStats returns detailed queue counts from the in-memory state (O(1), no DB).
type MemStats struct {
	Pending    int64
	InProgress int64
	Fetched    int64
	Done       int64
	Failed     int64
	DirtyItems int
}

func (s *State) MemStats() MemStats {
	s.mem.mu.Lock()
	dirtyLen := len(s.mem.dirty)
	s.mem.mu.Unlock()
	return MemStats{
		Pending:    s.mem.pendingN.Load(),
		InProgress: s.mem.inProgressN.Load(),
		Fetched:    s.mem.fetchedN.Load(),
		Done:       s.mem.doneN.Load(),
		Failed:     s.mem.failedN.Load(),
		DirtyItems: dirtyLen,
	}
}

// enqueueInMem adds a URL to the in-memory queue if it's not already known.
// Must be called with s.mem.mu held.
func (s *State) enqueueInMem(url, entityType string, priority int) {
	if _, exists := s.mem.items[url]; exists {
		return
	}
	it := &memItem{
		url:        url,
		entityType: entityType,
		priority:   priority,
		status:     "pending",
		isNew:      true,
	}
	s.mem.items[url] = it
	heap.Push(&s.mem.pendingHeap, it)
	s.mem.pendingN.Add(1)
	s.mem.dirty[url] = it
}

// markItemDone removes an item from memItems after recording in dirty + visitedBuf.
// Must be called with s.mem.mu held.
func (s *State) markItemDone(it *memItem, statusCode int) {
	it.status = "done"
	it.htmlPath = ""
	s.mem.dirty[it.url] = it
	s.mem.visitedBuf = append(s.mem.visitedBuf, visitedRow{
		url:        it.url,
		statusCode: statusCode,
		entityType: it.entityType,
		fetchedAt:  time.Now(),
	})
	delete(s.mem.items, it.url)
	s.mem.inProgressN.Add(-1)
	s.mem.doneN.Add(1)
}

// ── Checkpoint interval (exported so CLI can override) ────────────────────────

// DefaultCheckpointInterval is how often dirty state is flushed to DuckDB.
const DefaultCheckpointInterval = 1 * time.Second

// ── sql.NullString helper (already in db.go as nullStr but we need it here too)

// nullStrM is a local alias (sql.NullString is not exposed by db.go's nullStr).
func nullStrM(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// Unused — silence compiler if nullStrM is not used elsewhere.
var _ = nullStrM
