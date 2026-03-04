# FTS Dashboard Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace `search cc fts web` (search-only SPA) with `search cc fts dashboard` — a full admin panel for CC FTS pipeline management with real-time WebSocket progress.

**Architecture:** Go HTTP server (`pkg/index/web/`) with embedded single-file HTML+htmx SPA. WebSocket hub broadcasts job progress. JobManager runs pipeline operations as background goroutines, calling the same pkg functions as the CLI. Five tabs: Overview, Pipeline, Search, Browse, Crawls.

**Tech Stack:** Go net/http, gorilla/websocket (already in go.mod), htmx (CDN), Tailwind CSS (CDN), vanilla JS for WebSocket client. Brutalist design per spec/0652.

**Spec:** `spec/0653_fts_dashboard.md`

---

## Task 1: WebSocket Hub

**Files:**
- Create: `pkg/index/web/ws.go`
- Test: `pkg/index/web/ws_test.go`

**Step 1: Write the failing test**

```go
// pkg/index/web/ws_test.go
package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWSHub_BroadcastToSubscriber(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()

	// Start HTTP server with WebSocket endpoint.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws", hub.HandleWS)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Connect WebSocket client.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Subscribe to job "j1".
	conn.WriteJSON(map[string]any{"type": "subscribe", "job_ids": []string{"j1"}})

	// Give the hub time to register the subscription.
	time.Sleep(50 * time.Millisecond)

	// Broadcast a progress message for "j1".
	hub.Broadcast("j1", map[string]any{"type": "progress", "job_id": "j1", "pct": 0.5})

	// Read the message back.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg map[string]any
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["job_id"] != "j1" {
		t.Errorf("got job_id=%v, want j1", msg["job_id"])
	}
	if msg["pct"].(float64) != 0.5 {
		t.Errorf("got pct=%v, want 0.5", msg["pct"])
	}
}

func TestWSHub_UnsubscribedClientDoesNotReceive(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws", hub.HandleWS)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Subscribe to "j2" only.
	conn.WriteJSON(map[string]any{"type": "subscribe", "job_ids": []string{"j2"}})
	time.Sleep(50 * time.Millisecond)

	// Broadcast for "j1" — client should NOT receive.
	hub.Broadcast("j1", map[string]any{"type": "progress", "job_id": "j1"})

	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	var msg map[string]any
	err = conn.ReadJSON(&msg)
	if err == nil {
		t.Errorf("expected timeout, got message: %v", msg)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go test ./pkg/index/web/ -run TestWSHub -v -count=1`
Expected: FAIL — `NewWSHub` undefined

**Step 3: Write minimal implementation**

```go
// pkg/index/web/ws.go
package web

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSHub manages WebSocket clients and broadcasts messages to subscribers.
type WSHub struct {
	mu      sync.RWMutex
	clients map[*WSClient]struct{}
	done    chan struct{}
}

// WSClient represents a single WebSocket connection.
type WSClient struct {
	conn   *websocket.Conn
	hub    *WSHub
	mu     sync.RWMutex
	subs   map[string]bool // subscribed job IDs; "*" means all
	sendCh chan []byte
	done   chan struct{}
}

// NewWSHub creates a new WebSocket hub.
func NewWSHub() *WSHub {
	return &WSHub{
		clients: make(map[*WSClient]struct{}),
		done:    make(chan struct{}),
	}
}

// Close shuts down the hub and all client connections.
func (h *WSHub) Close() {
	close(h.done)
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		c.conn.Close()
		close(c.sendCh)
	}
	h.clients = make(map[*WSClient]struct{})
}

// HandleWS upgrades an HTTP connection to WebSocket and registers the client.
func (h *WSHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}

	client := &WSClient{
		conn:   conn,
		hub:    h,
		subs:   make(map[string]bool),
		sendCh: make(chan []byte, 64),
		done:   make(chan struct{}),
	}

	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()

	go client.writePump()
	go client.readPump()
}

// Broadcast sends a message to all clients subscribed to the given job ID.
func (h *WSHub) Broadcast(jobID string, msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		c.mu.RLock()
		subscribed := c.subs["*"] || c.subs[jobID]
		c.mu.RUnlock()
		if subscribed {
			select {
			case c.sendCh <- data:
			default:
				// Drop message if client is slow.
			}
		}
	}
}

func (h *WSHub) removeClient(c *WSClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// readPump reads messages from the WebSocket connection (subscribe/unsubscribe).
func (c *WSClient) readPump() {
	defer func() {
		c.hub.removeClient(c)
		c.conn.Close()
		close(c.done)
	}()

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			Type   string   `json:"type"`
			JobIDs []string `json:"job_ids"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		c.mu.Lock()
		switch msg.Type {
		case "subscribe":
			for _, id := range msg.JobIDs {
				c.subs[id] = true
			}
		case "unsubscribe":
			for _, id := range msg.JobIDs {
				delete(c.subs, id)
			}
		}
		c.mu.Unlock()
	}
}

