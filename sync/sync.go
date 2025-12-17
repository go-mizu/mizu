// Package sync provides authoritative, offline-first state synchronization.
//
// It defines a durable mutation pipeline, an ordered change log, and
// cursor-based replication so clients can converge to correct state
// across retries, disconnects, offline operation, and server restarts.
//
// The package is transport-agnostic. HTTP is the default transport,
// but correctness does not depend on realtime delivery. Realtime systems
// such as live may accelerate convergence but are optional.
//
// # Design principles
//
//   - Authoritative: All durable state changes are applied on the server
//   - Offline-first: Clients may enqueue and replay mutations safely
//   - Idempotent: Replayed mutations must not apply twice
//   - Pull-based: Clients converge by pulling changes since a cursor
//   - Scoped: All data and cursors are partitioned by scope
//
// # Basic usage
//
//	store := memory.NewStore()
//	log := memory.NewLog()
//	applied := memory.NewApplied()
//
//	engine := sync.New(sync.Options{
//	    Store:   store,
//	    Log:     log,
//	    Applied: applied,
//	    Mutator: myMutator,
//	})
//
//	// Mount HTTP handlers
//	engine.Mount(app)
//
// # Mutation flow
//
//  1. Client sends mutation via Push
//  2. Engine checks idempotency (Applied)
//  3. Mutator applies business logic to Store
//  4. Changes are recorded in Log
//  5. Result is stored in Applied
//  6. Notifier is called (if configured)
//
// # Client synchronization
//
// Clients maintain a cursor and call Pull to receive changes since that cursor.
// For initial sync or recovery, clients can call Snapshot to get full state.
package sync

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
)

// -----------------------------------------------------------------------------
// Errors
// -----------------------------------------------------------------------------

var (
	// ErrNotFound is returned when an entity is not found.
	ErrNotFound = errors.New("sync: not found")

	// ErrUnknownMutation is returned when a mutation name has no handler.
	ErrUnknownMutation = errors.New("sync: unknown mutation")

	// ErrInvalidMutation is returned when a mutation is malformed.
	ErrInvalidMutation = errors.New("sync: invalid mutation")

	// ErrConflict is returned when there is a conflict during mutation.
	ErrConflict = errors.New("sync: conflict")
)

// Error codes for internal use in HTTP responses.
const (
	codeOK           = ""
	codeNotFound     = "not_found"
	codeUnknown      = "unknown_mutation"
	codeInvalid      = "invalid_mutation"
	codeCursorTooOld = "cursor_too_old"
	codeConflict     = "conflict"
	codeInternal     = "internal_error"
)

// errCursorTooOld is returned when a cursor has been trimmed from the log.
var errCursorTooOld = errors.New("sync: cursor too old")

// -----------------------------------------------------------------------------
// Types
// -----------------------------------------------------------------------------

// Mutation represents a client request to change state.
// It is a command, not a state patch.
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
}

// Result describes the outcome of applying a mutation.
type Result struct {
	OK      bool     `json:"ok"`
	Cursor  uint64   `json:"cursor,omitempty"`
	Code    string   `json:"code,omitempty"`
	Error   string   `json:"error,omitempty"`
	Changes []Change `json:"changes,omitempty"`
}

// Change is a single durable state change recorded in the log.
type Change struct {
	Cursor uint64    `json:"cursor"`
	Scope  string    `json:"scope"`
	Entity string    `json:"entity"`
	ID     string    `json:"id"`
	Op     Op        `json:"op"`
	Data   []byte    `json:"data,omitempty"`
	Time   time.Time `json:"time"`
}

// Op defines the type of change operation.
type Op string

const (
	Create Op = "create"
	Update Op = "update"
	Delete Op = "delete"
)

// -----------------------------------------------------------------------------
// Interfaces
// -----------------------------------------------------------------------------

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

