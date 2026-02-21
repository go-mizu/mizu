package kestrel

import (
	"sort"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Per-bucket key index (for List operations)
// ---------------------------------------------------------------------------

func firstSegment(key string) string {
	if i := strings.IndexByte(key, '/'); i >= 0 {
		return key[:i]
	}
	return ""
}

type segmentKeys struct {
	keys   map[string]struct{}
	sorted []string
	dirty  bool
}

func (sk *segmentKeys) ensureSorted() {
	if !sk.dirty {
		return
	}
	sk.sorted = sk.sorted[:0]
	for k := range sk.keys {
		sk.sorted = append(sk.sorted, k)
	}
	sort.Strings(sk.sorted)
	sk.dirty = false
}

type bucketKeySet struct {
	mu       sync.RWMutex
	total    int
	segments map[string]*segmentKeys
	noSlash  *segmentKeys
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

type keyIndex struct {
	buckets sync.Map
}

func (ki *keyIndex) getBucketKeys(bucket string) *bucketKeySet {
	if v, ok := ki.buckets.Load(bucket); ok {
		return v.(*bucketKeySet)
	}
	bk := &bucketKeySet{
		segments: make(map[string]*segmentKeys, 16),
		noSlash:  &segmentKeys{keys: make(map[string]struct{}, 16), dirty: true},
	}
	actual, _ := ki.buckets.LoadOrStore(bucket, bk)
	return actual.(*bucketKeySet)
}

func (ki *keyIndex) add(bucket, key string) {
	bk := ki.getBucketKeys(bucket)
	bk.mu.RLock()
	sk := bk.getSegment(key)
	if _, exists := sk.keys[key]; exists {
		bk.mu.RUnlock()
		return
	}
	bk.mu.RUnlock()

	bk.mu.Lock()
	sk = bk.getSegment(key)
	if _, exists := sk.keys[key]; !exists {
		sk.keys[key] = struct{}{}
		sk.dirty = true
		bk.total++
	}
	bk.mu.Unlock()
}

func (ki *keyIndex) remove(bucket, key string) {
	bk := ki.getBucketKeys(bucket)
	bk.mu.Lock()
	sk := bk.getSegment(key)
	if _, exists := sk.keys[key]; exists {
		delete(sk.keys, key)
		sk.dirty = true
		bk.total--
	}
	bk.mu.Unlock()
}

func (ki *keyIndex) list(bucket, prefix string) []string {
	bk := ki.getBucketKeys(bucket)
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
	var results []string
	for i := start; i < len(sorted); i++ {
		key := sorted[i]
		if !strings.HasPrefix(key, prefix) {
			break
		}
		results = append(results, key)
	}
	return results
}

func (ki *keyIndex) hasBucket(bucket string) bool {
	v, ok := ki.buckets.Load(bucket)
	if !ok {
		return false
	}
	bk := v.(*bucketKeySet)
	bk.mu.RLock()
	n := bk.total
	bk.mu.RUnlock()
	return n > 0
}

func (ki *keyIndex) removeAllForBucket(bucket string) {
	ki.buckets.Delete(bucket)
}