// writePump writes messages from sendCh to the WebSocket connection.
func (c *WSClient) writePump() {
	defer c.conn.Close()
	for {
		select {
		case msg, ok := <-c.sendCh:
			if !ok {
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go test ./pkg/index/web/ -run TestWSHub -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/index/web/ws.go pkg/index/web/ws_test.go
git commit -m "feat(dashboard): add WebSocket hub for real-time job progress"
```

---

## Task 2: Job Manager

**Files:**
- Create: `pkg/index/web/jobs.go`
- Test: `pkg/index/web/jobs_test.go`

**Step 1: Write the failing test**

```go
// pkg/index/web/jobs_test.go
package web

import (
	"context"
	"testing"
	"time"
)

func TestJobManager_CreateAndList(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()
	jm := NewJobManager(hub, "/tmp/test-data", "CC-TEST")

	job := jm.Create(JobConfig{
		Type:    "index",
		CrawlID: "CC-TEST",
		Files:   "0",
		Engine:  "duckdb",
	})
	if job.ID == "" {
		t.Fatal("job ID should not be empty")
	}
	if job.Status != "queued" {
		t.Errorf("initial status = %q, want queued", job.Status)
	}

	jobs := jm.List()
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].ID != job.ID {
		t.Errorf("listed job ID = %q, want %q", jobs[0].ID, job.ID)
	}
}

func TestJobManager_CancelJob(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()
	jm := NewJobManager(hub, "/tmp/test-data", "CC-TEST")

	job := jm.Create(JobConfig{
		Type:    "download",
		CrawlID: "CC-TEST",
		Files:   "0",
	})

	// Simulate running: set a cancel func.
	ctx, cancel := context.WithCancel(context.Background())
	jm.mu.Lock()
	jm.jobs[job.ID].Status = "running"
	jm.jobs[job.ID].cancel = cancel
	jm.mu.Unlock()

	ok := jm.Cancel(job.ID)
	if !ok {
		t.Fatal("Cancel returned false")
	}

	// Context should be cancelled.
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("context was not cancelled")
	}

	got := jm.Get(job.ID)
	if got.Status != "cancelled" {
		t.Errorf("status = %q, want cancelled", got.Status)
	}
}

func TestJobManager_GetNonexistent(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()
	jm := NewJobManager(hub, "/tmp/test-data", "CC-TEST")

	got := jm.Get("nonexistent")
	if got != nil {
		t.Errorf("expected nil for nonexistent job, got %+v", got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go test ./pkg/index/web/ -run TestJobManager -v -count=1`
Expected: FAIL — `NewJobManager` undefined

**Step 3: Write minimal implementation**

```go
// pkg/index/web/jobs.go
package web

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JobConfig describes what a job should do.
type JobConfig struct {
	Type    string `json:"type"`    // download, markdown, pack, index
	CrawlID string `json:"crawl"`
	Files   string `json:"files"`   // "0", "0-4", "all"
	Engine  string `json:"engine"`  // for index jobs
	Source  string `json:"source"`  // for index jobs (files, parquet, bin, etc.)
	Format  string `json:"format"`  // for pack jobs
	Fast    bool   `json:"fast"`    // for markdown jobs
}

// Job represents a running or completed pipeline job.
type Job struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Status    string     `json:"status"` // queued, running, completed, failed, cancelled
	Config    JobConfig  `json:"config"`
	Progress  float64    `json:"progress"`           // 0.0–1.0
	Message   string     `json:"message"`
	Rate      float64    `json:"rate,omitempty"`      // items/sec
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Error     string     `json:"error,omitempty"`
	cancel    context.CancelFunc
}

// JobManager manages pipeline jobs in-memory.
type JobManager struct {
	mu      sync.RWMutex
	jobs    map[string]*Job
	order   []string // job IDs in creation order
	hub     *WSHub
	baseDir string
	crawlID string
}

// NewJobManager creates a new job manager.
func NewJobManager(hub *WSHub, baseDir, crawlID string) *JobManager {
	return &JobManager{
		jobs:    make(map[string]*Job),
		hub:     hub,
		baseDir: baseDir,
		crawlID: crawlID,
	}
}

// Create creates a new job in "queued" status.
func (m *JobManager) Create(cfg JobConfig) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	job := &Job{
		ID:        uuid.New().String()[:8],
		Type:      cfg.Type,
		Status:    "queued",
		Config:    cfg,
		StartedAt: time.Now(),
	}
	m.jobs[job.ID] = job
	m.order = append(m.order, job.ID)
	return job
}

// Get returns a job by ID, or nil if not found.
func (m *JobManager) Get(id string) *Job {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.jobs[id]
}

// List returns all jobs in creation order (newest first).
func (m *JobManager) List() []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Job, 0, len(m.order))
	for i := len(m.order) - 1; i >= 0; i-- {
		if j, ok := m.jobs[m.order[i]]; ok {
			result = append(result, j)
		}
	}
	return result
}

// Cancel cancels a running job.
func (m *JobManager) Cancel(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return false
	}
	if job.cancel != nil {
		job.cancel()
	}
	job.Status = "cancelled"
	now := time.Now()
	job.EndedAt = &now
	return true
}

// UpdateProgress updates a job's progress and broadcasts via WebSocket.
func (m *JobManager) UpdateProgress(id string, pct float64, msg string, rate float64) {
	m.mu.Lock()
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	job.Progress = pct
	job.Message = msg
	job.Rate = rate
	m.mu.Unlock()

	m.hub.Broadcast(id, map[string]any{
		"type":   "progress",
		"job_id": id,
		"pct":    pct,
		"msg":    msg,
		"rate":   rate,
	})
}

// Complete marks a job as completed.
func (m *JobManager) Complete(id string, msg string) {
	m.mu.Lock()
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	job.Status = "completed"
	job.Progress = 1.0
	job.Message = msg
	now := time.Now()
	job.EndedAt = &now
	m.mu.Unlock()

	m.hub.Broadcast(id, map[string]any{
		"type":   "complete",
		"job_id": id,
		"msg":    msg,
	})
}

// Fail marks a job as failed.
func (m *JobManager) Fail(id string, err error) {
	m.mu.Lock()
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	job.Status = "failed"
	job.Error = err.Error()
	now := time.Now()
	job.EndedAt = &now
	m.mu.Unlock()

	m.hub.Broadcast(id, map[string]any{
		"type":   "failed",
		"job_id": id,
		"error":  err.Error(),
	})
}