// Log records ordered changes and serves them by cursor.
type Log interface {
	// Append adds changes to the log and returns the final cursor.
	// Changes are assigned sequential cursor values starting after
	// the current cursor. The returned cursor is the last assigned.
	Append(ctx context.Context, scope string, changes []Change) (uint64, error)

	// Since returns changes after the given cursor for a scope.
	// Returns up to limit changes.
	Since(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, error)

	// Cursor returns the current latest cursor for a scope.
	Cursor(ctx context.Context, scope string) (uint64, error)

	// Trim removes changes before the given cursor (for compaction).
	Trim(ctx context.Context, scope string, before uint64) error
}

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

// Notifier is an optional integration hook.
// It is called after changes are committed to notify
// external systems (e.g., live/websocket connections).
type Notifier interface {
	Notify(scope string, cursor uint64)
}

// NotifierFunc wraps a function as a Notifier.
type NotifierFunc func(scope string, cursor uint64)

// Notify implements Notifier.
func (f NotifierFunc) Notify(scope string, cursor uint64) {
	f(scope, cursor)
}

// -----------------------------------------------------------------------------
// Engine
// -----------------------------------------------------------------------------

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
			return Result{OK: false, Code: codeInternal, Error: err.Error()}
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
			return Result{OK: false, Code: codeInternal, Error: err.Error()}
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
	code := codeInternal
	switch {
	case errors.Is(err, ErrNotFound):
		code = codeNotFound
	case errors.Is(err, ErrUnknownMutation):
		code = codeUnknown
	case errors.Is(err, ErrInvalidMutation):
		code = codeInvalid
	case errors.Is(err, ErrConflict):
		code = codeConflict
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

// -----------------------------------------------------------------------------
// HTTP Transport
// -----------------------------------------------------------------------------

// HTTP request/response types (internal wire format)
type pushRequest struct {
	Mutations []Mutation `json:"mutations"`
}

type pushResponse struct {
	Results []Result `json:"results"`
}

type pullRequest struct {
	Scope  string `json:"scope,omitempty"`
	Cursor uint64 `json:"cursor"`
	Limit  int    `json:"limit,omitempty"`
}

type pullResponse struct {
	Changes []Change `json:"changes"`
	HasMore bool     `json:"has_more"`
}

type snapshotRequest struct {
	Scope string `json:"scope,omitempty"`
}

type snapshotResponse struct {
	Data   map[string]map[string][]byte `json:"data"`
	Cursor uint64                       `json:"cursor"`
}

// Mount registers sync routes on a Mizu app at /_sync/*.
func (e *Engine) Mount(app *mizu.App) {
	e.MountAt(app, "/_sync")
}

// MountAt registers sync routes at a custom prefix.
func (e *Engine) MountAt(app *mizu.App, prefix string) {
	app.Post(prefix+"/push", e.handlePush())
	app.Post(prefix+"/pull", e.handlePull())
	app.Post(prefix+"/snapshot", e.handleSnapshot())
}

func (e *Engine) handlePush() mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req pushRequest
		if err := c.BindJSON(&req, 1<<20); err != nil { // 1MB max
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		if len(req.Mutations) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "no mutations provided",
			})
		}

		results, err := e.Push(c.Context(), req.Mutations)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, pushResponse{Results: results})
	}
}

func (e *Engine) handlePull() mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req pullRequest
		if err := c.BindJSON(&req, 1<<16); err != nil { // 64KB max
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		changes, hasMore, err := e.Pull(c.Context(), req.Scope, req.Cursor, req.Limit)
		if err != nil {
			code := http.StatusInternalServerError
			errCode := codeInternal
			if errors.Is(err, errCursorTooOld) {
				code = http.StatusGone
				errCode = codeCursorTooOld
			}
			return c.JSON(code, map[string]string{
				"code":  errCode,
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, pullResponse{
			Changes: changes,
			HasMore: hasMore,
		})
	}
}

func (e *Engine) handleSnapshot() mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req snapshotRequest
		if err := c.BindJSON(&req, 1<<16); err != nil { // 64KB max
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		data, cursor, err := e.Snapshot(c.Context(), req.Scope)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, snapshotResponse{
			Data:   data,
			Cursor: cursor,
		})
	}
}
