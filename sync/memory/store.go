package memory

import (
	"context"
	"sync"

	gosync "github.com/go-mizu/mizu/sync"
)

// Store is an in-memory implementation of sync.Store.
type Store struct {
	mu   sync.RWMutex
	data map[string]map[string]map[string][]byte // scope -> entity -> id -> data
}

// NewStore creates a new in-memory store.
func NewStore() *Store {
	return &Store{
		data: make(map[string]map[string]map[string][]byte),
	}
}

// Get retrieves an entity by scope/entity/id.
func (s *Store) Get(ctx context.Context, scope, entity, id string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entities, ok := s.data[scope]
	if !ok {
		return nil, gosync.ErrNotFound
	}
	items, ok := entities[entity]
	if !ok {
		return nil, gosync.ErrNotFound
	}
	data, ok := items[id]
	if !ok {
		return nil, gosync.ErrNotFound
	}

	// Return a copy to avoid mutation
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, nil
}

// Set stores an entity.
func (s *Store) Set(ctx context.Context, scope, entity, id string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data[scope] == nil {
		s.data[scope] = make(map[string]map[string][]byte)
	}
	if s.data[scope][entity] == nil {
		s.data[scope][entity] = make(map[string][]byte)
	}

	// Store a copy
	cp := make([]byte, len(data))
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
func (s *Store) Snapshot(ctx context.Context, scope string) (map[string]map[string][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entities, ok := s.data[scope]
	if !ok {
		return make(map[string]map[string][]byte), nil
	}

	// Deep copy
	result := make(map[string]map[string][]byte, len(entities))
	for entity, items := range entities {
		result[entity] = make(map[string][]byte, len(items))
		for id, data := range items {
			cp := make([]byte, len(data))
			copy(cp, data)
			result[entity][id] = cp
		}
	}
	return result, nil
}

// Clear removes all data from the store.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]map[string]map[string][]byte)
}
