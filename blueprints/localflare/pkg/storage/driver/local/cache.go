//go:build !windows

// File: driver/local/cache.go
package local

import (
	"container/list"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// HOT OBJECT CACHE
// =============================================================================
// LRU cache for recently accessed objects to avoid repeated disk I/O.
// Critical for mixed workloads where the same objects are read repeatedly.

const (
	// CacheMaxBytes is the maximum total size of cached objects (256MB).
	CacheMaxBytes = 256 * 1024 * 1024

	// CacheMaxItems is the maximum number of cached entries.
	CacheMaxItems = 50000

	// CacheableThreshold is the maximum size of cacheable objects (128KB).
	// Objects larger than this bypass the cache.
	CacheableThreshold = 128 * 1024

	// CacheShards is the number of cache shards for concurrency.
	CacheShards = 64
)

// ObjectCache is a sharded LRU cache for object data.
type ObjectCache struct {
	shards    [CacheShards]*cacheShard
	maxBytes  int64
	maxItems  int
	enabled   atomic.Bool
	hits      atomic.Int64
	misses    atomic.Int64
	evictions atomic.Int64
}

type cacheShard struct {
	mu       sync.RWMutex
	entries  map[string]*cacheEntry
	lru      *list.List
	size     int64
	maxSize  int64
	maxItems int
}

type cacheEntry struct {
	key     string
	data    []byte
	size    int64
	modTime time.Time
	element *list.Element
}

// globalObjectCache is the shared cache instance.
var globalObjectCache = newObjectCache()

func newObjectCache() *ObjectCache {
	c := &ObjectCache{
		maxBytes: CacheMaxBytes,
		maxItems: CacheMaxItems,
	}
	c.enabled.Store(true)

	shardBytes := int64(CacheMaxBytes / CacheShards)
	shardItems := CacheMaxItems / CacheShards

	for i := range c.shards {
		c.shards[i] = &cacheShard{
			entries:  make(map[string]*cacheEntry, shardItems/2),
			lru:      list.New(),
			maxSize:  shardBytes,
			maxItems: shardItems,
		}
	}
	return c
}

// shardIndex returns the shard index for a key.
func (c *ObjectCache) shardIndex(key string) int {
	// Simple FNV-1a hash
	h := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return int(h % CacheShards)
}

// Get retrieves an object from cache if present.
// Returns the data, modification time, and whether it was found.
func (c *ObjectCache) Get(bucketKey string) ([]byte, time.Time, bool) {
	if !c.enabled.Load() {
		return nil, time.Time{}, false
	}

	shard := c.shards[c.shardIndex(bucketKey)]
	shard.mu.RLock()
	entry, ok := shard.entries[bucketKey]
	if !ok {
		shard.mu.RUnlock()
		c.misses.Add(1)
		return nil, time.Time{}, false
	}

	// Copy data to avoid race conditions (caller may hold reference)
	data := make([]byte, len(entry.data))
	copy(data, entry.data)
	modTime := entry.modTime
	shard.mu.RUnlock()

	// Update LRU position (upgrade to write lock)
	shard.mu.Lock()
	if entry.element != nil {
		shard.lru.MoveToFront(entry.element)
	}
	shard.mu.Unlock()

	c.hits.Add(1)
	return data, modTime, true
}

// Put stores an object in the cache.
// Objects larger than CacheableThreshold are not cached.
func (c *ObjectCache) Put(bucketKey string, data []byte, modTime time.Time) {
	if !c.enabled.Load() {
		return
	}

	size := int64(len(data))
	if size > CacheableThreshold || size == 0 {
		return
	}

	// Copy data to avoid mutations affecting cached data
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	shard := c.shards[c.shardIndex(bucketKey)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Check if already cached
	if existing, ok := shard.entries[bucketKey]; ok {
		// Update existing entry
		shard.size -= existing.size
		existing.data = dataCopy
		existing.size = size
		existing.modTime = modTime
		shard.size += size
		shard.lru.MoveToFront(existing.element)
		return
	}

	// Evict entries if needed
	for shard.size+size > shard.maxSize || len(shard.entries) >= shard.maxItems {
		if shard.lru.Len() == 0 {
			break
		}
		c.evictOldest(shard)
	}

	// Add new entry
	entry := &cacheEntry{
		key:     bucketKey,
		data:    dataCopy,
		size:    size,
		modTime: modTime,
	}
	entry.element = shard.lru.PushFront(entry)
	shard.entries[bucketKey] = entry
	shard.size += size
}

// Invalidate removes an object from the cache.
func (c *ObjectCache) Invalidate(bucketKey string) {
	if !c.enabled.Load() {
		return
	}

	shard := c.shards[c.shardIndex(bucketKey)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if entry, ok := shard.entries[bucketKey]; ok {
		shard.lru.Remove(entry.element)
		shard.size -= entry.size
		delete(shard.entries, bucketKey)
	}
}

// InvalidatePrefix removes all objects with a given prefix from the cache.
func (c *ObjectCache) InvalidatePrefix(prefix string) {
	if !c.enabled.Load() {
		return
	}

	// Check all shards (expensive but rare operation)
	for _, shard := range c.shards {
		shard.mu.Lock()
		toDelete := make([]string, 0)
		for key := range shard.entries {
			if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
				toDelete = append(toDelete, key)
			}
		}
		for _, key := range toDelete {
			if entry, ok := shard.entries[key]; ok {
				shard.lru.Remove(entry.element)
				shard.size -= entry.size
				delete(shard.entries, key)
			}
		}
		shard.mu.Unlock()
	}
}

// evictOldest removes the least recently used entry from a shard.
// Must be called with shard.mu held.
func (c *ObjectCache) evictOldest(shard *cacheShard) {
	oldest := shard.lru.Back()
	if oldest == nil {
		return
	}

	entry := oldest.Value.(*cacheEntry)
	shard.lru.Remove(oldest)
	shard.size -= entry.size
	delete(shard.entries, entry.key)
	c.evictions.Add(1)
}

// Stats returns cache statistics.
func (c *ObjectCache) Stats() (hits, misses, evictions int64) {
	return c.hits.Load(), c.misses.Load(), c.evictions.Load()
}

// Enable enables or disables the cache.
func (c *ObjectCache) Enable(enable bool) {
	c.enabled.Store(enable)
}

// Clear removes all entries from the cache.
func (c *ObjectCache) Clear() {
	for _, shard := range c.shards {
		shard.mu.Lock()
		shard.entries = make(map[string]*cacheEntry, shard.maxItems/2)
		shard.lru = list.New()
		shard.size = 0
		shard.mu.Unlock()
	}
}

// cacheKey generates a cache key from bucket name and object key.
func cacheKey(bucket, key string) string {
	// Use simple concatenation with separator for efficiency
	return bucket + "\x00" + key
}

// =============================================================================
// CACHED READER
// =============================================================================
// In-memory reader for cached objects.

// cachedReader reads from in-memory cached data.
type cachedReader struct {
	data []byte
	pos  int
}

// Read implements io.Reader.
func (r *cachedReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// Close implements io.Closer.
func (r *cachedReader) Close() error {
	return nil
}

// WriteTo implements io.WriterTo for optimized streaming.
func (r *cachedReader) WriteTo(w io.Writer) (int64, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n, err := w.Write(r.data[r.pos:])
	r.pos = len(r.data)
	return int64(n), err
}
