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
//	mutator := sync.NewMutatorMap()
//	mutator.Register("todo/create", handleTodoCreate)
//	mutator.Register("todo/toggle", handleTodoToggle)
//
//	engine := sync.New(sync.Options{
//	    Store:   store,
//	    Log:     log,
//	    Applied: applied,
//	    Mutator: mutator,
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
