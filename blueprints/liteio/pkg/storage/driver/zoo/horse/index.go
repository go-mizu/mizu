package horse

import (
	"sort"
	"strings"
	"sync"
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
}

// segmentKeys holds keys for one path segment with lazy sorting.
type segmentKeys struct {
	keys   map[string]struct{} // O(1) add/remove
	sorted []string            // lazily built on list
	dirty  bool                // true if sorted is stale
}

// bucketKeySet tracks all keys for a bucket, segmented by first path component.
// Keys like "write/123" go into segment "write", "scale-1000/45/001" into "scale-1000".
// List operations only sort within the matching segment, not all keys in the bucket.
type bucketKeySet struct {
	mu       sync.RWMutex
	total    int                       // total key count across all segments
	segments map[string]*segmentKeys   // first_segment → keys
	noSlash  *segmentKeys              // keys without any "/" go here
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

// compositeKey creates a lookup key from bucket and object key.
func compositeKey(bucket, key string) string {
	return bucket + "\x00" + key
}

// shardFor returns the shard index using FNV-1a (inlined for speed).
func shardFor(ck string) uint32 {
	const offset32 = 2166136261
	const prime32 = 16777619
	h := uint32(offset32)
	for i := 0; i < len(ck); i++ {
		h ^= uint32(ck[i])
		h *= prime32
	}
	return h % shardCount
}

// firstSegment returns the portion of key before the first '/'.
// For "write/123" → "write", for "abc" → "".
func firstSegment(key string) string {
	if i := strings.IndexByte(key, '/'); i >= 0 {
		return key[:i]
	}
	return "" // no slash
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
	ck := compositeKey(bucket, key)
	si := shardFor(ck)
	s := &idx.shards[si]

	s.mu.Lock()
	old, exists := s.entries[ck]
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
	ck := compositeKey(bucket, key)
	si := shardFor(ck)
	s := &idx.shards[si]

	s.mu.RLock()
	e, ok := s.entries[ck]
	s.mu.RUnlock()
	return e, ok
}

func (idx *shardedIndex) remove(bucket, key string) bool {
	ck := compositeKey(bucket, key)
	si := shardFor(ck)
	s := &idx.shards[si]

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
// Uses segmented index: only sorts keys within the matching segment.
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

	// Binary search for the first key >= prefix.
	start := sort.SearchStrings(sorted, prefix)

	var results []listResult
	for i := start; i < len(sorted); i++ {
		key := sorted[i]
		if !strings.HasPrefix(key, prefix) {
			break
		}

		ck := compositeKey(bucket, key)
		si := shardFor(ck)
		s := &idx.shards[si]
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
