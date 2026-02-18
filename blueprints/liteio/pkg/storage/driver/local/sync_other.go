//go:build !linux && !darwin && !windows

// File: driver/local/sync_other.go
// Fallback sync for platforms without specific optimizations.
package local

import (
	"os"
	"sync"
)

// syncFile falls back to standard Sync (fsync) on generic platforms.
func syncFile(f *os.File) error {
	return f.Sync()
}

// groupSync syncs files in parallel on generic platforms.
// Parallel execution lets the kernel coalesce I/O operations.
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
