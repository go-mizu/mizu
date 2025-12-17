package sync

// Persistence defines the interface for persisting client state.
type Persistence interface {
	// SaveQueue persists the mutation queue.
	SaveQueue(mutations []Mutation, seq uint64) error

	// LoadQueue loads the persisted mutation queue.
	LoadQueue() ([]Mutation, uint64, error)

	// SaveCursor persists the current cursor.
	SaveCursor(cursor uint64) error

	// LoadCursor loads the persisted cursor.
	LoadCursor() (uint64, error)

	// SaveStore persists the full store snapshot.
	SaveStore(data map[string]map[string][]byte) error

	// LoadStore loads the persisted store snapshot.
	LoadStore() (map[string]map[string][]byte, error)

	// SaveClientID persists the client identifier.
	SaveClientID(id string) error

	// LoadClientID loads the persisted client identifier.
	LoadClientID() (string, error)
}

// MemoryPersistence is an ephemeral in-memory persistence implementation.
type MemoryPersistence struct {
	mutations []Mutation
	seq       uint64
	cursor    uint64
	store     map[string]map[string][]byte
	clientID  string
}

// NewMemoryPersistence creates a new memory persistence.
func NewMemoryPersistence() *MemoryPersistence {
	return &MemoryPersistence{
		store: make(map[string]map[string][]byte),
	}
}

func (p *MemoryPersistence) SaveQueue(mutations []Mutation, seq uint64) error {
	p.mutations = make([]Mutation, len(mutations))
	copy(p.mutations, mutations)
	p.seq = seq
	return nil
}

func (p *MemoryPersistence) LoadQueue() ([]Mutation, uint64, error) {
	result := make([]Mutation, len(p.mutations))
	copy(result, p.mutations)
	return result, p.seq, nil
}

func (p *MemoryPersistence) SaveCursor(cursor uint64) error {
	p.cursor = cursor
	return nil
}

func (p *MemoryPersistence) LoadCursor() (uint64, error) {
	return p.cursor, nil
}

func (p *MemoryPersistence) SaveStore(data map[string]map[string][]byte) error {
	p.store = make(map[string]map[string][]byte, len(data))
	for entity, items := range data {
		p.store[entity] = make(map[string][]byte, len(items))
		for id, bytes := range items {
			cp := make([]byte, len(bytes))
			copy(cp, bytes)
			p.store[entity][id] = cp
		}
	}
	return nil
}

func (p *MemoryPersistence) LoadStore() (map[string]map[string][]byte, error) {
	result := make(map[string]map[string][]byte, len(p.store))
	for entity, items := range p.store {
		result[entity] = make(map[string][]byte, len(items))
		for id, bytes := range items {
			cp := make([]byte, len(bytes))
			copy(cp, bytes)
			result[entity][id] = cp
		}
	}
	return result, nil
}

func (p *MemoryPersistence) SaveClientID(id string) error {
	p.clientID = id
	return nil
}

func (p *MemoryPersistence) LoadClientID() (string, error) {
	return p.clientID, nil
}

// NopPersistence is a no-op persistence implementation.
type NopPersistence struct{}

func (NopPersistence) SaveQueue([]Mutation, uint64) error                    { return nil }
func (NopPersistence) LoadQueue() ([]Mutation, uint64, error)                { return nil, 0, nil }
func (NopPersistence) SaveCursor(uint64) error                               { return nil }
func (NopPersistence) LoadCursor() (uint64, error)                           { return 0, nil }
func (NopPersistence) SaveStore(map[string]map[string][]byte) error          { return nil }
func (NopPersistence) LoadStore() (map[string]map[string][]byte, error)      { return nil, nil }
func (NopPersistence) SaveClientID(string) error                             { return nil }
func (NopPersistence) LoadClientID() (string, error)                         { return "", nil }
