package herd

import (
	"sort"
	"strings"
	"sync"
	"unsafe"
)

const shardCount = 256

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

// shardBucketKeys tracks keys for a single bucket within a single shard.
// The sorted array is rebuilt lazily on list() after writes dirty it.
type shardBucketKeys struct {
	keys   map[string]struct{}
	sorted []string
	dirty  bool
}

// shard is one segment of the sharded hash index.
// Key tracking is per-shard (under the same shard lock) — zero extra contention
// compared to the old per-bucket lock that caused C100 timeouts.
type shard struct {
	mu         sync.RWMutex
	entries    map[string]*indexEntry                // "bucket\x00key" → entry
	bucketKeys map[string]*shardBucketKeys           // bucket → key set for list
	_          [40]byte                              // padding to avoid false sharing
}

// shardedIndex is a 256-shard concurrent hash index.
type shardedIndex struct {
	shards [shardCount]shard
}

func newIndex() *shardedIndex {
	idx := &shardedIndex{}
	for i := range idx.shards {
		idx.shards[i].entries = make(map[string]*indexEntry, 64)
		idx.shards[i].bucketKeys = make(map[string]*shardBucketKeys, 4)
	}
	return idx
}

// shardForParts computes shard index from bucket+key without allocation.
func shardForParts(bucket, key string) uint32 {
	const offset32 = 2166136261
	const prime32 = 16777619
	h := uint32(offset32)
	for i := 0; i < len(bucket); i++ {
		h ^= uint32(bucket[i])
		h *= prime32
	}
	h ^= 0
	h *= prime32
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= prime32
	}
	return h % shardCount
}

func compositeKey(bucket, key string) string {
	return bucket + "\x00" + key
}

func compositeKeyBuf(buf []byte, bucket, key string) []byte {
	n := len(bucket) + 1 + len(key)
	if cap(buf) >= n {
		buf = buf[:n]
	} else {
		buf = make([]byte, n)
	}
	copy(buf, bucket)
	buf[len(bucket)] = 0
	copy(buf[len(bucket)+1:], key)
	return buf
}

func unsafeString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func (idx *shardedIndex) put(bucket, key string, e *indexEntry) {
	si := shardForParts(bucket, key)
	s := &idx.shards[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	s.mu.Lock()
	old, exists := s.entries[ck]
	if !exists {
		ck = compositeKey(bucket, key)
		// Track key for list — under same shard lock, zero extra contention.
		sbk := s.bucketKeys[bucket]
		if sbk == nil {
			sbk = &shardBucketKeys{keys: make(map[string]struct{}, 64)}
			s.bucketKeys[bucket] = sbk
		}
		sbk.keys[key] = struct{}{}
		sbk.dirty = true
	}
	s.entries[ck] = e
	s.mu.Unlock()

	if exists {
		releaseIndexEntry(old)
	}
}

func (idx *shardedIndex) get(bucket, key string) (*indexEntry, bool) {
	si := shardForParts(bucket, key)
	s := &idx.shards[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	s.mu.RLock()
	e, ok := s.entries[ck]
	s.mu.RUnlock()
	return e, ok
}

func (idx *shardedIndex) remove(bucket, key string) bool {
	si := shardForParts(bucket, key)
	s := &idx.shards[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	s.mu.Lock()
	old, exists := s.entries[ck]
	if exists {
		delete(s.entries, ck)
		if sbk := s.bucketKeys[bucket]; sbk != nil {
			delete(sbk.keys, key)
			sbk.dirty = true
		}
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
		sbk := s.bucketKeys[bucket]
		if sbk == nil || len(sbk.keys) == 0 {
			s.mu.RUnlock()
			continue
		}

		if sbk.dirty {
			// Need to rebuild sorted list. Upgrade to write lock.
			s.mu.RUnlock()
			s.mu.Lock()
			// Double-check after acquiring write lock.
			sbk = s.bucketKeys[bucket]
			if sbk != nil && sbk.dirty {
				sbk.sorted = sbk.sorted[:0]
				for k := range sbk.keys {
					sbk.sorted = append(sbk.sorted, k)
				}
				sort.Strings(sbk.sorted)
				sbk.dirty = false
			}
			// Downgrade: collect results under write lock (safe, just slower).
			if sbk != nil && len(sbk.sorted) > 0 {
				idx.collectResults(s, bucket, prefix, sbk.sorted, &results)
			}
			s.mu.Unlock()
			continue
		}

		// Fast path: sorted cache is valid.
		idx.collectResults(s, bucket, prefix, sbk.sorted, &results)
		s.mu.RUnlock()
	}
	return results
}

// collectResults appends matching list results from a sorted key slice.
// Uses binary search for O(log n + m) prefix matching.
func (idx *shardedIndex) collectResults(s *shard, bucket, prefix string, sorted []string, results *[]listResult) {
	if prefix == "" {
		for _, key := range sorted {
			var buf [256]byte
			ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))
			if e, ok := s.entries[ck]; ok {
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
		var buf [256]byte
		ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))
		if e, ok := s.entries[ck]; ok {
			*results = append(*results, listResult{key: key, entry: e})
		}
	}
}

func (idx *shardedIndex) hasBucket(bucket string) bool {
	for i := range idx.shards {
		s := &idx.shards[i]
		s.mu.RLock()
		sbk := s.bucketKeys[bucket]
		has := sbk != nil && len(sbk.keys) > 0
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
