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
	// Don't pool inline entries — their []byte would need special handling.
}

type indexShard struct {
	mu      sync.RWMutex
	entries map[string]*indexEntry // "bucket\x00key" → entry
	_       [40]byte              // cache-line padding
}

// index is the per-stripe sharded hash index.
// No bucketKeySet — listing scans shards on demand (saves ~100ns/write).
type index struct {
	shards [shardCount]indexShard
}

func newIndex() *index {
	idx := &index{}
	for i := range idx.shards {
		idx.shards[i].entries = make(map[string]*indexEntry, 64)
	}
	return idx
}

// shardForParts computes shard index from bucket+key without allocation (FNV-1a).
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

func (idx *index) put(bucket, key string, e *indexEntry) {
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
		releaseEntry(old)
	}
}

func (idx *index) get(bucket, key string) (*indexEntry, bool) {
	si := shardForParts(bucket, key)
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
		releaseEntry(old)
	}
	return exists
}

// list scans all shards for entries matching bucket+prefix, sorted by key.
// No pre-built key sets — O(entries_in_stripe) scan, fast for typical workloads.
func (idx *index) list(bucket, prefix string) []listResult {
	ckPrefix := bucket + "\x00" + prefix
	bucketPrefixLen := len(bucket) + 1

	var results []listResult
	for i := range idx.shards {
		s := &idx.shards[i]
		s.mu.RLock()
		for k, e := range s.entries {
			if strings.HasPrefix(k, ckPrefix) {
				key := k[bucketPrefixLen:]
				results = append(results, listResult{key: key, entry: e})
			}
		}
		s.mu.RUnlock()
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].key < results[j].key
	})

	return results
}

// hasBucket scans shards to check if any keys exist for the bucket.
func (idx *index) hasBucket(bucket string) bool {
	prefix := bucket + "\x00"
	for i := range idx.shards {
		s := &idx.shards[i]
		s.mu.RLock()
		for k := range s.entries {
			if strings.HasPrefix(k, prefix) {
				s.mu.RUnlock()
				return true
			}
		}
		s.mu.RUnlock()
	}
	return false
}

// bucketNames returns all unique bucket names across all shards.
func (idx *index) bucketNames() []string {
	seen := make(map[string]struct{})
	for i := range idx.shards {
		s := &idx.shards[i]
		s.mu.RLock()
		for k := range s.entries {
			if sep := strings.IndexByte(k, 0); sep >= 0 {
				seen[k[:sep]] = struct{}{}
			}
		}
		s.mu.RUnlock()
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

type listResult struct {
	key   string
	entry *indexEntry
}
