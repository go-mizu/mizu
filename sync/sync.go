package sync

import (
	"context"
	"time"
)

// Engine is the main sync coordinator.
type Engine struct {
	store     Store
	changelog ChangeLog
	mutator   Mutator
	broker    PokeBroker
}

// Options configures the sync engine.
type Options struct {
	// Store is the data store. Required.
	Store Store

	// ChangeLog is the change log. Required.
	ChangeLog ChangeLog

	// Mutator processes mutations. Required.
	Mutator Mutator

	// Broker notifies live connections. Optional.
	Broker PokeBroker
}

// New creates a new sync engine.
func New(opts Options) *Engine {
	broker := opts.Broker
	if broker == nil {
		broker = NopBroker{}
	}

	return &Engine{
		store:     opts.Store,
		changelog: opts.ChangeLog,
		mutator:   opts.Mutator,
		broker:    broker,
	}
}

// Push applies mutations and returns results.
func (e *Engine) Push(ctx context.Context, mutations []Mutation) ([]MutationResult, error) {
	results := make([]MutationResult, len(mutations))
	affectedScopes := make(map[string]uint64) // scope -> max cursor

	for i, mut := range mutations {
		result := MutationResult{}

		// Apply mutation
		changes, err := e.mutator.Apply(ctx, e.store, mut)
		if err != nil {
			result.OK = false
			result.Error = err.Error()
			results[i] = result
			continue
		}

		// Append changes to log
		var maxCursor uint64
		for _, change := range changes {
			change.Timestamp = time.Now()
			if change.Scope == "" {
				change.Scope = mut.Scope
			}

			cursor, err := e.changelog.Append(ctx, change)
			if err != nil {
				result.OK = false
				result.Error = err.Error()
				break
			}

			change.Cursor = cursor
			if cursor > maxCursor {
				maxCursor = cursor
			}

			// Track affected scope
			if change.Scope != "" {
				if cursor > affectedScopes[change.Scope] {
					affectedScopes[change.Scope] = cursor
				}
			}
		}

		if result.Error == "" {
			result.OK = true
			result.Cursor = maxCursor
			result.Changes = changes
		}
		results[i] = result
	}

	// Poke affected scopes
	for scope, cursor := range affectedScopes {
		e.broker.Poke(scope, cursor)
	}

	return results, nil
}

// Pull returns changes since a cursor.
func (e *Engine) Pull(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, uint64, bool, error) {
	if limit <= 0 {
		limit = 100
	}

	changes, err := e.changelog.Since(ctx, scope, cursor, limit+1)
	if err != nil {
		return nil, 0, false, err
	}

	hasMore := len(changes) > limit
	if hasMore {
		changes = changes[:limit]
	}

	var newCursor uint64
	if len(changes) > 0 {
		newCursor = changes[len(changes)-1].Cursor
	} else {
		newCursor = cursor
	}

	return changes, newCursor, hasMore, nil
}

// Snapshot returns all data in a scope.
func (e *Engine) Snapshot(ctx context.Context, scope string) (map[string]map[string]any, uint64, error) {
	cursor, err := e.changelog.Cursor(ctx)
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
func (e *Engine) Store() Store {
	return e.store
}

// ChangeLog returns the underlying change log.
func (e *Engine) ChangeLog() ChangeLog {
	return e.changelog
}

// Cursor returns the current change log cursor.
func (e *Engine) Cursor(ctx context.Context) (uint64, error) {
	return e.changelog.Cursor(ctx)
}
