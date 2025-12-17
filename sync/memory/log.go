package memory

import (
	"context"
	"sync"

	gosync "github.com/go-mizu/mizu/sync"
)

// Log is an in-memory implementation of sync.Log.
type Log struct {
	mu      sync.RWMutex
	entries map[string][]gosync.Change // scope -> changes
	cursors map[string]uint64          // scope -> current cursor
	global  uint64                     // global cursor counter
}

// NewLog creates a new in-memory log.
func NewLog() *Log {
	return &Log{
		entries: make(map[string][]gosync.Change),
		cursors: make(map[string]uint64),
	}
}

// Append adds changes to the log and returns the final cursor.
func (l *Log) Append(ctx context.Context, scope string, changes []gosync.Change) (uint64, error) {
	if len(changes) == 0 {
		return l.Cursor(ctx, scope)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for i := range changes {
		l.global++
		changes[i].Cursor = l.global
		if changes[i].Scope == "" {
			changes[i].Scope = scope
		}
		l.entries[scope] = append(l.entries[scope], changes[i])
	}

	l.cursors[scope] = l.global
	return l.global, nil
}

// Since returns changes after the given cursor for a scope.
func (l *Log) Since(ctx context.Context, scope string, cursor uint64, limit int) ([]gosync.Change, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	entries := l.entries[scope]
	var result []gosync.Change

	for _, entry := range entries {
		if entry.Cursor <= cursor {
			continue
		}
		result = append(result, entry)
		if len(result) >= limit {
			break
		}
	}

	return result, nil
}

// Cursor returns the current latest cursor for a scope.
func (l *Log) Cursor(ctx context.Context, scope string) (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.cursors[scope], nil
}

// Trim removes changes before the given cursor.
func (l *Log) Trim(ctx context.Context, scope string, before uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries := l.entries[scope]
	idx := 0
	for i, entry := range entries {
		if entry.Cursor >= before {
			idx = i
			break
		}
		// If we reach the end, all entries are before 'before'
		if i == len(entries)-1 {
			idx = len(entries)
		}
	}

	if idx > 0 {
		l.entries[scope] = entries[idx:]
	}
	return nil
}

// Len returns the number of entries in the log for a scope.
func (l *Log) Len(scope string) int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries[scope])
}

// Clear removes all entries from the log.
func (l *Log) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = make(map[string][]gosync.Change)
	l.cursors = make(map[string]uint64)
	l.global = 0
}
