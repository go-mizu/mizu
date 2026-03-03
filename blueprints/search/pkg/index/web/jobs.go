package web

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ── Job Types ────────────────────────────────────────────────────────────

// JobConfig describes the parameters for a pipeline job.
type JobConfig struct {
	Type    string `json:"type"`    // download, markdown, pack, index
	CrawlID string `json:"crawl"`
	Files   string `json:"files"`   // "0", "0-4", "all"
	Engine  string `json:"engine"`  // for index jobs
	Source  string `json:"source"`  // for index jobs (files, parquet, bin, etc.)
	Format  string `json:"format"`  // for pack jobs
	Fast    bool   `json:"fast"`    // for markdown jobs
}

// Job represents a single pipeline job tracked by the JobManager.
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

// ── JobManager ───────────────────────────────────────────────────────────

// JobManager manages pipeline jobs in-memory. It is safe for concurrent use.
type JobManager struct {
	mu      sync.RWMutex
	jobs    map[string]*Job
	order   []string // job IDs in creation order
	hub     *WSHub
	baseDir string
	crawlID string
}

// NewJobManager creates a new JobManager that broadcasts updates via hub.
func NewJobManager(hub *WSHub, baseDir, crawlID string) *JobManager {
	return &JobManager{
		jobs:    make(map[string]*Job),
		hub:     hub,
		baseDir: baseDir,
		crawlID: crawlID,
	}
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
	m.mu.Unlock()

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
	// Reverse iteration for newest-first ordering.
	for i := len(m.order) - 1; i >= 0; i-- {
		if job, ok := m.jobs[m.order[i]]; ok {
			result = append(result, job)
		}
	}
	return result
}

// Cancel cancels a job. If the job has a cancel function (i.e., it is running),
// the function is called. Returns false if the job ID is not found.
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
	m.mu.Unlock()

	if cancelFn != nil {
		cancelFn()
	}

	m.hub.Broadcast(id, map[string]any{
		"type":   "job_update",
		"job_id": id,
		"status": "cancelled",
	})
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

	m.hub.Broadcast(id, map[string]any{
		"type":     "job_progress",
		"job_id":   id,
		"progress": pct,
		"message":  msg,
		"rate":     rate,
	})
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
	m.mu.Unlock()

	m.hub.Broadcast(id, map[string]any{
		"type":   "job_update",
		"job_id": id,
		"status": "completed",
	})
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
	m.mu.Unlock()

	m.hub.Broadcast(id, map[string]any{
		"type":   "job_update",
		"job_id": id,
		"status": "failed",
		"error":  err.Error(),
	})
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
	m.mu.Unlock()

	m.hub.Broadcast(id, map[string]any{
		"type":   "job_update",
		"job_id": id,
		"status": "running",
	})
}
