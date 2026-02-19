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

// bucketKeyList tracks all keys for a bucket.
// Uses a pending buffer to avoid re-sorting the entire key set on every mutation.
// After a full sort (ensureSorted), new additions go into pending.
// List operations use binary search on sorted + linear scan of pending,
// avoiding catastrophic re-sorts when only a few keys were added.
type bucketKeyList struct {
	keys    map[string]struct{}
	sorted  []string // last fully sorted snapshot
	pending []string // keys added since last ensureSorted
}

func (bkl *bucketKeyList) add(key string) {
	if _, exists := bkl.keys[key]; !exists {
		bkl.keys[key] = struct{}{}
		bkl.pending = append(bkl.pending, key)
	}
}

func (bkl *bucketKeyList) remove(key string) {
	delete(bkl.keys, key)
	// sorted may have stale entry; filtered during shard lookup.
}

// ensureSorted rebuilds the full sorted list from the keys map and clears pending.
func (bkl *bucketKeyList) ensureSorted() []string {
	if len(bkl.pending) == 0 && len(bkl.sorted) == len(bkl.keys) {
		return bkl.sorted
	}
	bkl.sorted = bkl.sorted[:0]
	for k := range bkl.keys {
		bkl.sorted = append(bkl.sorted, k)
	}
	sort.Strings(bkl.sorted)
	bkl.pending = bkl.pending[:0]
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
// Three-tier strategy to avoid catastrophic re-sorts:
//  1. No pending keys: binary search the sorted snapshot (fastest).
//  2. Small pending (< 10K): binary search sorted + scan pending (fast).
//  3. Large pending: full rebuild via ensureSorted (one-time cost).
//
// After a full rebuild, pending is cleared. Subsequent small additions
// (e.g., Scale/Write adding 10-1000 keys) stay in the fast pending path.
func (idx *index) list(bucket, prefix string) []listResult {
	idx.keysMu.Lock()
	bkl := idx.bucketKeys[bucket]
	if bkl == nil {
		idx.keysMu.Unlock()
		return nil
	}

	var matching []string
	npend := len(bkl.pending)

	switch {
	case npend == 0 && len(bkl.sorted) == len(bkl.keys):
		// Fast path: sorted snapshot is up to date.
		matching = listBinarySearch(bkl.sorted, prefix)

	case npend > 0 && npend < 10000:
		// Medium path: binary search sorted + scan small pending.
		matching = listBinarySearch(bkl.sorted, prefix)
		for _, k := range bkl.pending {
			if prefix == "" || strings.HasPrefix(k, prefix) {
				matching = append(matching, k)
			}
		}
		if len(matching) > 1 {
			sort.Strings(matching)
			// Deduplicate (handles re-add after delete edge case).
			j := 0
			for i := range matching {
				if i == 0 || matching[i] != matching[i-1] {
					matching[j] = matching[i]
					j++
				}
			}
			matching = matching[:j]
		}

	default:
		// Slow path: full rebuild (first call after many writes).
		sorted := bkl.ensureSorted()
		matching = listBinarySearch(sorted, prefix)
	}
	idx.keysMu.Unlock()

	if len(matching) == 0 {
		return nil
	}

	// Look up entries for matching keys.
	// Shard lookup also filters stale entries (deleted keys still in sorted).
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

	return results
}

// listBinarySearch finds all keys in sorted that match the given prefix.
func listBinarySearch(sorted []string, prefix string) []string {
	if len(sorted) == 0 {
		return nil
	}
	start := sort.SearchStrings(sorted, prefix)
	var matching []string
	for i := start; i < len(sorted); i++ {
		if prefix != "" && !strings.HasPrefix(sorted[i], prefix) {
			break
		}
		matching = append(matching, sorted[i])
	}
	return matching
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