// SetRunning marks a job as running with a cancel function.
func (m *JobManager) SetRunning(id string, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job, ok := m.jobs[id]; ok {
		job.Status = "running"
		job.cancel = cancel
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go test ./pkg/index/web/ -run TestJobManager -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/index/web/jobs.go pkg/index/web/jobs_test.go
git commit -m "feat(dashboard): add JobManager for pipeline job lifecycle"
```

---

## Task 3: Data Scanner

**Files:**
- Create: `pkg/index/web/scanner.go`
- Test: `pkg/index/web/scanner_test.go`

**Step 1: Write the failing test**

```go
// pkg/index/web/scanner_test.go
package web

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDataDir(t *testing.T) {
	// Create temp directory structure mimicking CC data layout.
	tmp := t.TempDir()
	crawlDir := filepath.Join(tmp, "CC-TEST")

	// Create warc/ with 2 fake files.
	warcDir := filepath.Join(crawlDir, "warc")
	os.MkdirAll(warcDir, 0755)
	os.WriteFile(filepath.Join(warcDir, "00000.warc.gz"), make([]byte, 1024), 0644)
	os.WriteFile(filepath.Join(warcDir, "00001.warc.gz"), make([]byte, 2048), 0644)

	// Create markdown/ with 1 shard containing 2 files.
	mdDir := filepath.Join(crawlDir, "markdown", "00000")
	os.MkdirAll(mdDir, 0755)
	os.WriteFile(filepath.Join(mdDir, "a.md"), make([]byte, 100), 0644)
	os.WriteFile(filepath.Join(mdDir, "b.md"), make([]byte, 200), 0644)

	// Create fts/duckdb/00000/ with 1 file.
	ftsDir := filepath.Join(crawlDir, "fts", "duckdb", "00000")
	os.MkdirAll(ftsDir, 0755)
	os.WriteFile(filepath.Join(ftsDir, "index.db"), make([]byte, 512), 0644)

	summary := ScanDataDir(crawlDir)

	if summary.WARCCount != 2 {
		t.Errorf("WARCCount = %d, want 2", summary.WARCCount)
	}
	if summary.WARCTotalSize != 3072 {
		t.Errorf("WARCTotalSize = %d, want 3072", summary.WARCTotalSize)
	}
	if summary.MDShards != 1 {
		t.Errorf("MDShards = %d, want 1", summary.MDShards)
	}
	if summary.FTSEngines["duckdb"] != 512 {
		t.Errorf("FTSEngines[duckdb] = %d, want 512", summary.FTSEngines["duckdb"])
	}
}

func TestScanDataDir_Empty(t *testing.T) {
	summary := ScanDataDir(t.TempDir())
	if summary.WARCCount != 0 {
		t.Errorf("WARCCount = %d, want 0", summary.WARCCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go test ./pkg/index/web/ -run TestScanDataDir -v -count=1`
Expected: FAIL — `ScanDataDir` undefined

**Step 3: Write minimal implementation**

```go
// pkg/index/web/scanner.go
package web

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DataSummary holds aggregated stats about a crawl's local data.
type DataSummary struct {
	CrawlID       string           `json:"crawl_id"`
	WARCCount     int              `json:"warc_count"`
	WARCTotalSize int64            `json:"warc_total_size"`
	MDShards      int              `json:"md_shards"`
	MDTotalSize   int64            `json:"md_total_size"`
	MDDocEstimate int              `json:"md_doc_estimate"`
	PackFormats   map[string]int64 `json:"pack_formats"`
	FTSEngines    map[string]int64 `json:"fts_engines"`
	FTSShardCount map[string]int   `json:"fts_shard_count"`
}

// ScanDataDir scans a crawl directory and returns a DataSummary.
// This is a lightweight scan — it does not open DuckDB files.
func ScanDataDir(crawlDir string) DataSummary {
	s := DataSummary{
		PackFormats:   make(map[string]int64),
		FTSEngines:    make(map[string]int64),
		FTSShardCount: make(map[string]int),
	}
	s.CrawlID = filepath.Base(crawlDir)

	// Scan warc/
	s.WARCCount, s.WARCTotalSize = countFilesAndSize(filepath.Join(crawlDir, "warc"))

	// Scan markdown/
	mdBase := filepath.Join(crawlDir, "markdown")
	if entries, err := os.ReadDir(mdBase); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				s.MDShards++
				count, size := countFilesAndSize(filepath.Join(mdBase, e.Name()))
				s.MDDocEstimate += count
				s.MDTotalSize += size
			}
		}
	}

	// Scan pack/
	packBase := filepath.Join(crawlDir, "pack")
	if entries, err := os.ReadDir(packBase); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				_, size := countFilesAndSize(filepath.Join(packBase, e.Name()))
				s.PackFormats[e.Name()] = size
			}
		}
	}

	// Scan fts/
	ftsBase := filepath.Join(crawlDir, "fts")
	if engines, err := os.ReadDir(ftsBase); err == nil {
		for _, eng := range engines {
			if !eng.IsDir() {
				continue
			}
			engineDir := filepath.Join(ftsBase, eng.Name())
			shards, err := os.ReadDir(engineDir)
			if err != nil {
				continue
			}
			shardCount := 0
			var totalSize int64
			for _, sh := range shards {
				if sh.IsDir() {
					shardCount++
					_, sz := countFilesAndSize(filepath.Join(engineDir, sh.Name()))
					totalSize += sz
				}
			}
			s.FTSEngines[eng.Name()] = totalSize
			s.FTSShardCount[eng.Name()] = shardCount
		}
	}

	return s
}

// countFilesAndSize walks a directory and returns file count and total size.
func countFilesAndSize(dir string) (int, int64) {
	var count int
	var total int64
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		count++
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return count, total
}

// FormatBytes formats a byte count as a human-readable string.
func FormatBytes(b int64) string {
	if b == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	v := float64(b)
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if i == 0 {
		return strings.TrimRight(strings.TrimRight(
			formatFloat(v, 0), "0"), ".") + " B"
	}
	return formatFloat(v, 1) + " " + units[i]
}

func formatFloat(v float64, prec int) string {
	if prec == 0 {
		return strings.TrimRight(strings.TrimRight(
			strconv.FormatFloat(v, 'f', 1, 64), "0"), ".")
	}
	return strconv.FormatFloat(v, 'f', prec, 64)
}
```

Note: `FormatBytes` needs `"strconv"` imported. The function is intentionally separate from the existing `formatBytes` in server.go (lowercase, unexported) — we export it for use in templates. We can clean up the duplication later.

**Step 4: Run test to verify it passes**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go test ./pkg/index/web/ -run TestScanDataDir -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/index/web/scanner.go pkg/index/web/scanner_test.go
git commit -m "feat(dashboard): add data directory scanner for stats"
```

---

## Task 4: Job Executors

Wire JobManager to real pipeline functions. Each job type calls the same
underlying pkg functions as the CLI.

**Files:**
- Create: `pkg/index/web/executors.go`

**Step 1: Write the executors**

The executors wrap CLI-level pipeline functions with JobManager progress callbacks.
They don't need unit tests of their own — they delegate to well-tested pkg functions.
Integration testing happens via the API in Task 6.

