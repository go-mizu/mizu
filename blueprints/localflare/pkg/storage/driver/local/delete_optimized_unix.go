//go:build !windows

// File: driver/local/delete_optimized_unix.go
package local

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// =============================================================================
// BATCH DELETE WITH UNLINKAT
// =============================================================================
// Uses directory file descriptors and unlinkat for faster batch deletes.

// batchDeleteFiles deletes multiple files efficiently using unlinkat.
// Files are grouped by directory to minimize syscalls.
func batchDeleteFiles(files []string) error {
	if len(files) == 0 {
		return nil
	}

	// Group files by directory
	byDir := make(map[string][]string)
	for _, f := range files {
		dir := filepath.Dir(f)
		base := filepath.Base(f)
		byDir[dir] = append(byDir[dir], base)
	}

	// Delete files in each directory using unlinkat
	for dir, bases := range byDir {
		fd, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY, 0)
		if err != nil {
			// Fall back to regular delete
			for _, base := range bases {
				os.Remove(filepath.Join(dir, base))
			}
			continue
		}

		for _, base := range bases {
			// Use unlinkat with pre-opened directory fd
			unix.Unlinkat(fd, base, 0)
		}

		unix.Close(fd)
	}

	return nil
}

// =============================================================================
// ASYNC DELETE QUEUE
// =============================================================================
// Non-blocking delete queue for when durability isn't critical.

const (
	deleteQueueSize    = 10000
	deleteWorkers      = 4
	deleteFlushTimeout = 100 * time.Millisecond
)

type asyncDeleteQueue struct {
	ch      chan string
	done    chan struct{}
	wg      sync.WaitGroup
	started bool
	mu      sync.Mutex
}

var globalDeleteQueue = &asyncDeleteQueue{
	ch:   make(chan string, deleteQueueSize),
	done: make(chan struct{}),
}

// startDeleteWorkers starts background workers for async deletion.
func (q *asyncDeleteQueue) start() {
	q.mu.Lock()
	if q.started {
		q.mu.Unlock()
		return
	}
	q.started = true
	q.mu.Unlock()

	for i := 0; i < deleteWorkers; i++ {
		q.wg.Add(1)
		go q.worker()
	}
}

// worker processes delete requests from the queue.
func (q *asyncDeleteQueue) worker() {
	defer q.wg.Done()

	batch := make([]string, 0, 100)
	ticker := time.NewTicker(deleteFlushTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-q.done:
			// Flush remaining
			if len(batch) > 0 {
				batchDeleteFiles(batch)
			}
			return

		case path := <-q.ch:
			batch = append(batch, path)
			// Flush when batch is full
			if len(batch) >= 100 {
				batchDeleteFiles(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			// Periodic flush
			if len(batch) > 0 {
				batchDeleteFiles(batch)
				batch = batch[:0]
			}
		}
	}
}

// deleteAsync queues a file for async deletion.
// Returns immediately. Falls back to sync delete if queue is full.
func deleteAsync(path string) {
	globalDeleteQueue.start()

	select {
	case globalDeleteQueue.ch <- path:
		// Queued successfully
	default:
		// Queue full, delete synchronously
		os.Remove(path)
	}
}

// stopDeleteQueue stops the async delete workers and flushes remaining deletes.
func stopDeleteQueue() {
	close(globalDeleteQueue.done)
	globalDeleteQueue.wg.Wait()
}

// =============================================================================
// OPTIMIZED SINGLE DELETE
// =============================================================================

// deleteWithUnlink deletes a single file using unlinkat for potential speedup.
func deleteWithUnlink(path string) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	fd, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		// Fall back to regular delete
		return os.Remove(path)
	}
	defer unix.Close(fd)

	err = unix.Unlinkat(fd, base, 0)
	if err != nil {
		return &os.PathError{Op: "unlinkat", Path: path, Err: err}
	}
	return nil
}

// =============================================================================
// RECURSIVE DELETE OPTIMIZATION
// =============================================================================

// deleteRecursiveFast deletes a directory tree using optimized syscalls.
func deleteRecursiveFast(root string) error {
	// Collect all files first
	var files []string
	var dirs []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			dirs = append(dirs, path)
		} else {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Delete files in batches
	batchDeleteFiles(files)

	// Delete directories in reverse order (deepest first)
	for i := len(dirs) - 1; i >= 0; i-- {
		os.Remove(dirs[i])
	}

	return nil
}
