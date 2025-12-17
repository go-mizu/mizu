package sync

import "context"

// Mutator contains application business logic.
// It processes mutations and returns the resulting changes.
type Mutator interface {
	// Apply processes a mutation and returns the resulting changes.
	// The mutator should:
	//   1. Validate the mutation
	//   2. Apply changes to the store
	//   3. Return the list of changes for the log
	Apply(ctx context.Context, store Store, m Mutation) ([]Change, error)
}

// MutatorFunc is a function that implements Mutator.
type MutatorFunc func(context.Context, Store, Mutation) ([]Change, error)

// Apply implements Mutator.
func (f MutatorFunc) Apply(ctx context.Context, s Store, m Mutation) ([]Change, error) {
	return f(ctx, s, m)
}

// MutatorMap dispatches to registered handlers by mutation name.
type MutatorMap struct {
	handlers map[string]MutatorFunc
}

// NewMutatorMap creates a new MutatorMap.
func NewMutatorMap() *MutatorMap {
	return &MutatorMap{handlers: make(map[string]MutatorFunc)}
}

// Register adds a handler for a mutation name.
func (m *MutatorMap) Register(name string, handler MutatorFunc) {
	m.handlers[name] = handler
}

// Apply dispatches to the registered handler.
func (m *MutatorMap) Apply(ctx context.Context, store Store, mut Mutation) ([]Change, error) {
	handler, ok := m.handlers[mut.Name]
	if !ok {
		return nil, ErrUnknownMutation
	}
	return handler(ctx, store, mut)
}
