// Package memory provides in-memory implementations of sync interfaces.
//
// These implementations are suitable for development, testing, and
// single-server deployments. For production use with persistence
// or horizontal scaling, use database-backed implementations.
package memory

import (
	"context"
	"encoding/json"
	gosync "sync"

	"github.com/go-mizu/mizu/sync"
)

// -----------------------------------------------------------------------------
// Store
// -----------------------------------------------------------------------------

// Store is an in-memory implementation of sync.Store.
type Store struct {
	mu   gosync.RWMutex
	data map[string]map[string]map[string]json.RawMessage // scope -> entity -> id -> data
}

// NewStore creates a new in-memory store.
func NewStore() *Store {
	return &Store{
		data: make(map[string]map[string]map[string]json.RawMessage),
	}
}

// Get retrieves an entity by scope/entity/id.
func (s *Store) Get(ctx context.Context, scope, entity, id string) (json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entities, ok := s.data[scope]
	if !ok {
		return nil, sync.ErrNotFound
	}
	items, ok := entities[entity]
	if !ok {
		return nil, sync.ErrNotFound
	}
	data, ok := items[id]
	if !ok {
		return nil, sync.ErrNotFound
	}

	// Return a copy to avoid mutation
	cp := make(json.RawMessage, len(data))
	copy(cp, data)
	return cp, nil
}

// Set stores an entity.
func (s *Store) Set(ctx context.Context, scope, entity, id string, data json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data[scope] == nil {
		s.data[scope] = make(map[string]map[string]json.RawMessage)
	}
	if s.data[scope][entity] == nil {
		s.data[scope][entity] = make(map[string]json.RawMessage)
	}

	// Store a copy
	cp := make(json.RawMessage, len(data))
	copy(cp, data)
	s.data[scope][entity][id] = cp
	return nil
}

// Delete removes an entity.
func (s *Store) Delete(ctx context.Context, scope, entity, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entities, ok := s.data[scope]; ok {
		if items, ok := entities[entity]; ok {
			delete(items, id)
			if len(items) == 0 {
				delete(entities, entity)
			}
			if len(entities) == 0 {
				delete(s.data, scope)
			}
		}
	}
	return nil
}

// Snapshot returns all data in a scope.
func (s *Store) Snapshot(ctx context.Context, scope string) (map[string]map[string]json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entities, ok := s.data[scope]
	if !ok {
		return make(map[string]map[string]json.RawMessage), nil
	}

	// Deep copy
	result := make(map[string]map[string]json.RawMessage, len(entities))
	for entity, items := range entities {
		result[entity] = make(map[string]json.RawMessage, len(items))
		for id, data := range items {
			cp := make(json.RawMessage, len(data))
			copy(cp, data)
			result[entity][id] = cp
		}
	}
	return result, nil
}

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
// Applied
// -----------------------------------------------------------------------------

// Applied is an in-memory implementation of sync.Applied.
type Applied struct {
	mu    gosync.RWMutex
	store map[string]map[string]sync.Result // scope -> key -> result
}

// NewApplied creates a new in-memory applied tracker.
func NewApplied() *Applied {
	return &Applied{
		store: make(map[string]map[string]sync.Result),
	}
}

// Get retrieves a stored result for a mutation key.
func (a *Applied) Get(ctx context.Context, scope, key string) (sync.Result, bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	scoped, ok := a.store[scope]
	if !ok {
		return sync.Result{}, false, nil
	}
	result, ok := scoped[key]
	return result, ok, nil
}

// Put stores a result for a mutation key.
func (a *Applied) Put(ctx context.Context, scope, key string, res sync.Result) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.store[scope] == nil {
		a.store[scope] = make(map[string]sync.Result)
	}
	a.store[scope][key] = res
	return nil
}
