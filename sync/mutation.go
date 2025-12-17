package sync

import (
	"context"
	"time"
)

// Mutation represents a client-originated state change request.
type Mutation struct {
	// Name identifies the mutation type (e.g., "todo/toggle", "user/update").
	Name string `json:"name"`

	// Args contains mutation-specific arguments.
	Args map[string]any `json:"args,omitempty"`

	// ClientID is the originating client identifier (for deduplication).
	ClientID string `json:"client_id,omitempty"`

	// ClientSeq is the client's sequence number (for ordering).
	ClientSeq uint64 `json:"client_seq,omitempty"`

	// Scope identifies the data partition this mutation affects.
	Scope string `json:"scope,omitempty"`
}

// MutationResult is returned after applying a mutation.
type MutationResult struct {
	// OK indicates success.
	OK bool `json:"ok"`

	// Cursor is the new change log cursor after this mutation.
	Cursor uint64 `json:"cursor"`

	// Error contains any error message.
	Error string `json:"error,omitempty"`

	// Changes lists the entities affected by this mutation.
	Changes []Change `json:"changes,omitempty"`
}

// Change represents a single entity change in the log.
type Change struct {
	// Cursor is the unique, monotonically increasing position.
	Cursor uint64 `json:"cursor"`

	// Scope identifies the data partition.
	Scope string `json:"scope"`

	// Entity is the entity type (e.g., "todo", "user").
	Entity string `json:"entity"`

	// ID is the entity identifier.
	ID string `json:"id"`

	// Op is the operation type.
	Op ChangeOp `json:"op"`

	// Data contains the entity data (for create/update).
	Data any `json:"data,omitempty"`

	// Timestamp is when the change occurred.
	Timestamp time.Time `json:"ts"`
}

// ChangeOp defines the type of change.
type ChangeOp string

const (
	OpCreate ChangeOp = "create"
	OpUpdate ChangeOp = "update"
	OpDelete ChangeOp = "delete"
)

// Mutator applies mutations to the store and returns changes.
type Mutator interface {
	// Apply processes a mutation and returns the resulting changes.
	// The mutator should:
	// 1. Validate the mutation
	// 2. Apply changes to the store
	// 3. Return the list of changes for the log
	Apply(ctx context.Context, store Store, m Mutation) ([]Change, error)
}

// MutatorFunc is a function that implements Mutator.
type MutatorFunc func(ctx context.Context, store Store, m Mutation) ([]Change, error)

// Apply implements Mutator.
func (f MutatorFunc) Apply(ctx context.Context, store Store, m Mutation) ([]Change, error) {
	return f(ctx, store, m)
}

// MutatorMap is a Mutator that dispatches to registered handlers by mutation name.
type MutatorMap struct {
	handlers map[string]MutatorFunc
}

// NewMutatorMap creates a new MutatorMap.
func NewMutatorMap() *MutatorMap {
	return &MutatorMap{
		handlers: make(map[string]MutatorFunc),
	}
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
