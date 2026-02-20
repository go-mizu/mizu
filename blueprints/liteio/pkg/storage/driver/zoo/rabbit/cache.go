package rabbit

import (
	"container/list"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// HOT CACHE (L1) - Lock-Free Ring Buffer
// =============================================================================
// Ultra-fast cache for frequently accessed objects using atomic operations.

type HotCache struct {
	slots    [HotCacheSlots]hotSlot
	nextSlot atomic.Uint64
	hits     atomic.Int64
	misses   atomic.Int64
}

type hotSlot struct {
	key     atomic.Pointer[string]
	data    atomic.Pointer[[]byte]
	modTime atomic.Int64 // Unix nano
	hash    atomic.Uint64
}

func newHotCache() *HotCache {
	return &HotCache{}
}

// Get retrieves data from hot cache (lock-free).
func (c *HotCache) Get(key string) ([]byte, time.Time, bool) {
	hash := uint64(fnv1a(key))
	slot := hash % HotCacheSlots

	// Check if slot matches our key
	s := &c.slots[slot]
	if s.hash.Load() != hash {
		c.misses.Add(1)
		return nil, time.Time{}, false
	}

	keyPtr := s.key.Load()
	if keyPtr == nil || *keyPtr != key {
		c.misses.Add(1)
		return nil, time.Time{}, false
	}

	dataPtr := s.data.Load()
	if dataPtr == nil {
		c.misses.Add(1)
		return nil, time.Time{}, false
	}

	c.hits.Add(1)
	modTime := time.Unix(0, s.modTime.Load())
	return *dataPtr, modTime, true
}

// Put stores data in hot cache (lock-free).
func (c *HotCache) Put(key string, data []byte, modTime time.Time) {
	hash := uint64(fnv1a(key))
	slot := hash % HotCacheSlots

	// Copy data to avoid mutations
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	s := &c.slots[slot]
	s.hash.Store(hash)
	s.key.Store(&key)
	s.data.Store(&dataCopy)
	s.modTime.Store(modTime.UnixNano())
}

// GetHot provides zero-copy access to hot cache data.
// The returned slice is valid only until the next Put to the same slot.
func (c *HotCache) GetHot(key string) ([]byte, time.Time, bool) {
	hash := uint64(fnv1a(key))
	slot := hash % HotCacheSlots

	s := &c.slots[slot]
	if s.hash.Load() != hash {
		return nil, time.Time{}, false
	}

	keyPtr := s.key.Load()
	if keyPtr == nil || *keyPtr != key {
		return nil, time.Time{}, false
	}

	dataPtr := s.data.Load()
	if dataPtr == nil {
		return nil, time.Time{}, false
	}

	modTime := time.Unix(0, s.modTime.Load())
	return *dataPtr, modTime, true
}

// Invalidate removes an entry from hot cache.
func (c *HotCache) Invalidate(key string) {
	hash := uint64(fnv1a(key))
	slot := hash % HotCacheSlots

	s := &c.slots[slot]
	if s.hash.Load() == hash {
		keyPtr := s.key.Load()
		if keyPtr != nil && *keyPtr == key {
			s.hash.Store(0)
			s.key.Store(nil)
			s.data.Store(nil)
		}
	}
}

// =============================================================================
// WARM CACHE (L2) - Sharded LRU
// =============================================================================

type WarmCache struct {
	shards    [NumCacheShards]*cacheShard
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
	key         string
	data        []byte
	size        int64
	modTime     time.Time
	element     *list.Element
	accessCount int32
}

func newWarmCache(maxBytes int64, maxItems int) *WarmCache {
	c := &WarmCache{
		maxBytes: maxBytes,
		maxItems: maxItems,
	}
	c.enabled.Store(true)

	shardBytes := maxBytes / NumCacheShards
	shardItems := maxItems / NumCacheShards

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

func (c *WarmCache) shardIndex(key string) int {
	return int(fnv1a(key) % NumCacheShards)
}

// Get retrieves data from warm cache.
func (c *WarmCache) Get(key string) ([]byte, time.Time, bool) {
	if !c.enabled.Load() {
		return nil, time.Time{}, false
	}

	shard := c.shards[c.shardIndex(key)]
	shard.mu.RLock()
	entry, ok := shard.entries[key]
	if !ok {
		shard.mu.RUnlock()
		c.misses.Add(1)
		return nil, time.Time{}, false
	}

	// Copy data
	data := make([]byte, len(entry.data))
	copy(data, entry.data)
	modTime := entry.modTime

	// Lazy LRU
	entry.accessCount++
	shouldUpdate := entry.accessCount >= LazyLRUThreshold
	if shouldUpdate {
		entry.accessCount = 0
	}
	shard.mu.RUnlock()

	if shouldUpdate && entry.element != nil {
		shard.mu.Lock()
		shard.lru.MoveToFront(entry.element)
		shard.mu.Unlock()
	}

	c.hits.Add(1)
	return data, modTime, true
}

// Put stores data in warm cache.
func (c *WarmCache) Put(key string, data []byte, modTime time.Time) {
	if !c.enabled.Load() {
		return
	}

	size := int64(len(data))
	if size > SmallThreshold || size == 0 {
		return // Don't cache large or empty objects
	}

	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	shard := c.shards[c.shardIndex(key)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Update existing
	if existing, ok := shard.entries[key]; ok {
		shard.size -= existing.size
		existing.data = dataCopy
		existing.size = size
		existing.modTime = modTime
		shard.size += size
		shard.lru.MoveToFront(existing.element)
		return
	}

	// Evict if needed
	for shard.size+size > shard.maxSize || len(shard.entries) >= shard.maxItems {
		if shard.lru.Len() == 0 {
			break
		}
		c.evictOldest(shard)
	}

	// Add new entry
	entry := &cacheEntry{
		key:     key,
		data:    dataCopy,
		size:    size,
		modTime: modTime,
	}
	entry.element = shard.lru.PushFront(entry)
	shard.entries[key] = entry
	shard.size += size
}

// Invalidate removes an entry from warm cache.
func (c *WarmCache) Invalidate(key string) {
	if !c.enabled.Load() {
		return
	}

	shard := c.shards[c.shardIndex(key)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if entry, ok := shard.entries[key]; ok {
		shard.lru.Remove(entry.element)
		shard.size -= entry.size
		delete(shard.entries, key)
	}
}

// InvalidatePrefix removes all entries with prefix.
func (c *WarmCache) InvalidatePrefix(prefix string) {
	if !c.enabled.Load() {
		return
	}

	for _, shard := range c.shards {
		shard.mu.Lock()
		var toDelete []string
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

func (c *WarmCache) evictOldest(shard *cacheShard) {
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

// =============================================================================
// CACHED READERS
// =============================================================================

// zeroCopyReader provides zero-copy access to cached data.
type zeroCopyReader struct {
	data []byte
	pos  int
}

func (r *zeroCopyReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *zeroCopyReader) Close() error { return nil }

func (r *zeroCopyReader) WriteTo(w io.Writer) (int64, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n, err := w.Write(r.data[r.pos:])
	r.pos = len(r.data)
	return int64(n), err
}

// cachedReader reads from copied cached data.
type cachedReader struct {
	data []byte
	pos  int
}

func (r *cachedReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *cachedReader) Close() error { return nil }

func (r *cachedReader) WriteTo(w io.Writer) (int64, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n, err := w.Write(r.data[r.pos:])
	r.pos = len(r.data)
	return int64(n), err
}

// =============================================================================
// DIRECTORY CACHE
// =============================================================================

type DirCache struct {
	shards [NumShards]dirCacheShard
	hits   atomic.Int64
	misses atomic.Int64
}

type dirCacheShard struct {
	mu      sync.RWMutex
	entries map[string]time.Time
}

var globalDirCache = newDirCache()

func newDirCache() *DirCache {
	c := &DirCache{}
	for i := range c.shards {
		c.shards[i].entries = make(map[string]time.Time, 64)
	}
	return c
}

func (c *DirCache) shardIndex(path string) uint32 {
	return fnv1a(path) % NumShards
}

func (c *DirCache) Check(path string) bool {
	shard := &c.shards[c.shardIndex(path)]
	shard.mu.RLock()
	t, ok := shard.entries[path]
	shard.mu.RUnlock()

	if ok && time.Since(t) < DirCacheTTL {
		c.hits.Add(1)
		return true
	}
	c.misses.Add(1)
	return false
}

func (c *DirCache) Add(path string) {
	shard := &c.shards[c.shardIndex(path)]
	shard.mu.Lock()
	if len(shard.entries) >= DirCacheMaxSize/NumShards {
		// Simple eviction: clear half
		for k := range shard.entries {
			delete(shard.entries, k)
			if len(shard.entries) < DirCacheMaxSize/NumShards/2 {
				break
			}
		}
	}
	shard.entries[path] = time.Now()
	shard.mu.Unlock()
}

func (c *DirCache) Invalidate(path string) {
	shard := &c.shards[c.shardIndex(path)]
	shard.mu.Lock()
	delete(shard.entries, path)
	shard.mu.Unlock()
}
