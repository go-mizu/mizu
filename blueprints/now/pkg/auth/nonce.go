package auth

import (
	"errors"
	"sync"
	"time"
)

// NonceStore tracks nonces to prevent replay attacks.
type NonceStore interface {
	// Check returns an error if the nonce has been seen before.
	// Records the nonce if it is new.
	Check(nonce string, timestamp int64) error
}

// MemNonceStore is an in-memory NonceStore.
type MemNonceStore struct {
	mu     sync.Mutex
	seen   map[string]int64
	window time.Duration
}

// NewMemNonceStore returns a NonceStore with the given expiry window.
func NewMemNonceStore(window time.Duration) *MemNonceStore {
	return &MemNonceStore{
		seen:   make(map[string]int64),
		window: window,
	}
}

// Check rejects reused nonces and records new ones.
func (s *MemNonceStore) Check(nonce string, timestamp int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cleanup expired entries.
	cutoff := time.Now().Add(-s.window).UnixMilli()
	for k, ts := range s.seen {
		if ts < cutoff {
			delete(s.seen, k)
		}
	}

	if _, ok := s.seen[nonce]; ok {
		return errors.New("nonce reused")
	}

	s.seen[nonce] = timestamp
	return nil
}