```go
// pkg/index/web/executors.go
package web

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

// RunJob dispatches a job to the appropriate executor in a background goroutine.
// It manages the job lifecycle: queued → running → completed/failed/cancelled.
func (m *JobManager) RunJob(job *Job) {
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		m.SetRunning(job.ID, cancel)

		var err error
		switch job.Config.Type {
		case "download":
			err = m.execDownload(ctx, job)
		case "markdown":
			err = m.execMarkdown(ctx, job)
		case "pack":
			err = m.execPack(ctx, job)
		case "index":
			err = m.execIndex(ctx, job)
		default:
			err = fmt.Errorf("unknown job type: %s", job.Config.Type)
		}

		if err != nil {
			if ctx.Err() != nil {
				return // already cancelled
			}
			m.Fail(job.ID, err)
		} else {
			m.Complete(job.ID, fmt.Sprintf("%s completed", job.Config.Type))
		}
	}()
}

func (m *JobManager) resolveBaseDir(cfg JobConfig) string {
	crawl := cfg.CrawlID
	if crawl == "" {
		crawl = m.crawlID
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "data", "common-crawl", crawl)
}

func (m *JobManager) execDownload(ctx context.Context, job *Job) error {
	baseDir := m.resolveBaseDir(job.Config)
	client := cc.NewClient("", 4)

	crawl := job.Config.CrawlID
	if crawl == "" {
		crawl = m.crawlID
	}

	paths, err := client.DownloadManifest(ctx, crawl, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}

	selected, err := parseFileSelector(job.Config.Files, len(paths))
	if err != nil {
		return err
	}

	warcDir := filepath.Join(baseDir, "warc")
	os.MkdirAll(warcDir, 0755)

	for i, idx := range selected {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		remotePath := paths[idx]
		localPath := filepath.Join(warcDir, filepath.Base(remotePath))

		m.UpdateProgress(job.ID,
			float64(i)/float64(len(selected)),
			fmt.Sprintf("downloading %s (%d/%d)", filepath.Base(remotePath), i+1, len(selected)),
			0)

		err := client.DownloadFile(ctx, remotePath, localPath, func(recv, total int64) {
			filePct := float64(recv) / float64(max(total, 1))
			overallPct := (float64(i) + filePct) / float64(len(selected))
			m.UpdateProgress(job.ID, overallPct,
				fmt.Sprintf("downloading %s — %s / %s",
					filepath.Base(remotePath),
					FormatBytes(recv), FormatBytes(total)),
				0)
		})
		if err != nil {
			return fmt.Errorf("download %s: %w", filepath.Base(remotePath), err)
		}
	}
	return nil
}

func (m *JobManager) execMarkdown(ctx context.Context, job *Job) error {
	// Markdown extraction is complex (2-phase pipeline). For v1, we report
	// progress at the file level. The actual conversion is done by warc_md pkg.
	m.UpdateProgress(job.ID, 0, "starting markdown extraction...", 0)

	// TODO: wire to warc_md.RunPipeline when refactored for progress callbacks.
	// For now, this is a placeholder that the frontend can display.
	return fmt.Errorf("markdown extraction via dashboard not yet implemented — use CLI: search cc warc markdown")
}

func (m *JobManager) execPack(ctx context.Context, job *Job) error {
	baseDir := m.resolveBaseDir(job.Config)
	crawl := job.Config.CrawlID
	if crawl == "" {
		crawl = m.crawlID
	}

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawl, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	selected, err := parseFileSelector(job.Config.Files, len(paths))
	if err != nil {
		return err
	}

	format := job.Config.Format
	if format == "" {
		format = "parquet"
	}

	for i, idx := range selected {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		warcIdx := warcIndexFromPath(paths[idx], idx)
		markdownDir := filepath.Join(baseDir, "markdown", warcIdx)
		packDir := filepath.Join(baseDir, "pack")

		packFile, err := packFilePath(packDir, format, warcIdx)
		if err != nil {
			return err
		}

		m.UpdateProgress(job.ID,
			float64(i)/float64(len(selected)),
			fmt.Sprintf("packing %s [%s] (%d/%d)", warcIdx, format, i+1, len(selected)),
			0)

		progress := func(stats *index.PipelineStats) {
			done := stats.DocsIndexed.Load()
			total := stats.TotalFiles.Load()
			elapsed := time.Since(stats.StartTime).Seconds()
			rate := float64(0)
			if elapsed > 0 {
				rate = float64(done) / elapsed
			}
			pct := float64(0)
			if total > 0 {
				pct = float64(done) / float64(total)
			}
			overallPct := (float64(i) + pct) / float64(len(selected))
			m.UpdateProgress(job.ID, overallPct,
				fmt.Sprintf("packing %s [%s] %d/%d docs", warcIdx, format, done, total),
				rate)
		}

		workers := runtime.NumCPU()
		batchSize := 5000
		switch format {
		case "parquet":
			_, err = index.PackParquet(ctx, markdownDir, packFile, workers, batchSize, progress)
		case "bin":
			_, err = index.PackFlatBin(ctx, markdownDir, packFile, workers, batchSize, progress)
		case "markdown":
			_, err = index.PackFlatBinGz(ctx, markdownDir, packFile, workers, batchSize, progress)
		default:
			err = fmt.Errorf("unsupported format: %s", format)
		}
		if err != nil {
			return fmt.Errorf("pack %s: %w", format, err)
		}
	}
	return nil
}

func (m *JobManager) execIndex(ctx context.Context, job *Job) error {
	baseDir := m.resolveBaseDir(job.Config)
	crawl := job.Config.CrawlID
	if crawl == "" {
		crawl = m.crawlID
	}
	engineName := job.Config.Engine
	if engineName == "" {
		engineName = "duckdb"
	}
	source := job.Config.Source
	if source == "" {
		source = "files"
	}

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawl, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	selected, err := parseFileSelector(job.Config.Files, len(paths))
	if err != nil {
		return err
	}

	for i, idx := range selected {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		warcIdx := warcIndexFromPath(paths[idx], idx)
		outputDir := filepath.Join(baseDir, "fts", engineName, warcIdx)

		eng, err := index.NewEngine(engineName)
		if err != nil {
			return err
		}
		if err := eng.Open(ctx, outputDir); err != nil {
			return fmt.Errorf("open engine: %w", err)
		}

		if source == "files" {
			sourceDir := filepath.Join(baseDir, "markdown", warcIdx)
			cfg := index.PipelineConfig{
				SourceDir: sourceDir,
				BatchSize: 5000,
				Workers:   0, // auto
			}
			progress := func(stats *index.PipelineStats) {
				done := stats.DocsIndexed.Load()
				total := stats.TotalFiles.Load()
				elapsed := time.Since(stats.StartTime).Seconds()
				rate := float64(0)
				if elapsed > 0 {
					rate = float64(done) / elapsed
				}
				pct := float64(0)
				if total > 0 {
					pct = float64(done) / float64(total)
				}
				overallPct := (float64(i) + pct) / float64(len(selected))
				m.UpdateProgress(job.ID, overallPct,
					fmt.Sprintf("indexing %s [%s] %d/%d docs", warcIdx, engineName, done, total),
					rate)
			}
			_, err = index.RunPipeline(ctx, eng, cfg, progress)
		} else {
			// Pack-based indexing: find pack file and stream from it.
			packDir := filepath.Join(baseDir, "pack")
			packFile, perr := packFilePath(packDir, source, warcIdx)
			if perr != nil {
				eng.Close()
				return perr
			}
			progress := func(done, total int64, elapsed time.Duration) {
				secs := elapsed.Seconds()
				rate := float64(0)
				if secs > 0 {
					rate = float64(done) / secs
				}
				pct := float64(0)
				if total > 0 {
					pct = float64(done) / float64(total)
				}
				overallPct := (float64(i) + pct) / float64(len(selected))
				m.UpdateProgress(job.ID, overallPct,
					fmt.Sprintf("indexing %s [%s←%s] %d docs", warcIdx, engineName, source, done),
					rate)
			}
			switch source {
			case "parquet":
				_, err = index.RunPipelineFromParquet(ctx, eng, packFile, 5000, progress)
			case "bin":
				_, err = index.RunPipelineFromFlatBin(ctx, eng, packFile, 5000, progress)
			case "markdown":
				_, err = index.RunPipelineFromFlatBinGz(ctx, eng, packFile, 5000, progress)
			default:
				err = fmt.Errorf("unsupported source: %s", source)
			}
		}

		eng.Close()
		if err != nil {
			return fmt.Errorf("index %s: %w", warcIdx, err)
		}
	}
	return nil
}

// --- helpers (duplicated from cli/ to avoid import cycle) ---

func warcIndexFromPath(warcPath string, fallback int) string {
	base := filepath.Base(warcPath)
	name := strings.TrimSuffix(strings.TrimSuffix(base, ".gz"), ".warc")
	parts := strings.Split(name, "-")
	if last := parts[len(parts)-1]; len(last) == 5 {
		if _, err := strconv.Atoi(last); err == nil {
			return last
		}
	}
	return fmt.Sprintf("%05d", fallback)
}

func packFilePath(packDir, format, warcIdx string) (string, error) {
	switch format {
	case "parquet":
		return filepath.Join(packDir, "parquet", warcIdx+".parquet"), nil
	case "bin":
		return filepath.Join(packDir, "bin", warcIdx+".bin"), nil
	case "duckdb":
		return filepath.Join(packDir, "duckdb", warcIdx+".duckdb"), nil
	case "markdown":
		return filepath.Join(packDir, "markdown", warcIdx+".bin.gz"), nil
	default:
		return "", fmt.Errorf("unknown format %q", format)
	}
}

func parseFileSelector(sel string, total int) ([]int, error) {
	if sel == "all" {
		result := make([]int, total)
		for i := range result {
			result[i] = i
		}
		return result, nil
	}
	if strings.Contains(sel, "-") {
		parts := strings.SplitN(sel, "-", 2)
		from, err1 := strconv.Atoi(parts[0])
		to, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("invalid range: %s", sel)
		}
		if from < 0 || to >= total || from > to {
			return nil, fmt.Errorf("range %d-%d out of bounds (0-%d)", from, to, total-1)
		}
		result := make([]int, to-from+1)
		for i := range result {
			result[i] = from + i
		}
		return result, nil
	}
	idx, err := strconv.Atoi(sel)
	if err != nil {
		return nil, fmt.Errorf("invalid file index: %s", sel)
	}
	if idx < 0 || idx >= total {
		return nil, fmt.Errorf("file index %d out of bounds (0-%d)", idx, total-1)
	}
	return []int{idx}, nil
}
```

