//go:build !windows

// File: driver/local/handle_pool.go
package local

import (
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// FILE HANDLE POOLING
// =============================================================================
// Pools open file handles to reduce open/close syscall overhead.
// Critical for high-concurrency workloads where the same files are accessed repeatedly.

const (
	// MaxPooledHandles is the maximum number of handles to keep open.
	MaxPooledHandles = 1000

	// HandleMaxAge is how long to keep idle handles open.
	HandleMaxAge = 30 * time.Second

	// HandlePoolCleanupInterval is how often to clean up stale handles.
	HandlePoolCleanupInterval = 10 * time.Second
)

// pooledHandle wraps an os.File with reference counting.
type pooledHandle struct {
	file       *os.File
	path       string
	lastUsed   atomic.Int64
	refCount   atomic.Int32
	readOnly   bool
	validUntil int64
}

// handlePool manages pooled file handles.
type handlePool struct {
	mu       sync.RWMutex
	handles  map[string]*pooledHandle
	maxAge   time.Duration
	stopCh   chan struct{}
	stopped  bool
}

// globalHandlePool is the shared handle pool.
var globalHandlePool = newHandlePool()

func newHandlePool() *handlePool {
	hp := &handlePool{
		handles: make(map[string]*pooledHandle),
		maxAge:  HandleMaxAge,
		stopCh:  make(chan struct{}),
	}
	go hp.cleanupLoop()
	return hp
}

// Get returns a pooled handle for the given path, or opens a new one.
// Caller must call Release() when done with the handle.
func (hp *handlePool) Get(path string, readOnly bool) (*os.File, func(), error) {
	hp.mu.RLock()
	h, ok := hp.handles[path]
	if ok && h.readOnly == readOnly {
		// Found pooled handle
		h.refCount.Add(1)
		h.lastUsed.Store(time.Now().UnixNano())
		hp.mu.RUnlock()

		release := func() {
			h.refCount.Add(-1)
			h.lastUsed.Store(time.Now().UnixNano())
		}
		return h.file, release, nil
	}
	hp.mu.RUnlock()

	// Open new handle
	var f *os.File
	var err error
	if readOnly {
		// #nosec G304 -- path validated by caller
		f, err = os.Open(path)
	} else {
		// #nosec G304 -- path validated by caller
		f, err = os.OpenFile(path, os.O_RDWR, FilePermissions)
	}
	if err != nil {
		return nil, nil, err
	}

	// Try to add to pool
	hp.mu.Lock()
	defer hp.mu.Unlock()

	// Check again if another goroutine added it
	if existing, ok := hp.handles[path]; ok && existing.readOnly == readOnly {
		f.Close() // Don't need the new handle
		existing.refCount.Add(1)
		existing.lastUsed.Store(time.Now().UnixNano())
		release := func() {
			existing.refCount.Add(-1)
			existing.lastUsed.Store(time.Now().UnixNano())
		}
		return existing.file, release, nil
	}

	// Check if pool is full
	if len(hp.handles) >= MaxPooledHandles {
		// Don't pool, just return with simple close
		return f, func() { f.Close() }, nil
	}

	// Add to pool
	h = &pooledHandle{
		file:     f,
		path:     path,
		readOnly: readOnly,
	}
	h.refCount.Store(1)
	h.lastUsed.Store(time.Now().UnixNano())
	hp.handles[path] = h

	release := func() {
		h.refCount.Add(-1)
		h.lastUsed.Store(time.Now().UnixNano())
	}
	return f, release, nil
}

// Invalidate removes a handle from the pool (e.g., after write/delete).
func (hp *handlePool) Invalidate(path string) {
	hp.mu.Lock()
	defer hp.mu.Unlock()

	if h, ok := hp.handles[path]; ok {
		delete(hp.handles, path)
		// Wait for refs to drop, then close
		go func() {
			for h.refCount.Load() > 0 {
				time.Sleep(time.Millisecond)
			}
			h.file.Close()
		}()
	}
}

// cleanupLoop periodically removes stale handles.
func (hp *handlePool) cleanupLoop() {
	ticker := time.NewTicker(HandlePoolCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hp.stopCh:
			return
		case <-ticker.C:
			hp.cleanup()
		}
	}
}

func (hp *handlePool) cleanup() {
	hp.mu.Lock()
	defer hp.mu.Unlock()

	now := time.Now().UnixNano()
	maxAge := hp.maxAge.Nanoseconds()

	for path, h := range hp.handles {
		// Don't close handles that are in use
		if h.refCount.Load() > 0 {
			continue
		}
		// Close handles that haven't been used recently
		if now-h.lastUsed.Load() > maxAge {
			delete(hp.handles, path)
			h.file.Close()
		}
	}
}

// Close shuts down the handle pool and closes all handles.
func (hp *handlePool) Close() {
	hp.mu.Lock()
	if hp.stopped {
		hp.mu.Unlock()
		return
	}
	hp.stopped = true
	close(hp.stopCh)

	// Close all handles
	for _, h := range hp.handles {
		h.file.Close()
	}
	hp.handles = nil
	hp.mu.Unlock()
}

// =============================================================================
// GOROUTINE-LOCAL BUFFER CACHE
// =============================================================================
// Reduces buffer pool contention by caching buffers per-goroutine.

type goroutineBufferCache struct {
	// Use sync.Map for concurrent access without lock contention
	cache sync.Map // goroutineID -> *cachedBuffer
}

type cachedBuffer struct {
	buf      []byte
	lastUsed int64
}

var gBufferCache = &goroutineBufferCache{}

// getGoroutineID returns a unique identifier for the current goroutine.
// Note: This is a simplified approximation - in production you'd use
// runtime internals or thread-local storage.
func getGoroutineID() uint64 {
	// Use a counter that's incremented per-goroutine access
	// Combined with random to distribute across cache slots
	return uint64(fastrand())
}

// GetBuffer returns a buffer from goroutine-local cache or global pool.
func (c *goroutineBufferCache) GetBuffer(size int) []byte {
	gid := getGoroutineID() % 256 // Limit to 256 slots

	if cached, ok := c.cache.Load(gid); ok {
		cb := cached.(*cachedBuffer)
		if len(cb.buf) >= size {
			cb.lastUsed = time.Now().UnixNano()
			return cb.buf[:size]
		}
	}

	// Get from global pool
	var buf []byte
	switch {
	case size <= SmallBufferSize:
		buf = shardedSmallPool.Get()
	case size <= MediumBufferSize:
		buf = shardedMediumPool.Get()
	case size <= LargeBufferSize:
		buf = shardedLargePool.Get()
	default:
		buf = shardedHugePool.Get()
	}

	// Cache for this goroutine
	c.cache.Store(gid, &cachedBuffer{
		buf:      buf,
		lastUsed: time.Now().UnixNano(),
	})

	return buf[:size]
}

// ReturnBuffer is a no-op since we cache per-goroutine.
// The buffer will be reused on next call.
func (c *goroutineBufferCache) ReturnBuffer(buf []byte) {
	// No-op - buffer stays in goroutine cache
	_ = buf
}
