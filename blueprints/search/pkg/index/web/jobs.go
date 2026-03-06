package web

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore"
	"github.com/google/uuid"
)

// ── Job Types ────────────────────────────────────────────────────────────

// JobConfig describes the parameters for a pipeline job.
type JobConfig struct {
	Type    string `json:"type"` // download, markdown, pack, index
	CrawlID string `json:"crawl"`
	Files   string `json:"files"`  // "0", "0-4", "all"
	Engine  string `json:"engine"` // for index jobs
	Source  string `json:"source"` // for index jobs (files, parquet, bin, etc.)
	Format  string `json:"format"` // for pack jobs
	Fast    bool   `json:"fast"`   // for markdown jobs
}

// Job represents a single pipeline job tracked by the JobManager.
type Job struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Status    string     `json:"status"` // queued, running, completed, failed, cancelled
	Config    JobConfig  `json:"config"`
	Progress  float64    `json:"progress"` // 0.0–1.0
	Message   string     `json:"message"`
	Rate      float64    `json:"rate,omitempty"` // items/sec
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Error     string     `json:"error,omitempty"`
	cancel    context.CancelFunc
}

// JobCompleteHook is called when a job transitions to completed status.
type JobCompleteHook func(job *Job, crawlID, crawlDir string)

// ── WebSocket event types ────────────────────────────────────────────────

