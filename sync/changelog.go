package sync

import (
	"context"
	"sync"
)

// ChangeLog stores and retrieves changes.
type ChangeLog interface {
	// Append adds a change and returns the assigned cursor.
	Append(ctx context.Context, change Change) (uint64, error)

	// Since returns changes after the given cursor for a scope.
	// If scope is empty, returns changes for all scopes.
	Since(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, error)

	// Cursor returns the current latest cursor.
	Cursor(ctx context.Context) (uint64, error)

	// Trim removes changes older than the given cursor (for compaction).
	Trim(ctx context.Context, beforeCursor uint64) error
}

// MemoryChangeLog is an in-memory implementation of ChangeLog.
type MemoryChangeLog struct {
	mu      sync.RWMutex
	entries []Change
	cursor  uint64
}

// NewMemoryChangeLog creates a new in-memory change log.
func NewMemoryChangeLog() *MemoryChangeLog {
	return &MemoryChangeLog{
		entries: make([]Change, 0),
	}
}

// Append adds a change and returns the assigned cursor.
func (c *MemoryChangeLog) Append(ctx context.Context, change Change) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cursor++
	change.Cursor = c.cursor
	c.entries = append(c.entries, change)

	return c.cursor, nil
}

// Since returns changes after the given cursor for a scope.
func (c *MemoryChangeLog) Since(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	var result []Change
	for _, entry := range c.entries {
		if entry.Cursor <= cursor {
			continue
		}
		if scope != "" && entry.Scope != scope {
			continue
		}
		result = append(result, entry)
		if len(result) >= limit {
			break
		}
	}

	return result, nil
}

// Cursor returns the current latest cursor.
func (c *MemoryChangeLog) Cursor(ctx context.Context) (uint64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cursor, nil
}

// Trim removes changes older than the given cursor.
func (c *MemoryChangeLog) Trim(ctx context.Context, beforeCursor uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find the first entry at or after beforeCursor
	idx := 0
	for i, entry := range c.entries {
		if entry.Cursor >= beforeCursor {
			idx = i
			break
		}
	}

	if idx > 0 {
		c.entries = c.entries[idx:]
	}

	return nil
}

// Len returns the number of entries in the log.
func (c *MemoryChangeLog) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
