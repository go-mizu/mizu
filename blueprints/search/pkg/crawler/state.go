package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CrawlState holds the persistent state of a crawl for resume capability.
type CrawlState struct {
	StartURL  string     `json:"start_url"`
	StartedAt time.Time  `json:"started_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Stats     CrawlStats `json:"stats"`
	Visited   []string   `json:"visited"`
	Pending   []URLEntry `json:"pending"`
}

// SaveState saves crawl state to a JSON file.
func SaveState(path string, state *CrawlState) error {
	state.UpdatedAt = time.Now()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	// Write to temp file, then rename for atomicity
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming state file: %w", err)
	}

	return nil
}

// LoadState loads crawl state from a JSON file.
// Returns nil if the file does not exist.
func LoadState(path string) (*CrawlState, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var state CrawlState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}

	return &state, nil
}

// RemoveState deletes the state file.
func RemoveState(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// StateExists checks if a state file exists.
func StateExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
