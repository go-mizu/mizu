//go:build !windows

// File: driver/local/sync_batcher.go
// Group commit sync batcher for high-throughput concurrent writes.
//
// Instead of each writer calling fdatasync individually, concurrent writers
// batch their sync requests. On ext4 (Linux), the first fdatasync triggers
// a journal commit covering ALL dirty inodes — one sync makes the entire
// batch durable. On other platforms, each file is synced individually.
//
// This is the same "group commit" pattern used by databases (PostgreSQL,
// MySQL InnoDB) for high write throughput while maintaining durability.
package local

import (
	"os"
	"sync"
	"time"
)

// batchWait is the maximum time to wait for concurrent writers to accumulate
// before flushing the batch. Only applied when multiple writers are pending.
// 100µs captures most concurrent writers while adding negligible latency.
const batchWait = 100 * time.Microsecond

// syncBatcher coalesces fdatasync calls from concurrent writers.
// Each writer blocks until its data is durable (same guarantee as direct fsync).
type syncBatcher struct {
	mu      sync.Mutex
	pending []syncRequest
	wake    chan struct{}
}

// syncRequest represents a pending sync for one file.
type syncRequest struct {
	file *os.File
	done chan error
}

// getDoneChan allocates a fresh done channel per sync request.
func getDoneChan() chan error {
	return make(chan error, 1)
}

// globalSyncBatcher is the shared batcher for all write operations.
var globalSyncBatcher = newSyncBatcher()

func newSyncBatcher() *syncBatcher {
	b := &syncBatcher{
		wake:    make(chan struct{}, 1),
		pending: make([]syncRequest, 0, 64),
	}
	go b.loop()
	return b
}

// BatchSync adds a file to the sync batch and blocks until its data is durable.
// Multiple concurrent callers are batched together for efficiency.
// Returns immediately if NoFsync is set.
//
// OPTIMIZATIONS vs naive approach:
// - No time.After timeout (eliminates timer goroutine creation per write)
// - Batcher goroutine processes on a separate thread (better NVMe scheduling)
func (b *syncBatcher) BatchSync(f *os.File) error {
	if NoFsync {
		return nil
	}

	done := getDoneChan()

	b.mu.Lock()
	b.pending = append(b.pending, syncRequest{file: f, done: done})
	b.mu.Unlock()

	// Wake the flusher (non-blocking send)
	select {
	case b.wake <- struct{}{}:
	default:
	}

	// Block until our sync completes.
	return <-done
}

// loop is the background flusher goroutine.
func (b *syncBatcher) loop() {
	for range b.wake {
		b.processBatches()
	}
}

// processBatches drains all pending sync requests in waves.
// Keeps looping until no more requests are pending to handle
// requests whose wake signals were dropped (channel was full).
func (b *syncBatcher) processBatches() {
	for {
		b.mu.Lock()
		n := len(b.pending)
		b.mu.Unlock()

		if n == 0 {
			return
		}

		// Wait briefly to accumulate more writers into the batch.
		// For single writers, process immediately (no added latency).
		if n > 1 {
			time.Sleep(batchWait)
		}

		// Drain all pending requests
		b.mu.Lock()
		batch := b.pending
		b.pending = make([]syncRequest, 0, max(len(batch), 8))
		b.mu.Unlock()

		if len(batch) == 0 {
			return
		}

		// groupSync is platform-specific:
		// - Linux: single fdatasync (ext4 journal covers all dirty inodes)
		// - macOS: parallel F_BARRIERFSYNC (NVMe coalescing)
		// - Other: parallel fsync
		groupSync(batch)
	}
}
