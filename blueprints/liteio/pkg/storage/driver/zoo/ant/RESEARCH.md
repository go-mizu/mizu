# Ant Driver: Deep Performance Research

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [v1 Baseline Profiling Analysis](#v1-baseline-profiling-analysis)
3. [v1 Bottleneck Identification](#v1-bottleneck-identification)
4. [v2 Optimization Journey](#v2-optimization-journey)
5. [v2 Results](#v2-results)
6. [Lessons Learned](#lessons-learned)
7. [Appendix: Profile Commands](#appendix-profile-commands)

---

## Architecture Overview

Ant is an Adaptive Radix Tree (ART) storage driver inspired by the SMART ART paper (OSDI 2023). It provides O(key_length) lookups by decomposing keys byte-by-byte through four adaptive node types.

### Storage Layout (v1)

```
store (single global RWMutex)
 └── artTree
      ├── artNode (2,744 bytes EACH, union of all 4 types)
      │    ├── Node4:   keys[16], children[48]       ← wastes 2,704 bytes
      │    ├── Node16:  keys[16], children[48]       ← wastes 2,564 bytes
      │    ├── Node48:  childIndex[256], children[48] ← wastes 2,048 bytes
      │    └── Node256: children256[256]              ← uses full 2,744 bytes
      ├── leafData (80 bytes, separate heap alloc)
      │    └── key []byte (composite key copy)
      ├── values.dat (append-only, per-op fsync)
      └── wal.log (per-op fsync, per-op make([]byte))
```

### Key Design Decisions (v1)

| Component | Design | Problem |
|-----------|--------|---------|
| Single artTree | Global RWMutex | **Kills parallelism** (C1→C200 = 182x drop) |
| Union artNode | All 4 types in one struct | **2,744 bytes per node** (Node4 needs ~72) |
| Separate leafData | Heap-allocated per leaf | Extra pointer + GC pressure |
| compositeKey | `bucket + "\x00" + key` | Per-op []byte allocation |
| appendValue | `make([]byte, totalSize)` | Per-op heap allocation |
| appendWAL | `make([]byte, entrySize)` | Per-op heap allocation |
| readValue | `make([]byte, totalSize)` | Per-op heap allocation (even for Stat!) |
| Per-op fsync | vlog.Sync() + wal.Sync() | 2 syscalls per write |
| No buffer pool | Fresh allocations everywhere | GC overhead compounds |

### Data Flow (v1)

**Write path:**
```
Write() → cleanKey()                     [string processing, allocation]
        → compositeKey(bucket, key)      [make([]byte), concat]
        → artSearch() under RLock        [check existing for created time]
        → io.ReadFull/ReadAll(src)       [make([]byte, size), full buffer]
        → appendValue(data, ct, ...)     [make([]byte, totalSize), WriteAt, Sync]
        → appendWAL(op, key, ...)        [make([]byte, entrySize), Write, Sync]
        → artInsert() under Lock         [allocate artNode 2,744B + leafData 80B]
```

**Read path:**
```
Open() → compositeKey(bucket, key)       [make([]byte), concat]
       → artSearch() under RLock         [tree traversal]
       → readValue(offset, totalSize)    [make([]byte, totalSize), ReadAt]
       → bytes.NewReader(data)           [wrap in ReadCloser]
```

---

## v1 Baseline Profiling Analysis

**Environment:** Go 1.26.0, darwin/arm64, 10 CPUs, benchtime=1-2s, concurrency=200

### Benchmark Results

| Benchmark | Throughput | Latency P50 | Latency P99 |
|-----------|------------|-------------|-------------|
| **Write/1KB** | 221.0K ops/s (215.8 MB/s) | 3.3us | 14.8us |
| **Write/64KB** | 13.1K ops/s (816.0 MB/s) | 25.6us | 344.9us |
| **Write/1MB** | 479 ops/s (478.9 MB/s) | 698.2us | 12.2ms |
| **Write/10MB** | 42 ops/s (420.4 MB/s) | 10.2ms | 242.0ms |
| **Write/100MB** | 5 ops/s (469.0 MB/s) | 179.3ms | 413.1ms |
| **Read/1KB** | 557.7K ops/s (544.6 MB/s) | 875ns | 17.3us |
| **Read/64KB** | 55.1K ops/s (3.4 GB/s) | 8.5us | 108.7us |
| **Read/1MB** | 3.3K ops/s (3.3 GB/s) | 218.8us | 1.5ms |
| **Read/10MB** | 270 ops/s (2.7 GB/s) | 2.8ms | 17.6ms |
| **Read/100MB** | 24 ops/s (2.4 GB/s) | 32.0ms | 79.9ms |
| **Stat** | 633.1K ops/s | — | — |
| **Delete** | 274.2K ops/s | — | — |
| **Copy/1KB** | 0.87 MB/s | — | — |
| **List/100** | 77.7K ops/s | — | — |

### Parallel Write Scalability (CRITICAL FAILURE)

| Concurrency | Throughput | vs C1 |
|-------------|------------|-------|
| C1 | 83.7 MB/s | 1.0x |
| C10 | 5.1 MB/s | **0.06x** |
| C25 | 2.2 MB/s | **0.03x** |
| C50 | 1.5 MB/s | **0.02x** |
| C100 | 0.89 MB/s | **0.01x** |
| C200 | 0.46 MB/s | **0.005x** |

**The global RWMutex causes a 182x throughput collapse from C1 to C200.** This is the single largest performance problem. At C200, 200 goroutines contend for a single lock.

### Parallel Read Scalability

| Concurrency | Throughput | vs C1 |
|-------------|------------|-------|
| C1 | 356.4 MB/s | 1.0x |
| C10 | 58.5 MB/s | 0.16x |
| C25 | 56.7 MB/s | 0.16x |
| C50 | 81.7 MB/s | 0.23x |
| C100 | 81.3 MB/s | 0.23x |
| C200 | 65.8 MB/s | 0.18x |

Reads use RLock (shared) but still degrade because: (1) Write operations hold exclusive Lock which blocks all readers, (2) readValue() allocates on heap per call, creating GC pressure that affects all goroutines.

### Resource Usage

| Metric | Value |
|--------|-------|
| Peak RSS | 6,109 MB |
| Go Heap | 5,221 MB |
| Go Sys | 10,434 MB |
| Disk Used | 9,388 MB |
| GC Cycles | 57 |

**5.2 GB Go heap is catastrophic.** The 100MB target requires a 52x reduction.

### Memory Budget Analysis

**artNode struct: 2,744 bytes per node**

| Field | Size | Used By | Waste for Node4 |
|-------|------|---------|-----------------|
| kind | 1B | All | 0 |
| numChildren | 2B | All | 0 |
| prefix (slice) | 24B | All | 0 |
| keys[16] | 16B | Node4, Node16 | 12B (only needs 4) |
| children[48] | 384B | Node4/16/48 | 352B (only needs 32) |
| childIndex[256] | 256B | Node48 only | **256B (unused)** |
| children256[256] | 2,048B | Node256 only | **2,048B (unused)** |
| leaf | 8B | All | 0 |
| **Total** | **2,744B** | | **2,668B waste for Node4** |

**Type-specific sizes (what each node actually needs):**

| Node Type | Fields Needed | Actual Size | vs Current |
|-----------|---------------|-------------|------------|
| Node4 | kind + count + prefix + keys[4] + children[4] + leaf | ~72B | **38x smaller** |
| Node16 | kind + count + prefix + keys[16] + children[16] + leaf | ~184B | **15x smaller** |
| Node48 | kind + count + prefix + childIndex[256] + children[48] + leaf | ~680B | **4x smaller** |
| Node256 | kind + count + prefix + children256[256] + leaf | ~2,088B | 1.3x smaller |

**Typical distribution** (80% Node4, 15% Node16, 4% Node48, 1% Node256):
- 100K nodes current: 100K × 2,744 = **274.4 MB**
- 100K nodes optimized: 80K×72 + 15K×184 + 4K×680 + 1K×2,088 = **13.4 MB** (20x reduction)

### Per-Operation Allocation Analysis

**Write/1KB path allocations:**

| Operation | Allocation | Size | Per-Op? |
|-----------|-----------|------|---------|
| `compositeKey()` | `[]byte(bucket + "\x00" + key)` | ~20B | Yes |
| `io.ReadFull()` | `make([]byte, size)` | 1,024B | Yes |
| `appendValue()` | `make([]byte, totalSize)` | ~1,050B | Yes |
| `appendWAL()` | `make([]byte, entrySize)` | ~50B | Yes |
| `newNode4()` | `&artNode{}` | 2,744B | Yes |
| `leafData` | `&leafData{key: compositeKey}` | 80B + ~20B key | Yes |
| **Total per write** | | **~5,000B** | |

At 221K writes/s: **~1.1 GB/s of allocations.** This is why GC has 57 cycles.

**Read/1KB path allocations:**

| Operation | Allocation | Size | Per-Op? |
|-----------|-----------|------|---------|
| `compositeKey()` | `[]byte(bucket + "\x00" + key)` | ~20B | Yes |
| `readValue()` | `make([]byte, totalSize)` | ~1,050B | Yes |
| `bytes.NewReader` | wrapper struct | ~16B | Yes |
| **Total per read** | | **~1,086B** | |

**Stat path (reads ENTIRE value just for metadata!):**

The `Stat()` method at line 1491 calls `readValue()` which reads the FULL value from disk just to extract `contentType`, `created`, and `updated`. For a 100MB object, Stat reads 100MB into heap.

---

## v1 Bottleneck Identification

### Bottleneck 1: Global RWMutex (CRITICAL — 182x parallel collapse)

**Impact:** Parallel write throughput drops 182x (C1→C200). Parallel read drops 5.4x.

**Root cause:** Single `artTree.mu sync.RWMutex` serializes ALL tree operations across ALL buckets, ALL keys. The write path holds exclusive Lock during `artInsert`, blocking all concurrent reads and writes.

Additionally, the Write path acquires the lock TWICE:
1. Line 1356: `tree.mu.RLock()` to check existing key (preserve created time)
2. Line 1384: `tree.mu.Lock()` to insert → calls `artSearch` AGAIN inside the lock

**Solution:** Shard the ART by first byte of composite key (256 shards). Each shard has its own RWMutex.

### Bottleneck 2: Union artNode Struct (2,744B per node)

**Impact:** 274 MB for 100K nodes. Exceeds 100MB budget on its own.

**Root cause:** All four node types share one struct. Node256's `children256 [256]*artNode` (2,048B) is allocated for every Node4.

**Solution:** Type-specific structs via interface.

### Bottleneck 3: Per-Operation Heap Allocations (~5KB/write)

**Impact:** ~1.1 GB/s allocation rate → 57 GC cycles → GC pauses affect all goroutines.

**Root cause:** Every Write does: make(compositeKey) + make(valueData) + make(vlogBuf) + make(walBuf) + new(artNode) + new(leafData). Every Read does: make(compositeKey) + make(readBuf).

**Solution:** Buffer pools (sync.Pool or mutex-guarded free lists). Reuse buffers across operations.

### Bottleneck 4: Per-Operation fsync (2 syncs per write)

**Impact:** Each write calls `vlog.Sync()` + `wal.Sync()` when sync!=none. On macOS, fsync→F_FULLFSYNC is ~1ms each.

**Root cause:** No write batching. Each operation syncs independently.

**Solution:** WAL batching — accumulate entries, flush periodically or on buffer full.

### Bottleneck 5: Stat Reads Full Value from Disk

**Impact:** Stat for a 100MB object reads 100MB into heap just to get 16 bytes of metadata.

**Root cause:** `readValue()` reads the entire vlog entry. No way to read just metadata.

**Solution:** Store metadata (size, contentType, timestamps) in the leaf node itself. No disk I/O needed for Stat.

### Bottleneck 6: Leaf Key Duplication

**Impact:** Each leaf stores a full copy of the composite key as `[]byte`. For 100K objects with 30-byte keys: 3 MB of duplicated key data.

**Root cause:** `leafData.key` stores a copy for verification in `artSearch` (line 214: `bytes.Equal(cur.leaf.key, key)`).

**Solution:** The key is implicit in the tree path. With correct path compression, the tree path reconstructs the key exactly. Remove `leafData.key` and verify via path.

### Bottleneck 7: No Value Size in Leaf

**Impact:** Even to return `Object.Size` in List, we must call `computeValueSize()` which computes from `totalSize - overhead`. This is fragile and prevents direct size queries.

**Solution:** Store actual value size directly in the leaf.

---

## v2 Optimization Journey

### Optimization 1: Type-Specific Node Structs

**Problem:** Every artNode is 2,744 bytes regardless of type. 97% is waste for Node4.

**Solution:** Use an interface with type-specific structs:

```go
type artNode interface {
    findChild(b byte) artNode
    addChild(b byte, child artNode)
    kind() nodeKind
    // ...
}

type node4 struct {
    prefix      []byte
    numChildren uint8
    keys        [4]byte
    children    [4]artNode   // interface, 16B each
    leaf        *leafData
}
// Size: ~120B (vs 2,744B) — 23x smaller
```

**Actual sizes with interface children (16B each):**

| Type | Size | Savings vs v1 |
|------|------|---------------|
| node4 | ~120B | **23x** |
| node16 | ~344B | **8x** |
| node48 | ~1,064B | **2.6x** |
| node256 | ~4,136B | 0.66x (larger due to interface) |

With typical distribution (80/15/4/1): 100K nodes = **23.4 MB** (vs 274 MB, **11.7x reduction**).

**Further optimization — embed leaf in node (eliminate leafData pointer):**

```go
type leafEntry struct {
    valueOffset int64
    valueSize   int32    // actual value size (max 2GB)
    ctIndex     uint16   // index into content-type string table
    created     int64
    updated     int64
}
// Size: 32B (vs 80B + 8B pointer = 88B)
```

With embedded leaf (no separate allocation): 100K objects ≈ **16.6 MB** for nodes + leaves.

### Optimization 2: Sharded ART (16 Shards)

**Problem:** Global RWMutex causes 182x parallel collapse.

**Solution:** 16 independent ART shards, selected by FNV-1a hash of composite key:

```go
type shardedART struct {
    shards [16]artShard
}

type artShard struct {
    mu   sync.RWMutex
    root artNode
    size int64
}

func (s *shardedART) shardFor(key []byte) *artShard {
    h := fnv1a(key)
    return &s.shards[h & 0x0F]
}
```

**Expected improvement:**
- Contention reduced 16x (200 goroutines / 16 shards = 12.5 per shard)
- Lock hold time unchanged but lock acquisition contention massively reduced
- Read/write parallelism across shards is fully concurrent

### Optimization 3: Buffer Pool (Eliminate Per-Op Allocations)

**Problem:** ~5KB allocated per write operation, ~1KB per read.

**Solution:** sync.Pool for hot-path buffers:

```go
var bufPool = sync.Pool{
    New: func() any { return make([]byte, 0, 4096) },
}

func (s *store) appendValue(data []byte, ...) (int64, int64, error) {
    buf := bufPool.Get().([]byte)
    defer bufPool.Put(buf[:0])
    buf = buf[:totalSize] // reuse capacity
    // ... encode into buf ...
}
```

**Also pool:** WAL entry buffers, read buffers, composite key buffers.

### Optimization 4: Metadata-Only Stat (No Disk I/O)

**Problem:** Stat reads entire value from disk.

**Solution:** Store value size and content-type in the leaf. Stat becomes a pure in-memory operation:

```go
func (b *bucket) Stat(...) (*storage.Object, error) {
    leaf := shard.search(compositeK) // in-memory only
    return &storage.Object{
        Size:        int64(leaf.valueSize),
        ContentType: s.ctTable[leaf.ctIndex],
        Created:     time.Unix(0, leaf.created),
        Updated:     time.Unix(0, leaf.updated),
    }, nil
}
```

Content-type interning via string table: most objects share a few content types ("application/octet-stream", "text/plain", etc.). Store an index into a deduplicated table.

### Optimization 5: Mmap Value Log for Reads

**Problem:** Each Read allocates `make([]byte, totalSize)` to read from disk.

**Solution:** Mmap the value log for reads. Return a slice of the mmap'd region — zero allocation, zero copy.

```go
type mmapVlog struct {
    data     []byte   // mmap'd region
    size     int64    // written size
    fd       *os.File // for writes (pwrite)
    mu       sync.Mutex
}

func (v *mmapVlog) readSlice(offset, size int64) []byte {
    return v.data[offset : offset+size] // zero alloc, zero copy
}
```

**Remap on growth** (lesson from herd v4): when the value log grows beyond the current mmap region, remap to cover the new size. Old mappings can leak — bounded by geometric growth.

### Optimization 6: WAL Batching

**Problem:** Each Write calls WAL.Write() + WAL.Sync() — 2 syscalls.

**Solution:** Ring buffer of WAL entries. Background flusher writes + syncs periodically:

```go
type walBatcher struct {
    buf     []byte    // pre-allocated WAL buffer
    pos     int
    mu      sync.Mutex
    flushed chan struct{}
}
```

Flush triggers: buffer full, 1ms timer, or explicit sync request. This reduces WAL syscalls from 1-per-op to 1-per-batch.

### Optimization 7: Eliminate Leaf Key Duplication

**Problem:** Each leaf stores composite key as `[]byte` (24B slice header + key data).

**Solution:** Remove `leafData.key`. The tree path already encodes the key. For verification, reconstruct the key from the path during traversal (or skip verification since the tree is correct by construction).

### Optimization 8: Inline compositeKey Construction

**Problem:** `compositeKey()` creates `[]byte(bucket + "\x00" + key)` — heap allocation.

**Solution:** Pre-compute composite key into pooled buffer:

```go
func compositeKeyInto(buf []byte, bucket, key string) []byte {
    buf = buf[:0]
    buf = append(buf, bucket...)
    buf = append(buf, 0)
    buf = append(buf, key...)
    return buf
}
```

Or better: compute hash directly without materializing the full key:

```go
func shardForParts(bucket, key string) int {
    h := fnv1aString(bucket)
    h = fnv1aByte(h, 0)
    h = fnv1aString(key)
    return int(h & 0x0F)
}
```

---

## v2 Results

### Architecture Changes (v2 → v2b)

The initial v2 kept a global WAL mutex and global vlog mutex, which still serialized all 200 goroutines through 4 lock acquisitions per write. v2b restructured to **per-shard vlogs with embedded WAL metadata**, achieving:

- **ONE lock per write** (shard.mu.Lock → vlog append → ART insert → unlock)
- **ZERO global locks** in the write path
- **Zero-copy reads** from mmap (no `make([]byte)` + `copy()` for reads)
- **No separate WAL** — recovery scans shard vlog entries directly

### Performance Comparison

| Benchmark | v1 Baseline | v2b Optimized | Improvement |
|-----------|-------------|---------------|-------------|
| Write/1KB | 221.0K ops/s | **1,400K ops/s** | **6.3x** |
| Write/64KB | 13.1K ops/s (816 MB/s) | **34.4K ops/s (2.1 GB/s)** | **2.6x** |
| Write/1MB | — | **437 ops/s (436.9 MB/s)** | — |
| Read/1KB | 557.7K ops/s | **3,600K ops/s** | **6.5x** |
| Read/64KB | — | **715.7K ops/s (44.7 GB/s)** | — |
| Read/1MB | — | **53.1K ops/s (53.1 GB/s)** | — |
| Stat | 633.1K ops/s | **6,300K ops/s** | **10.0x** |
| Delete | 274.2K ops/s | **3,900K ops/s** | **14.2x** |
| Copy/1KB | — | **409.5 MB/s** | — |
| List/100 | 77.7K ops/s | **72.2K ops/s** | 0.93x |

### Parallel Write Scalability

| Concurrency | v1 Baseline | v2b Optimized | Improvement |
|-------------|-------------|---------------|-------------|
| C1 | 83.7 MB/s | **500.7 MB/s** | **6.0x** |
| C10 | — | **83.4 MB/s** | — |
| C50 | — | **14.7 MB/s** | — |
| C100 | — | **15.9 MB/s** | — |
| C200 | 0.46 MB/s | **6.3 MB/s** | **13.7x** |

### Resource Usage

| Metric | v1 Baseline | v2b Optimized | Change |
|--------|-------------|---------------|--------|
| Peak RSS (full bench) | 6,109 MB | 6,962 MB | +14% (mmap) |
| Go Heap (full bench) | 5,221 MB | 3,007 MB | **-42%** |
| GC Cycles (full bench) | 57 | 140 | +2.5x (more iters) |
| Go Heap (100K×1KB) | — | **22.4 MB** | — |
| Go Sys (100K×1KB) | — | **50.6 MB** | — |

### Key Insight: Per-Shard Vlog Entry Format

```
Put entry:
  [4B] entrySize  [1B] op=0  [2B] keyLen  [NB] key
  [1B] ctLen  [MB] contentType  [8B] created  [8B] updated  [VB] value
  Total: 24 + N + M + V bytes

Delete entry:
  [4B] entrySize  [1B] op=1  [2B] keyLen  [NB] key  [8B] timestamp
  Total: 15 + N bytes
```

Recovery scans each shard's vlog sequentially (16 independent files), rebuilding
the per-shard ART. No separate WAL file needed.

---

## Lessons Learned

### From v1 Analysis:

1. **Union structs are catastrophic for memory** — 2,744B per node when most need 72B. Always use type-specific structs for polymorphic data.

2. **Global locks kill parallelism** — Even RWMutex. At C200, lock contention dominates. Shard early.

3. **Per-operation allocations compound through GC** — 5KB × 221K ops/s = 1.1 GB/s of GC pressure. Pool everything on the hot path.

4. **Stat should never touch disk** — Store all metadata in-memory. The index exists for exactly this purpose.

5. **Fsync batching is essential** — 2 fsyncs per write is 2ms overhead on macOS. Batch to amortize.

### From v2/v2b Optimization:

6. **Per-shard everything** — Having per-shard ART but global WAL/vlog still serializes writes. The shard boundary must encompass ALL mutable state (ART + vlog + WAL) to eliminate global contention.

7. **Single lock per operation** — v2 used 4 locks per write (shard.RLock, vlog.mu, walMu, shard.Lock). v2b uses 1 lock (shard.Lock). Fewer lock acquisitions = less contention overhead.

8. **Eliminate the WAL** — Embedding key metadata in the vlog entry makes the vlog self-describing. Recovery just scans the vlog. One less file, one less lock, simpler recovery.

9. **Zero-copy mmap reads** — Returning a slice of mmap'd memory instead of `make([]byte) + copy()` eliminates the biggest allocation in the read path. Old mappings are intentionally leaked (bounded by geometric growth) to keep references safe.

10. **Parallel write collapse is fundamental** — Even with 16 independent shards, 200 goroutines still contend ~12.5 per shard. Further improvement requires lock-free data structures or partitioned goroutine assignment.

---

## Appendix: Profile Commands

### Running Benchmarks

```bash
# Full benchmark with profiling
go run ./cmd/bench --drivers ant --profile --resource-tracking \
  --benchtime 2s --output ./report/ant_v2_optimized \
  --formats markdown,json --progress

# Quick benchmark (no profiling)
go run ./cmd/bench --drivers ant --benchtime 1s --formats markdown

# Specific benchmark only
go run ./cmd/bench --drivers ant --filter "Write/1KB" --benchtime 5s

# Scale tests (data growth behavior)
go run ./cmd/bench --drivers ant --filter "Scale" --scales "1000,10000,100000"
```

### Viewing Profiles

```bash
# CPU profile flamegraph
go tool pprof -http=:8080 report/ant_v1_baseline/ant/cpu.pprof

# Top allocation sites
go tool pprof -top -cum -nodecount=30 report/ant_v1_baseline/ant/allocs.pprof

# Heap in-use
go tool pprof -http=:8080 report/ant_v1_baseline/ant/heap.pprof

# Lock contention
go tool pprof -top report/ant_v1_baseline/ant/block.pprof

# Compare baseline vs optimized
go tool pprof -base report/ant_v1_baseline/ant/cpu.pprof \
              report/ant_v2_optimized/ant/cpu.pprof

# Investigate attribution
go tool pprof -peek "artInsert" report/ant_v1_baseline/ant/cpu.pprof
```

### Memory Analysis

```bash
# Check artNode struct size
go run -v <<'EOF'
package main
import ("fmt"; "unsafe")
type artNode struct { /* ... */ }
func main() { fmt.Println(unsafe.Sizeof(artNode{})) }
EOF

# Runtime memory stats during benchmark
GODEBUG=gctrace=1 go run ./cmd/bench --drivers ant --benchtime 2s 2>&1 | grep gc
```
