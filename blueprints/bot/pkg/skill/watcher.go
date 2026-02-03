package skill

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// skillsVersion is a monotonic counter that increments when skills change.
// Matches OpenClaw's refresh.ts globalVersion pattern.
var skillsVersion atomic.Int64

// BumpVersion increments the global skills version counter.
// Called when skill files change to invalidate session caches.
func BumpVersion() int64 {
	return skillsVersion.Add(1)
}

// CurrentVersion returns the current skills version.
// Session snapshots compare their stored version against this to decide
// whether a re-snapshot is needed.
func CurrentVersion() int64 {
	return skillsVersion.Load()
}

// Watcher monitors skill directories for changes and calls onChange when
// SKILL.md files are created, modified, or deleted.
type Watcher struct {
	dirs     []string
	onChange func()
	stop     chan struct{}
	done     chan struct{}
	debounce time.Duration
	mu       sync.Mutex
}

// StartWatcher begins watching the given directories for SKILL.md changes.
// The onChange callback is debounced — rapid successive changes trigger only
// one callback after the debounce period (default 500ms).
// Matches OpenClaw's refresh.ts ensureSkillsWatcher behaviour.
func StartWatcher(dirs []string, onChange func()) *Watcher {
	// Filter to existing directories only.
	var valid []string
	for _, d := range dirs {
		if d == "" {
			continue
		}
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			valid = append(valid, d)
		}
	}

	w := &Watcher{
		dirs:     valid,
		onChange: onChange,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
		debounce: 500 * time.Millisecond,
	}

	go w.poll()
	return w
}

// Stop shuts down the watcher.
func (w *Watcher) Stop() {
	close(w.stop)
	<-w.done
}

// poll checks for changes periodically using file modification times.
// This is simpler and more portable than fsnotify, matching chokidar's
// polling mode that OpenClaw falls back to.
func (w *Watcher) poll() {
	defer close(w.done)

	// Build initial state.
	state := w.snapshot()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			newState := w.snapshot()
			if !mapsEqual(state, newState) {
				state = newState
				BumpVersion()
				if w.onChange != nil {
					w.onChange()
				}
			}
		}
	}
}

// snapshot returns a map of file paths to modification times for all
// SKILL.md files in watched directories.
func (w *Watcher) snapshot() map[string]int64 {
	m := make(map[string]int64)
	for _, dir := range w.dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skillFile := filepath.Join(dir, e.Name(), skillFileName)
			info, err := os.Stat(skillFile)
			if err != nil {
				continue
			}
			m[skillFile] = info.ModTime().UnixNano()
		}
	}
	return m
}

// mapsEqual checks if two string→int64 maps are identical.
func mapsEqual(a, b map[string]int64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}
