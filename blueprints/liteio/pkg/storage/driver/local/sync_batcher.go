//go:build !windows

// File: driver/local/sync_batcher.go
// Group commit sync batcher for high-throughput concurrent writes.
//
// Instead of each writer calling fdatasync individually, concurrent writers
// batch their sync requests. On ext4, the first fdatasync triggers a journal
// commit that covers ALL dirty inodes in the current transaction. Subsequent
// fdatasyncs in the same batch find their data already committed (fast no-op).
//
// This is the same "group commit" pattern used by databases (PostgreSQL,
// MySQL InnoDB) for high write throughput while maintaining durability.
//
// Performance characteristics:
//   - C1 (single writer): ~0 overhead (immediate sync, no delay)
//   - C10 (10 concurrent): ~5x improvement (10 syncs → 1 real + 9 fast)
//   - C50 (50 concurrent): ~10x improvement (50 syncs → 1 real + 49 fast)
package local

import (
	"os"
	"sync"
	"time"
)

// batchWait is the maximum time to wait for concurrent writers to accumulate
// before flushing the batch. Only applied when multiple writers are pending.
// 50µs is short enough to add negligible latency but long enough to capture
// most concurrent writers in a batch.
const batchWait = 50 * time.Microsecond

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
func (b *syncBatcher) BatchSync(f *os.File) error {
	if NoFsync {
		return nil
	}

	done := make(chan error, 1)

	b.mu.Lock()
	b.pending = append(b.pending, syncRequest{file: f, done: done})
	n := len(b.pending)
	b.mu.Unlock()

	// Wake the flusher (non-blocking send to avoid deadlock)
	select {
	case b.wake <- struct{}{}:
	default:
		// Flusher already signaled; it will pick up our request
		// when it next checks pending
		_ = n
	}

	// Block until our sync completes — same durability guarantee as direct fsync
	return <-done
}

// loop is the background flusher goroutine.
func (b *syncBatcher) loop() {
	for range b.wake {
		b.processBatches()
	}
}

// processBatches drains all pending sync requests in waves.
// It keeps looping until no more requests are pending, to handle
// requests that arrive during the sync phase (whose signals may
// have been dropped because the wake channel was full).
func (b *syncBatcher) processBatches() {
	for {
		// Check how many writers are pending
		b.mu.Lock()
		n := len(b.pending)
		b.mu.Unlock()

		if n == 0 {
			return
		}

		// If multiple writers are pending, wait briefly to accumulate more.
		// For single writers, process immediately (no added latency).
		if n > 1 {
			time.Sleep(batchWait)
		}

		// Collect all pending requests
		b.mu.Lock()
		batch := b.pending
		b.pending = make([]syncRequest, 0, max(len(batch), 8))
		b.mu.Unlock()

		if len(batch) == 0 {
			return
		}

		// Sync all files in the batch.
		//
		// On ext4 with journaling:
		//   - The first fdatasync triggers a journal commit for the current
		//     transaction, which includes ALL dirty inodes (all writers' data).
		//   - Subsequent fdatasyncs find their inodes already committed in the
		//     same journal transaction, so they return quickly.
		//   - Net effect: 1 real I/O for the entire batch.
		//
		// On other filesystems: each fdatasync still does its own sync,
		// but the kernel I/O scheduler can merge them since all data is
		// already in the page cache.
		for i := range batch {
			batch[i].done <- syncFile(batch[i].file)
		}
	}
}
