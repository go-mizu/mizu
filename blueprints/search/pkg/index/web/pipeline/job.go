package pipeline

import (
	"context"
	"time"
)

// JobConfig describes the parameters for a pipeline task.
type JobConfig struct {
	Type    string `json:"type"` // download, markdown, pack, index, scrape, scrape_markdown
	CrawlID string `json:"crawl"`
	Files   string `json:"files"`  // "0", "0-4", "all"
	Engine  string `json:"engine"` // for index jobs
	Source  string `json:"source"` // for index/scrape jobs
	Format  string `json:"format"` // for pack jobs
	Domain  string `json:"domain,omitempty"` // for scrape jobs
}

// Job represents a single pipeline task tracked by the Manager.
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
	Cancel    context.CancelFunc `json:"-"`
}

// CompleteHook is called when a job transitions to completed status.
type CompleteHook func(job *Job, crawlID, crawlDir string)

// Broadcaster delivers real-time updates to connected clients.
type Broadcaster interface {
	Broadcast(jobID string, msg any)
	BroadcastAll(msg any)
}

// jobUpdate is the WS payload for status transitions.
type jobUpdate struct {
	Type   string `json:"type"`
	JobID  string `json:"job_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// jobProgress is the WS payload for in-flight progress updates.
type jobProgress struct {
	Type     string  `json:"type"`
	JobID    string  `json:"job_id"`
	Progress float64 `json:"progress"`
	Message  string  `json:"message"`
	Rate     float64 `json:"rate"`
}
