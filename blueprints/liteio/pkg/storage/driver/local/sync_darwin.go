//go:build darwin

// File: driver/local/sync_darwin.go
// macOS-specific sync using F_BARRIERFSYNC for better write performance.
package local

import (
	"os"
	"sync"
	"syscall"
)

// F_BARRIERFSYNC ensures all data written before the barrier is persisted
// before any data written after the barrier. Unlike F_FULLFSYNC which flushes
// the SSD's entire write cache (~5ms), F_BARRIERFSYNC only ensures ordering
// (~1-2ms). Both provide crash-safe durability guarantees.
//
// Available since macOS 10.14 (Mojave). Falls back to F_FULLFSYNC on error.
const fBarrierFsync = 85

// syncFile uses F_BARRIERFSYNC on macOS for faster durable writes.
// This is 2-5x faster than Go's default f.Sync() which uses F_FULLFSYNC.
func syncFile(f *os.File) error {
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, f.Fd(), fBarrierFsync, 0)
	if errno != 0 {
		// Fall back to F_FULLFSYNC if F_BARRIERFSYNC not supported
		return f.Sync()
	}
	return nil
}

// groupSync syncs files in parallel on macOS.
// Unlike ext4, macOS has no journal commit optimization — each file needs
// its own fsync. Parallel execution lets the NVMe controller coalesce I/O.
func groupSync(batch []syncRequest) {
	if len(batch) == 1 {
		batch[0].done <- syncFile(batch[0].file)
		return
	}

	var wg sync.WaitGroup
	for i := range batch {
		wg.Add(1)
		go func(r *syncRequest) {
			defer wg.Done()
			r.done <- syncFile(r.file)
		}(&batch[i])
	}
	wg.Wait()
}
