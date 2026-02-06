package crawler

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crawl_state.json")

	state := &CrawlState{
		StartURL:  "https://example.com",
		StartedAt: time.Now().Truncate(time.Second),
		Stats: CrawlStats{
			PagesTotal:   10,
			PagesSuccess: 8,
			PagesFailed:  2,
		},
		Visited: []string{
			"https://example.com/a",
			"https://example.com/b",
		},
		Pending: []URLEntry{
			{URL: "https://example.com/c", Depth: 1, Priority: 0},
		},
	}

	if err := SaveState(path, state); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState error: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadState returned nil")
	}

	if loaded.StartURL != state.StartURL {
		t.Errorf("StartURL = %q, want %q", loaded.StartURL, state.StartURL)
	}
	if loaded.Stats.PagesTotal != 10 {
		t.Errorf("PagesTotal = %d, want 10", loaded.Stats.PagesTotal)
	}
	if len(loaded.Visited) != 2 {
		t.Errorf("Visited = %d, want 2", len(loaded.Visited))
	}
	if len(loaded.Pending) != 1 {
		t.Errorf("Pending = %d, want 1", len(loaded.Pending))
	}
}

func TestLoadStateNotFound(t *testing.T) {
	state, err := LoadState("/nonexistent/path/state.json")
	if err != nil {
		t.Fatalf("LoadState error: %v", err)
	}
	if state != nil {
		t.Error("expected nil for nonexistent file")
	}
}

func TestRemoveState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	os.WriteFile(path, []byte("{}"), 0o644)

	if !StateExists(path) {
		t.Fatal("state file should exist")
	}

	if err := RemoveState(path); err != nil {
		t.Fatalf("RemoveState error: %v", err)
	}

	if StateExists(path) {
		t.Error("state file should not exist after removal")
	}

	// Removing nonexistent should not error
	if err := RemoveState(path); err != nil {
		t.Fatalf("RemoveState nonexistent error: %v", err)
	}
}
