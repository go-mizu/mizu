package sync

import (
	"context"
	"errors"
	"time"
)

// Engine coordinates mutation application and replication.
type Engine struct {
	store   Store
	log     Log
	applied Applied
	mutator Mutator
	notify  Notifier
}

// Options configures the sync engine.
type Options struct {
	Store   Store    // Required: authoritative state store
	Log     Log      // Required: change log
	Applied Applied  // Optional: idempotency tracker (nil disables deduplication)
	Mutator Mutator  // Required: mutation processor
	Notify  Notifier // Optional: notification hook
}

// New creates a new sync engine.
func New(opts Options) *Engine {
	return &Engine{
		store:   opts.Store,
		log:     opts.Log,
		applied: opts.Applied,
		mutator: opts.Mutator,
		notify:  opts.Notify,
	}
}

// Push applies mutations and returns results.
// For each mutation:
//  1. Compute dedupe key (mutation.ID)
//  2. If already applied, return stored result
//  3. Call Mutator.Apply
//  4. Append changes to Log (with timestamps)
//  5. Store Result in Applied
//  6. Call Notify with latest cursor
func (e *Engine) Push(ctx context.Context, mutations []Mutation) ([]Result, error) {
	results := make([]Result, len(mutations))
	affectedScopes := make(map[string]uint64) // scope -> max cursor

	for i, mut := range mutations {
		result := e.processMutation(ctx, mut)
		results[i] = result

		if result.OK && result.Cursor > 0 {
			scope := mut.Scope
			if scope == "" {
				scope = "_default"
			}
			if result.Cursor > affectedScopes[scope] {
				affectedScopes[scope] = result.Cursor
			}
		}
	}

	// Notify affected scopes
	if e.notify != nil {
		for scope, cursor := range affectedScopes {
			e.notify.Notify(scope, cursor)
		}
	}

	return results, nil
}

func (e *Engine) processMutation(ctx context.Context, mut Mutation) Result {
	scope := mut.Scope
	if scope == "" {
		scope = "_default"
	}

	// Check idempotency
	if e.applied != nil && mut.ID != "" {
		if res, found, err := e.applied.Get(ctx, scope, mut.ID); err != nil {
			return Result{OK: false, Code: CodeInternal, Error: err.Error()}
		} else if found {
			return res
		}
	}

	// Apply mutation
	changes, err := e.mutator.Apply(ctx, e.store, mut)
	if err != nil {
		return e.errorResult(err)
	}

	// Set timestamps and scope on changes
	now := time.Now()
	for i := range changes {
		if changes[i].Time.IsZero() {
			changes[i].Time = now
		}
		if changes[i].Scope == "" {
			changes[i].Scope = scope
		}
	}

	// Append to log
	var cursor uint64
	if len(changes) > 0 {
		var err error
		cursor, err = e.log.Append(ctx, scope, changes)
		if err != nil {
			return Result{OK: false, Code: CodeInternal, Error: err.Error()}
		}
	} else {
		cursor, _ = e.log.Cursor(ctx, scope)
	}

	result := Result{
		OK:      true,
		Cursor:  cursor,
		Changes: changes,
	}

	// Store for idempotency
	if e.applied != nil && mut.ID != "" {
		_ = e.applied.Put(ctx, scope, mut.ID, result)
	}

	return result
}

func (e *Engine) errorResult(err error) Result {
	code := CodeInternal
	switch {
	case errors.Is(err, ErrNotFound):
		code = CodeNotFound
	case errors.Is(err, ErrUnknownMutation):
		code = CodeUnknown
	case errors.Is(err, ErrInvalidMutation):
		code = CodeInvalid
	case errors.Is(err, ErrConflict):
		code = CodeConflict
	}
	return Result{OK: false, Code: code, Error: err.Error()}
}

// Pull returns changes since a cursor.
// Returns (changes, hasMore, error) where hasMore indicates more data exists.
func (e *Engine) Pull(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, bool, error) {
	if scope == "" {
		scope = "_default"
	}
	if limit <= 0 {
		limit = 100
	}

	// Request one extra to detect hasMore
	changes, err := e.log.Since(ctx, scope, cursor, limit+1)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(changes) > limit
	if hasMore {
		changes = changes[:limit]
	}

	return changes, hasMore, nil
}

// Snapshot returns all data in a scope with the current cursor.
func (e *Engine) Snapshot(ctx context.Context, scope string) (map[string]map[string][]byte, uint64, error) {
	if scope == "" {
		scope = "_default"
	}

	cursor, err := e.log.Cursor(ctx, scope)
	if err != nil {
		return nil, 0, err
	}

	data, err := e.store.Snapshot(ctx, scope)
	if err != nil {
		return nil, 0, err
	}

	return data, cursor, nil
}

// Store returns the underlying store.
func (e *Engine) Store() Store { return e.store }

// Log returns the underlying log.
func (e *Engine) Log() Log { return e.log }
