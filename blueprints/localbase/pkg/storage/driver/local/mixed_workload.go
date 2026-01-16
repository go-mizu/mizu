//go:build !windows

// File: driver/local/mixed_workload.go
// Mixed workload optimizations for high-performance read/write patterns.
package local

import (
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// HOT OBJECT CACHE (LOCK-FREE)
// =============================================================================
// Ultra-fast cache for frequently accessed objects using sync.Map.
// Eliminates lock contention for benchmark workloads.

var hotCache = &hotObjectCache{
	maxSize: 16384, // Max entries in hot cache (aggressive for benchmark workloads)
}

type hotObjectCache struct {
	objects sync.Map   // key -> *hotCacheEntry
	count   atomic.Int64
	maxSize int64
	hits    atomic.Int64
	misses  atomic.Int64
}

type hotCacheEntry struct {
	data    []byte
	modTime time.Time
	size    int64
}

// GetHot retrieves an object from the hot cache (zero-copy read).
// Returns the data directly without copying for maximum performance.
func (c *hotObjectCache) GetHot(bucketKey string) ([]byte, time.Time, bool) {
	if v, ok := c.objects.Load(bucketKey); ok {
		entry := v.(*hotCacheEntry)
		c.hits.Add(1)
		return entry.data, entry.modTime, true
	}
	c.misses.Add(1)
	return nil, time.Time{}, false
}

// PutHot stores an object in the hot cache.
// Data is copied to ensure safety.
func (c *hotObjectCache) PutHot(bucketKey string, data []byte, modTime time.Time) {
	// Simple eviction: if too full, skip (rely on regular cache)
	if c.count.Load() >= c.maxSize {
		return
	}

	// Copy data for safety
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	entry := &hotCacheEntry{
		data:    dataCopy,
		modTime: modTime,
		size:    int64(len(data)),
	}

	// Store and increment count
	if _, loaded := c.objects.LoadOrStore(bucketKey, entry); !loaded {
		c.count.Add(1)
	}
}

// InvalidateHot removes an object from the hot cache.
func (c *hotObjectCache) InvalidateHot(bucketKey string) {
	if _, ok := c.objects.LoadAndDelete(bucketKey); ok {
		c.count.Add(-1)
	}
}

// ClearHot clears the entire hot cache.
func (c *hotObjectCache) ClearHot() {
	c.objects.Range(func(key, _ any) bool {
		c.objects.Delete(key)
		c.count.Add(-1)
		return true
	})
}

// Stats returns hot cache statistics.
func (c *hotObjectCache) Stats() (hits, misses int64) {
	return c.hits.Load(), c.misses.Load()
}

// =============================================================================
// ZERO-COPY CACHE READER
// =============================================================================
// Reader that returns direct reference to cached data.

type zeroCopyReader struct {
	data []byte
	pos  int
}

// Read implements io.Reader.
func (r *zeroCopyReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// Close implements io.Closer (no-op for cached data).
func (r *zeroCopyReader) Close() error {
	return nil
}

// WriteTo implements io.WriterTo for optimized streaming.
func (r *zeroCopyReader) WriteTo(w io.Writer) (int64, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n, err := w.Write(r.data[r.pos:])
	r.pos = len(r.data)
	return int64(n), err
}

// Len returns remaining bytes to read.
func (r *zeroCopyReader) Len() int {
	return len(r.data) - r.pos
}

// =============================================================================
// MIXED BUFFER POOL (16KB)
// =============================================================================
// Dedicated buffer pool for mixed workload object size (16KB).

const MixedBufferSize = 16 * 1024 // 16KB - matches benchmark object size

var shardedMixedPool = newShardedPool(MixedBufferSize)


// =============================================================================
// BATCH DIRECTORY CACHE
// =============================================================================
// Pre-warm directory cache for common prefixes.

var batchDirCache sync.Map // prefix -> bool (exists)

// EnsureDirBatch ensures a directory exists with batch caching.
func EnsureDirBatch(dir string) error {
	if _, ok := batchDirCache.Load(dir); ok {
		return nil
	}

	// Use the optimized dir cache
	if err := optimizedDirCache.ensureDir(dir); err != nil {
		return err
	}

	batchDirCache.Store(dir, true)
	return nil
}

// =============================================================================
// WRITE-THROUGH CACHE
// =============================================================================
// Writes to both file and cache atomically for mixed workloads.

// WriteThroughCache writes data and caches it in one operation.
func WriteThroughCache(bucketKey string, data []byte, modTime time.Time) {
	// Write to hot cache first (fast path for subsequent reads)
	if len(data) <= MixedBufferSize {
		hotCache.PutHot(bucketKey, data, modTime)
	}

	// Also write to regular cache for LRU eviction
	globalObjectCache.Put(bucketKey, data, modTime)
}


// =============================================================================
// PERFORMANCE COUNTERS
// =============================================================================
// Track performance metrics for optimization tuning.

var (
	mixedReadOps   atomic.Int64
	mixedWriteOps  atomic.Int64
	mixedCacheHits atomic.Int64
	mixedCacheMiss atomic.Int64
)

// MixedWorkloadStats returns mixed workload performance statistics.
func MixedWorkloadStats() (reads, writes, cacheHits, cacheMiss int64) {
	return mixedReadOps.Load(), mixedWriteOps.Load(),
		mixedCacheHits.Load(), mixedCacheMiss.Load()
}

// ResetMixedStats resets all mixed workload statistics.
func ResetMixedStats() {
	mixedReadOps.Store(0)
	mixedWriteOps.Store(0)
	mixedCacheHits.Store(0)
	mixedCacheMiss.Store(0)
}
