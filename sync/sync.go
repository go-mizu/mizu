// Package sync provides authoritative, offline-first state synchronization.
//
// It defines a durable mutation pipeline, an ordered change log, and
// cursor-based replication so clients can converge to correct state
// across retries, disconnects, offline operation, and server restarts.
//
// # Design principles
//
//   - Authoritative: All durable state changes are applied on the server
//   - Offline-first: Clients may enqueue and replay mutations safely
//   - Idempotent: Replayed mutations must not apply twice
//   - Pull-based: Clients converge by pulling changes since a cursor
//   - Scoped: All data and cursors are partitioned by scope
//
// # Core primitives
//
//  1. Mutation: client intent with stable ID for dedupe
//  2. ApplyFunc: application logic that mutates state and emits changes
//  3. Log: ordered changes served by cursor
//  4. SnapshotFunc: bootstrap when cursor is too old
//
// # Basic usage
//
//	log := memory.NewLog()
//	dedupe := memory.NewDedupe()
//
//	engine := sync.New(sync.Options{
//	    Log:   log,
//	    Dedupe: dedupe,
//	    Apply: func(ctx context.Context, m sync.Mutation) ([]sync.Change, error) {
//	        // Application logic here
//	        return changes, nil
//	    },
//	})
//
// # Mutation flow
//
//  1. Client sends mutation via Push
//  2. Engine checks idempotency (Dedupe.Seen)
//  3. ApplyFunc processes mutation and returns changes
//  4. Changes are recorded in Log
//  5. Dedupe.Mark is called for idempotency
//
// # Client synchronization
//
// Clients maintain a cursor and call Pull to receive changes since that cursor.
// For initial sync or recovery, clients can call Snapshot to get full state.
package sync

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// -----------------------------------------------------------------------------
// Errors
// -----------------------------------------------------------------------------

var (
	// ErrNotFound is returned when an entity is not found.
	ErrNotFound = errors.New("sync: not found")

	// ErrInvalidMutation is returned when a mutation is malformed.
	ErrInvalidMutation = errors.New("sync: invalid mutation")

	// ErrConflict is returned when there is a conflict during mutation.
	ErrConflict = errors.New("sync: conflict")

	// ErrCursorTooOld is returned when a cursor has been trimmed from the log.
	// Log.Since implementations must return this error when the requested
	// cursor is below the log's retention window.
	ErrCursorTooOld = errors.New("sync: cursor too old")
)

// DefaultScope is used when no scope is specified.
const DefaultScope = "_default"

// -----------------------------------------------------------------------------
// Types
// -----------------------------------------------------------------------------

// Mutation represents a client request to change state.
// It is a command, not a state patch.
type Mutation struct {
	// ID uniquely identifies this mutation for idempotency.
	ID string `json:"id"`

	// Scope identifies the data partition.
	Scope string `json:"scope,omitempty"`

	// Name identifies the mutation type.
	Name string `json:"name"`

	// Args contains mutation-specific arguments as opaque JSON.
	Args json.RawMessage `json:"args,omitempty"`
}

// Result describes the outcome of applying a mutation.
type Result struct {
	OK      bool     `json:"ok"`
	Cursor  uint64   `json:"cursor,omitempty"`
	Error   string   `json:"error,omitempty"`
	Changes []Change `json:"changes,omitempty"`
}

// Change is a single durable state change recorded in the log.
// The Data field is opaque - applications define their own payload schema.
type Change struct {
	Cursor uint64          `json:"cursor"`
	Scope  string          `json:"scope"`
	Time   time.Time       `json:"time"`
	Data   json.RawMessage `json:"data,omitempty"`
}

// -----------------------------------------------------------------------------
// Interfaces
// -----------------------------------------------------------------------------

// Log records ordered changes and serves them by cursor.
type Log interface {
	// Append adds changes to the log and returns the final cursor.
	// Implementations MUST assign sequential Change.Cursor values in-place
	// starting after the current cursor. The returned cursor is the last assigned.
	// This allows callers to see the assigned cursors in the input slice.
	Append(ctx context.Context, scope string, changes []Change) (uint64, error)

	// Since returns changes after the given cursor for a scope.
	// Returns up to limit changes.
	// Returns ErrCursorTooOld if the cursor has been trimmed from the log.
	Since(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, error)

	// Cursor returns the current latest cursor for a scope.
	Cursor(ctx context.Context, scope string) (uint64, error)

	// Trim removes changes before the given cursor (for compaction).
	Trim(ctx context.Context, scope string, before uint64) error
}

