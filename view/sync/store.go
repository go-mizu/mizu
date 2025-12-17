package sync

import (
	"sync"
)

// Store is the local state container for synchronized data.
type Store struct {
	mu       sync.RWMutex
	data     map[string]map[string][]byte // entity -> id -> data
	version  *Signal[uint64]              // Global version for reactivity
	onChange func(entity, id string, op Op)
}

// Op defines the type of change operation.
type Op string

const (
	OpCreate Op = "create"
	OpUpdate Op = "update"
	OpDelete Op = "delete"
)

// NewStore creates a new empty store.
func NewStore() *Store {
	return &Store{
		data:    make(map[string]map[string][]byte),
		version: NewSignal[uint64](0),
	}
}

// Get retrieves an entity by entity type and id.
func (s *Store) Get(entity, id string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.data[entity]
	if !ok {
		return nil, false
	}
	data, ok := items[id]
	if !ok {
		return nil, false
	}

	// Return a copy to avoid mutation
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, true
}

// Set stores an entity.
func (s *Store) Set(entity, id string, data []byte) {
	s.mu.Lock()

	if s.data[entity] == nil {
		s.data[entity] = make(map[string][]byte)
	}

	_, exists := s.data[entity][id]
	op := OpCreate
	if exists {
		op = OpUpdate
	}

	// Store a copy
	cp := make([]byte, len(data))
	copy(cp, data)
	s.data[entity][id] = cp

	s.mu.Unlock()

	// Update version to trigger reactivity
	s.version.Update(func(v uint64) uint64 { return v + 1 })

	if s.onChange != nil {
		s.onChange(entity, id, op)
	}
}

// Delete removes an entity.
func (s *Store) Delete(entity, id string) {
	s.mu.Lock()

	deleted := false
	if items, ok := s.data[entity]; ok {
		if _, exists := items[id]; exists {
			delete(items, id)
			deleted = true
			if len(items) == 0 {
				delete(s.data, entity)
			}
		}
	}

	s.mu.Unlock()

	if deleted {
		s.version.Update(func(v uint64) uint64 { return v + 1 })
		if s.onChange != nil {
			s.onChange(entity, id, OpDelete)
		}
	}
}

// Has checks if an entity exists.
func (s *Store) Has(entity, id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.data[entity]
	if !ok {
		return false
	}
	_, ok = items[id]
	return ok
}

// List returns all IDs for an entity type.
func (s *Store) List(entity string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.data[entity]
	if !ok {
		return nil
	}

	ids := make([]string, 0, len(items))
	for id := range items {
		ids = append(ids, id)
	}
	return ids
}

// All returns all data for an entity type.
func (s *Store) All(entity string) map[string][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.data[entity]
	if !ok {
		return nil
	}

	// Deep copy
	result := make(map[string][]byte, len(items))
	for id, data := range items {
		cp := make([]byte, len(data))
		copy(cp, data)
		result[id] = cp
	}
	return result
}

// Snapshot returns all data in the store.
func (s *Store) Snapshot() map[string]map[string][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Deep copy
	result := make(map[string]map[string][]byte, len(s.data))
	for entity, items := range s.data {
		result[entity] = make(map[string][]byte, len(items))
		for id, data := range items {
			cp := make([]byte, len(data))
			copy(cp, data)
			result[entity][id] = cp
		}
	}
	return result
}

// Load replaces all data in the store.
func (s *Store) Load(data map[string]map[string][]byte) {
	s.mu.Lock()

	// Clear existing data
	s.data = make(map[string]map[string][]byte, len(data))

	// Deep copy incoming data
	for entity, items := range data {
		s.data[entity] = make(map[string][]byte, len(items))
		for id, bytes := range items {
			cp := make([]byte, len(bytes))
			copy(cp, bytes)
			s.data[entity][id] = cp
		}
	}

	s.mu.Unlock()

	// Trigger reactivity
	s.version.Update(func(v uint64) uint64 { return v + 1 })
}

// Clear removes all data from the store.
func (s *Store) Clear() {
	s.mu.Lock()
	s.data = make(map[string]map[string][]byte)
	s.mu.Unlock()

	s.version.Update(func(v uint64) uint64 { return v + 1 })
}

// Version returns the reactive version signal.
func (s *Store) Version() *Signal[uint64] {
	return s.version
}

// SetOnChange sets the change callback.
func (s *Store) SetOnChange(fn func(entity, id string, op Op)) {
	s.mu.Lock()
	s.onChange = fn
	s.mu.Unlock()
}

// Count returns the number of entities of a given type.
func (s *Store) Count(entity string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data[entity])
}

// EntityTypes returns all entity type names in the store.
func (s *Store) EntityTypes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	types := make([]string, 0, len(s.data))
	for entity := range s.data {
		types = append(types, entity)
	}
	return types
}
