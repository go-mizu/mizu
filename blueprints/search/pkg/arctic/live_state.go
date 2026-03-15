package arctic

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Phase values for StateSnapshot.Phase and LiveCurrent.Phase.
const (
	PhaseIdle        = "idle"
	PhaseDownloading = "downloading"
	PhaseValidating  = "validating"
	PhaseProcessing  = "processing"
	PhaseCommitting  = "committing"
	PhaseRetrying    = "retrying"
	PhaseDone        = "done"
)

// LiveCurrent describes what is actively being processed.
type LiveCurrent struct {
	YM         string `json:"ym"`
	Type       string `json:"type"`
	Phase      string `json:"phase"`
	Shard      int    `json:"shard,omitempty"`
	Rows       int64  `json:"rows,omitempty"`
	BytesDone  int64  `json:"bytes_done,omitempty"`
	BytesTotal int64  `json:"bytes_total,omitempty"`
}

// SessionStats holds aggregate counters for the running session.
type SessionStats struct {
	Committed    int   `json:"committed"`
	Skipped      int   `json:"skipped"`
	Retries      int   `json:"retries"`
	TotalRows    int64 `json:"total_rows"`
	TotalBytes   int64 `json:"total_bytes"`
	TotalMonths  int   `json:"total_months"`
}

// StateSnapshot is the JSON-serializable point-in-time view of the session.
type StateSnapshot struct {
	SessionID string       `json:"session_id"`
	StartedAt time.Time    `json:"started_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Phase     string       `json:"phase"`
	Current   *LiveCurrent `json:"current,omitempty"`
	Stats     SessionStats `json:"stats"`
}

// LiveState holds mutable session state, safe for concurrent use.
type LiveState struct {
	mu   sync.RWMutex
	snap StateSnapshot
}

// NewLiveState creates a LiveState initialised for a new session.
func NewLiveState(totalMonths int) *LiveState {
	now := time.Now().UTC()
	return &LiveState{
		snap: StateSnapshot{
			SessionID: now.Format(time.RFC3339),
			StartedAt: now,
			UpdatedAt: now,
			Phase:     PhaseIdle,
			Stats:     SessionStats{TotalMonths: totalMonths},
		},
	}
}

// Update applies fn under a write lock and stamps UpdatedAt.
func (s *LiveState) Update(fn func(*StateSnapshot)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.snap)
	s.snap.UpdatedAt = time.Now().UTC()
}

// Snapshot returns a deep copy of the current state under a read lock.
func (s *LiveState) Snapshot() StateSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := s.snap
	if s.snap.Current != nil {
		cur := *s.snap.Current
		cp.Current = &cur
	}
	return cp
}

// WriteStateJSON atomically writes snap to {RepoRoot}/states.json.
func WriteStateJSON(cfg Config, snap StateSnapshot) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tmp, err := os.CreateTemp(cfg.RepoRoot, ".states-*.json")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmp.Name(), cfg.StatesJSONPath()); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
