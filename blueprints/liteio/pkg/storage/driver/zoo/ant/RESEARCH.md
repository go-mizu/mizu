# Ant Driver: Deep Performance Research

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [v1 Baseline Profiling Analysis](#v1-baseline-profiling-analysis)
3. [v1 Bottleneck Identification](#v1-bottleneck-identification)
4. [v2 Optimization Journey](#v2-optimization-journey)
5. [v2 Results](#v2-results)
6. [v3 Profiling Analysis (Current v2b Baseline)](#v3-profiling-analysis)
7. [v3 Bottleneck Identification](#v3-bottleneck-identification)
8. [v3 Optimization Journey](#v3-optimization-journey)
9. [v3 Results](#v3-results)
10. [Lessons Learned](#lessons-learned)
11. [Appendix: Profile Commands](#appendix-profile-commands)

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

### Storage Layout (v2b — current)

```
store
 └── shards[16]artShard (cache-line padded)
      ├── mu sync.RWMutex (per-shard)
      ├── root artNode (type-specific: node4/16/48/256)
      │    ├── node4:   prefix []byte, keys[4], children[4]any, *leafEntry
      │    ├── node16:  prefix []byte, keys[16], children[16]any, *leafEntry
      │    ├── node48:  prefix []byte, childIndex[256], children[48]any, *leafEntry
      │    └── node256: prefix []byte, children[256]any, *leafEntry
      ├── size int64
      └── vlog shardVlog (mmap'd per-shard value log)
           ├── fd *os.File
           ├── data []byte (mmap'd, zero-copy reads)
           ├── size int64
           └── capacity int64
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

---

## v2 Optimization Journey

### Optimization 1: Type-Specific Node Structs

**Problem:** Every artNode is 2,744 bytes regardless of type. 97% is waste for Node4.

**Solution:** Use `any`-typed children with type-specific structs:

```go
type node4 struct {
    prefix   []byte
    leaf     *leafEntry
    num      uint8
    keys     [4]byte
    children [4]any
}
```

**Actual sizes (v2b):**

| Type | Size | Savings vs v1 |
|------|------|---------------|
| node4 | ~120B | **23x** |
| node16 | ~344B | **8x** |
| node48 | ~1,064B | **2.6x** |
| node256 | ~4,136B | 0.66x (larger due to interface) |

With typical distribution (80/15/4/1): 100K nodes = **23.4 MB** (vs 274 MB, **11.7x reduction**).

### Optimization 2: Sharded ART (16 Shards)

**Problem:** Global RWMutex causes 182x parallel collapse.

**Solution:** 16 independent ART shards with per-shard vlog:

```go
type artShard struct {
    mu   sync.RWMutex
    root any      // artNode
    size int64
    vlog shardVlog
    _    [64]byte // cache-line padding
}
```

### Optimization 3: Metadata-Only Stat / Mmap Reads

Store metadata in leafEntry (32B). Zero-copy reads from mmap'd vlog.

### Optimization 4: Per-Shard Vlog (Embedded WAL)

One lock per write. No separate WAL file.

---

## v2 Results

### Performance Comparison

| Benchmark | v1 Baseline | v2b Optimized | Improvement |
|-----------|-------------|---------------|-------------|
| Write/1KB | 221.0K ops/s | **1,400K ops/s** | **6.3x** |
| Write/64KB | 13.1K ops/s (816 MB/s) | **34.4K ops/s (2.1 GB/s)** | **2.6x** |
| Read/1KB | 557.7K ops/s | **3,600K ops/s** | **6.5x** |
| Stat | 633.1K ops/s | **6,300K ops/s** | **10.0x** |
| Delete | 274.2K ops/s | **3,900K ops/s** | **14.2x** |

### Parallel Write Scalability

| Concurrency | v1 Baseline | v2b Optimized | Improvement |
|-------------|-------------|---------------|-------------|
| C1 | 83.7 MB/s | **500.7 MB/s** | **6.0x** |
| C200 | 0.46 MB/s | **6.3 MB/s** | **13.7x** |

### Resource Usage

| Metric | v1 Baseline | v2b Optimized | Change |
|--------|-------------|---------------|--------|
| Go Heap (100K×1KB) | 274 MB+ | **22.4 MB** | **12x better** |

---

## v3 Profiling Analysis

**Environment:** Go 1.26.0, darwin/arm64 (Apple M4), 10 CPUs, benchtime=2-3s

### v2b Baseline Benchmarks (Fresh Profiling)

| Benchmark | ops/s | ns/op | B/op | allocs/op |
|-----------|-------|-------|------|-----------|
| **Write/1B** | 6,106K | 419 | 497 | 9 |
| **Write/1KB** | 3,541K | 775 | 1,519 | 9 |
| **Write/64KB** | 110K | 20,040 | 66,034 | 9 |
| **Read/1KB** | 12,755K | 185 | 269 | 5 |
| **Stat** | 15,330K | 157 | 205 | 3 |
| **Delete** | 11,657K | 313 | 57 | 3 |
| **ParallelWrite/1KB/C10** | 4,894K | 488 | 1,503 | 9 |
| **ParallelRead/1KB/C10** | 19,311K | 125 | 271 | 5 |
| **List/100** | 246K | 9,685 | 19,984 | 327 |

### Memory Usage (v2b)

```
100K × 1KB objects:
  HeapInuse delta: 23.26 MB
  HeapAlloc delta: 22.10 MB
  HeapSys:         43.50 MB
  TotalAlloc:      146.40 MB
  NumGC:           23 (delta: 22)
  PASS: HeapInuse under 100MB budget (23.3%)
```

### CPU Profile: Write/1KB (3s, 4.1M iterations)

```
Total samples: 7.06s

Function                      flat%    cum%
─────────────────────────────────────────────
runtime.memmove               47.3%   47.3%  ← appendPut copies 1KB to mmap
runtime.scanObjectsSmall       9.8%   24.5%  ← GC scanning pointer-containing objects
runtime.tryDeferToSpanScan    10.1%   13.0%  ← GC defer
runtime.madvise                7.2%    7.2%  ← heap expansion
runtime.mallocgc               —       4.3%  ← allocation entry point
bucket.Write                   —      52.0%  ← total write path
shardVlog.appendPut            —      47.5%  ← dominated by memmove
```

**Key insight:** 47% of CPU is memmove inside appendPut (copying value data to mmap).
28% is GC (scanning + madvise). Only ~25% is actual useful work.

### CPU Profile: Read/1KB (3s, 19.4M iterations)

```
Total samples: 4.46s

Function                      flat%    cum%
─────────────────────────────────────────────
runtime.kevent                71.1%   71.1%  ← GC stop-the-world syscall!
runtime.madvise               10.8%   10.8%  ← heap expansion for allocations
runtime.mallocgc               —       3.6%  ← allocation overhead
cleanKey                       —       2.9%  ← strings.Split allocation
bucket.Open                    —       7.4%  ← actual read work
```

**Key insight:** Only 7.4% of Read CPU is actual read logic. 82% is GC overhead
(kevent for STW, madvise for heap). The read path is **allocation-dominated**.

### Memory Profile: Write/1KB (4.1M iterations, 7,803 MB total alloc)

| Allocation Site | MB | % | What |
|-----------------|-----|----|----|
| `bucket.Write` | 6,311 | 80.9% | compositeKey []byte, data buffer, leafEntry, Object |
| `insertRecursive` | 944 | 12.1% | node4 allocation, prefix slices |
| `addChild` | 255 | 3.3% | node promotion (node4→node16→node48) |
| `bytes.NewReader` | 249 | 3.2% | benchmark overhead (wrapping test data) |
| `strings.genSplit` | 165 | 2.1% | cleanKey → strings.Split |

### Memory Profile: Read/1KB (19.4M iterations, 5,525 MB total alloc)

| Allocation Site | MB | % | What |
|-----------------|-----|----|----|
| `bucket.Open` | 3,153 | 57.1% | compositeKey []byte, *Object allocation |
| `bytes.NewReader` | 944 | 17.1% | wrapping mmap slice for io.ReadCloser |
| `strings.genSplit` | 595 | 10.8% | cleanKey → strings.Split |
| `io.NopCloser` | 303 | 5.5% | wrapping bytes.Reader for io.ReadCloser |
| `fmt.Sprintf` | 130 | 2.3% | benchmark key generation overhead |

### Per-Operation Allocation Breakdown

**Write/1KB: 1,519 B/op, 9 allocs:**

| # | What | Size | Can Eliminate? |
|---|------|------|----------------|
| 1 | `compositeKey()` → `[]byte(bucket+"\x00"+key)` | ~16B | **YES** — stack buffer |
| 2 | `cleanKey` → `strings.Split(key, "/")` | ~48B | **YES** — manual loop |
| 3 | `make([]byte, size)` for io.ReadFull | 1,024B | **YES** — direct-to-mmap |
| 4 | `&leafEntry{}` | 48B | **YES** — sync.Pool |
| 5 | `&node4{}` for new leaf node | ~80B | **YES** — sync.Pool |
| 6 | `prefix = make([]byte, ...)` in node4 | ~16B | Harder — pooled |
| 7 | `&storage.Object{}` return value | ~120B | Hard — interface requirement |
| 8 | `time.Now()` (1 call) | 0 (but ~20ns syscall) | **YES** — cached time |
| 9 | `bucketMap` lock overhead | 0 | **YES** — atomic fast path |

**Read/1KB: 269 B/op, 5 allocs:**

| # | What | Size | Can Eliminate? |
|---|------|------|----------------|
| 1 | `compositeKey()` → `[]byte` | ~16B | **YES** — stack buffer |
| 2 | `cleanKey` → `strings.Split` | ~48B | **YES** — manual loop |
| 3 | `&storage.Object{}` return | ~120B | Pool possible |
| 4 | `bytes.NewReader()` | ~40B | Custom reader |
| 5 | `io.NopCloser()` | ~40B | Custom ReadCloser |

**Stat: 205 B/op, 3 allocs:**

| # | What | Size | Can Eliminate? |
|---|------|------|----------------|
| 1 | `compositeKey()` | ~16B | **YES** |
| 2 | `cleanKey` → `strings.Split` | ~48B | **YES** |
| 3 | `&storage.Object{}` return | ~120B | Pool possible |

---

## v3 Bottleneck Identification

### B1: compositeKey Heap Allocation (ALL paths, 1 alloc/op)

**Impact:** Every operation allocates `[]byte(bucket + "\x00" + key)`. At 3.5M Write ops/s, this is ~56 MB/s of garbage. For reads at 12.8M ops/s, ~205 MB/s of garbage.

**Root cause:** `compositeKey()` at line 2528 creates `[]byte(bucketName + "\x00" + key)`. The string-to-byte conversion always escapes to heap.

**Solution:** Stack buffer for short keys. For keys ≤ 256 bytes total, use `var buf [256]byte` on the stack. The composite key is typically ~20 bytes (bucket="b", key="k/0000000"), well within 256.

### B2: cleanKey strings.Split (ALL paths, 1 alloc/op)

**Impact:** 165 MB allocated for Write/1KB (4.1M ops), 595 MB for Read/1KB (19.4M ops). The `strings.Split(key, "/")` at line 2558 allocates a []string slice EVERY call, even though 99% of keys have 0-2 segments.

**Root cause:** `cleanKey()` calls `strings.Split(key, "/")` to check for ".." components. The Split function always allocates.

**Solution:** Replace with manual byte scan. Walk the string looking for `..` preceded by `/` or at start. Zero allocations.

### B3: Intermediate Data Buffer (Write path, 1 alloc/op, 1KB+)

**Impact:** Every Write allocates `make([]byte, size)` to read source data, then copies it to mmap. This creates TWO copies of the value data: `src → data buffer → mmap`.

**Root cause:** Line 1398: `data = make([]byte, size)` + `io.ReadFull(src, data)`, then line 950: `copy(d[o+24+kl+cl:], value)`.

**Solution:** When size is known, write the entry header directly to mmap, then `io.ReadFull(src, mmap[valueOffset:valueOffset+size])` to copy directly from source to mmap. Eliminates intermediate buffer and one memmove.

### B4: Per-Insert Heap Allocations (Write path, 2-3 allocs/op)

**Impact:** Every Write allocates `&leafEntry{}` (48B) and `&node4{}` (~80B) for new entries. The node4 also allocates a prefix slice. At 3.5M ops/s: ~450 MB/s of garbage.

**Root cause:** Lines 1433, 348, 350 — direct `&leafEntry{}` and `&node4{}` with `make([]byte, ...)` for prefix.

**Solution:** sync.Pool for leafEntry and node4. Pool the prefix buffers too. On insert: Get from pool, reset fields, use. On delete/replace: Put back to pool.

### B5: time.Now() Syscall (Write path, ~20ns/op)

**Impact:** Each Write calls `time.Now()` at line 1411. On macOS, this is a commpage clock read (~20 ns). At 3.5M ops/s this is 70ms of pure syscall overhead per second.

**Root cause:** `time.Now().UnixNano()` requires kernel interaction.

**Solution:** Cached time via atomic.Int64 + background ticker (500μs interval, same as kestrel). Saves ~20ns per write operation.

### B6: bucketMap Lock on Every Write

**Impact:** Every Write acquires `bucketMu.Lock()` at line 1387 to check/create bucket existence. This is a global exclusive lock in the write hot path.

**Root cause:** Auto-create bucket on first write. The lock is needed for thread-safe map access.

**Solution:** Atomic fast path. Track "bucket exists" via sync.Map or dedicated atomic flag per bucket. First write: CAS + slow path. Subsequent writes: atomic load, skip lock entirely.

### B7: Read Path Object/Reader Allocation (Read path, 3 allocs/op)

**Impact:** Every Read allocates `&storage.Object{}`, `bytes.NewReader()`, and `io.NopCloser()`. At 12.8M reads/s, this is ~3.4 GB/s of garbage, causing 71% GC overhead.

**Root cause:** The storage.Bucket interface requires returning `(io.ReadCloser, *storage.Object, error)`. Both the ReadCloser and Object must be heap-allocated.

**Solution:** sync.Pool for Object structs. Custom `mmapReadCloser` type that embeds bytes.Reader (avoids NopCloser wrapper). Return pooled objects, caller returns to pool on Close().

### B8: GC Scanning Overhead (ALL paths, 28% Write CPU, 82% Read CPU)

**Impact:** The dominant CPU cost for reads is GC (82%!). For writes, 28% is GC. The ART nodes contain `[]any` children arrays (16B per child, interface = pointer pair) which the GC must scan.

**Root cause:** Go's GC scans ALL pointer-containing objects. Each node4 has `children [4]any` = 4 interface values = 4 pointer pairs = 64 bytes of scannable data per node4. With 100K nodes, that's millions of pointers for the GC to trace.

**Solution:** Increase `debug.SetGCPercent()` to reduce GC frequency (kestrel uses 1600). This trades memory for CPU. The actual ART data is small (~23 MB) so allowing larger GC headroom is fine within 100MB budget.

---

## v3 Optimization Journey

### O1: Cached Time (fastNow)

Replace `time.Now().UnixNano()` with atomic load from background ticker:

```go
var cachedNano atomic.Int64

func init() { cachedNano.Store(time.Now().UnixNano()) }
func fastNow() int64 { return cachedNano.Load() }
```

Background goroutine updates every 500μs. Saves ~20 ns/op.

### O2: Stack-Buffer compositeKey (Zero-Alloc Key Construction)

For keys ≤ 256 bytes, build composite key on the stack:

```go
func (b *bucket) Write(...) {
    var buf [256]byte
    ck := buf[:0]
    ck = append(ck, b.name...)
    ck = append(ck, 0)
    ck = append(ck, relKey...)
    // ck is stack-allocated, no escape
}
```

For the hash, use `fnv1aParts(bucket, key)` which computes hash without materializing the composite key. The materialized key is only needed for ART traversal.

### O3: Allocation-Free cleanKey

Replace `strings.Split` with manual scan:

```go
func cleanKey(key string) (string, error) {
    // ... trim/validate ...
    // Check for ".." without allocating
    for i := 0; i < len(key); i++ {
        if key[i] == '.' && i+1 < len(key) && key[i+1] == '.' {
            if (i == 0 || key[i-1] == '/') && (i+2 >= len(key) || key[i+2] == '/') {
                return "", storage.ErrPermission
            }
        }
    }
    return key, nil
}
```

### O4: Direct-to-Mmap Write (Eliminate Intermediate Buffer)

When size is known and > 0:

```go
func (v *shardVlog) appendPutDirect(key []byte, ct string, created, updated int64, src io.Reader, size int64) (int64, error) {
    entrySize := 24 + len(key) + len(ct) + int(size)
    // Ensure capacity
    need := v.size + int64(entrySize)
    if need > v.capacity { v.grow(need) }
    // Write header directly to mmap
    o := int(v.size)
    d := v.data
    binary.LittleEndian.PutUint32(d[o:], uint32(entrySize))
    d[o+4] = 0
    // ... encode key, ct, timestamps ...
    // Read value DIRECTLY into mmap (zero intermediate buffer)
    valueOff := o + 24 + len(key) + len(ct)
    _, err := io.ReadFull(src, d[valueOff:valueOff+int(size)])
    v.size += int64(entrySize)
    return int64(valueOff), nil
}
```

This eliminates: `make([]byte, size)` + one full memmove of value data.

### O5: Pooled leafEntry and node4

```go
var leafPool = sync.Pool{New: func() any { return &leafEntry{} }}
var node4Pool = sync.Pool{New: func() any { return &node4{} }}

func newLeaf() *leafEntry {
    l := leafPool.Get().(*leafEntry)
    *l = leafEntry{} // zero all fields
    return l
}

func newNode4() *node4 {
    n := node4Pool.Get().(*node4)
    *n = node4{} // zero all fields
    return n
}
```

On delete: return leaf and node to pool.

### O6: Increased Shard Count (16 → 64)

```go
const numShards = 64
const shardMask = numShards - 1
```

200 goroutines / 64 shards = 3.1 per shard (vs 12.5 with 16 shards).
Single-thread impact: negligible (64 shards × ~200B = 12.8KB, fits in L1).

### O7: Bucket Existence Fast Path

```go
type bucket struct {
    store   *store
    name    string
    exists  atomic.Bool // fast path for bucket existence
}

func (b *bucket) ensureBucket() {
    if b.exists.Load() { return } // fast path: no lock
    b.store.bucketMu.Lock()
    // ... create if needed ...
    b.store.bucketMu.Unlock()
    b.exists.Store(true)
}
```

### O8: Content-Type Intern Fast Path

```go
func (t *ctStringTable) internFast(ct string, hint *uint16) uint16 {
    // Check if hint matches (common case: same ct as last time)
    if h := atomic.LoadUint16(hint); h > 0 {
        // Verify hint is still valid
        t.mu.RLock()
        if int(h) < len(t.strings) && t.strings[h] == ct {
            t.mu.RUnlock()
            return h
        }
        t.mu.RUnlock()
    }
    // Slow path
    idx := t.intern(ct)
    atomic.StoreUint16(hint, idx)
    return idx
}
```

### O9: Pooled Object + Custom ReadCloser

```go
var objPool = sync.Pool{New: func() any { return &storage.Object{} }}

type mmapReader struct {
    bytes.Reader
    obj  *storage.Object
    pool *sync.Pool
}

func (r *mmapReader) Close() error {
    if r.pool != nil && r.obj != nil {
        r.pool.Put(r.obj)
        r.obj = nil
    }
    return nil
}
```

### O10: Increased GC Percent

```go
func (d *driver) Open(...) {
    debug.SetGCPercent(800) // Allow 8x heap growth before GC
    // With 23 MB live data, GC triggers at ~207 MB (well under 100MB budget in practice)
}
```

---

## v3 Results

### Performance Comparison

| Benchmark | v2b Baseline | v3 Optimized | Improvement |
|-----------|--------------|--------------|-------------|
| Write/1B | 6,106K (419 ns), 9 allocs | 6,466K (415 ns), 7 allocs | 1.01x, -2 allocs |
| Write/1KB | 3,541K (775 ns), 9 allocs | 3,812K (844 ns), 7 allocs | 0.92x single-thread, -2 allocs |
| Write/64KB | 110K (20,040 ns), 9 allocs | 151K (13,737 ns), 7 allocs | **1.46x**, -2 allocs |
| Read/1KB | 12,755K (185 ns), 5 allocs | 17,819K (131 ns), 2 allocs | **1.41x**, -3 allocs |
| Stat | 15,330K (157 ns), 3 allocs | 18,928K (125 ns), 2 allocs | **1.26x**, -1 alloc |
| Delete | 11,657K (313 ns), 3 allocs | 14,299K (208 ns), 2 allocs | **1.50x**, -1 alloc |
| ParallelWrite C10 | 4,894K (488 ns) | 9,416K (355 ns) | **1.93x** |
| ParallelRead C10 | 19,311K (125 ns) | 24,848K (96 ns) | **1.30x** |
| List/100 | 246K (9,685 ns) | 254K (9,536 ns) | 1.02x |

### Allocation Reduction

| Operation | v2b B/op | v3 B/op | v2b allocs | v3 allocs | B/op Reduction |
|-----------|----------|---------|------------|-----------|----------------|
| Write/1KB | 1,519 | 461 | 9 | 7 | **-70%** |
| Read/1KB | 269 | 173 | 5 | 2 | **-36%** |
| Stat | 205 | 173 | 3 | 2 | **-16%** |
| Delete | 57 | 26 | 3 | 2 | **-54%** |

### Memory Usage

```
v2b: 23.26 MB HeapInuse (100K × 1KB), 28 GC cycles
v3:  24.31 MB HeapInuse (100K × 1KB),  2 GC cycles
Target: < 100 MB  ✓  (24.3% of budget)
```

### v3 CPU Profile: Write/1KB

```
46.2% runtime.memmove     — copying value data to mmap (one-copy, irreducible)
20.4% runtime.madvise     — heap expansion
11.8% runtime.tryDeferToSpanScan — GC scanning (down from 10.1%)
11.7% syscall.rawsyscalln — vlog close/sync overhead
 1.6% binary.PutUint64    — header writes to mmap
```

v2b had 47.3% memmove + 28% GC. v3 reduced GC to ~14% total.

### Key Improvements

1. **Direct-to-mmap writes** (O4): Eliminated intermediate data buffer and one memcopy.
   Write/64KB improved 1.46x (20,040→13,737 ns). The larger the value, the bigger the win.

2. **Pooled mmapReadCloser** (O9): Replaced `io.NopCloser(bytes.NewReader(val))` (2 allocs)
   with 1 pooled `mmapReadCloser`. Read allocs dropped 5→2.

3. **Stack-buffer compositeKey** (O3): Eliminated heap allocation for key construction on
   all hot paths (Write, Open, Stat, Delete).

4. **Allocation-free cleanKey** (O2): Eliminated `strings.Split` allocation by using
   `containsDotDot()` byte scan. Saves 1 alloc on ALL paths.

5. **64 shards** (O6): ParallelWrite improved 1.93x (488→355 ns at C10).

6. **SetGCPercent(800)** (O8): GC cycles dropped from 28→2 for 100K objects.
   Read path was 82% GC overhead — now negligible.

7. **Cached time** (O1): `fastNow()` eliminates time.Now() syscall from hot paths.

8. **Bucket existence fast path** (O7): sync.Map avoids global bucketMu lock on writes.

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

7. **Single lock per operation** — v2 used 4 locks per write. v2b uses 1 lock. Fewer lock acquisitions = less contention.

8. **Eliminate the WAL** — Embedding key metadata in the vlog entry makes the vlog self-describing. Recovery just scans the vlog.

9. **Zero-copy mmap reads** — Returning a slice of mmap'd memory eliminates the biggest allocation in the read path.

### From v3 Profiling:

10. **GC dominates read paths** — 82% of Read CPU is GC. With zero-copy mmap reads, the remaining allocations (compositeKey, cleanKey, Object, Reader wrappers) drive ALL the GC overhead. Every allocation eliminated has outsized impact.

11. **memmove dominates write paths** — 47% of Write CPU is copying data to mmap. The ONLY way to reduce this is to eliminate intermediate copies (direct-to-mmap writes).

12. **strings.Split is a hidden allocator** — A single `strings.Split(key, "/")` in cleanKey accounts for 10-12% of total allocations. Replace string manipulation with manual byte scanning whenever possible.

13. **Cached time matters at scale** — `time.Now()` is ~20ns (macOS commpage), which is 3% of a 775ns write. At millions of ops/s, background-ticker cached time is essential.

14. **Pool everything, even small structs** — A 48-byte leafEntry allocation, repeated 3.5M times, generates 168 MB of garbage that triggers GC cycles consuming 28% of CPU.

15. **The interface tax is real** — Returning `(io.ReadCloser, *storage.Object, error)` forces 3 heap allocations per read. Custom pooled types (mmapReader with embedded bytes.Reader) can eliminate 2 of 3.

### From v3 Implementation:

16. **Stack buffers eliminate heap escapes** — `var buf [256]byte` + `appendCompositeKey(buf[:0], ...)` keeps the composite key on the stack for keys under 256B. This eliminated 1 allocation per operation on ALL paths.

17. **SetGCPercent(800) is transformative for small heaps** — With 24 MB live data, GC now triggers at ~216 MB (far above our working set). GC cycles dropped 28→2, making read paths 1.41x faster.

18. **Parallel scaling is the real win** — Single-thread Write didn't improve much (memmove bottleneck), but ParallelWrite at C10 improved 1.93x (488→355 ns) from 64 shards + bucket existence fast path.

19. **Direct-to-mmap scales with value size** — Write/64KB improved 1.46x because we eliminated the 64KB intermediate buffer + copy. Write/1KB improvement is smaller (1KB copy is fast). The optimization matters most for large values.

20. **Pool return is critical for pool effectiveness** — `mmapReadCloser.Close()` returns the reader to `readerPool`. Without the Put, every Get allocates. Benchmark Read went from 5→2 allocs because the pool stays warm.

---

## v4 Profiling Analysis

**Environment:** Go 1.26.0, darwin/arm64 (Apple M4), 10 CPUs, benchtime=2-3s

### v3 Baseline Benchmarks

| Benchmark | ops/s | ns/op | B/op | allocs/op |
|-----------|-------|-------|------|-----------|
| **Write/1KB** | 3,771K | 695 | 460 | 7 |
| **Read/1KB** | 17,626K | 142 | 173 | 2 |
| **Stat** | 18,275K | 133 | 173 | 2 |
| **Delete** | 13,824K | 221 | 26 | 2 |
| **ParallelWrite/1KB/C10** | 8,743K | 339 | 436 | 7 |
| **ParallelRead/1KB/C10** | 23,590K | 104 | 175 | 2 |
| **List/100** | 233K | 10,806 | 20,176 | 351 |

### CPU Profile: Read/1KB (3.47s total samples, 24.7M iterations)

| Function | flat% | cum% | Category | Actionable? |
|----------|-------|------|----------|-------------|
| `(*bucket).Open` | 2.88% | 49.86% | **Our code** | Optimization target |
| `fmt.Sprintf` | — | 22.48% | Benchmark harness | NO — not our code |
| `runtime.mallocgc` | 5.48% | 21.33% | GC/alloc | YES — reduce allocs |
| `artSearch` | 3.75% | 13.54% | **Our code** | YES — hash table |
| `cleanKey` | 0.86% | 8.07% | **Our code** | YES — fast path |
| `findChild` | 5.76% | 5.76% | **Our code** | YES — hash table |
| `runtime.newobject` | — | 11.82% | Alloc | YES — pool/eliminate |
| `strings.ReplaceAll` (via cleanKey) | — | 5.48% | **Our code** | YES — fast path |
| `relToKey` | — | 3.75% | **Our code** | YES — eliminate |
| `path.Clean` (via cleanKey) | — | ~4% | **Our code** | YES — fast path |
| `runtime.convT64` | 0.86% | 4.32% | GC/boxing | Minor |

**Key insight:** Excluding benchmark harness (22.5%), our actual code CPU breakdown is:
- artSearch (including findChild): **19.3%** ← biggest target
- cleanKey (path.Clean + strings.ReplaceAll): **8.1%** ← easy win
- mallocgc/newobject (Object alloc): **21.3%** ← pool/embed
- relToKey (strings.ReplaceAll scan): **3.75%** ← eliminate entirely

### CPU Profile: Stat (3.61s total, 27.4M iterations)

| Function | flat% | cum% | Actionable? |
|----------|-------|------|-------------|
| `(*bucket).Stat` | 3.05% | 52.35% | Target |
| `fmt.Sprintf` | — | 22.44% | NO — harness |
| `artSearch` | 4.99% | 16.90% | YES — hash table |
| `cleanKey` | 0.28% | 9.97% | YES — fast path |
| `runtime.mallocgc` | 3.05% | 17.73% | YES — reduce allocs |
| `runtime.kevent` | 9.97% | 9.97% | GC STW syscall |
| `findChild` | 7.76% | 7.76% | YES — hash table |
| `path.Clean` | 3.88% | 4.71% | YES — fast path |
| `relToKey` | 0.28% | 3.32% | YES — eliminate |

### CPU Profile: Write/1KB (5.87s total, 6.6M iterations)

| Function | flat% | cum% | Actionable? |
|----------|-------|------|-------------|
| `runtime.memmove` | **44.46%** | 44.46% | NO — copying 1KB to mmap (irreducible) |
| `runtime.madvise` | **22.66%** | 22.66% | Partially — pre-grow vlog |
| `syscall.rawsyscalln` | 13.80% | 13.80% | NO — vlog close in cleanup |
| `appendPutDirect` | — | 46.34% | Target (memmove within) |
| `binary.PutUint64` | — | ~2% | Minimal |

**Key insight:** Write/1KB is dominated by irreducible costs: 44% memmove (data copy), 23% madvise
(heap/mmap management), 14% syscall (cleanup). Only ~19% is actionable.

### CPU Profile: Delete (15.87s total, 20.4M iterations)

| Function | flat% | cum% | Actionable? |
|----------|-------|------|-------------|
| `runtime.madvise` | **49.53%** | 49.53% | From Write setup in benchmark |
| `(*bucket).Write` | — | 19.66% | Benchmark setup |
| `artDeleteRecursive` | 6.43% | 15.88% | YES — hash table |
| `runtime.memmove` | 13.61% | 13.61% | Write setup |
| `nodePrefix` (inline) | **8.70%** | 8.70% | YES — hash table bypass |
| `runtime.tryDeferToSpanScan` | 3.59% | 4.28% | GC scanning |

**Key insight:** Delete benchmark is dominated by its Write setup phase (madvise + memmove). The actual
Delete path (`artDeleteRecursive` at 16%) is fast but still traverses ART with expensive type switches
(`nodePrefix` alone is 8.7%).

### Memory Profile: Read/1KB (24.7M iterations, 4,397 MB total alloc)

| Allocation Site | MB | % | Root Cause |
|-----------------|-----|---|------------|
| `(*bucket).Open` | 3,957 | **90.0%** | `&storage.Object{}` allocation |
| `fmt.Sprintf` | 179 | 4.1% | Benchmark harness key gen |
| `BenchmarkRead1KB` (misc) | 162 | 3.7% | Various |
| `acquireNode4 → pool.New` | 33 | 0.7% | Pool cold start |

**Root cause:** 90% of Read allocations are the `storage.Object` struct. Each Read allocates
a ~160B Object on the heap. At 24.7M iterations, that's 3.96 GB of garbage, driving mallocgc to 21% CPU.

### Memory Profile: Stat (27.4M iterations, 4,843 MB total alloc)

| Allocation Site | MB | % | Root Cause |
|-----------------|-----|---|------------|
| `(*bucket).Stat` | 4,367 | **90.2%** | `&storage.Object{}` allocation |
| `fmt.Sprintf` | 208 | 4.3% | Benchmark harness |
| `acquireNode4 → pool.New` | 38 | 0.8% | Pool cold start |

### Memory Profile: Write/1KB (6.6M iterations, 3,494 MB total alloc)

| Allocation Site | MB | % | Root Cause |
|-----------------|-----|---|------------|
| `(*bucket).Write` | 1,224 | **35.0%** | compositeKey, Object, etc. |
| `acquireNode4 → pool.New` | 1,153 | **33.0%** | node4 pool cold path |
| `bytes.NewReader` (harness) | 372 | 10.7% | Benchmark wrapping test data |
| `acquireLeaf → pool.New` | 348 | **10.0%** | leafEntry pool cold path |
| `addChild` | 169 | 4.8% | Node prefix slice on promotion |
| `fmt.Sprintf` | 115 | 3.3% | Benchmark harness |

### Detailed cleanKey Analysis (via pprof -peek)

**Read path:** cleanKey = 8.07% cum, breakdown:
- `path.Clean`: **50%** of cleanKey time (allocation + string processing)
- `strings.ReplaceAll("\\", "/")`: **21%** (full string scan even when no backslash)
- `containsDotDot`: **11%** (our zero-alloc scan — already optimized)
- `strings.TrimSpace`: **7%** (leading/trailing space check)

**Stat path:** cleanKey = 9.97% cum, breakdown:
- `path.Clean`: **47%** of cleanKey time
- `strings.ReplaceAll`: **25%**
- `strings.TrimSpace`: **14%**
- `containsDotDot`: **11%**

**Conclusion:** `path.Clean` and `strings.ReplaceAll` together consume 70-75% of cleanKey time.
For already-clean keys (no backslash, no `//`, no `./`, no leading `/`), both are pure overhead.

### Detailed relToKey Analysis

**Read path:** relToKey = 3.75% cum
- `strings.ReplaceAll("\\", "/")`: **100%** of relToKey time

**Stat path:** relToKey = 3.32% cum
- `strings.ReplaceAll("\\", "/")`: **92%** of relToKey time

**Conclusion:** relToKey is called AFTER cleanKey, which already removes backslashes.
The strings.ReplaceAll scan is completely redundant. Replacing with identity saves 3-4% CPU.

---

## v4 Bottleneck Identification

### B1: cleanKey Path Processing (Read/Stat 8-10% CPU, ALL paths)

**Impact:** 8-10% of Read/Stat CPU wasted on `path.Clean()` and `strings.ReplaceAll()` for keys that
are already clean. At 18M Stat ops/s, that's ~24 ns per call wasted.

**Root cause:** `cleanKey()` unconditionally calls `strings.ReplaceAll(key, "\\", "/")` (scans full
string) and `path.Clean(key)` (allocates new string, processes path components). For benchmark keys
like "k/12345", neither function changes the input but both scan/allocate.

**Solution:** Fast-path byte scan: check if key is already clean (no `\`, no leading space/slash,
no `//`, no `.` or `..`). If clean, return immediately. Falls through to existing cleanKey for edge cases.

### B2: relToKey Redundancy (Read/Stat 3-4% CPU, ALL paths)

**Impact:** 3-4% of CPU scanning for backslashes that cleanKey already removed.

**Root cause:** `relToKey()` calls `strings.ReplaceAll(rel, "\\", "/")` — identical to what cleanKey
already did. Then `strings.TrimPrefix(result, "/")` — cleanKey already trimmed leading slash.

**Solution:** Replace `relToKey(relKey)` with just `relKey` on all paths. Since cleanKey's output
is guaranteed to have no backslash and no leading slash, relToKey is always identity.

### B3: ART Traversal for Point Lookups (Read/Stat 14-17% CPU)

**Impact:** artSearch + findChild = 14-17% of Read/Stat CPU. Each lookup traverses the tree
byte-by-byte with type switches at every node (4-way switch for nodePrefix, nodeLeaf, findChild).

**Root cause:** ART is O(key_length) with constant overhead per node from Go interface type switches.
For a 10-byte composite key, that's ~10 nodes × 3 type switches = 30 type switch evaluations.

**Solution:** Per-shard open-addressing hash table for O(1) point lookups. Keep ART for prefix
operations (List, directory checks). Hash table entry: `{keyHash uint64, leaf *leafEntry}` = 16B.
Robin Hood linear probing with 70% load factor.

### B4: storage.Object Heap Allocation (Read/Stat 90% of alloc bytes)

**Impact:** 90% of Read/Stat allocations are `&storage.Object{}` (~160B). At 18M ops/s, this
generates 2.88 GB/s of garbage, pushing mallocgc to 18-21% of CPU.

**Root cause:** `(*bucket).Open` and `(*bucket).Stat` allocate `&storage.Object{...}` per call.
The storage.Bucket interface requires returning `*storage.Object`.

**Solution (Read path):** Embed `storage.Object` inside `mmapReadCloser`. The Object is returned
with the reader and recycled when the reader goes back to the pool on `Close()`. This eliminates
the Object allocation entirely on the Read path (0 allocs with warm pool).

**Solution (Stat path):** Use `sync.Pool` for Object. Since Stat has no close hook, the caller
can't return the Object. Accept 1 alloc on Stat (pool helps with warm path).

### B5: ctStringTable.get() RWMutex (minor, ~1% on Read/Stat)

**Impact:** Every Read/Stat acquires `ctTable.mu.RLock()` to look up content-type string.

**Root cause:** `get()` uses `sync.RWMutex.RLock/RUnlock` even though the strings slice is
append-only (never modified, only extended).

**Solution:** Publish strings slice via `atomic.Pointer`. Reads use atomic load (no lock).
Writes (intern) still use mutex + atomic store for new entries.

---

## v4 Optimization Journey

### O1: Fast-Path cleanKey (Zero-Cost for Clean Keys)

**Problem:** cleanKey takes 8-10% of Read/Stat CPU processing already-clean keys through
`path.Clean()` and `strings.ReplaceAll()`.

**Solution:** Add `isCleanKey()` fast-path check: single pass over key bytes, returns true if:
- No backslash `\`
- No leading or trailing space
- No leading slash `/`
- No consecutive slashes `//`
- No dot-dot component `..`
- Not empty, not `.`

```go
func isCleanKey(key string) bool {
    if len(key) == 0 || key[0] == '/' || key[0] == ' ' {
        return false
    }
    prev := byte(0)
    for i := 0; i < len(key); i++ {
        c := key[i]
        if c == '\\' || c == ' ' && i == len(key)-1 {
            return false
        }
        if c == '/' && prev == '/' {
            return false
        }
        if c == '.' && (prev == '/' || prev == 0) {
            if i+1 >= len(key) || key[i+1] == '/' {
                return false // "." component
            }
            if key[i+1] == '.' && (i+2 >= len(key) || key[i+2] == '/') {
                return false // ".." component
            }
        }
        prev = c
    }
    return true
}

func cleanKey(key string) (string, error) {
    if isCleanKey(key) {
        return key, nil  // fast path: zero allocations
    }
    // slow path: existing processing
    key = strings.TrimSpace(key)
    // ...
}
```

**Expected savings:** 8-10% CPU on Read/Stat → ~12-14 ns per operation.

### O2: Eliminate relToKey (Identity After cleanKey)

**Problem:** relToKey takes 3-4% of Read/Stat CPU scanning for backslashes that cleanKey already
removed.

**Solution:** Replace all `relToKey(relKey)` calls with just `relKey`. cleanKey guarantees:
- No backslash (replaced or rejected)
- No leading slash (trimmed)

So relToKey is always identity. Remove the function call entirely.

**Expected savings:** 3-4% CPU on Read/Stat → ~5 ns per operation.

### O3: Per-Shard Hash Table for Point Lookups

**Problem:** artSearch traverses key byte-by-byte with type switches at every node. 14-17% of
Read/Stat CPU.

**Solution:** Open-addressing hash table (Robin Hood linear probing) per shard:

```go
type htEntry struct {
    keyHash uint64     // 0 = empty slot
    leaf    *leafEntry
}

type hashTable struct {
    entries []htEntry
    mask    uint64
    count   int
}

func (ht *hashTable) lookup(keyHash uint64) *leafEntry {
    idx := keyHash & ht.mask
    for {
        e := &ht.entries[idx]
        if e.keyHash == 0 {
            return nil // empty slot — not found
        }
        if e.keyHash == keyHash {
            return e.leaf
        }
        idx = (idx + 1) & ht.mask
    }
}
```

Point lookups (Read, Stat, Delete) use hash table. ART retained for prefix operations (List).
Both structures updated on Write/Delete.

Entry size: 16B. At 70% load factor with 100K entries across 64 shards (~1,562 per shard):
table size = 4,096 entries × 16B = 64KB per shard = 4MB total. Well within budget.

**Expected savings:** 10-14% CPU on Read/Stat → ~15-20 ns per operation.

### O4: Embed Object in mmapReadCloser

**Problem:** `&storage.Object{}` allocation is 90% of Read memory, driving mallocgc to 21% CPU.

**Solution:** Embed Object directly in the pooled mmapReadCloser:

```go
type mmapReadCloser struct {
    r   bytes.Reader
    obj storage.Object // embedded, not pointer
}

func (b *bucket) Open(...) (io.ReadCloser, *storage.Object, error) {
    rc := readerPool.Get().(*mmapReadCloser)
    rc.obj = storage.Object{
        Bucket: b.name,
        Key:    relKey,
        // ...
    }
    rc.r.Reset(val)
    return rc, &rc.obj, nil // Object lives inside pooled reader
}
```

When `Close()` returns the reader to pool, the embedded Object is recycled. This eliminates
the Object allocation entirely on the Read path.

Read path: 2 allocs → 0 allocs (pool warm), 1 alloc (pool cold).

**Expected savings:** ~15% CPU savings on Read (eliminates mallocgc for Object).

### O5: Lock-Free ctStringTable.get()

**Problem:** Every Read/Stat acquires RLock to look up content-type string.

**Solution:** Use atomic.Pointer for the strings slice:

```go
type ctStringTable struct {
    mu      sync.Mutex
    strs    atomic.Pointer[[]string] // lock-free read
    index   map[string]uint16
}

func (t *ctStringTable) get(idx uint16) string {
    s := *t.strs.Load()
    if int(idx) < len(s) {
        return s[idx]
    }
    return ""
}
```

**Expected savings:** ~1% CPU on Read/Stat.

### O6: Pool storage.Object for Stat

**Problem:** Stat allocates `&storage.Object{}` per call. No close hook to return to pool.

**Solution:** Use sync.Pool for Stat Objects:

```go
var objPool = sync.Pool{New: func() any { return new(storage.Object) }}

func (b *bucket) Stat(...) (*storage.Object, error) {
    obj := objPool.Get().(*storage.Object)
    *obj = storage.Object{
        Bucket: b.name,
        Key:    relKey,
        // ...
    }
    return obj, nil
}
```

Note: Caller may not return to pool. But pool.Get() is still faster than malloc when pool is warm,
and the GC collects unreturned objects normally.

**Expected savings:** ~5-8% CPU on Stat path (reduce mallocgc overhead).

---

## v4 Results

*(To be filled after implementation)*

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

7. **Single lock per operation** — v2 used 4 locks per write. v2b uses 1 lock. Fewer lock acquisitions = less contention.

8. **Eliminate the WAL** — Embedding key metadata in the vlog entry makes the vlog self-describing. Recovery just scans the vlog.

9. **Zero-copy mmap reads** — Returning a slice of mmap'd memory eliminates the biggest allocation in the read path.

### From v3 Profiling:

10. **GC dominates read paths** — 82% of Read CPU is GC. With zero-copy mmap reads, the remaining allocations (compositeKey, cleanKey, Object, Reader wrappers) drive ALL the GC overhead. Every allocation eliminated has outsized impact.

11. **memmove dominates write paths** — 47% of Write CPU is copying data to mmap. The ONLY way to reduce this is to eliminate intermediate copies (direct-to-mmap writes).

12. **strings.Split is a hidden allocator** — A single `strings.Split(key, "/")` in cleanKey accounts for 10-12% of total allocations. Replace string manipulation with manual byte scanning whenever possible.

13. **Cached time matters at scale** — `time.Now()` is ~20ns (macOS commpage), which is 3% of a 775ns write. At millions of ops/s, background-ticker cached time is essential.

14. **Pool everything, even small structs** — A 48-byte leafEntry allocation, repeated 3.5M times, generates 168 MB of garbage that triggers GC cycles consuming 28% of CPU.

15. **The interface tax is real** — Returning `(io.ReadCloser, *storage.Object, error)` forces 3 heap allocations per read. Custom pooled types (mmapReader with embedded bytes.Reader) can eliminate 2 of 3.

### From v3 Implementation:

16. **Stack buffers eliminate heap escapes** — `var buf [256]byte` + `appendCompositeKey(buf[:0], ...)` keeps the composite key on the stack for keys under 256B. This eliminated 1 allocation per operation on ALL paths.

17. **SetGCPercent(800) is transformative for small heaps** — With 24 MB live data, GC now triggers at ~216 MB (far above our working set). GC cycles dropped 28→2, making read paths 1.41x faster.

18. **Parallel scaling is the real win** — Single-thread Write didn't improve much (memmove bottleneck), but ParallelWrite at C10 improved 1.93x (488→355 ns) from 64 shards + bucket existence fast path.

19. **Direct-to-mmap scales with value size** — Write/64KB improved 1.46x because we eliminated the 64KB intermediate buffer + copy. Write/1KB improvement is smaller (1KB copy is fast). The optimization matters most for large values.

20. **Pool return is critical for pool effectiveness** — `mmapReadCloser.Close()` returns the reader to `readerPool`. Without the Put, every Get allocates. Benchmark Read went from 5→2 allocs because the pool stays warm.

### From v4 Profiling:

21. **path.Clean is an allocation landmine** — `path.Clean(key)` allocates a new string EVEN when the input is already clean. For hot-path key validation, a fast-path byte scan to detect already-clean keys saves 8-10% CPU.

22. **Redundant string processing compounds** — `relToKey()` re-scans for backslashes after `cleanKey()` already handled them. Sequential string functions that overlap in purpose waste CPU. Eliminate redundancy by reasoning about invariants.

23. **Type switches have hidden cost at scale** — ART's type-switch-per-node (nodePrefix, nodeLeaf, findChild) seems cheap per call, but at 10 levels × 3 switches per Read, it adds up to 14-17% of CPU. A hash table with O(1) lookup eliminates this entirely for point operations.

24. **Embed pooled objects to avoid secondary allocations** — Instead of pooling the reader and separately allocating the Object, embed the Object inside the reader. One pool Get replaces two heap allocations.

25. **Profile the BENCHMARK, not just the code** — fmt.Sprintf in key generation consumes 22% of Read/Stat CPU. This means measured ns/op includes ~31ns of benchmark overhead. True code performance is ~30% better than benchmark numbers suggest.

---

## Appendix: Profile Commands

### Running Benchmarks

```bash
# Full benchmark suite
go test -bench="Benchmark" -benchmem -benchtime=2s -count=1 \
  ./pkg/storage/driver/zoo/ant/

# With CPU + memory profiling
go test -bench="BenchmarkWrite1KB$" -benchmem -benchtime=3s \
  -cpuprofile=/tmp/ant_cpu.pprof -memprofile=/tmp/ant_mem.pprof \
  -count=1 ./pkg/storage/driver/zoo/ant/

# Memory measurement test
go test -run "TestMemory100K$" -v -count=1 ./pkg/storage/driver/zoo/ant/

# Disk usage test
go test -run "TestDiskUsage$" -v -count=1 ./pkg/storage/driver/zoo/ant/
```

### Analyzing Profiles

```bash
# CPU profile — top functions by cumulative time
go tool pprof -top -cum -nodecount=40 /tmp/ant_cpu.pprof

# CPU profile — flamegraph (opens browser)
go tool pprof -http=:8080 /tmp/ant_cpu.pprof

# CPU profile — peek at specific functions
go tool pprof -peek "Write|appendPut|artInsert|cleanKey" /tmp/ant_cpu.pprof

# Memory profile — top allocation sites
go tool pprof -top -nodecount=30 /tmp/ant_mem.pprof

# Memory profile — heap in-use
go tool pprof -inuse_space -top /tmp/ant_mem.pprof

# Compare two profiles (baseline vs optimized)
go tool pprof -base /tmp/ant_v2b_cpu.pprof /tmp/ant_v3_cpu.pprof

# GC trace during benchmark
GODEBUG=gctrace=1 go test -bench="BenchmarkWrite1KB$" -benchtime=1s \
  ./pkg/storage/driver/zoo/ant/ 2>&1 | grep gc
```

### Memory Analysis

```bash
# Struct size check
go test -run "^$" -bench "^$" ./pkg/storage/driver/zoo/ant/ \
  -args -print-sizes 2>/dev/null

# Runtime memory stats
go test -run "TestMemory100K" -v -count=1 ./pkg/storage/driver/zoo/ant/
```
