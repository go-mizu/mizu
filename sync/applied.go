package sync

import "context"

// Applied tracks mutations already processed.
// This enables strict idempotency - replayed mutations
// return their original result without re-execution.
type Applied interface {
	// Get retrieves a stored result for a mutation key.
	// Returns (result, true, nil) if found.
	// Returns (Result{}, false, nil) if not found.
	Get(ctx context.Context, scope, key string) (Result, bool, error)

	// Put stores a result for a mutation key.
	Put(ctx context.Context, scope, key string, res Result) error
}
