package memory

import (
	"context"
	"sync"

	gosync "github.com/go-mizu/mizu/sync"
)

// Applied is an in-memory implementation of sync.Applied.
type Applied struct {
	mu    sync.RWMutex
	store map[string]map[string]gosync.Result // scope -> key -> result
}

// NewApplied creates a new in-memory applied tracker.
func NewApplied() *Applied {
	return &Applied{
		store: make(map[string]map[string]gosync.Result),
	}
}

// Get retrieves a stored result for a mutation key.
func (a *Applied) Get(ctx context.Context, scope, key string) (gosync.Result, bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	scoped, ok := a.store[scope]
	if !ok {
		return gosync.Result{}, false, nil
	}
	result, ok := scoped[key]
	return result, ok, nil
}

// Put stores a result for a mutation key.
func (a *Applied) Put(ctx context.Context, scope, key string, res gosync.Result) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.store[scope] == nil {
		a.store[scope] = make(map[string]gosync.Result)
	}
	a.store[scope][key] = res
	return nil
}

// Clear removes all stored results.
func (a *Applied) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.store = make(map[string]map[string]gosync.Result)
}

// Len returns the number of stored results for a scope.
func (a *Applied) Len(scope string) int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.store[scope])
}