**Step 2: Commit**

```bash
git add pkg/index/web/executors.go
git commit -m "feat(dashboard): add job executors for download/pack/index pipelines"
```

---

## Task 5: Enhanced Server with Dashboard API Routes

**Files:**
- Modify: `pkg/index/web/server.go` — add new fields, routes, handlers
- Test: `pkg/index/web/server_test.go`

**Step 1: Write the failing test**

```go
// pkg/index/web/server_test.go
package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleOverview(t *testing.T) {
	tmp := t.TempDir()
	crawlDir := filepath.Join(tmp, "CC-TEST")
	os.MkdirAll(filepath.Join(crawlDir, "warc"), 0755)
	os.WriteFile(filepath.Join(crawlDir, "warc", "00000.warc.gz"), make([]byte, 1024), 0644)

	srv := &Server{
		EngineName: "duckdb",
		CrawlID:    "CC-TEST",
		FTSBase:    filepath.Join(crawlDir, "fts", "duckdb"),
		MDBase:     filepath.Join(crawlDir, "markdown"),
		CrawlDir:   crawlDir,
		Hub:        NewWSHub(),
	}
	srv.Jobs = NewJobManager(srv.Hub, crawlDir, "CC-TEST")

	req := httptest.NewRequest("GET", "/api/overview", nil)
	rec := httptest.NewRecorder()
	srv.handleOverview(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var result map[string]any
	json.NewDecoder(rec.Body).Decode(&result)
	if result["crawl_id"] != "CC-TEST" {
		t.Errorf("crawl_id = %v, want CC-TEST", result["crawl_id"])
	}
}

func TestHandleEngines(t *testing.T) {
	srv := &Server{Hub: NewWSHub()}
	srv.Jobs = NewJobManager(srv.Hub, "", "")

	req := httptest.NewRequest("GET", "/api/engines", nil)
	rec := httptest.NewRecorder()
	srv.handleEngines(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var result map[string]any
	json.NewDecoder(rec.Body).Decode(&result)
	engines, ok := result["engines"]
	if !ok {
		t.Fatal("missing engines key")
	}
	if arr, ok := engines.([]any); !ok || len(arr) == 0 {
		t.Error("engines should be non-empty array")
	}
}

func TestHandleJobs_Empty(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()
	srv := &Server{Hub: hub}
	srv.Jobs = NewJobManager(hub, "", "")

	req := httptest.NewRequest("GET", "/api/jobs", nil)
	rec := httptest.NewRecorder()
	srv.handleListJobs(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var result map[string]any
	json.NewDecoder(rec.Body).Decode(&result)
	jobs := result["jobs"].([]any)
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go test ./pkg/index/web/ -run "TestHandle(Overview|Engines|Jobs)" -v -count=1`
Expected: FAIL — `CrawlDir`, `Hub`, `Jobs` fields undefined on Server

