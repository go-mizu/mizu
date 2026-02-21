package herd

import (
	"sort"
	"strings"
	"sync"
)

const shardCount = 256
const shardMask = shardCount - 1 // v4: bitmask for power-of-2 shard selection

// indexEntry stores the location and metadata for a single object.
type indexEntry struct {
	valueOffset int64  // offset in volume file where value bytes start (0 for inline)
	size        int64  // value size in bytes
	contentType string // interned content type
	created     int64  // UnixNano
	updated     int64  // UnixNano
	inline      []byte // inline value data (≤inlineMax), nil for volume-backed
}

// indexEntryPool reduces GC pressure by recycling indexEntry allocations.
var indexEntryPool = sync.Pool{
	New: func() any { return &indexEntry{} },
}

func acquireIndexEntry() *indexEntry {
	e := indexEntryPool.Get().(*indexEntry)
	*e = indexEntry{}
	return e
}

func releaseIndexEntry(e *indexEntry) {
	if e != nil {
		e.inline = nil // help GC
		indexEntryPool.Put(e)
	}
}

// shardBucket is the per-bucket data within a shard.
// v3 optimization: merged bucketKeys into entries — single map lookup path,
// eliminates compositeKey string concatenation (was 3.99% heap = 1325MB).
type shardBucket struct {
	entries map[string]*indexEntry // key → entry (NO composite key needed)
	sorted  []string              // lazy-rebuilt sorted key cache
	dirty   bool                  // true after put/remove
}

// shard is one segment of the sharded hash index.
// v3: Two-level map (bucket → key → entry) eliminates composite key allocation.
type shard struct {
	mu      sync.RWMutex
	buckets map[string]*shardBucket // bucket → per-bucket data
	_       [40]byte                // padding to avoid false sharing
}

// shardedIndex is a 256-shard concurrent hash index.
type shardedIndex struct {
	shards [shardCount]shard
}

func newIndex() *shardedIndex {
	idx := &shardedIndex{}
	for i := range idx.shards {
		idx.shards[i].buckets = make(map[string]*shardBucket, 4)
	}
	return idx
}

// shardForParts computes shard index from bucket+key without allocation.
// v4: bitmask instead of modulo, proper separator byte.
func shardForParts(bucket, key string) uint32 {
	const offset32 = 2166136261
	const prime32 = 16777619
	h := uint32(offset32)
	for i := 0; i < len(bucket); i++ {
		h ^= uint32(bucket[i])
		h *= prime32
	}
	h ^= 0xFF // v4: proper separator (was no-op h ^= 0)
	h *= prime32
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= prime32
	}
	return h & shardMask
}

func (idx *shardedIndex) put(bucket, key string, e *indexEntry) {
	si := shardForParts(bucket, key)
	s := &idx.shards[si]

	s.mu.Lock()
	sb := s.buckets[bucket]
	if sb == nil {
		sb = &shardBucket{entries: make(map[string]*indexEntry, 64)}
		s.buckets[bucket] = sb
	}
	old, exists := sb.entries[key]
	sb.entries[key] = e
	if !exists {
		sb.dirty = true
	}
	s.mu.Unlock()

	if exists {
		releaseIndexEntry(old)
	}
}

func (idx *shardedIndex) get(bucket, key string) (*indexEntry, bool) {
	si := shardForParts(bucket, key)
	s := &idx.shards[si]

	s.mu.RLock()
	sb := s.buckets[bucket]
	if sb == nil {
		s.mu.RUnlock()
		return nil, false
	}
	e, ok := sb.entries[key]
	s.mu.RUnlock()
	return e, ok
}

func (idx *shardedIndex) remove(bucket, key string) bool {
	si := shardForParts(bucket, key)
	s := &idx.shards[si]

	s.mu.Lock()
	sb := s.buckets[bucket]
	if sb == nil {
		s.mu.Unlock()
		return false
	}
	old, exists := sb.entries[key]
	if exists {
		delete(sb.entries, key)
		sb.dirty = true
	}
	s.mu.Unlock()

	if exists {
		releaseIndexEntry(old)
	}
	return exists
}

// list returns keys matching bucket+prefix using sorted arrays with binary search.
// Sorted arrays are rebuilt lazily (only when dirty from writes), giving O(log n + m)
// per shard for prefix queries instead of O(n).
func (idx *shardedIndex) list(bucket, prefix string) []listResult {
	var results []listResult
	for i := range idx.shards {
		s := &idx.shards[i]

		// First try RLock — fast path when sorted cache is valid.
		s.mu.RLock()
		sb := s.buckets[bucket]
		if sb == nil || len(sb.entries) == 0 {
			s.mu.RUnlock()
			continue
		}

		if sb.dirty {
			// Need to rebuild sorted list. Upgrade to write lock.
			s.mu.RUnlock()
			s.mu.Lock()
			// Double-check after acquiring write lock.
			sb = s.buckets[bucket]
			if sb != nil && sb.dirty {
				sb.sorted = sb.sorted[:0]
				for k := range sb.entries {
					sb.sorted = append(sb.sorted, k)
				}
				sort.Strings(sb.sorted)
				sb.dirty = false
			}
			// Downgrade: collect results under write lock (safe, just slower).
			if sb != nil && len(sb.sorted) > 0 {
				idx.collectResults(sb, prefix, &results)
			}
			s.mu.Unlock()
			continue
		}

		// Fast path: sorted cache is valid.
		idx.collectResults(sb, prefix, &results)
		s.mu.RUnlock()
	}
	return results
}

// collectResults appends matching list results from a sorted key slice.
// Uses binary search for O(log n + m) prefix matching.
func (idx *shardedIndex) collectResults(sb *shardBucket, prefix string, results *[]listResult) {
	sorted := sb.sorted
	if prefix == "" {
		for _, key := range sorted {
			if e, ok := sb.entries[key]; ok {
				*results = append(*results, listResult{key: key, entry: e})
			}
		}
		return
	}

	start := sort.SearchStrings(sorted, prefix)
	for j := start; j < len(sorted); j++ {
		key := sorted[j]
		if !strings.HasPrefix(key, prefix) {
			break
		}
		if e, ok := sb.entries[key]; ok {
			*results = append(*results, listResult{key: key, entry: e})
		}
	}
}

func (idx *shardedIndex) hasBucket(bucket string) bool {
	for i := range idx.shards {
		s := &idx.shards[i]
		s.mu.RLock()
		sb := s.buckets[bucket]
		has := sb != nil && len(sb.entries) > 0
		s.mu.RUnlock()
		if has {
			return true
		}
	}
	return false
}

type listResult struct {
	key   string
	entry *indexEntry
}
