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
	contentType string
	created     int64 // UnixNano
	updated     int64 // UnixNano
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

// shard is one segment of the sharded hash index.
type shard struct {
	mu      sync.RWMutex
	entries map[string]*indexEntry // "bucket\x00key" → entry
	_       [40]byte              // padding to avoid false sharing
}

// segmentKeys holds keys for one path segment with lazy sorting.
type segmentKeys struct {
	keys   map[string]struct{}
	sorted []string
	dirty  bool
}

// bucketKeySet tracks all keys for a bucket, segmented by first path component.
type bucketKeySet struct {
	mu       sync.RWMutex
	total    int
	segments map[string]*segmentKeys
	noSlash  *segmentKeys
}

// shardedIndex is a 256-shard concurrent hash index.
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

func firstSegment(key string) string {
	if i := strings.IndexByte(key, '/'); i >= 0 {
		return key[:i]
	}
	return ""
}

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

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	s.mu.Lock()
	old, exists := s.entries[ck]
	if !exists {
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

func (sk *segmentKeys) ensureSorted() {
	if !sk.dirty {
		return
	}
	sk.sorted = sk.sorted[:0] // reuse underlying array
	for k := range sk.keys {
		sk.sorted = append(sk.sorted, k)
	}
	sort.Strings(sk.sorted)
	sk.dirty = false
}

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