**Step 3: Modify server.go**

Add new fields to Server, new routes to Handler(), and new handler methods.
Keep all existing handlers and routes unchanged.

Key changes to `server.go`:
1. Add `CrawlDir`, `Hub`, `Jobs` fields to `Server`
2. Add `NewDashboard()` constructor (enhanced version of `New()`)
3. Add new routes in `Handler()`
4. Add handler methods: `handleOverview`, `handleEngines`, `handleListJobs`, `handleCreateJob`, `handleGetJob`, `handleCancelJob`, `handleCrawls`, `handleCrawlWarcs`, `handleCrawlData`

```go
// Add to Server struct:
//   CrawlDir string   // ~/data/common-crawl/{crawlID}
//   Hub      *WSHub
//   Jobs     *JobManager

// NewDashboard creates a Server configured as a full dashboard.
func NewDashboard(engineName, crawlID, addr, baseDir string) *Server {
	s := New(engineName, crawlID, addr, baseDir)
	s.CrawlDir = baseDir
	s.Hub = NewWSHub()
	s.Jobs = NewJobManager(s.Hub, baseDir, crawlID)
	return s
}

// Update Handler() to add new routes (after existing ones):
//   mux.HandleFunc("GET /api/overview", s.handleOverview)
//   mux.HandleFunc("GET /api/crawls", s.handleCrawls)
//   mux.HandleFunc("GET /api/crawl/{id}/warcs", s.handleCrawlWarcs)
//   mux.HandleFunc("GET /api/crawl/{id}/data", s.handleCrawlData)
//   mux.HandleFunc("GET /api/engines", s.handleEngines)
//   mux.HandleFunc("GET /api/jobs", s.handleListJobs)
//   mux.HandleFunc("GET /api/jobs/{id}", s.handleGetJob)
//   mux.HandleFunc("POST /api/jobs", s.handleCreateJob)
//   mux.HandleFunc("DELETE /api/jobs/{id}", s.handleCancelJob)
//   if s.Hub != nil {
//       mux.HandleFunc("GET /ws", s.Hub.HandleWS)
//   }
```

See spec/0653 for handler implementations. Each handler is a thin wrapper:
- `handleOverview` → calls `ScanDataDir(s.CrawlDir)`
- `handleEngines` → calls `index.List()`
- `handleListJobs` → calls `s.Jobs.List()`
- `handleCreateJob` → decodes JSON body → `s.Jobs.Create(cfg)` → `s.Jobs.RunJob(job)`
- `handleGetJob` → `s.Jobs.Get(id)`
- `handleCancelJob` → `s.Jobs.Cancel(id)`
- `handleCrawls` → `cc.NewClient("", 4).ListCrawls(ctx)`
- `handleCrawlWarcs` → `cc.NewClient().DownloadManifest()` + check local files
- `handleCrawlData` → `ScanDataDir(crawlDir)` for specific crawl

**Step 4: Run test to verify it passes**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go test ./pkg/index/web/ -run "TestHandle(Overview|Engines|Jobs)" -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/index/web/server.go pkg/index/web/server_test.go
git commit -m "feat(dashboard): add API routes for overview, engines, jobs, crawls"
```

---

## Task 6: CLI Command — `search cc fts dashboard`

**Files:**
- Modify: `cli/cc_fts_web.go` — add `dashboard` command alongside existing `web`
- Modify: `cli/cc_fts.go` — register `dashboard` command

**Step 1: Add dashboard command**

In `cli/cc_fts_web.go`, add `newCCFTSDashboard()`:

```go
func newCCFTSDashboard() *cobra.Command {
	var (
		port    int
		engine  string
		crawlID string
		addr    string
		open    bool
	)

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch admin dashboard for FTS pipeline management",
		Long: `Start the FTS dashboard — a web interface for managing the full
CC FTS pipeline: download WARCs, extract markdown, pack data, build indexes,
search, and browse documents. Real-time progress via WebSocket.`,
		Example: `  search cc fts dashboard
  search cc fts dashboard --port 8080
  search cc fts dashboard --open`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if crawlID == "" {
				crawlID = detectLatestCrawl()
			}
			homeDir, _ := os.UserHomeDir()
			baseDir := filepath.Join(homeDir, "data", "common-crawl", crawlID)

			srv := web.NewDashboard(engine, crawlID, addr, baseDir)

			url := fmt.Sprintf("http://localhost:%d", port)
			fmt.Fprintf(os.Stderr, "FTS Dashboard\n")
			fmt.Fprintf(os.Stderr, "  url:     %s\n", url)
			fmt.Fprintf(os.Stderr, "  engine:  %s\n", engine)
			fmt.Fprintf(os.Stderr, "  crawl:   %s\n", crawlID)
			fmt.Fprintf(os.Stderr, "  data:    %s\n", baseDir)
			fmt.Fprintf(os.Stderr, "\nPress Ctrl+C to stop.\n")

			if open {
				openBrowser(url)
			}

			return srv.ListenAndServe(cmd.Context(), port)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3456, "Listen port")
	cmd.Flags().StringVar(&engine, "engine", "duckdb", "Default FTS engine")
	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&addr, "addr", "", "External engine address")
	cmd.Flags().BoolVar(&open, "open", false, "Open browser on start")
	return cmd
}
```

In `cli/cc_fts.go`, add `cmd.AddCommand(newCCFTSDashboard())` in `newCCFTS()`.

**Step 2: Verify it compiles**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go build ./cmd/search/`
Expected: builds successfully

**Step 3: Commit**

```bash
git add cli/cc_fts_web.go cli/cc_fts.go
git commit -m "feat(cli): add 'search cc fts dashboard' command"
```

---

## Task 7: Dashboard HTML — Layout + Tab Navigation

Replace `pkg/index/web/static/index.html` with the full dashboard SPA.
This is a large file. Build it incrementally: layout + tabs first, then
fill in each tab's content.

**Files:**
- Modify: `pkg/index/web/static/index.html`

**Step 1: Write the dashboard HTML skeleton**

