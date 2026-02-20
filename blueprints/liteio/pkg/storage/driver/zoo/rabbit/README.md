# Rabbit - Cache-First Filesystem Storage

High-performance in-process filesystem driver with L1/L2 tiered caching. Targets sub-millisecond read latency for small-to-medium objects by keeping frequently accessed data in memory.

## Architecture

```
Write Path (size-based routing):
  ┌─ Empty (0B)     → create file, optional fsync
  ├─ Tiny  (≤8KB)   → pool buffer → Hot + Warm cache → disk
  ├─ Small (≤128KB) → pool buffer → Warm cache only → disk
  └─ Large (>128KB) → temp file → fsync → atomic rename (no cache)

Read Path (cache-first):
  ┌─ Hot cache hit   → atomic load, ~50ns (lock-free ring)
  ├─ Warm cache hit  → RLock + copy, ~1-10µs (sharded LRU)
  └─ Disk            → open + read, ~100µs-10ms
```

### Two-Tier Cache

| Layer | Type | Size | Access | Concurrency |
|-------|------|------|--------|-------------|
| **L1 Hot** | Lock-free ring buffer | 4,096 slots | Atomic pointer CAS | Zero contention |
| **L2 Warm** | Sharded LRU | 256MB (64 shards × 4MB) | Per-shard RWMutex | Low contention |

- **Hot cache**: FNV-1a hash → slot index. Collision = overwrite (no chaining). Objects ≤8KB only.
- **Warm cache**: Lazy LRU (8 accesses before front-move). Objects ≤128KB. `InvalidatePrefix()` for batch eviction.

### Concurrency

- **256 key index shards** (sorted string arrays, FNV-1a distribution)
- **64 cache shards** (warm cache)
- **32 pool shards** (buffer pools, lock-free LCG shard selection)
- **sync.Map** for bucket lookup (zero-copy fast path)
- **Cached timestamps**: background goroutine updates every 5ms (avoids `time.Now()` syscalls)

### Buffer Pools

5 tiered `sync.Pool` sizes: 4KB, 64KB, 256KB, 2MB, 8MB. Each pool sharded 32 ways to reduce contention.

## DSN

```
rabbit:///path/to/data
rabbit:///path/to/data?nofsync=true
```

| Param | Default | Description |
|-------|---------|-------------|
| `nofsync` | `false` | Skip all fsync calls (unsafe, benchmarking only) |

## Limitations

- **No metadata persistence**: Content-type, custom headers lost on restart
- **No segment compaction**: Files stored individually (no log merging)
- **Hot cache collisions**: 32-bit FNV-1a into 4,096 slots = high collision rate at scale
- **Key index is O(n) insert/delete**: Binary search on sorted arrays, not trees
- **Directory listing loads all entries**: No filesystem-level pagination
- **Global nofsync flag**: No per-operation durability control
- **Multipart uploads have no expiry**: Orphaned uploads accumulate on disk
- **Upload ID counter resets on restart**: Potential collision with active uploads

## Enhancement Opportunities

1. **Hot cache**: Replace 32-bit with 64-bit hash; add chaining or open addressing
2. **Key index**: Replace sorted arrays with skip list or B-tree for O(log n) mutations
3. **Metadata persistence**: Sidecar `.meta` files for content-type, custom headers
4. **Upload cleanup**: Background goroutine to reap uploads older than 24h
5. **Streaming list**: Return iterator instead of pre-allocated slice
6. **Cache snapshots**: Serialize warm cache to manifest for fast restart warm-up
7. **Compression**: Optional at-rest compression for text-heavy workloads

## Performance Profile

| Operation | Expected Latency | Notes |
|-----------|-----------------|-------|
| Read (hot hit) | ~50ns | 4 atomic loads + 1 string compare |
| Read (warm hit) | 1-10µs | RLock + map lookup + memcpy |
| Read (disk, SSD) | 100-1000µs | open + read + seek |
| Write tiny (≤8KB) | 10-100µs | pool buffer + fsync |
| Write large (>128KB) | 1-10ms | temp file + fsync + rename |
| List (prefix) | O(n) | full scan + sort |

**Memory baseline**: ~260MB (hot cache ~4MB + warm cache 256MB + key index overhead)
