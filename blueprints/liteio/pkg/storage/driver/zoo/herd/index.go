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
	valueOffset int64       // offset in volume file where value bytes start (0 for inline)
	size        int64       // value size in bytes
	contentType string      // interned content type
	created     int64       // UnixNano
	updated     int64       // UnixNano
	inline      []byte      // inline value data (≤inlineMax), nil for volume-backed
	flNext      *indexEntry // v5: freelist link (used only when entry is in freelist)
}

// globalFreelist is the default indexEntry freelist for standalone usage (volume recovery).
// Per-index freelists (shardedIndex.fl) are preferred for the hot path.
var globalFreelist indexEntryFreelist

// acquireIndexEntry gets an indexEntry from the global freelist (used in recovery + multipart).
func acquireIndexEntry() *indexEntry {
	return globalFreelist.acquire()
}

// releaseIndexEntry returns an indexEntry to the global freelist.
func releaseIndexEntry(e *indexEntry) {
	globalFreelist.release(e)
}

// shardBucket is the per-bucket data within a shard.
// v3 optimization: merged bucketKeys into entries — single map lookup path,
// eliminates compositeKey string concatenation (was 3.99% heap = 1325MB).
// v5: swiss table replaces Go map, incremental sorted insert, no dirty flag.
type shardBucket struct {
	entries swissTable // v5: open-addressing hash table (replaces map[string]*indexEntry)
	sorted  []string   // v5: always sorted (incremental insert/remove)
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
	fl     indexEntryFreelist // v5: per-index lock-free freelist (never drained by GC)
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
		sb = &shardBucket{
			entries: newSwissTable(256),
			sorted:  make([]string, 0, 256),
		}
		s.buckets[bucket] = sb
	}
	old, exists := sb.entries.put(key, e)
	if !exists {
		// v5: incremental sorted insert via binary search + copy shift.
		pos := sort.SearchStrings(sb.sorted, key)
		sb.sorted = append(sb.sorted, "")
		copy(sb.sorted[pos+1:], sb.sorted[pos:])
		sb.sorted[pos] = key
	}
	s.mu.Unlock()

	if exists {
		idx.fl.release(old)
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
	e, ok := sb.entries.get(key)
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
	old, exists := sb.entries.remove(key)
	if exists {
		// v5: incremental sorted removal via binary search + copy shift.
		pos := sort.SearchStrings(sb.sorted, key)
		if pos < len(sb.sorted) && sb.sorted[pos] == key {
			copy(sb.sorted[pos:], sb.sorted[pos+1:])
			sb.sorted = sb.sorted[:len(sb.sorted)-1]
		}
	}
	s.mu.Unlock()

	if exists {
		idx.fl.release(old)
	}
	return exists
}

// list returns keys matching bucket+prefix using sorted arrays with binary search.
// v5: sorted arrays are maintained incrementally (no dirty/rebuild), so list is purely RLock.
// O(log n + m) per shard for prefix queries.
func (idx *shardedIndex) list(bucket, prefix string) []listResult {
	var results []listResult
	for i := range idx.shards {
		s := &idx.shards[i]

		s.mu.RLock()
		sb := s.buckets[bucket]
		if sb == nil || sb.entries.len() == 0 {
			s.mu.RUnlock()
			continue
		}
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
			if e, ok := sb.entries.get(key); ok {
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
		if e, ok := sb.entries.get(key); ok {
			*results = append(*results, listResult{key: key, entry: e})
		}
	}
}

func (idx *shardedIndex) hasBucket(bucket string) bool {
	for i := range idx.shards {
		s := &idx.shards[i]
		s.mu.RLock()
		sb := s.buckets[bucket]
		has := sb != nil && sb.entries.len() > 0
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
