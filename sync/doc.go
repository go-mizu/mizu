// Package sync provides offline-first data synchronization with push/pull semantics.
//
// The sync package implements a change log based synchronization protocol that allows
// clients to operate offline and converge to consistent state when connectivity returns.
//
// # Core Concepts
//
//   - Mutation: A client-originated state change request
//   - Change: A logged state change with a monotonically increasing cursor
//   - ChangeLog: Stores and retrieves changes for a scope
//   - Store: Scoped key-value storage for entity data
//   - Mutator: Business logic that applies mutations and produces changes
//   - PokeBroker: Notifies live connections when data changes
//
// # Integration with mizu/live
//
// The sync package integrates with mizu/live via the PokeBroker interface.
// When mutations are applied, the sync engine calls broker.Poke() to notify
// live connections that data has changed, triggering an immediate pull.
//
// # Example Usage
//
//	// Create a mutator
//	mutator := sync.MutatorFunc(func(ctx context.Context, store sync.Store, m sync.Mutation) ([]sync.Change, error) {
//	    switch m.Name {
//	    case "todo/create":
//	        // Apply mutation and return changes
//	    }
//	    return nil, fmt.Errorf("unknown mutation: %s", m.Name)
//	})
//
//	// Create engine
//	engine := sync.New(sync.Options{
//	    Store:     sync.NewMemoryStore(),
//	    ChangeLog: sync.NewMemoryChangeLog(),
//	    Mutator:   mutator,
//	    Broker:    pokeBroker,
//	})
//
//	// Mount on app
//	engine.Mount(app)
package sync
