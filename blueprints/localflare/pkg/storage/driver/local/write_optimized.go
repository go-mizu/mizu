//go:build !windows

// File: driver/local/write_optimized.go
package local

import (
	"hash/fnv"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// SHARDED BUFFER POOLS
// =============================================================================
// Reduces lock contention by sharding pools across CPUs.

const numPoolShards = 32

type shardedBufferPool struct {
	pools [numPoolShards]sync.Pool
	size  int
}

func newShardedPool(size int) *shardedBufferPool {
	p := &shardedBufferPool{size: size}
	for i := range p.pools {
		sz := size
		p.pools[i].New = func() interface{} {
			buf := make([]byte, sz)
			return &buf
		}
	}
	return p
}

func (p *shardedBufferPool) Get() []byte {
	// Use fast CPU-local shard selection
	shard := fastrand() % numPoolShards
	return *p.pools[shard].Get().(*[]byte)
}

func (p *shardedBufferPool) Put(buf []byte) {
	if cap(buf) != p.size {
		return
	}
	shard := fastrand() % numPoolShards
	p.pools[shard].Put(&buf)
}

// Optimized buffer pools with sharding
var (
	shardedSmallPool  = newShardedPool(SmallBufferSize)
	shardedMediumPool = newShardedPool(MediumBufferSize)
	shardedLargePool  = newShardedPool(LargeBufferSize)
	shardedHugePool   = newShardedPool(HugeBufferSize)
)

// getShardedBuffer returns a buffer from sharded pool based on size.
func getShardedBuffer(size int64) []byte {
	switch {
	case size <= TinyFileThreshold:
		return shardedSmallPool.Get()
	case size <= SmallFileThreshold:
		return shardedMediumPool.Get()
	case size <= LargeFileThreshold:
		return shardedLargePool.Get()
	default:
		return shardedHugePool.Get()
	}
}

// putShardedBuffer returns a buffer to the appropriate sharded pool.
func putShardedBuffer(buf []byte) {
	switch cap(buf) {
	case SmallBufferSize:
		shardedSmallPool.Put(buf)
	case MediumBufferSize:
		shardedMediumPool.Put(buf)
	case LargeBufferSize:
		shardedLargePool.Put(buf)
	case HugeBufferSize:
		shardedHugePool.Put(buf)
	}
}

// =============================================================================
// LOCK-FREE DIRECTORY CACHE
// =============================================================================

const numDirCacheShards = 256

type lockFreeDirCache struct {
	shards   [numDirCacheShards]dirCacheShard
	hits     atomic.Int64
	misses   atomic.Int64
	maxItems int
}

type dirCacheShard struct {
	mu      sync.RWMutex
	entries map[string]time.Time
}

var optimizedDirCache = &lockFreeDirCache{
	maxItems: DirCacheMaxSize / numDirCacheShards,
}

func init() {
	for i := range optimizedDirCache.shards {
		optimizedDirCache.shards[i].entries = make(map[string]time.Time, 64)
	}
}

func (c *lockFreeDirCache) shardIndex(path string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(path))
	return h.Sum32() % numDirCacheShards
}

func (c *lockFreeDirCache) ensureDir(path string) error {
	shard := &c.shards[c.shardIndex(path)]

	// Fast path: check cache with read lock only
	shard.mu.RLock()
	if t, ok := shard.entries[path]; ok && time.Since(t) < DirCacheTTL {
		shard.mu.RUnlock()
		c.hits.Add(1)
		return nil
	}
	shard.mu.RUnlock()
	c.misses.Add(1)

	// Slow path: create directory
	if err := os.MkdirAll(path, DirPermissions); err != nil {
		return err
	}

	// Update cache
	shard.mu.Lock()
	// Evict if too full (simple random eviction)
	if len(shard.entries) >= c.maxItems {
		for k := range shard.entries {
			delete(shard.entries, k)
			if len(shard.entries) < c.maxItems/2 {
				break
			}
		}
	}
	shard.entries[path] = time.Now()
	shard.mu.Unlock()

	return nil
}

func (c *lockFreeDirCache) invalidate(path string) {
	shard := &c.shards[c.shardIndex(path)]
	shard.mu.Lock()
	delete(shard.entries, path)
	shard.mu.Unlock()
}

// =============================================================================
// PER-CPU TEMP DIRECTORIES
// =============================================================================
// Reduces inode contention during parallel writes.

var (
	tempDirSetup sync.Once
	tempDirs     []string
)

func setupTempDirs(baseDir string) {
	tempDirSetup.Do(func() {
		numCPU := runtime.NumCPU()
		tempDirs = make([]string, numCPU)
		for i := 0; i < numCPU; i++ {
			dir := filepath.Join(baseDir, ".tmp", "cpu"+string(rune('0'+i%10)))
			os.MkdirAll(dir, DirPermissions)
			tempDirs[i] = dir
		}
	})
}

// getTempDirForCPU returns a temp directory for the current CPU.
func getTempDirForCPU(baseDir string) string {
	setupTempDirs(baseDir)
	if len(tempDirs) == 0 {
		return baseDir
	}
	// Use fastrand for CPU-local selection without actual pinning
	idx := fastrand() % uint32(len(tempDirs))
	return tempDirs[idx]
}

// =============================================================================
// FAST RANDOM NUMBER GENERATOR
// =============================================================================
// Used for shard selection without lock contention.
// Uses atomic counter + xorshift for fast pseudo-random distribution.

var fastrandState atomic.Uint64

func init() {
	// Seed with current time
	fastrandState.Store(uint64(time.Now().UnixNano()))
}

func fastrand() uint32 {
	// Simple xorshift* for fast pseudo-random numbers
	for {
		old := fastrandState.Load()
		// xorshift*
		x := old
		x ^= x >> 12
		x ^= x << 25
		x ^= x >> 27
		if fastrandState.CompareAndSwap(old, x) {
			return uint32(x * 0x2545F4914F6CDD1D >> 32)
		}
	}
}

// =============================================================================
// WRITE TRACKING (Eliminate post-write stat)
// =============================================================================

type writeResult struct {
	written int64
	modTime time.Time
}

// trackingWriter wraps a writer and tracks bytes written.
type trackingWriter struct {
	w       *os.File
	written int64
}

func (t *trackingWriter) Write(p []byte) (int, error) {
	n, err := t.w.Write(p)
	t.written += int64(n)
	return n, err
}

func (t *trackingWriter) Close() error {
	return t.w.Close()
}

func (t *trackingWriter) Sync() error {
	return t.w.Sync()
}

func (t *trackingWriter) Name() string {
	return t.w.Name()
}

// =============================================================================
// OBJECT POOL FOR RESPONSE OBJECTS
// =============================================================================
// Reduces allocations in hot paths.

var objectPool = sync.Pool{
	New: func() interface{} {
		return &objectPoolEntry{}
	},
}

type objectPoolEntry struct {
	bucket      string
	key         string
	size        int64
	contentType string
	created     time.Time
	updated     time.Time
}

func (e *objectPoolEntry) reset() {
	e.bucket = ""
	e.key = ""
	e.size = 0
	e.contentType = ""
	e.created = time.Time{}
	e.updated = time.Time{}
}
