package horse

import (
	"sort"
	"strings"
	"sync"
	"unsafe"
)

const shardCount = 256

// indexEntry stores the location and metadata for a single object.
type indexEntry struct {
	valueOffset int64  // offset in volume file where value bytes start
	size        int64  // value size in bytes
	contentType string
	created     int64 // UnixNano
	updated     int64 // UnixNano
}

// indexEntryPool reduces GC pressure by recycling indexEntry allocations.
var indexEntryPool = sync.Pool{
	New: func() any { return &indexEntry{} },
}

// acquireIndexEntry gets a zeroed indexEntry from the pool.
func acquireIndexEntry() *indexEntry {
	e := indexEntryPool.Get().(*indexEntry)
	*e = indexEntry{}
	return e
}

// releaseIndexEntry returns an indexEntry to the pool.
func releaseIndexEntry(e *indexEntry) {
	if e != nil {
		indexEntryPool.Put(e)
	}
}

// shard is one segment of the sharded hash index.
type shard struct {
	mu      sync.RWMutex
	entries map[string]*indexEntry // "bucket\x00key" → entry
	_       [40]byte              // padding to avoid false sharing (cache line = 64 bytes)
}

// segmentKeys holds keys for one path segment with lazy sorting.
type segmentKeys struct {
	keys   map[string]struct{} // O(1) add/remove
	sorted []string            // lazily built on list
	dirty  bool                // true if sorted is stale
}

// bucketKeySet tracks all keys for a bucket, segmented by first path component.
type bucketKeySet struct {
	mu       sync.RWMutex
	total    int                     // total key count across all segments
	segments map[string]*segmentKeys // first_segment → keys
	noSlash  *segmentKeys            // keys without any "/" go here
}

// shardedIndex is a 256-shard concurrent hash index (KeyDir).
type shardedIndex struct {
	shards  [shardCount]shard
	buckets sync.Map // bucket name → *bucketKeySet
}

func newIndex() *shardedIndex {
	idx := &shardedIndex{}
	for i := range idx.shards {
		idx.shards[i].entries = make(map[string]*indexEntry, 64)
	}
	return idx
}

// shardForParts computes shard index directly from bucket+key without allocating a compositeKey string.
// Equivalent to shardFor(bucket + "\x00" + key) but zero-alloc.
func shardForParts(bucket, key string) uint32 {
	const offset32 = 2166136261
	const prime32 = 16777619
	h := uint32(offset32)
	for i := 0; i < len(bucket); i++ {
		h ^= uint32(bucket[i])
		h *= prime32
	}
	// Hash the null separator.
	h ^= 0
	h *= prime32
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= prime32
	}
	return h % shardCount
}

// compositeKey creates a lookup key from bucket and object key.
// Used only for map operations where a string key is required.
func compositeKey(bucket, key string) string {
	return bucket + "\x00" + key
}

// compositeKeyBuf writes a composite key into buf, returning the result slice.
// Avoids heap allocation when buf is large enough (stack-allocated by caller).
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

// unsafeString creates a string from a byte slice without copying.
// The caller must ensure the byte slice is not modified while the string is in use.
func unsafeString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// firstSegment returns the portion of key before the first '/'.
func firstSegment(key string) string {
	if i := strings.IndexByte(key, '/'); i >= 0 {
		return key[:i]
	}
	return ""
}

// getBucketKeys returns (or creates) the per-bucket key set.
func (idx *shardedIndex) getBucketKeys(bucket string) *bucketKeySet {
	if v, ok := idx.buckets.Load(bucket); ok {
		return v.(*bucketKeySet)
	}
	bk := &bucketKeySet{
		segments: make(map[string]*segmentKeys, 16),
		noSlash:  &segmentKeys{keys: make(map[string]struct{}, 16), dirty: true},
	}
	actual, _ := idx.buckets.LoadOrStore(bucket, bk)
	return actual.(*bucketKeySet)
}

// getSegment returns (or creates) the segment for the given key.
func (bk *bucketKeySet) getSegment(key string) *segmentKeys {
	seg := firstSegment(key)
	if seg == "" {
		return bk.noSlash
	}
	sk, ok := bk.segments[seg]
	if !ok {
		sk = &segmentKeys{keys: make(map[string]struct{}, 64), dirty: true}
		bk.segments[seg] = sk
	}
	return sk
}

func (idx *shardedIndex) put(bucket, key string, e *indexEntry) {
	si := shardForParts(bucket, key)
	s := &idx.shards[si]

	// Build composite key using stack buffer to avoid heap alloc in common case.
	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	s.mu.Lock()
	old, exists := s.entries[ck]
	if !exists {
		// Only allocate the string for the map key on first insert.
		ck = compositeKey(bucket, key)
	}
	s.entries[ck] = e
	s.mu.Unlock()

	if exists {
		releaseIndexEntry(old)
	} else {
		bk := idx.getBucketKeys(bucket)
		bk.mu.Lock()
		sk := bk.getSegment(key)
		sk.keys[key] = struct{}{}
		sk.dirty = true
		bk.total++
		bk.mu.Unlock()
	}
}

func (idx *shardedIndex) get(bucket, key string) (*indexEntry, bool) {
	si := shardForParts(bucket, key)
	s := &idx.shards[si]

	// Use stack buffer for map lookup — no heap alloc.
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
	}
	s.mu.Unlock()

	if exists {
		releaseIndexEntry(old)
		bk := idx.getBucketKeys(bucket)
		bk.mu.Lock()
		sk := bk.getSegment(key)
		delete(sk.keys, key)
		sk.dirty = true
		bk.total--
		bk.mu.Unlock()
	}

	return exists
}

// ensureSorted rebuilds the sorted key list for a segment if dirty.
func (sk *segmentKeys) ensureSorted() {
	if !sk.dirty {
		return
	}
	sk.sorted = make([]string, 0, len(sk.keys))
	for k := range sk.keys {
		sk.sorted = append(sk.sorted, k)
	}
	sort.Strings(sk.sorted)
	sk.dirty = false
}

// list returns all entries matching bucket and prefix, sorted by key.
func (idx *shardedIndex) list(bucket, prefix string) []listResult {
	bk := idx.getBucketKeys(bucket)

	seg := firstSegment(prefix)

	bk.mu.Lock()
	var sk *segmentKeys
	if seg == "" {
		sk = bk.noSlash
	} else {
		sk = bk.segments[seg]
	}
	if sk == nil || len(sk.keys) == 0 {
		bk.mu.Unlock()
		return nil
	}

	sk.ensureSorted()
	sorted := sk.sorted
	bk.mu.Unlock()

	start := sort.SearchStrings(sorted, prefix)

	var results []listResult
	for i := start; i < len(sorted); i++ {
		key := sorted[i]
		if !strings.HasPrefix(key, prefix) {
			break
		}

		si := shardForParts(bucket, key)
		s := &idx.shards[si]

		var buf [256]byte
		ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

		s.mu.RLock()
		e, ok := s.entries[ck]
		s.mu.RUnlock()

		if ok {
			results = append(results, listResult{key: key, entry: e})
		}
	}

	return results
}

// hasBucket returns true if any keys exist for the given bucket.
func (idx *shardedIndex) hasBucket(bucket string) bool {
	v, ok := idx.buckets.Load(bucket)
	if !ok {
		return false
	}
	bk := v.(*bucketKeySet)
	bk.mu.RLock()
	n := bk.total
	bk.mu.RUnlock()
	return n > 0
}

type listResult struct {
	key   string
	entry *indexEntry
}
