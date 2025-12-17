package live

import (
	"sync"
	"time"
)

// SessionStore persists session data.
type SessionStore interface {
	// Get retrieves a session by ID.
	Get(id string) (*sessionBase, bool)

	// Set stores a session.
	Set(id string, session *sessionBase)

	// Delete removes a session.
	Delete(id string)

	// Touch updates the session's last-seen time.
	Touch(id string)

	// Count returns the number of active sessions.
	Count() int

	// Cleanup removes expired sessions.
	Cleanup(maxAge time.Duration) int
}

// MemoryStore is the default in-memory session store.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*storeEntry
}

type storeEntry struct {
	session  *sessionBase
	lastSeen time.Time
}

// NewMemoryStore creates a new in-memory session store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*storeEntry),
	}
}

// Get retrieves a session by ID.
func (s *MemoryStore) Get(id string) (*sessionBase, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.sessions[id]
	if !ok {
		return nil, false
	}
	return entry.session, true
}

// Set stores a session.
func (s *MemoryStore) Set(id string, session *sessionBase) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = &storeEntry{
		session:  session,
		lastSeen: time.Now(),
	}
}

// Delete removes a session.
func (s *MemoryStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

// Touch updates the session's last-seen time.
func (s *MemoryStore) Touch(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry, ok := s.sessions[id]; ok {
		entry.lastSeen = time.Now()
	}
}

// Count returns the number of active sessions.
func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// Cleanup removes expired sessions.
func (s *MemoryStore) Cleanup(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	count := 0

	for id, entry := range s.sessions {
		if entry.lastSeen.Before(cutoff) {
			delete(s.sessions, id)
			count++
		}
	}

	return count
}

// All returns all sessions (for testing).
func (s *MemoryStore) All() []*sessionBase {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*sessionBase, 0, len(s.sessions))
	for _, entry := range s.sessions {
		result = append(result, entry.session)
	}
	return result
}