// ApplyFunc processes a mutation and returns the resulting changes.
// It closes over whatever storage the application uses.
type ApplyFunc func(ctx context.Context, m Mutation) ([]Change, error)

// SnapshotFunc returns a snapshot of all data in a scope with the current cursor.
// Used for initial sync or recovery when cursor is too old.
type SnapshotFunc func(ctx context.Context, scope string) (json.RawMessage, uint64, error)

// Dedupe tracks mutations already processed for idempotency.
// This enables "at-most-once" semantics - replayed mutations become no-ops.
type Dedupe interface {
	// Seen returns true if the mutation has already been processed.
	Seen(ctx context.Context, scope, id string) (bool, error)

	// Mark records that a mutation has been processed.
	Mark(ctx context.Context, scope, id string) error
}

// -----------------------------------------------------------------------------
// Engine
// -----------------------------------------------------------------------------

// Engine coordinates mutation application and replication.
type Engine struct {
	log      Log
	apply    ApplyFunc
	snapshot SnapshotFunc
	dedupe   Dedupe
	now      func() time.Time
}

// Options configures the sync engine.
type Options struct {
	Log      Log         // Required: change log
	Apply    ApplyFunc   // Required: mutation processor
	Snapshot SnapshotFunc // Optional: snapshot provider
	Dedupe   Dedupe      // Optional: idempotency tracker (nil disables deduplication)

	// Now returns the current time. Defaults to time.Now if nil.
	// Useful for testing and deterministic timestamps.
	Now func() time.Time
}

// New creates a new sync engine.
func New(opts Options) *Engine {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Engine{
		log:      opts.Log,
		apply:    opts.Apply,
		snapshot: opts.Snapshot,
		dedupe:   opts.Dedupe,
		now:      now,
	}
}

// Push applies mutations and returns results.
// For each mutation:
//  1. Compute dedupe key (mutation.ID)
//  2. If already applied (Dedupe.Seen), return success without re-executing
//  3. Call ApplyFunc
//  4. Append changes to Log (with timestamps)
//  5. Call Dedupe.Mark
func (e *Engine) Push(ctx context.Context, mutations []Mutation) ([]Result, error) {
	results := make([]Result, len(mutations))

	for i, mut := range mutations {
		results[i] = e.processMutation(ctx, mut)
	}

	return results, nil
}

func (e *Engine) processMutation(ctx context.Context, mut Mutation) Result {
	scope := mut.Scope
	if scope == "" {
		scope = DefaultScope
	}

	// Check idempotency
	if e.dedupe != nil && mut.ID != "" {
		seen, err := e.dedupe.Seen(ctx, scope, mut.ID)
		if err != nil {
			return Result{OK: false, Error: err.Error()}
		}
		if seen {
			// Return success for duplicate - mutation was already processed
			cursor, _ := e.log.Cursor(ctx, scope)
			return Result{OK: true, Cursor: cursor}
		}
	}

	// Apply mutation
	changes, err := e.apply(ctx, mut)
	if err != nil {
		return e.errorResult(err)
	}

	// Set timestamps and scope on changes
	now := e.now()
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
			return Result{OK: false, Error: err.Error()}
		}
	} else {
		cursor, _ = e.log.Cursor(ctx, scope)
	}

	result := Result{
		OK:      true,
		Cursor:  cursor,
		Changes: changes,
	}

	// Mark for idempotency
	if e.dedupe != nil && mut.ID != "" {
		if err := e.dedupe.Mark(ctx, scope, mut.ID); err != nil {
			return Result{OK: false, Error: "failed to mark idempotency key"}
		}
	}

	return result
}

func (e *Engine) errorResult(err error) Result {
	return Result{OK: false, Error: err.Error()}
}

// Pull returns changes since a cursor.
// Returns (changes, hasMore, error) where hasMore indicates more data exists.
func (e *Engine) Pull(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, bool, error) {
	if scope == "" {
		scope = DefaultScope
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
// Returns (data, cursor, error).
func (e *Engine) Snapshot(ctx context.Context, scope string) (json.RawMessage, uint64, error) {
	if scope == "" {
		scope = DefaultScope
	}

	if e.snapshot == nil {
		// Return empty snapshot if no snapshot function configured
		cursor, err := e.log.Cursor(ctx, scope)
		if err != nil {
			return nil, 0, err
		}
		return json.RawMessage("{}"), cursor, nil
	}

	return e.snapshot(ctx, scope)
}

// Log returns the underlying log for direct access if needed.
func (e *Engine) Log() Log {
	return e.log
}
