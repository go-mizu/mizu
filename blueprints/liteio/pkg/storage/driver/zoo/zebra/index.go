package zebra

import (
	"sort"
	"strings"
	"sync"
	"unsafe"
)

const shardCount = 64

// indexEntry stores the location and metadata for a single object.
type indexEntry struct {
	valueOffset int64
	size        int64
	contentType string
	created     int64 // UnixNano
	updated     int64 // UnixNano
	inline      []byte // non-nil for inline values (≤ inlineMax)
}

// entryPool reduces GC pressure for non-inline entries.
var entryPool = sync.Pool{
	New: func() any { return &indexEntry{} },
}

func acquireEntry() *indexEntry {
	e := entryPool.Get().(*indexEntry)
	*e = indexEntry{}
	return e
}

func releaseEntry(e *indexEntry) {
	if e != nil && e.inline == nil {
		entryPool.Put(e)
	}
}

type indexShard struct {
	mu      sync.RWMutex
	entries map[string]*indexEntry // "bucket\x00key" → entry
	_       [40]byte              // cache-line padding
}

// bucketKeyList tracks all keys for a bucket with lazy sorting.
type bucketKeyList struct {
	keys   map[string]struct{}
	sorted []string
	dirty  bool
}

func (bkl *bucketKeyList) add(key string) {
	if _, exists := bkl.keys[key]; !exists {
		bkl.keys[key] = struct{}{}
		bkl.dirty = true
	}
}

func (bkl *bucketKeyList) remove(key string) {
	if _, exists := bkl.keys[key]; exists {
		delete(bkl.keys, key)
		bkl.dirty = true
	}
}

func (bkl *bucketKeyList) ensureSorted() []string {
	if !bkl.dirty {
		return bkl.sorted
	}
	bkl.sorted = bkl.sorted[:0]
	for k := range bkl.keys {
		bkl.sorted = append(bkl.sorted, k)
	}
	sort.Strings(bkl.sorted)
	bkl.dirty = false
	return bkl.sorted
}

// index is the per-stripe sharded hash index.
// Stripe-level bucket key tracking enables fast list.
type index struct {
	shards [shardCount]indexShard

	// Stripe-level bucket key tracking for efficient list operations.
	// Separate from shard locks to avoid holding shard locks during sort.
	keysMu     sync.RWMutex
	bucketKeys map[string]*bucketKeyList // bucket → sorted keys
}

func newIndex() *index {
	idx := &index{
		bucketKeys: make(map[string]*bucketKeyList, 4),
	}
	for i := range idx.shards {
		idx.shards[i].entries = make(map[string]*indexEntry, 64)
	}
	return idx
}

// shardForParts computes shard index using high 32 bits of 64-bit FNV-1a.
// Must match getH/putH which use (h>>32) % shardCount.
func shardForParts(bucket, key string) uint32 {
	h := fnvHash(bucket, key)
	return uint32(h>>32) % shardCount
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

func (idx *index) put(bucket, key string, e *indexEntry) {
	si := shardForParts(bucket, key)
	idx.putToShard(si, bucket, key, e)
}

// putH uses a pre-computed 64-bit hash (high 32 bits) for shard selection.
func (idx *index) putH(h uint64, bucket, key string, e *indexEntry) {
	si := uint32(h>>32) % shardCount
	idx.putToShard(si, bucket, key, e)
}

func (idx *index) putToShard(si uint32, bucket, key string, e *indexEntry) {
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

	// Track key in bucket key list (separate lock from shard).
	if !exists {
		idx.keysMu.Lock()
		bkl := idx.bucketKeys[bucket]
		if bkl == nil {
			bkl = &bucketKeyList{keys: make(map[string]struct{}, 64)}
			idx.bucketKeys[bucket] = bkl
		}
		bkl.add(key)
		idx.keysMu.Unlock()
	}

	if exists {
		releaseEntry(old)
	}
}

func (idx *index) get(bucket, key string) (*indexEntry, bool) {
	si := shardForParts(bucket, key)
	return idx.getFromShard(si, bucket, key)
}

// getH uses a pre-computed 64-bit hash (high 32 bits) for shard selection.
func (idx *index) getH(h uint64, bucket, key string) (*indexEntry, bool) {
	si := uint32(h>>32) % shardCount
	return idx.getFromShard(si, bucket, key)
}

func (idx *index) getFromShard(si uint32, bucket, key string) (*indexEntry, bool) {
	s := &idx.shards[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	s.mu.RLock()
	e, ok := s.entries[ck]
	s.mu.RUnlock()
	return e, ok
}

func (idx *index) remove(bucket, key string) bool {
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
		// Remove from bucket key list.
		idx.keysMu.Lock()
		if bkl := idx.bucketKeys[bucket]; bkl != nil {
			bkl.remove(key)
		}
		idx.keysMu.Unlock()
		releaseEntry(old)
	}
	return exists
}

// list returns entries matching bucket+prefix, sorted by key.
// Uses stripe-level sorted key list with binary search for efficient prefix matching.
// The sorted list is built lazily on first call and reused until new keys are added.
func (idx *index) list(bucket, prefix string) []listResult {
	idx.keysMu.Lock()
	bkl := idx.bucketKeys[bucket]
	if bkl == nil {
		idx.keysMu.Unlock()
		return nil
	}

	sorted := bkl.ensureSorted()
	// Binary search for prefix start.
	start := sort.SearchStrings(sorted, prefix)

	var matching []string
	for i := start; i < len(sorted); i++ {
		if prefix != "" && !strings.HasPrefix(sorted[i], prefix) {
			break
		}
		matching = append(matching, sorted[i])
	}
	idx.keysMu.Unlock()

	if len(matching) == 0 {
		return nil
	}

	// Look up entries for matching keys.
	results := make([]listResult, 0, len(matching))
	for _, key := range matching {
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

	// Already sorted from binary search.
	return results
}

// hasBucket checks if any keys exist for the bucket.
func (idx *index) hasBucket(bucket string) bool {
	idx.keysMu.RLock()
	bkl := idx.bucketKeys[bucket]
	n := 0
	if bkl != nil {
		n = len(bkl.keys)
	}
	idx.keysMu.RUnlock()
	return n > 0
}

type listResult struct {
	key   string
	entry *indexEntry
}
