package usagi

import (
	"hash/fnv"
	"sort"
	"sync"
)

const indexShardCount = 256

type indexShard struct {
	mu    sync.RWMutex
	items map[string]*entry
	keys  []string
}

type shardedIndex struct {
	shards [indexShardCount]indexShard
}

func newShardedIndex() *shardedIndex {
	idx := &shardedIndex{}
	for i := range idx.shards {
		idx.shards[i].items = make(map[string]*entry)
	}
	return idx
}

func (s *shardedIndex) shard(key string) *indexShard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return &s.shards[h.Sum32()%indexShardCount]
}

func (s *shardedIndex) Get(key string) (*entry, bool) {
	sh := s.shard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	v, ok := sh.items[key]
	return v, ok
}

func (s *shardedIndex) Set(key string, e *entry) {
	sh := s.shard(key)
	sh.mu.Lock()
	if _, ok := sh.items[key]; !ok {
		insertKeySorted(&sh.keys, key)
	}
	sh.items[key] = e
	sh.mu.Unlock()
}

func (s *shardedIndex) Delete(key string) {
	sh := s.shard(key)
	sh.mu.Lock()
	delete(sh.items, key)
	removeKeySorted(&sh.keys, key)
	sh.mu.Unlock()
}

func (s *shardedIndex) Len() int {
	total := 0
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		total += len(sh.items)
		sh.mu.RUnlock()
	}
	return total
}

func (s *shardedIndex) Keys(prefix string) []string {
	if prefix == "" {
		return mergeShardKeys(s.shards[:], "")
	}
	return mergeShardKeys(s.shards[:], prefix)
}

func (s *shardedIndex) Snapshot() map[string]*entry {
	out := make(map[string]*entry)
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		for k, v := range sh.items {
			out[k] = v
		}
		sh.mu.RUnlock()
	}
	return out
}

func insertKeySorted(keys *[]string, key string) {
	idx := sort.SearchStrings(*keys, key)
	if idx < len(*keys) && (*keys)[idx] == key {
		return
	}
	*keys = append(*keys, "")
	copy((*keys)[idx+1:], (*keys)[idx:])
	(*keys)[idx] = key
}

func removeKeySorted(keys *[]string, key string) {
	idx := sort.SearchStrings(*keys, key)
	if idx < len(*keys) && (*keys)[idx] == key {
		*keys = append((*keys)[:idx], (*keys)[idx+1:]...)
	}
}

func mergeShardKeys(shards []indexShard, prefix string) []string {
	type shardIter struct {
		keys []string
		idx  int
	}
	iters := make([]shardIter, 0, len(shards))
	for i := range shards {
		sh := &shards[i]
		sh.mu.RLock()
		var slice []string
		if prefix == "" {
			slice = append([]string(nil), sh.keys...)
		} else {
			slice = prefixSlice(sh.keys, prefix)
		}
		sh.mu.RUnlock()
		if len(slice) > 0 {
			iters = append(iters, shardIter{keys: slice, idx: 0})
		}
	}

	if len(iters) == 0 {
		return nil
	}

	h := &keyHeap{}
	for i := range iters {
		heapPush(h, heapItem{key: iters[i].keys[0], shard: i})
	}

	merged := make([]string, 0)
	for h.len() > 0 {
		item := heapPop(h)
		merged = append(merged, item.key)
		iter := &iters[item.shard]
		iter.idx++
		if iter.idx < len(iter.keys) {
			heapPush(h, heapItem{key: iter.keys[iter.idx], shard: item.shard})
		}
	}
	return merged
}

func prefixSlice(keys []string, prefix string) []string {
	start := sort.SearchStrings(keys, prefix)
	endPrefix := nextPrefix(prefix)
	if endPrefix == "" {
		return keys[start:]
	}
	end := sort.SearchStrings(keys, endPrefix)
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	return keys[start:end]
}

func nextPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}
	b := []byte(prefix)
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] < 0xFF {
			b[i]++
			return string(b[:i+1])
		}
	}
	return ""
}

type heapItem struct {
	key   string
	shard int
}

type keyHeap struct {
	items []heapItem
}

func (h *keyHeap) len() int {
	return len(h.items)
}

func heapPush(h *keyHeap, item heapItem) {
	h.items = append(h.items, item)
	up := len(h.items) - 1
	for up > 0 {
		parent := (up - 1) / 2
		if h.items[parent].key <= h.items[up].key {
			break
		}
		h.items[parent], h.items[up] = h.items[up], h.items[parent]
		up = parent
	}
}

func heapPop(h *keyHeap) heapItem {
	item := h.items[0]
	last := h.items[len(h.items)-1]
	h.items = h.items[:len(h.items)-1]
	if len(h.items) == 0 {
		return item
	}
	h.items[0] = last
	down := 0
	for {
		left := 2*down + 1
		right := left + 1
		smallest := down
		if left < len(h.items) && h.items[left].key < h.items[smallest].key {
			smallest = left
		}
		if right < len(h.items) && h.items[right].key < h.items[smallest].key {
			smallest = right
		}
		if smallest == down {
			break
		}
		h.items[down], h.items[smallest] = h.items[smallest], h.items[down]
		down = smallest
	}
	return item
}