type wsJobUpdate struct {
	Type   string `json:"type"`
	JobID  string `json:"job_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type wsJobProgress struct {
	Type     string  `json:"type"`
	JobID    string  `json:"job_id"`
	Progress float64 `json:"progress"`
	Message  string  `json:"message"`
	Rate     float64 `json:"rate"`
}

// ── JobManager ───────────────────────────────────────────────────────────

// JobManager manages pipeline jobs with in-memory caching and optional
// persistence via metastore. It is safe for concurrent use.
type JobManager struct {
	mu      sync.RWMutex
	jobs    map[string]*Job
	order   []string // job IDs in creation order
	hub     *WSHub
	baseDir string
	crawlID string
	store   metastore.Store // optional persistence

	persistCh chan metastore.JobRecord // batched persist channel
	stopOnce  sync.Once

	onComplete JobCompleteHook

	manifestMu    sync.Mutex
	manifestCache map[string]manifestCacheEntry
	manifestFetch func(ctx context.Context, crawlID string) ([]string, error)
}

type manifestCacheEntry struct {
	paths     []string
	fetchedAt time.Time
}

// NewJobManager creates a new JobManager that broadcasts updates via hub.
func NewJobManager(hub *WSHub, baseDir, crawlID string) *JobManager {
	return &JobManager{
		jobs:          make(map[string]*Job),
		hub:           hub,
		baseDir:       baseDir,
		crawlID:       crawlID,
		manifestCache: make(map[string]manifestCacheEntry),
	}
}

// SetStore configures persistence and starts the async flush goroutine.
// Call before LoadHistory.
func (m *JobManager) SetStore(s metastore.Store) {
	m.mu.Lock()
	m.store = s
	m.persistCh = make(chan metastore.JobRecord, 256)
	m.mu.Unlock()
	go m.persistFlusher(s)
}

// persistFlusher drains the persist channel in batches.
func (m *JobManager) persistFlusher(s metastore.Store) {
	for rec := range m.persistCh {
		// Drain any additional queued records into a batch.
		batch := []metastore.JobRecord{rec}
		drain := true
		for drain {
			select {
			case r, ok := <-m.persistCh:
				if !ok {
					drain = false
				} else {
					batch = append(batch, r)
				}
			default:
				drain = false
			}
		}
		// Deduplicate: keep last record per job ID.
		seen := make(map[string]int, len(batch))
		for i, r := range batch {
			seen[r.ID] = i
		}
		for _, idx := range seen {
			if err := s.PutJob(context.Background(), batch[idx]); err != nil {
				logErrorf("jobs persist id=%s err=%v", batch[idx].ID, err)
			}
		}
	}
}

// StopPersist closes the persist channel and flushes remaining records.
func (m *JobManager) StopPersist() {
	m.stopOnce.Do(func() {
		m.mu.RLock()
		ch := m.persistCh
		m.mu.RUnlock()
		if ch != nil {
			close(ch)
		}
	})
}

// LoadHistory loads completed/failed/cancelled jobs from the store into memory.
// Running/queued jobs are not restored (they can't resume after restart).
func (m *JobManager) LoadHistory(ctx context.Context) {
	m.mu.RLock()
	s := m.store
	m.mu.RUnlock()
	if s == nil {
		return
	}

	recs, err := s.ListJobs(ctx)
	if err != nil {
		logErrorf("jobs load-history err=%v", err)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rec := range recs {
		if rec.Status == "queued" || rec.Status == "running" {
			continue
		}
		if _, exists := m.jobs[rec.ID]; exists {
			continue
		}
		var cfg JobConfig
		_ = json.Unmarshal([]byte(rec.Config), &cfg)
		job := &Job{
			ID:        rec.ID,
			Type:      rec.Type,
			Status:    rec.Status,
			Config:    cfg,
			Progress:  rec.Progress,
			Message:   rec.Message,
			Rate:      rec.Rate,
			StartedAt: rec.StartedAt,
			EndedAt:   rec.EndedAt,
			Error:     rec.Error,
		}
		m.jobs[rec.ID] = job
		m.order = append(m.order, rec.ID)
	}
	logInfof("jobs loaded %d history records from store", len(recs))
}

// Create adds a new job with a unique short ID in "queued" status.
func (m *JobManager) Create(cfg JobConfig) *Job {
	id := uuid.New().String()[:8]

	job := &Job{
		ID:        id,
		Type:      cfg.Type,
		Status:    "queued",
		Config:    cfg,
		StartedAt: time.Now(),
	}

	m.mu.Lock()
	m.jobs[id] = job
	m.order = append(m.order, id)
	rec := snapshotJob(job)
	ch := m.persistCh
	m.mu.Unlock()

	enqueuePersist(ch, rec)
	return job
}

// Get returns the job with the given ID, or nil if not found.
func (m *JobManager) Get(id string) *Job {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.jobs[id]
}

// List returns all jobs, newest first.
func (m *JobManager) List() []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Job, 0, len(m.order))
	for i := len(m.order) - 1; i >= 0; i-- {
		if job, ok := m.jobs[m.order[i]]; ok {
			result = append(result, job)
		}
	}
	return result
}

// Cancel cancels a job. Returns false if the job ID is not found.
func (m *JobManager) Cancel(id string) bool {
	m.mu.Lock()
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return false
	}
	now := time.Now()
	job.Status = "cancelled"
	job.EndedAt = &now
	cancelFn := job.cancel
	rec := snapshotJob(job)
	ch := m.persistCh
	m.mu.Unlock()

	if cancelFn != nil {
		cancelFn()
	}

	enqueuePersist(ch, rec)
	m.hub.Broadcast(id, wsJobUpdate{Type: "job_update", JobID: id, Status: "cancelled"})
	logInfof("job lifecycle id=%s status=cancelled", id)
	return true
}

// UpdateProgress updates a job's progress and broadcasts the change.
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

	m.hub.Broadcast(id, wsJobProgress{Type: "job_progress", JobID: id, Progress: pct, Message: msg, Rate: rate})
}

// Complete marks a job as completed with a final message.
func (m *JobManager) Complete(id string, msg string) {
	m.mu.Lock()
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	now := time.Now()
	job.Status = "completed"
	job.Progress = 1.0
	job.Message = msg
	job.EndedAt = &now
	hook := m.onComplete
	crawlID := m.resolveCrawlID(job)
	crawlDir := m.resolveCrawlDir(crawlID)
	rec := snapshotJob(job)
	ch := m.persistCh
	m.mu.Unlock()

	enqueuePersist(ch, rec)
	m.hub.Broadcast(id, wsJobUpdate{Type: "job_update", JobID: id, Status: "completed"})
	logInfof("job lifecycle id=%s status=completed msg=%q", id, msg)

	if hook != nil {
		hook(job, crawlID, crawlDir)
	}
}

// Fail marks a job as failed with the given error.
func (m *JobManager) Fail(id string, err error) {
	m.mu.Lock()
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	now := time.Now()
	job.Status = "failed"
	job.Error = err.Error()
	job.EndedAt = &now
	rec := snapshotJob(job)
	ch := m.persistCh
	m.mu.Unlock()

	enqueuePersist(ch, rec)
	m.hub.Broadcast(id, wsJobUpdate{Type: "job_update", JobID: id, Status: "failed", Error: err.Error()})
	logErrorf("job lifecycle id=%s status=failed err=%v", id, err)
}

// SetRunning marks a job as running and stores its cancel function.
func (m *JobManager) SetRunning(id string, cancel context.CancelFunc) {
	m.mu.Lock()
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	job.Status = "running"
	job.cancel = cancel
	rec := snapshotJob(job)
	ch := m.persistCh
	m.mu.Unlock()

	enqueuePersist(ch, rec)
	m.hub.Broadcast(id, wsJobUpdate{Type: "job_update", JobID: id, Status: "running"})
	logInfof("job lifecycle id=%s status=running", id)
}

// Clear removes all non-active jobs from memory and store.
func (m *JobManager) Clear() int {
	m.mu.Lock()
	var kept []string
	var activeRecs []metastore.JobRecord
	removed := 0
	for _, id := range m.order {
		job, ok := m.jobs[id]
		if !ok {
			continue
		}
		if job.Status == "running" || job.Status == "queued" {
			kept = append(kept, id)
			activeRecs = append(activeRecs, snapshotJob(job))
		} else {
			delete(m.jobs, id)
			removed++
		}
	}
	m.order = kept
	s := m.store
	m.mu.Unlock()

	if s != nil {
		go func() {
			if err := s.DeleteAllJobs(context.Background()); err != nil {
				logErrorf("jobs clear-store err=%v", err)
				return
			}
			for _, rec := range activeRecs {
				if err := s.PutJob(context.Background(), rec); err != nil {
					logErrorf("jobs re-persist id=%s err=%v", rec.ID, err)
				}
			}
		}()
	}

	logInfof("jobs cleared %d history entries", removed)
	return removed
}

// SetCompleteHook sets a callback fired whenever a job completes successfully.
func (m *JobManager) SetCompleteHook(h JobCompleteHook) {
	m.mu.Lock()
	m.onComplete = h
	m.mu.Unlock()
}

func (m *JobManager) resolveCrawlID(job *Job) string {
	if job != nil && job.Config.CrawlID != "" {
		return job.Config.CrawlID
	}
	return m.crawlID
}

func (m *JobManager) resolveCrawlDir(crawlID string) string {
	if crawlID == m.crawlID {
		return m.baseDir
	}
	return filepath.Join(filepath.Dir(m.baseDir), crawlID)
}

func (m *JobManager) setManifestFetcher(fn func(ctx context.Context, crawlID string) ([]string, error)) {
	m.mu.Lock()
	m.manifestFetch = fn
	m.mu.Unlock()
}

// snapshotJob creates a JobRecord snapshot. Must be called while holding m.mu.
func snapshotJob(job *Job) metastore.JobRecord {
	cfgJSON, _ := json.Marshal(job.Config)
	rec := metastore.JobRecord{
		ID:        job.ID,
		Type:      job.Type,
		Status:    job.Status,
		Config:    string(cfgJSON),
		Progress:  job.Progress,
		Message:   job.Message,
		Rate:      job.Rate,
		Error:     job.Error,
		StartedAt: job.StartedAt,
	}
	if job.EndedAt != nil {
		t := *job.EndedAt
		rec.EndedAt = &t
	}
	return rec
}

// enqueuePersist sends a job record to the persist channel. No-op if channel is nil.
func enqueuePersist(ch chan metastore.JobRecord, rec metastore.JobRecord) {
	if ch == nil {
		return
	}
	select {
	case ch <- rec:
	default:
		logErrorf("jobs persist queue full, dropping id=%s", rec.ID)
	}
}
