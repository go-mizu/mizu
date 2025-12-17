// Package sync provides a client-side synchronization runtime for offline-first
// interactive applications.
//
// It integrates:
//   - sync package as the authoritative correctness layer
//   - Reactive state management (Signal, Computed, Effect) for UI binding
//   - Optional live package as a latency accelerator
//
// # Core Concepts
//
// Client is the main runtime coordinating local state, mutation queue, and sync:
//
//	client := sync.New(sync.Options{
//	    BaseURL: "https://api.example.com/_sync",
//	    Scope:   "user:123",
//	})
//	client.Start(ctx)
//
// Signal provides reactive state that notifies dependents on change:
//
//	count := sync.NewSignal(0)
//	count.Set(1) // Triggers dependents
//
// Computed derives values that recompute when dependencies change:
//
//	doubled := sync.NewComputed(func() int {
//	    return count.Get() * 2
//	})
//
// Effect runs side effects when dependencies change:
//
//	sync.NewEffect(func() {
//	    fmt.Println("Count:", count.Get())
//	})
//
// Collection manages synchronized entities:
//
//	todos := sync.NewCollection[Todo](client, "todo")
//	todo := todos.Create("abc", Todo{Title: "Buy milk"})
//
// # Offline Support
//
// The client operates fully offline. Mutations are queued locally and pushed
// when connectivity is restored. The sync protocol ensures idempotent,
// conflict-free convergence.
//
// # Live Integration
//
// When a live connection is provided, the client receives push notifications
// for immediate sync triggers, reducing latency compared to polling.
package sync
