package sync

import "context"

// Store is the authoritative state store.
// All data is stored as JSON bytes to avoid type ambiguity.
type Store interface {
	// Get retrieves an entity by scope/entity/id.
	// Returns ErrNotFound if the entity does not exist.
	Get(ctx context.Context, scope, entity, id string) ([]byte, error)

	// Set stores an entity.
	Set(ctx context.Context, scope, entity, id string, data []byte) error

	// Delete removes an entity.
	// Returns nil if the entity does not exist.
	Delete(ctx context.Context, scope, entity, id string) error

	// Snapshot returns all data in a scope.
	// Returns map[entity]map[id]data.
	Snapshot(ctx context.Context, scope string) (map[string]map[string][]byte, error)
}
