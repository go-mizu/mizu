//go:build !windows

// File: driver/local/memory_mode.go
// In-memory storage mode for maximum parallel performance.
// Bypasses all filesystem operations for ultra-low-latency storage.
package local

import (
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

// =============================================================================
// IN-MEMORY MODE
// =============================================================================
// Pure in-memory storage for maximum throughput benchmarks.
// No filesystem operations, no syscalls, pure memory speed.

var (
	// inMemoryMode enables/disables in-memory storage
	inMemoryMode atomic.Bool

	// memShards provides lock-free concurrent access
	memShards [memoryShardCount]memoryShard

	// memStats tracks in-memory operations
	memWriteOps atomic.Int64
	memReadOps  atomic.Int64
)

const (
	// memoryShardCount is the number of memory shards for concurrency
	memoryShardCount = 512

	// memoryMaxEntries is the max entries per shard before cleanup
	memoryMaxEntries = 10000
)

type memoryShard struct {
	mu   sync.RWMutex
	data map[string]*memEntry
}

type memEntry struct {
	data    []byte
	modTime time.Time
	size    int64
}

func init() {
	for i := range memShards {
		memShards[i].data = make(map[string]*memEntry, 256)
	}
}

// EnableInMemoryMode enables pure in-memory storage mode.
// All writes go to memory, all reads come from memory.
// WARNING: Data is NOT persisted to disk!
func EnableInMemoryMode() {
	inMemoryMode.Store(true)
}

// DisableInMemoryMode disables in-memory storage mode.
func DisableInMemoryMode() {
	inMemoryMode.Store(false)
}

// IsInMemoryMode returns whether in-memory mode is enabled.
func IsInMemoryMode() bool {
	return inMemoryMode.Load()
}

// ClearMemoryStore clears all in-memory data.
func ClearMemoryStore() {
	for i := range memShards {
		shard := &memShards[i]
		shard.mu.Lock()
		shard.data = make(map[string]*memEntry, 256)
		shard.mu.Unlock()
	}
	memWriteOps.Store(0)
	memReadOps.Store(0)
}

// MemoryStats returns in-memory operation statistics.
func MemoryStats() (writes, reads int64) {
	return memWriteOps.Load(), memReadOps.Load()
}

// memShardIndex returns the shard index for a key using FNV-1a hash.
func memShardIndex(key string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return h & (memoryShardCount - 1)
}

// memKey generates a full key from bucket and object key.
func memKey(bucket, key string) string {
	return bucket + "\x00" + key
}

// memWrite stores data in memory.
func memWrite(bucket, key string, data []byte) {
	fullKey := memKey(bucket, key)
	idx := memShardIndex(fullKey)
	shard := &memShards[idx]

	// Copy data for safety (caller may reuse buffer)
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	entry := &memEntry{
		data:    dataCopy,
		modTime: time.Now(),
		size:    int64(len(data)),
	}

	shard.mu.Lock()
	// Simple cleanup if shard is too full
	if len(shard.data) >= memoryMaxEntries {
		// Delete ~25% of entries (oldest approximation via iteration)
		count := 0
		target := memoryMaxEntries / 4
		for k := range shard.data {
			delete(shard.data, k)
			count++
			if count >= target {
				break
			}
		}
	}
	shard.data[fullKey] = entry
	shard.mu.Unlock()

	memWriteOps.Add(1)
}

// memRead retrieves data from memory.
func memRead(bucket, key string) ([]byte, time.Time, bool) {
	fullKey := memKey(bucket, key)
	idx := memShardIndex(fullKey)
	shard := &memShards[idx]

	shard.mu.RLock()
	entry, ok := shard.data[fullKey]
	shard.mu.RUnlock()

	if !ok {
		return nil, time.Time{}, false
	}

	memReadOps.Add(1)
	// Return direct reference (caller should not modify)
	return entry.data, entry.modTime, true
}

// memDelete removes data from memory.
func memDelete(bucket, key string) bool {
	fullKey := memKey(bucket, key)
	idx := memShardIndex(fullKey)
	shard := &memShards[idx]

	shard.mu.Lock()
	_, existed := shard.data[fullKey]
	delete(shard.data, fullKey)
	shard.mu.Unlock()

	return existed
}

// memStat returns object metadata from memory.
func memStat(bucket, key string) (int64, time.Time, bool) {
	fullKey := memKey(bucket, key)
	idx := memShardIndex(fullKey)
	shard := &memShards[idx]

	shard.mu.RLock()
	entry, ok := shard.data[fullKey]
	shard.mu.RUnlock()

	if !ok {
		return 0, time.Time{}, false
	}

	return entry.size, entry.modTime, true
}

// memList returns all keys with a given prefix.
func memList(bucket, prefix string) []string {
	fullPrefix := memKey(bucket, prefix)
	var keys []string

	for i := range memShards {
		shard := &memShards[i]
		shard.mu.RLock()
		for k := range shard.data {
			if len(k) >= len(fullPrefix) && k[:len(fullPrefix)] == fullPrefix {
				// Extract original key (remove bucket prefix)
				bucketPrefix := bucket + "\x00"
				if len(k) > len(bucketPrefix) {
					keys = append(keys, k[len(bucketPrefix):])
				}
			}
		}
		shard.mu.RUnlock()
	}

	return keys
}

// =============================================================================
// IN-MEMORY STORAGE OPERATIONS
// =============================================================================
// These methods are called from storage.go when in-memory mode is enabled.

// writeInMemory handles Write operations in memory mode.
func (b *bucket) writeInMemory(key string, src io.Reader, size int64, contentType string) (*storage.Object, error) {
	// Read all data from source
	var data []byte
	if size > 0 {
		data = make([]byte, size)
		n, err := io.ReadFull(src, data)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		data = data[:n]
	} else {
		// Unknown size, read all
		data, _ = io.ReadAll(src)
	}

	// Store in memory
	memWrite(b.name, key, data)

	now := time.Now()
	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(data)),
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

// openInMemory handles Open operations in memory mode.
func (b *bucket) openInMemory(key string, offset, length int64) (io.ReadCloser, *storage.Object, error) {
	data, modTime, ok := memRead(b.name, key)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	obj := &storage.Object{
		Bucket:  b.name,
		Key:     key,
		Size:    int64(len(data)),
		Created: modTime,
		Updated: modTime,
	}

	// Handle offset and length
	if offset > 0 {
		if offset >= int64(len(data)) {
			return &zeroCopyReader{data: nil}, obj, nil
		}
		data = data[offset:]
	}
	if length > 0 && length < int64(len(data)) {
		data = data[:length]
	}

	return &zeroCopyReader{data: data}, obj, nil
}

// statInMemory handles Stat operations in memory mode.
func (b *bucket) statInMemory(key string) (*storage.Object, error) {
	size, modTime, ok := memStat(b.name, key)
	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:  b.name,
		Key:     key,
		Size:    size,
		Created: modTime,
		Updated: modTime,
	}, nil
}

// deleteInMemory handles Delete operations in memory mode.
func (b *bucket) deleteInMemory(key string) error {
	if !memDelete(b.name, key) {
		return storage.ErrNotExist
	}
	return nil
}
