package usagi

import (
	"sort"
	"sync"
	"sync/atomic"
)

const indexShardCount = 256

type indexShard struct {
	mu    sync.RWMutex
	items map[string]*entry
}

type shardedIndex struct {
	shards       [indexShardCount]indexShard
	cacheMu      sync.Mutex
	cacheKeys    []string
	cacheVersion uint64
	modVersion   uint64
}

func newShardedIndex() *shardedIndex {
	idx := &shardedIndex{}
	for i := range idx.shards {
		idx.shards[i].items = make(map[string]*entry)
	}
	return idx
}

func (s *shardedIndex) shard(key string) *indexShard {
	return &s.shards[fnv32a(key)%indexShardCount]
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
	sh.items[key] = e
	sh.mu.Unlock()
	atomic.AddUint64(&s.modVersion, 1)
}

func (s *shardedIndex) Delete(key string) {
	sh := s.shard(key)
	sh.mu.Lock()
	if _, ok := sh.items[key]; ok {
		delete(sh.items, key)
		atomic.AddUint64(&s.modVersion, 1)
	}
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
	version := atomic.LoadUint64(&s.modVersion)
	s.cacheMu.Lock()
	if s.cacheVersion != version {
		keys := make([]string, 0)
		for i := range s.shards {
			sh := &s.shards[i]
			sh.mu.RLock()
			for k := range sh.items {
				keys = append(keys, k)
			}
			sh.mu.RUnlock()
		}
		sort.Strings(keys)
		s.cacheKeys = keys
		s.cacheVersion = version
	}
	keys := append([]string(nil), s.cacheKeys...)
	s.cacheMu.Unlock()
	if prefix == "" {
		return keys
	}
	return prefixSlice(keys, prefix)
}

// KeysView returns a read-only view of the cached key slice.
// Callers must treat the returned slice as immutable.
func (s *shardedIndex) KeysView(prefix string) []string {
	version := atomic.LoadUint64(&s.modVersion)
	s.cacheMu.Lock()
	if s.cacheVersion != version {
		keys := make([]string, 0)
		for i := range s.shards {
			sh := &s.shards[i]
			sh.mu.RLock()
			for k := range sh.items {
				keys = append(keys, k)
			}
			sh.mu.RUnlock()
		}
		sort.Strings(keys)
		s.cacheKeys = keys
		s.cacheVersion = version
	}
	keys := s.cacheKeys
	s.cacheMu.Unlock()
	if prefix == "" {
		return keys
	}
	return prefixSlice(keys, prefix)
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

const (
	fnv32aOffset = 2166136261
	fnv32aPrime  = 16777619
)

func fnv32a(key string) uint32 {
	var h uint32 = fnv32aOffset
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= fnv32aPrime
	}
	return h
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
