package sync

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Mutation represents a client request to change state.
type Mutation struct {
	// ID uniquely identifies this mutation for idempotency.
	ID string `json:"id"`

	// Name identifies the mutation type.
	Name string `json:"name"`

	// Scope identifies the data partition.
	Scope string `json:"scope,omitempty"`

	// Client identifies the originating client.
	Client string `json:"client,omitempty"`

	// Seq is a client-local sequence number.
	Seq uint64 `json:"seq,omitempty"`

	// Args contains mutation-specific arguments.
	Args map[string]any `json:"args,omitempty"`

	// CreatedAt is when the mutation was created.
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// Queue manages pending mutations.
type Queue struct {
	mu        sync.RWMutex
	mutations []Mutation
	byID      map[string]int // id -> index for fast lookup
	seq       uint64         // sequence counter
	clientID  string         // client identifier
}

// NewQueue creates a new empty mutation queue.
func NewQueue() *Queue {
	return &Queue{
		byID:     make(map[string]int),
		clientID: generateClientID(),
	}
}

// Push adds a mutation to the queue and returns its ID.
func (q *Queue) Push(m Mutation) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Generate ID if not provided
	if m.ID == "" {
		m.ID = generateMutationID()
	}

	// Check for duplicate
	if _, exists := q.byID[m.ID]; exists {
		return m.ID
	}

	// Set sequence and client
	q.seq++
	m.Seq = q.seq
	m.Client = q.clientID

	// Set timestamp
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}

	// Add to queue
	q.byID[m.ID] = len(q.mutations)
	q.mutations = append(q.mutations, m)

	return m.ID
}

// Pending returns a copy of all pending mutations.
func (q *Queue) Pending() []Mutation {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]Mutation, len(q.mutations))
	copy(result, q.mutations)
	return result
}

// Len returns the number of pending mutations.
func (q *Queue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.mutations)
}

// Clear removes all pending mutations.
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.mutations = nil
	q.byID = make(map[string]int)
}

// Remove removes a specific mutation by ID.
func (q *Queue) Remove(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	idx, ok := q.byID[id]
	if !ok {
		return
	}

	// Remove from slice
	q.mutations = append(q.mutations[:idx], q.mutations[idx+1:]...)

	// Rebuild index map
	delete(q.byID, id)
	for i := idx; i < len(q.mutations); i++ {
		q.byID[q.mutations[i].ID] = i
	}
}

// RemoveUpTo removes all mutations up to and including the given sequence.
func (q *Queue) RemoveUpTo(seq uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Find cutoff point
	cutoff := 0
	for i, m := range q.mutations {
		if m.Seq <= seq {
			cutoff = i + 1
		}
	}

	if cutoff == 0 {
		return
	}

	// Remove IDs from map
	for i := 0; i < cutoff; i++ {
		delete(q.byID, q.mutations[i].ID)
	}

	// Truncate slice
	q.mutations = q.mutations[cutoff:]

	// Rebuild index map
	for i, m := range q.mutations {
		q.byID[m.ID] = i
	}
}

// Get returns a mutation by ID.
func (q *Queue) Get(id string) (Mutation, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	idx, ok := q.byID[id]
	if !ok {
		return Mutation{}, false
	}
	return q.mutations[idx], true
}

// Has checks if a mutation exists in the queue.
func (q *Queue) Has(id string) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	_, ok := q.byID[id]
	return ok
}

// ClientID returns the client identifier.
func (q *Queue) ClientID() string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.clientID
}

// SetClientID sets the client identifier.
func (q *Queue) SetClientID(id string) {
	q.mu.Lock()
	q.clientID = id
	q.mu.Unlock()
}

// CurrentSeq returns the current sequence number.
func (q *Queue) CurrentSeq() uint64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.seq
}

// SetSeq sets the sequence number (for persistence restoration).
func (q *Queue) SetSeq(seq uint64) {
	q.mu.Lock()
	q.seq = seq
	q.mu.Unlock()
}

// Load replaces all mutations (for persistence restoration).
func (q *Queue) Load(mutations []Mutation) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.mutations = make([]Mutation, len(mutations))
	copy(q.mutations, mutations)
	q.byID = make(map[string]int, len(mutations))

	for i, m := range q.mutations {
		q.byID[m.ID] = i
		if m.Seq > q.seq {
			q.seq = m.Seq
		}
	}
}

// generateMutationID generates a unique mutation ID.
func generateMutationID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// generateClientID generates a unique client ID.
func generateClientID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
