package rabbit

import (
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// SHARDED BUFFER POOLS
// =============================================================================
// Reduces lock contention by sharding pools across CPUs.

type shardedPool struct {
	pools [NumPoolShards]sync.Pool
	size  int
}

func newShardedPool(size int) *shardedPool {
	p := &shardedPool{size: size}
	for i := range p.pools {
		sz := size
		p.pools[i].New = func() interface{} {
			buf := make([]byte, sz)
			return &buf
		}
	}
	return p
}

func (p *shardedPool) Get() []byte {
	shard := fastrand() % NumPoolShards
	return *p.pools[shard].Get().(*[]byte)
}

func (p *shardedPool) Put(buf []byte) {
	if cap(buf) != p.size {
		return
	}
	shard := fastrand() % NumPoolShards
	p.pools[shard].Put(&buf)
}

// Global buffer pools
var (
	tinyPool   = newShardedPool(TinyBuffer)
	smallPool  = newShardedPool(SmallBuffer)
	mediumPool = newShardedPool(MediumBuffer)
	largePool  = newShardedPool(LargeBuffer)
	hugePool   = newShardedPool(HugeBuffer)
)

// getBuffer returns appropriately sized buffer from pool.
func getBuffer(size int64) []byte {
	switch {
	case size <= int64(TinyBuffer):
		return tinyPool.Get()
	case size <= int64(SmallBuffer):
		return smallPool.Get()
	case size <= int64(MediumBuffer):
		return mediumPool.Get()
	case size <= int64(LargeBuffer):
		return largePool.Get()
	default:
		return hugePool.Get()
	}
}

// putBuffer returns buffer to appropriate pool.
func putBuffer(buf []byte) {
	switch cap(buf) {
	case TinyBuffer:
		tinyPool.Put(buf)
	case SmallBuffer:
		smallPool.Put(buf)
	case MediumBuffer:
		mediumPool.Put(buf)
	case LargeBuffer:
		largePool.Put(buf)
	case HugeBuffer:
		hugePool.Put(buf)
	}
}

// =============================================================================
// FAST RANDOM NUMBER GENERATOR
// =============================================================================

var fastrandState atomic.Uint64

func init() {
	fastrandState.Store(uint64(time.Now().UnixNano()))
}

func fastrand() uint32 {
	for {
		old := fastrandState.Load()
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
// FAST TIME CACHE
// =============================================================================
// Reduces time.Now() calls under high concurrency.

var cachedTime atomic.Int64

func init() {
	cachedTime.Store(time.Now().UnixNano())
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		for range ticker.C {
			cachedTime.Store(time.Now().UnixNano())
		}
	}()
}

func fastNow() time.Time {
	return time.Unix(0, cachedTime.Load())
}

// =============================================================================
// FNV-1a HASH
// =============================================================================

func fnv1a(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