The HTML includes:
- htmx CDN (`<script src="https://unpkg.com/htmx.org@2.0.4">`)
- Tailwind CDN (same config as current: Geist font, zero border-radius)
- Header with 5 tab links: Overview, Pipeline, Search, Browse, Crawls
- `<main id="main">` content area
- Hash-based router (same pattern as current SPA)
- WebSocket client class for progress subscriptions
- All 5 tab renderers

The full HTML is ~800-1000 lines. Key sections:

**Header**: Fixed header with tabs. Active tab gets `border-b border-zinc-50` (brutalist).

**WebSocket client**:
```javascript
class WSClient {
  constructor() {
    this.ws = null;
    this.listeners = new Map(); // jobId → [callback]
  }
  connect() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    this.ws = new WebSocket(`${proto}//${location.host}/ws`);
    this.ws.onmessage = (e) => {
      const msg = JSON.parse(e.data);
      const jobId = msg.job_id;
      (this.listeners.get(jobId) || []).forEach(cb => cb(msg));
      (this.listeners.get('*') || []).forEach(cb => cb(msg));
    };
    this.ws.onclose = () => setTimeout(() => this.connect(), 2000);
  }
  subscribe(jobId, cb) {
    if (!this.listeners.has(jobId)) this.listeners.set(jobId, []);
    this.listeners.get(jobId).push(cb);
    this.ws?.send(JSON.stringify({type: 'subscribe', job_ids: [jobId]}));
  }
}
```

**Tab renderers**: Each tab fetches from the API and renders HTML. The Pipeline tab
uses the WSClient to update progress bars in real-time.

This is written as a complete file replacement — see the implementation step.

**Step 2: Build and verify**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go build ./cmd/search/`
Expected: builds (go:embed picks up new HTML)

**Step 3: Commit**

```bash
git add pkg/index/web/static/index.html
git commit -m "feat(dashboard): replace search SPA with full admin dashboard"
```

---

## Task 8: Overview Tab Content

**Files:**
- Modify: `pkg/index/web/static/index.html` (the renderOverview function)

The Overview tab shows:
1. Stat cards row (crawl ID, docs, disk, WARCs, engines)
2. Data breakdown table
3. Quick action buttons

Fetches from `/api/overview` and renders the DataSummary.

```javascript
async function renderOverview() {
  $('main').innerHTML = loadingHTML();
  try {
    const data = await fetch('/api/overview').then(r => r.json());
    $('main').innerHTML = `
      <div class="py-8 space-y-8 anim-fade-in">
        <h2 class="text-lg font-medium">Overview</h2>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
          ${statCard('Crawl', data.crawl_id)}
          ${statCard('Documents', (data.md_doc_estimate||0).toLocaleString())}
          ${statCard('WARC Files', data.warc_count)}
          ${statCard('Engines', Object.keys(data.fts_engines||{}).length)}
        </div>
        <div class="border border-zinc-800">
          <table class="w-full text-sm">
            <thead><tr class="border-b border-zinc-800 text-xs font-mono text-zinc-500">
              <th class="text-left p-3">Category</th>
              <th class="text-right p-3">Count</th>
              <th class="text-right p-3">Size</th>
            </tr></thead>
            <tbody>
              ${dataRow('WARC files', data.warc_count, data.warc_total_size)}
              ${dataRow('Markdown', data.md_shards + ' shards', data.md_total_size)}
              ${packRows(data.pack_formats)}
              ${ftsRows(data.fts_engines, data.fts_shard_count)}
            </tbody>
          </table>
        </div>
        <div class="flex gap-3">
          <button onclick="navigateTo('/pipeline')" class="px-4 py-2 text-sm border border-zinc-700 hover:bg-zinc-900 transition-colors">Open Pipeline</button>
          <button onclick="navigateTo('/search')" class="px-4 py-2 text-sm border border-zinc-700 hover:bg-zinc-900 transition-colors">Search</button>
        </div>
      </div>`;
  } catch(e) { $('main').innerHTML = errorHTML(e.message); }
}
```

**Commit:**

```bash
git add pkg/index/web/static/index.html
git commit -m "feat(dashboard): implement overview tab with stats and data table"
```

---

## Task 9: Pipeline Tab Content

**Files:**
- Modify: `pkg/index/web/static/index.html` (the renderPipeline function)

The Pipeline tab shows 4 steps vertically. Each step has:
- Status badges per WARC file
- A form to start the job (engine selector, file range, etc.)
- Real-time progress bars (WebSocket-driven)

```javascript
async function renderPipeline() {
  $('main').innerHTML = loadingHTML();
  const [overview, engines, jobs] = await Promise.all([
    fetch('/api/overview').then(r => r.json()),
    fetch('/api/engines').then(r => r.json()),
    fetch('/api/jobs').then(r => r.json()),
  ]);

  const activeJobs = (jobs.jobs || []).filter(j => j.status === 'running');

  $('main').innerHTML = `
    <div class="py-8 space-y-10 anim-fade-in">
      <h2 class="text-lg font-medium">Pipeline</h2>
      ${pipelineStep(1, 'Download WARC', 'download', overview, activeJobs)}
      ${pipelineStep(2, 'Extract Markdown', 'markdown', overview, activeJobs)}
      ${pipelineStep(3, 'Pack', 'pack', overview, activeJobs, engines)}
      ${pipelineStep(4, 'Build Index', 'index', overview, activeJobs, engines)}

      <div class="border-t border-zinc-800 pt-6">
        <h3 class="text-sm font-medium mb-4">Job History</h3>
        <div id="job-history">${renderJobList(jobs.jobs || [])}</div>
      </div>
    </div>`;

  // Subscribe to all active jobs.
  activeJobs.forEach(j => {
    wsClient.subscribe(j.id, (msg) => updateJobProgress(j.id, msg));
  });
}
```

Each `pipelineStep` renders a form that POSTs to `/api/jobs`:
```javascript
function startJob(type, formId) {
  const form = $(formId);
  const data = Object.fromEntries(new FormData(form));
  data.type = type;
  fetch('/api/jobs', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(data),
  }).then(r => r.json()).then(job => {
    wsClient.subscribe(job.id, msg => updateJobProgress(job.id, msg));
    renderPipeline(); // refresh
  });
}
```

Progress bars update via WebSocket callback:
```javascript
function updateJobProgress(jobId, msg) {
  const bar = $(`progress-${jobId}`);
  if (!bar) return;
  if (msg.type === 'progress') {
    bar.style.width = (msg.pct * 100) + '%';
    $(`msg-${jobId}`).textContent = msg.msg;
    $(`rate-${jobId}`).textContent = msg.rate ? msg.rate.toFixed(0) + ' docs/s' : '';
  } else if (msg.type === 'complete' || msg.type === 'failed') {
    renderPipeline(); // refresh on completion
  }
}
```

