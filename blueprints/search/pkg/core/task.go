package core

import "context"

// Task represents a unit of work executed by a worker, pipeline stage,
// or distributed runtime.
//
// A Task may emit intermediate state snapshots through the emit callback.
// The final result of the execution is returned as Metric.
//
// Generics:
//   State  – type describing the observable task state
//   Metric – type describing the final result/metrics of the task
type Task[State, Metric any] interface {

	// Run executes the task.
	//
	// ctx allows cancellation and deadline control.
	//
	// emit publishes intermediate state updates. Implementations
	// may call emit zero or more times during execution.
	//
	// The returned Metric typically represents the final summary
	// of the task (for example: number of pages crawled, bytes
	// processed, errors, latency statistics).
	Run(ctx context.Context, emit func(*State)) (Metric, error)
}