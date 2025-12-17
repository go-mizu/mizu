package sync

import (
	"context"
	"sync"
)

// Store provides scoped key-value storage for entity data.
type Store interface {
	// Get retrieves an entity by scope/entity/id.
	Get(ctx context.Context, scope, entity, id string) (any, error)

	// Set stores an entity.
	Set(ctx context.Context, scope, entity, id string, data any) error

	// Delete removes an entity.
	Delete(ctx context.Context, scope, entity, id string) error

	// List returns all entities of a type in a scope.
	List(ctx context.Context, scope, entity string) ([]any, error)

	// Snapshot returns all data in a scope (for full sync).
	// Returns map[entity]map[id]data
	Snapshot(ctx context.Context, scope string) (map[string]map[string]any, error)
}

// MemoryStore is an in-memory implementation of Store.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]map[string]map[string]any // scope -> entity -> id -> data
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]map[string]map[string]any),
	}
}

// Get retrieves an entity by scope/entity/id.
func (s *MemoryStore) Get(ctx context.Context, scope, entity, id string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entities, ok := s.data[scope]
	if !ok {
		return nil, ErrNotFound
	}

	items, ok := entities[entity]
	if !ok {
		return nil, ErrNotFound
	}

	data, ok := items[id]
	if !ok {
		return nil, ErrNotFound
	}

	return data, nil
}

// Set stores an entity.
func (s *MemoryStore) Set(ctx context.Context, scope, entity, id string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data[scope] == nil {
		s.data[scope] = make(map[string]map[string]any)
	}
	if s.data[scope][entity] == nil {
		s.data[scope][entity] = make(map[string]any)
	}

	s.data[scope][entity][id] = data
	return nil
}

// Delete removes an entity.
func (s *MemoryStore) Delete(ctx context.Context, scope, entity, id string) error {
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

// List returns all entities of a type in a scope.
func (s *MemoryStore) List(ctx context.Context, scope, entity string) ([]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entities, ok := s.data[scope]
	if !ok {
		return []any{}, nil
	}

	items, ok := entities[entity]
	if !ok {
		return []any{}, nil
	}

	result := make([]any, 0, len(items))
	for _, data := range items {
		result = append(result, data)
	}

	return result, nil
}

// Snapshot returns all data in a scope.
func (s *MemoryStore) Snapshot(ctx context.Context, scope string) (map[string]map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entities, ok := s.data[scope]
	if !ok {
		return make(map[string]map[string]any), nil
	}

	// Deep copy to avoid race conditions
	result := make(map[string]map[string]any)
	for entity, items := range entities {
		result[entity] = make(map[string]any)
		for id, data := range items {
			result[entity][id] = data
		}
	}

	return result, nil
}

// Clear removes all data from the store.
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]map[string]map[string]any)
}