**Commit:**

```bash
git add pkg/index/web/static/index.html
git commit -m "feat(dashboard): implement pipeline tab with job forms and progress"
```

---

## Task 10: Search Tab (migrate existing)

**Files:**
- Modify: `pkg/index/web/static/index.html`

Copy the existing search functionality (renderHome, doSearch, renderSearchResults,
renderPagination) into the dashboard as the Search tab. The API is unchanged
(`/api/search`). Minor adjustments:
- Search page renders inside `#main` instead of being the root
- Header search input works within dashboard context

**Commit:**

```bash
git add pkg/index/web/static/index.html
git commit -m "feat(dashboard): migrate search tab from existing fts web"
```

---

## Task 11: Browse Tab (migrate existing)

**Files:**
- Modify: `pkg/index/web/static/index.html`

Copy existing browse functionality (renderBrowse, renderShardList, loadShardFiles,
renderDoc) into the dashboard. Same API (`/api/browse`, `/api/doc`).

**Commit:**

```bash
git add pkg/index/web/static/index.html
git commit -m "feat(dashboard): migrate browse tab from existing fts web"
```

---

## Task 12: Crawls Tab

**Files:**
- Modify: `pkg/index/web/static/index.html`

The Crawls tab shows a table of available CC crawls:

```javascript
async function renderCrawls() {
  $('main').innerHTML = loadingHTML();
  try {
    const data = await fetch('/api/crawls').then(r => r.json());
    const crawls = data.crawls || [];
    $('main').innerHTML = `
      <div class="py-8 space-y-6 anim-fade-in">
        <h2 class="text-lg font-medium">Common Crawl Datasets</h2>
        <div class="border border-zinc-800">
          <table class="w-full text-sm">
            <thead><tr class="border-b border-zinc-800 text-xs font-mono text-zinc-500">
              <th class="text-left p-3">Crawl ID</th>
              <th class="text-left p-3">Name</th>
              <th class="text-right p-3">Status</th>
            </tr></thead>
            <tbody>
              ${crawls.map(c => `
                <tr class="border-b border-zinc-800/60 file-row">
                  <td class="p-3 font-mono text-xs">${esc(c.id)}</td>
                  <td class="p-3">${esc(c.name || '')}</td>
                  <td class="p-3 text-right">
                    <span class="text-xs font-mono text-zinc-500">${c.id === activeCrawl ? 'active' : ''}</span>
                  </td>
                </tr>`).join('')}
            </tbody>
          </table>
        </div>
      </div>`;
  } catch(e) { $('main').innerHTML = errorHTML(e.message); }
}
```

**Commit:**

```bash
git add pkg/index/web/static/index.html
git commit -m "feat(dashboard): implement crawls tab with CC dataset listing"
```

---

## Task 13: Integration Test — Full Lifecycle

**Files:**
- Create: `pkg/index/web/integration_test.go`

Test the full lifecycle: create job via API, receive WebSocket progress, verify completion.

```go
// pkg/index/web/integration_test.go
package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashboardRoutes(t *testing.T) {
	tmp := t.TempDir()
	srv := &Server{
		EngineName: "duckdb",
		CrawlID:    "CC-TEST",
		FTSBase:    tmp,
		MDBase:     tmp,
		CrawlDir:   tmp,
		Hub:        NewWSHub(),
	}
	srv.Jobs = NewJobManager(srv.Hub, tmp, "CC-TEST")

	handler := srv.Handler()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Test /api/overview
	resp, err := http.Get(ts.URL + "/api/overview")
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("overview status = %d", resp.StatusCode)
	}

	// Test /api/engines
	resp, _ = http.Get(ts.URL + "/api/engines")
	if resp.StatusCode != 200 {
		t.Errorf("engines status = %d", resp.StatusCode)
	}

	// Test /api/jobs (empty)
	resp, _ = http.Get(ts.URL + "/api/jobs")
	if resp.StatusCode != 200 {
		t.Errorf("jobs status = %d", resp.StatusCode)
	}
	var jobsResp map[string]any
	json.NewDecoder(resp.Body).Decode(&jobsResp)
	if jobs := jobsResp["jobs"].([]any); len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}

	// Test / serves HTML
	resp, _ = http.Get(ts.URL + "/")
	if resp.StatusCode != 200 {
		t.Errorf("index status = %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("index content-type = %s", ct)
	}
}
```

**Step 1: Run integration test**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go test ./pkg/index/web/ -run TestDashboardRoutes -v -count=1`
Expected: PASS

**Step 2: Commit**

```bash
git add pkg/index/web/integration_test.go
git commit -m "test(dashboard): add integration test for dashboard API routes"
```

---

## Task 14: Final Polish

**Files:**
- Modify: `pkg/index/web/static/index.html` — keyboard shortcuts, mobile layout, error states

Additions:
1. **Keyboard shortcuts**: Cmd+K focuses search (same as current), number keys 1-5 switch tabs
2. **Mobile responsive**: Stack stat cards vertically on small screens
3. **Error states**: "No data directory" message with CLI instructions when crawl dir missing
4. **Light/dark theme**: Same toggle as current SPA
5. **Auto-reconnect WebSocket**: Exponential backoff on disconnect

**Commit:**

```bash
git add pkg/index/web/static/index.html
git commit -m "feat(dashboard): add keyboard shortcuts, mobile layout, error states"
```

---

## Task 15: Verify and Clean Up

**Step 1: Run all tests**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go test ./pkg/index/web/ -v -count=1`
Expected: All tests pass

**Step 2: Build binary**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go build -o /tmp/search-dashboard ./cmd/search/`
Expected: Builds successfully

**Step 3: Smoke test**

Run: `/tmp/search-dashboard cc fts dashboard --port 3457`
Expected: Server starts, prints URL, serves dashboard HTML at localhost:3457

**Step 4: Remove duplicate formatBytes**

Clean up the duplicate `formatBytes` (lowercase) in server.go — replace calls with
`FormatBytes` (exported) from scanner.go.

**Step 5: Final commit**

```bash
git add -A
git commit -m "chore(dashboard): clean up duplicates, verify all tests pass"
```
