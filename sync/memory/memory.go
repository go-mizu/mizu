// Package memory provides in-memory implementations of sync interfaces.
//
// These implementations are suitable for development, testing, and
// single-server deployments. For production use with persistence
// or horizontal scaling, use database-backed implementations.
package memory

import (
	"context"
	gosync "sync"

	"github.com/go-mizu/mizu/sync"
)

// -----------------------------------------------------------------------------
// Log
// -----------------------------------------------------------------------------

// Log is an in-memory implementation of sync.Log.
type Log struct {
	mu        gosync.RWMutex
	entries   map[string][]sync.Change // scope -> changes
	cursors   map[string]uint64        // scope -> current cursor
	minCursor map[string]uint64        // scope -> minimum cursor after trim
	global    uint64                   // global cursor counter
}

// NewLog creates a new in-memory log.
func NewLog() *Log {
	return &Log{
		entries:   make(map[string][]sync.Change),
		cursors:   make(map[string]uint64),
		minCursor: make(map[string]uint64),
	}
}

// Append adds changes to the log and returns the final cursor.
func (l *Log) Append(ctx context.Context, scope string, changes []sync.Change) (uint64, error) {
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
// Returns ErrCursorTooOld if the cursor has been trimmed from the log.
func (l *Log) Since(ctx context.Context, scope string, cursor uint64, limit int) ([]sync.Change, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check if cursor is too old (has been trimmed)
	if minCursor, ok := l.minCursor[scope]; ok && cursor > 0 && cursor < minCursor {
		return nil, sync.ErrCursorTooOld
	}

	if limit <= 0 {
		limit = 100
	}

	entries := l.entries[scope]
	var result []sync.Change

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
		// Track minimum cursor for ErrCursorTooOld detection
		l.minCursor[scope] = before
	}
	return nil
}

// -----------------------------------------------------------------------------
// Dedupe
// -----------------------------------------------------------------------------

// Dedupe is an in-memory implementation of sync.Dedupe.
type Dedupe struct {
	mu   gosync.RWMutex
	seen map[string]map[string]bool // scope -> id -> seen
}

// NewDedupe creates a new in-memory dedupe tracker.
func NewDedupe() *Dedupe {
	return &Dedupe{
		seen: make(map[string]map[string]bool),
	}
}

// Seen returns true if the mutation has already been processed.
func (d *Dedupe) Seen(ctx context.Context, scope, id string) (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	scoped, ok := d.seen[scope]
	if !ok {
		return false, nil
	}
	return scoped[id], nil
}

// Mark records that a mutation has been processed.
func (d *Dedupe) Mark(ctx context.Context, scope, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.seen[scope] == nil {
		d.seen[scope] = make(map[string]bool)
	}
	d.seen[scope][id] = true
	return nil
}

