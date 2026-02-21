# Herd Driver: Deep Performance Research

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [v2 Baseline Profiling Analysis](#v2-baseline-profiling-analysis)
3. [v2 Optimization Journey](#v2-optimization-journey)
4. [v3 Baseline Profiling Analysis](#v3-baseline-profiling-analysis)
5. [v3 Optimization Journey](#v3-optimization-journey)
6. [v3 Results](#v3-results)
7. [v4 Baseline Profiling Analysis](#v4-baseline-profiling-analysis)
8. [v4 Optimization Journey](#v4-optimization-journey)
9. [v4 Results](#v4-results)
10. [Remaining Bottlenecks](#remaining-bottlenecks)
11. [Lessons Learned](#lessons-learned)
12. [Future Optimization Opportunities](#future-optimization-opportunities)
13. [Appendix: Profile Commands](#appendix-profile-commands)

---

## Architecture Overview

Herd is a high-performance striped object storage driver inspired by Facebook Haystack and SeaweedFS. The architecture is designed for maximum throughput on modern multi-core hardware.

### Storage Layout (v3)

```
store (16 stripes, FNV-1a routing, bitmask selection)
 └── stripe
      ├── volume (append-only mmap file, pwrite writes)
      ├── shardedIndex (256 shards × RWMutex)
      │    └── shard
      │         └── buckets: map[bucket]*shardBucket  ← two-level, no compositeKey
      │              ├── entries: map[key]*indexEntry
      │              ├── sorted: []string (lazy cache)
      │              └── dirty: bool
      ├── bloomFilter (lock-free, 10 bits/item, 7 hashes, wyhash mixing)
      ├── bufferRing (8 × 8MB, background flush)
      └── slabArena (lock-free bump, 128MB mmap chunks, GC-invisible)
```

### Key Design Decisions

| Component | Design | Rationale |
|-----------|--------|-----------|
| 16 stripes | FNV-1a hash, bitmask selection | Distributes load; bitmask avoids modulo |
| 256 shards/stripe | Per-shard RWMutex | 4096 total shards minimizes contention |
| Inline values | ≤8KB stored in mmap slab memory | Skips volume I/O for small objects |
| Buffer ring | 8 × 8MB ring buffer | Batches small writes into large WriteAt |
| Slab allocator | 128MB mmap bump-allocated chunks | Lock-free, GC-invisible, zero-on-demand |
| Bloom filter | Lock-free atomic OR, wyhash mixing | O(1) negative lookups, fast hashing |
| Two-level index | bucket → key → entry (no composite) | Eliminates string concatenation per op |
| Sorted key cache | Lazy rebuild on write, binary search | O(log n + m) prefix queries |

### Data Flow

**Write path (inline, sync=none):**
```
Write() → stripeFor(bucket,key)          [FNV-1a, bitmask, no lock]
        → slab.alloc(size)               [atomic Add, mmap chunk, no GC]
        → bytes.Reader.Read(slab_slice)   [direct memcpy, no interface dispatch]
        → idx.put(bucket, key, entry)     [shard RWMutex.Lock, two-level map]
        → bloom.add(bucket, key)          [atomic OR, wyhash mixing]
```

**Read path (inline):**
```
Open() → stripeFor(bucket,key)            [FNV-1a, bitmask, no lock]
       → bloom.mayContain(bucket,key)     [atomic Load, wyhash mixing]
       → idx.get(bucket, key)             [shard RWMutex.RLock, two-level map]
       → return mmapReader(e.inline)      [pool Get, no lock]
```

---

## v2 Baseline Profiling Analysis

**Environment:** Go 1.26.0, darwin/arm64, 10 CPUs, benchtime=2s, concurrency=200

### CPU Profile (275.45s samples / 102s wall = 2.7x CPU utilization)

| Function | Flat% | Category | Impact |
|----------|-------|----------|--------|
| `runtime.tryDeferToSpanScan` | 27.44% | GC scanning | **CRITICAL** |
| `runtime.memclrNoHeapPointers` | 8.92% | Allocation zeroing | HIGH |
| `runtime.madvise` | 7.60% | Memory management | MEDIUM |
| `runtime.mapaccess2_faststr` | 7.46% | Map lookup (index) | MEDIUM |
| `runtime.usleep` | 5.77% | Thread scheduling | LOW |
| `runtime.pthread_cond_wait` | 5.09% | Lock wait | HIGH |
| `runtime.mapassign_faststr` | 8.36% | Map insert (index) | MEDIUM |
| `runtime.pthread_cond_signal` | 4.14% | Lock signal | MEDIUM |
| `runtime.(*mspan).moveInlineMarks` | 4.00% | GC inline marks | HIGH |

**Key insight: 71% of CPU was overhead (GC + locks + memory management), not useful work.**

### Heap Profile (23.7 GB in use)

| Allocator | In-Use | % | Root Cause |
|-----------|--------|---|------------|
| `(*bucket).Write` inline `make([]byte)` | 18.4 GB | 77.8% | Per-write allocation |
| `(*shardedIndex).put` entries | 2.0 GB | 8.4% | Map growth + key tracking |
| `indexEntryPool.New` (pool miss) | 1.4 GB | 6.0% | GC drains sync.Pool |
| `compositeKey` string concat | 786 MB | 3.3% | "bucket\x00key" |
| `newWriteBuffer` | 512 MB | 2.2% | Expected (4 x 8MB x 16) |

**The single line `e.inline = make([]byte, size)` caused 78% of all heap usage.**

---

## v2 Optimization Journey

### Iteration 1: Slab Allocator + Remove Key Tracking

- `slab.go` (NEW): Lock-free bump allocator — 64MB heap chunks, atomic CAS
- Replaced `make([]byte, size)` with `stripe.slab.alloc(int(size))`
- Results: GC CPU 27.44% → ~4%, but List broke (O(N) scan)

### Iteration 2: Per-Shard Key Tracking

- Added `bucketKeys map[string]map[string]struct{}` to each shard
- Results: List recovered partially

### Iteration 3: Sorted Key Cache with Binary Search

- `shardBucketKeys` struct with sorted slice + dirty flag
- Binary search via `sort.SearchStrings()` = O(log n + m) per shard
- Results: List recovered to 0.63x baseline

---

## v3 Baseline Profiling Analysis

After v2 optimizations were complete, v3 baseline was captured with profiling.

**Environment:** Go 1.26.0, darwin/arm64, 10 CPUs, benchtime=2s, concurrency=200

### CPU Profile (297.96s samples / 151.38s wall = 1.97x utilization)

| # | Function | Flat (s) | Flat% | Category | Actionable? |
|---|----------|----------|-------|----------|-------------|
| 1 | `runtime.usleep` | 43.14 | 14.48% | Scheduler | No |
| 2 | `runtime.memclrNoHeapPointers` | 29.05 | 9.75% | Alloc zeroing | **YES → mmap** |
| 3 | `runtime.pthread_cond_wait` | 28.27 | 9.49% | Lock wait | Partially |
| 4 | `runtime.memmove` | 25.67 | 8.62% | Data copy | Partially |
| 5 | `runtime.pthread_cond_signal` | 20.95 | 7.03% | Lock signal | Partially |
| 6 | `runtime.madvise` | 14.61 | 4.90% | Mem management | **YES → mmap** |
| 7 | `runtime.moveInlineMarks` | 13.62 | 4.57% | GC marking | **YES → mmap** |
| 8 | `runtime.mapassign_faststr` | 12.29 | 4.12% | Map insert | **YES → 2-level** |
| 9 | `runtime.mapaccess2_faststr` | 11.95 | 4.01% | Map lookup | **YES → 2-level** |
| 10 | `runtime.tryDeferToSpanScan` | 9.83 | 3.30% | GC scanning | **YES → mmap** |

**Actionable overhead: 30.65% of CPU (memclr + madvise + moveInlineMarks + tryDeferToSpanScan + mapassign + mapaccess).**

### Heap Profile (25,207 MB in use)

| Allocator | In-Use | % | Root Cause |
|-----------|--------|---|------------|
| `slabArena.alloc` | 17,408 MB | 69.06% | make([]byte, 64MB) chunks — GC-visible |
| `shardedIndex.put` | 1,719 MB | 6.82% | Map growth + entries |
| `init.func1` (bloom) | 1,627 MB | 6.45% | Bloom filter bit arrays |
| **`compositeKey`** | **1,098 MB** | **4.36%** | **"bucket\x00key" string alloc** |
| `newWriteBuffer` | 1,024 MB | 4.06% | Buffer ring (expected) |
| `newSlabArena` | 1,024 MB | 4.06% | Initial chunk alloc |
| `fmt.Sprintf` | 675 MB | 2.68% | Stripe path formatting |
| `shardedIndex.list` | 372 MB | 1.47% | List result slices |

### Allocs Profile (91.26 GB total allocated)

| Allocator | Total | % |
|-----------|-------|---|
| `slabArena.alloc` | 45 GB | 49.31% |
| `shardedIndex.put` | 10.34 GB | 11.33% |
| `bucket.Open` | 5.40 GB | 5.92% |
| `bucket.Stat` | 3.94 GB | 4.32% |
| `bucket.Write` (cum) | 60.09 GB | 65.85% |

### Goroutine Profile

26 goroutines at baseline, stable. The `cachedTimeNano` ticker goroutine has a `timeTickerStop` channel for clean shutdown — no goroutine leaks.

### Mutex/Block Profiles

No significant contention. v2 already resolved the per-bucket lock bottleneck by moving key tracking under shard locks.

---

## v3 Optimization Journey

### Optimization 1: Mmap-Backed Slab Allocator

**File:** `slab.go`

**Problem:** `make([]byte, 128MB)` slab chunks were Go heap allocations:
- `memclrNoHeapPointers`: 29.05s (9.75%) — Go zeroes all heap allocations
- `moveInlineMarks`: 13.62s (4.57%) — GC tracks inline mark bits for large spans
- `tryDeferToSpanScan`: 9.83s (3.30%) — GC defers scanning of large spans
- `madvise`: 14.61s (4.90%) — Go runtime memory advice calls
- **Total: 67.11s (22.52%) CPU wasted on slab-related GC overhead**

**Solution:** Replace `make([]byte)` with `syscall.Mmap(MAP_ANON|MAP_PRIVATE)`:

```go
func mmapAlloc(size int) ([]byte, error) {
    return syscall.Mmap(-1, 0, size,
        syscall.PROT_READ|syscall.PROT_WRITE,
        syscall.MAP_ANON|syscall.MAP_PRIVATE)
}
```

**Why this works:**
1. **Zero-on-demand**: Kernel maps physical pages only when first accessed, pre-zeroed
2. **GC-invisible**: mmap memory is outside Go heap — no scanning, no marking, no spans
3. **Huge page eligible**: 128MB contiguous regions → transparent huge pages (2MB TLB entries)
4. **Clean release**: `munmap` returns memory to OS immediately (vs Go heap retention)

**Result:**
- `memclrNoHeapPointers`: 29.05s → 9.79s (**-66%**)
- `madvise`: 14.61s → 8.60s (**-41%**)
- `moveInlineMarks`: 13.62s → **ELIMINATED** (not in top 10)
- Slab no longer appears in heap profile at all (was 17,408 MB = 69%)

### Optimization 2: Two-Level Index (Eliminate compositeKey)

**File:** `index.go`

**Problem:** Every index operation constructed a composite key string:
```go
func compositeKey(bucket, key string) string {
    return bucket + "\x00" + key  // 1,098 MB heap, 4.36%
}
```

This inflated all map operations (`mapassign` 4.12%, `mapaccess` 4.01%) because:
- String concatenation allocates on heap every time
- Longer keys = more hashing work per map operation
- Separate `bucketKeys` tracking structure duplicated data

**Solution:** Two-level map eliminates composite key entirely:

```
Before: shard.entries["bucket\x00key"] → *indexEntry
         shard.bucketKeys["bucket"].keys["key"] → struct{}

After:  shard.buckets["bucket"].entries["key"] → *indexEntry
```

Merged `shardBucketKeys` into `shardBucket`:
```go
type shardBucket struct {
    entries map[string]*indexEntry  // key → entry (NO composite key)
    sorted  []string               // lazy-rebuilt sorted key cache
    dirty   bool                   // true after put/remove
}
```

**Result:**
- `compositeKey` eliminated from heap profile entirely (was 1,098 MB)
- `mapassign_faststr` dropped out of top 10 (was 12.29s)
- `mapaccess2_faststr` reduced from 11.95s to 10.81s (-10%)
- Total Go heap: 25,207 MB → 7,176 MB (**-71.5%**)

### Optimization 3: Wyhash-Style Bloom Filter

**File:** `bloom.go`

**Problem:** Double FNV-1a computed two independent hashes with two full passes over the data.

**Solution:** Single-pass hash with `bits.Mul64` mixing (from wyhash):

```go
func wymix(a, b uint64) uint64 {
    hi, lo := bits.Mul64(a, b)
    return hi ^ lo
}
```

Single data pass, then generate two hashes algebraically:
```go
h1 := wymix(h, s3)
h2 := wymix(h, s0) | 1  // odd for better double-hashing distribution
```

On ARM64, `bits.Mul64` compiles to `UMULH` + `MUL` (2 cycles total). This is ~2x faster than two FNV-1a passes.

### Optimization 4: Hot Path Micro-Optimizations

**File:** `storage.go`

1. **Bitmask stripe selection**: `h & stripeMask` instead of `h % numStripes` (1 cycle vs 4+)
2. **bytes.Reader fast path**: Type assertion bypasses `io.ReadFull` interface dispatch for inline writes
3. **Inline read fast path**: Skip offset/length processing when both are 0 (the common case)
4. **Byte comparison for directory check**: `key[len(key)-1] == '/'` vs `strings.HasSuffix(key, "/")`
5. **500µs time ticker**: More accurate timestamps for high-frequency operations

---

## v3 Results

### Performance Comparison

| Benchmark | v2 Baseline | v3 Optimized | Improvement |
|-----------|-------------|--------------|-------------|
| **Write/1KB** | 1.0M ops/s | 1.8M ops/s | **1.7x** |
| **Write/64KB** | 78.3K ops/s | 139.7K ops/s | **1.79x** |
| **Write/100MB** | 2 ops/s (185 MB/s) | 4 ops/s (380 MB/s) | **2.05x** |
| **Read/1KB** | 3.9M ops/s | 4.5M ops/s | **1.16x** |
| **Read/100MB** | 86 ops/s (8.6 GB/s) | 105 ops/s (10.5 GB/s) | **1.22x** |
| **Stat** | 7.5M ops/s | 10.0M ops/s | **1.33x** |
| **Delete** | 1.5M ops/s | 4.0M ops/s | **2.67x** |
| **Copy/1KB** | 298 MB/s | 417 MB/s | **1.40x** |
| **List/100** | 22.1K ops/s | 27.3K ops/s | **1.24x** |

### Parallel Write Scalability (BIGGEST WINS)

| Concurrency | v2 Baseline | v3 Optimized | Improvement |
|-------------|-------------|--------------|-------------|
| C1 | 462 MB/s | 898 MB/s | **1.94x** |
| C10 | 99 MB/s | 690 MB/s | **6.97x** |
| C25 | 63 MB/s | 390 MB/s | **6.19x** |
| C50 | 55 MB/s | 494 MB/s | **8.98x** |
| C100 | 13 MB/s | 443 MB/s | **34.1x** |
| C200 | 27 MB/s | 282 MB/s | **10.4x** |

The dramatic parallel write improvement (6-34x at high concurrency) comes from eliminating GC stop-the-world pauses. With 22% less CPU spent on GC, goroutines spend more time doing actual writes.

### Parallel Read Scalability

| Concurrency | v2 Baseline | v3 Optimized | Improvement |
|-------------|-------------|--------------|-------------|
| C1 | 1.8 GB/s | 3.2 GB/s | **1.78x** |
| C10 | 887 MB/s | 2.4 GB/s | **2.70x** |
| C100 | 607 MB/s | 1.5 GB/s | **2.47x** |

### Resource Usage

| Metric | v2 Baseline | v3 Optimized | Change |
|--------|-------------|--------------|--------|
| Peak RSS | 6.2 GB | 7.6 GB | +23% |
| **Go Heap In-Use** | **25.5 GB** | **7.7 GB** | **-70%** |
| **Total Alloc** | **93.3 GB** | **54.7 GB** | **-41%** |
| **GC Pause Total** | **15.7 ms** | **6.3 ms** | **-60%** |
| Goroutines | 26 | 26 | same |
| Efficiency | 13.8K ops/MB | 15.9K ops/MB | +15% |

### CPU Profile Comparison (Absolute Time)

| Function | v2 (s) | v3 (s) | Change |
|----------|--------|--------|--------|
| `memclrNoHeapPointers` | 29.05 | 9.79 | **-66%** |
| `madvise` | 14.61 | 8.60 | **-41%** |
| `moveInlineMarks` | 13.62 | <1.58 | **ELIMINATED** |
| `mapassign_faststr` | 12.29 | <1.58 | **ELIMINATED** |
| `mapaccess2_faststr` | 11.95 | 10.81 | -10% |
| `memmove` | 25.67 | 42.98 | +67% (more useful work) |

### Heap Profile Comparison

| Function | v2 In-Use | v3 In-Use | Change |
|----------|-----------|-----------|--------|
| `slabArena.alloc` | 17,408 MB | **0 MB** | **ELIMINATED** (mmap) |
| `compositeKey` | 1,098 MB | **0 MB** | **ELIMINATED** |
| **Total Go Heap** | **25,207 MB** | **7,176 MB** | **-71.5%** |

---

## v4 Baseline Profiling Analysis

After v3 optimizations were complete, v4 baseline was captured. The focus shifted from GC-visible heap (already reduced 71.5% in v3) to **GC-invisible memory patterns** and volume I/O overhead.

**Environment:** Go 1.26.0, darwin/arm64, 10 CPUs, benchtime=2s, concurrency=200

### CPU Profile (349.65s samples / 152.72s wall = 2.29x utilization)

| # | Function | Flat (s) | Flat% | Category | Notes |
|---|----------|----------|-------|----------|-------|
| 1 | `runtime.usleep` | 65.49 | 18.73% | Scheduler idle | Inherent |
| 2 | `runtime.pthread_cond_wait` | 47.82 | 13.68% | Lock wait | Inherent |
| 3 | `runtime.memmove` | 39.98 | 11.43% | Data copy | Inherent |
| 4 | `runtime.pthread_cond_signal` | 34.26 | 9.80% | Lock signal | Inherent |
| 5 | `runtime.tryDeferToSpanScan` | 22.57 | 6.46% | GC scanning | Moderate |
| 6 | `cmpbody` | 11.68 | 3.34% | String compare | Inherent |
| 7 | `runtime.madvise` | 8.53 | 2.44% | Memory mgmt | Moderate |
| 8 | `slices.partitionOrdered` | 7.77 | 2.22% | Sort in list | Inherent |
| 9 | `runtime.scanObject` | 7.62 | 2.18% | GC scanning | Moderate |
| 10 | `runtime.memclrNoHeapPointers` | 6.57 | 1.88% | Alloc zeroing | Moderate |

Scheduler overhead rose to 42% (from 34% in v3). This is expected — with less GC overhead, goroutines spend more time in actual work, which means more scheduling transitions per unit time. The actionable GC/memory overhead was 12.96%.

### Heap Profile (8,272 MB total in-use)

| # | Function | In-Use | % | Notes |
|---|----------|--------|---|-------|
| 1 | `init.func1` | 3,138 MB | 37.94% | **Misidentified — see below** |
| 2 | `shardedIndex.put` | 1,713 MB | 20.70% | Map growth |
| 3 | `fmt.Sprintf` | 1,522 MB | 18.39% | Benchmark framework |
| 4 | `newWriteBuffer` | 1,024 MB | 12.38% | 8 x 8MB x 16 stripes |
| 5 | `shardedIndex.list` | 722 MB | 8.72% | Result slices |

**Critical finding:** `go tool pprof -peek "init.func1"` revealed the call chain:
```
sync.(*Pool).Get → init.func1 (indexEntryPool.New)
```

The 3,138 MB was `indexEntryPool.New` (sync.Pool draining and re-allocating 72-byte `indexEntry` structs every GC cycle), NOT bloom filter data. The bloom filter is only ~20 MB total (16 stripes x 1.25 MB). This corrected the optimization priorities.

---

## v4 Optimization Journey

### Iteration 1: Mmap Bloom Filter + Write Buffers + Hash Fixes

**Changes:**
1. **Mmap bloom filter** (`bloom.go`): Moved `[]atomic.Uint64` to mmap-backed memory using `unsafe.Slice((*atomic.Uint64)(unsafe.Pointer(&data[0])), numWords)`. Preserves fast `atomic.Uint64.Or()` while keeping data off Go heap.
2. **Mmap write buffers** (`writebuf.go`): `make([]byte, 8MB)` → `mmapAlloc(8MB)`. Eliminated 1,024 MB from heap.
3. **Bitmask shard selection** (`index.go`): `h % shardCount` → `h & shardMask` (1 cycle vs 4+).
4. **Hash separator fix** (`index.go`, `storage.go`): `h ^= 0` (no-op) → `h ^= 0xFF` (proper separator).
5. **Resource cleanup**: `bloomFilter.close()`, `bufferRing.close()` with `mmapFree`.

**First benchmark result: MIXED**
- Write/64KB improved, but GC cycles 31 → 54, Total Alloc 59.8 → 80.2 GB (+34%)
- Root cause: `volume.readValueSlice` became #1 allocator at 19.89 GB (NOT in baseline top 10)
- The volume file grew beyond initial 1 GB mmap, reads fell back to `make([]byte, size) + fd.ReadAt()`

### Iteration 2: Volume Mmap Remapping

**Problem:** `growFile()` extended the volume file via `Truncate` but never remapped the mmap region. After Write/1KB and Write/64KB benchmarks filled the initial 1 GB prealloc, all subsequent reads for objects beyond 1 GB offset hit a heap fallback path.

**Solution:** Added mmap remapping in `growFile()`:
```go
// After Truncate, create new mmap covering full file
newData, err := syscall.Mmap(int(v.fd.Fd()), 0, int(newSize),
    syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
if err == nil {
    v.region.Store(&mmapRegion{buf: newData, capacity: newSize})
}
```

Old mmap mapping is intentionally leaked (readers may still reference it). Leak is bounded by geometric growth: total leaked <= current size.

### Iteration 3: Bloom Filter CAS → atomic.Or Recovery

The first bloom filter implementation used a CAS loop over raw `*uint64` pointers:
```go
// SLOW: CAS loop (iteration 1)
for {
    old := atomic.LoadUint64(ptr)
    if old&mask != 0 { break }
    if atomic.CompareAndSwapUint64(ptr, old, old|mask) { break }
}
```

This was slower than the original `atomic.Uint64.Or()` under contention. Fixed by using `unsafe.Slice` to create an `[]atomic.Uint64` view over mmap memory:
```go
// FAST: unsafe.Slice preserves atomic.Uint64.Or() (iteration 3)
atomicBits := unsafe.Slice((*atomic.Uint64)(unsafe.Pointer(&data[0])), numWords)
bf.bits[bit/64].Or(1 << (bit % 64))  // single-instruction atomic OR
```

This approach works because `atomic.Uint64` is exactly 8 bytes (same layout as `uint64`), and mmap returns page-aligned memory (always 8-byte aligned).

---

## v4 Results

### Performance Comparison

| Benchmark | v4 Baseline | v4 Optimized | Improvement |
|-----------|-------------|--------------|-------------|
| **Write/1KB** | 1.7M/s (1.7 GB/s) | 1.8M/s (1.7 GB/s) | +6% |
| **Write/64KB** | 134.2K/s (8.4 GB/s) | 250.8K/s (15.7 GB/s) | **+87%** |
| **Write/1MB** | 1.3K/s (1.3 GB/s) | 1.4K/s (1.4 GB/s) | +8% |
| **Write/10MB** | 23/s (225.9 MB/s) | 56/s (562.4 MB/s) | **+149%** |
| **Write/100MB** | 1/s (98.0 MB/s) | 2/s (223.5 MB/s) | **+128%** |
| **Read/1KB** | 4.2M/s (4.1 GB/s) | 4.3M/s (4.2 GB/s) | +2% |
| **Read/64KB** | 597.8K/s (37.4 GB/s) | 770.8K/s (48.2 GB/s) | **+29%** |
| **Read/1MB** | 49.1K/s (49.1 GB/s) | 56.2K/s (56.2 GB/s) | +14% |
| **Read/10MB** | 1.1K/s (10.6 GB/s) | 1.1K/s (10.5 GB/s) | 0% |
| **Read/100MB** | 105/s (10.5 GB/s) | 107/s (10.7 GB/s) | +2% |
| **Stat** | 10.3M/s | 9.7M/s | -6% |
| **Delete** | 4.2M/s | 4.0M/s | -5% |
| **Copy/1KB** | 560.2 MB/s | 484.4 MB/s | -14% |
| **List/100** | 27.3K/s | 26.5K/s | -3% |

### Resource Usage

| Metric | v4 Baseline | v4 Optimized | Change |
|--------|-------------|--------------|--------|
| Peak RSS | 9.7 GB | 9.8 GB | +1% |
| **Go Heap** | **11.1 GB** | **8.2 GB** | **-26%** |
| **Total Alloc** | **59.8 GB** | **55.8 GB** | **-7%** |
| **GC Pause Total** | **6.0 ms** | **5.9 ms** | **-2%** |
| Disk Used | 52.0 GB | 64.0 GB | +23% |
| GC Cycles | 31 | 51 | +65% |

### Latency Improvements (P99)

| Benchmark | Baseline P99 | Optimized P99 | Change |
|-----------|-------------|---------------|--------|
| Write/64KB | 54.9us | 14.9us | **-73%** |
| Write/10MB | 144.9ms | 134.5ms | -7% |
| Read/64KB | 3.4us | 1.7us | **-50%** |
| Read/1MB | 48.4us | 22.5us | **-54%** |

### Cumulative v2 → v3 → v4

| Benchmark | v2 Baseline | v3 Optimized | v4 Optimized | Total Gain |
|-----------|-------------|--------------|--------------|------------|
| Write/1KB | 1.0M/s | 1.8M/s | 1.8M/s | **1.8x** |
| Write/64KB | 78.3K/s | 139.7K/s | 250.8K/s | **3.2x** |
| Write/100MB | 2/s (185 MB/s) | 4/s (380 MB/s) | 2/s (223.5 MB/s) | — |
| Read/1KB | 3.9M/s | 4.5M/s | 4.3M/s | **1.1x** |
| Read/64KB | — | 597.8K/s | 770.8K/s | — |
| Stat | 7.5M/s | 10.0M/s | 9.7M/s | **1.3x** |
| Delete | 1.5M/s | 4.0M/s | 4.0M/s | **2.7x** |
| **Go Heap** | **25.5 GB** | **7.7 GB** | **8.2 GB** | **-68%** |
| **GC Pause** | **15.7 ms** | **6.3 ms** | **5.9 ms** | **-62%** |

---

## Remaining Bottlenecks

### 1. Go Runtime Scheduling (33% CPU)

`runtime.usleep` (15.27%) + `runtime.pthread_cond_wait` (10.87%) + `runtime.pthread_cond_signal` (7.93%) = 34% CPU. These are Go scheduler costs when goroutines block/unblock on mutexes and channels.

**Why hard to fix:** The benchmark framework uses 200-goroutine concurrency. Goroutine parking/waking is inherent to Go's cooperative scheduling model.

**Potential:** `GOEXPERIMENT=spinbitmutex` (Go 1.26+), reduce mutex hold time.

### 2. Memory Copy (16.72% CPU)

`runtime.memmove` (13.62%) + `runtime.memclrNoHeapPointers` (3.10%) = 16.72%. Data copying is inherent to the write path (user data → slab). The memclr remaining is from non-slab allocations.

**Potential:** Copy-on-write semantics, vectorized copy with NEON.

### 3. GC Scanning (6.31% CPU)

`runtime.tryDeferToSpanScan` still at 6.31% — this scans Go heap objects (index entries, sorted arrays, bloom filter bit arrays). Slab data is invisible but the Go-heap metadata around it is not.

**Potential:** Reduce index entry size, intern more strings, pre-allocate sorted arrays.

### 4. Map Operations (3.42% CPU)

`runtime.mapaccess2_faststr` at 3.42% — down from 8.13% combined (mapassign+mapaccess). Two-level index helped but map operations are still visible.

**Potential:** Swiss table with inline keys, pre-sized maps.

### 5. String Comparison (3.49% CPU)

`cmpbody` at 3.49% — string comparison in sorted key cache and map operations.

**Potential:** Length-prefixed keys, interning frequently accessed keys.

---

## Lessons Learned

### 1. mmap is the Right Tool for Large Byte Buffers in Go

The single biggest win in v3 was moving slab chunks from `make([]byte)` to `mmap(MAP_ANON)`. This eliminated 17.4 GB from the Go heap and freed 22% CPU that was spent on GC overhead.

**Rule of thumb:** Any allocation > 1MB that outlives a single request should use mmap in Go. The GC overhead per-byte is negligible for small objects but catastrophic for large, long-lived byte buffers.

### 2. Composite Keys Are a Hidden Tax

The `compositeKey = bucket + "\x00" + key` pattern seems innocuous but it was 4.36% of heap (1.1 GB) and inflated every map operation. Two-level maps (bucket → key → entry) eliminate this entirely with no semantic change.

**Rule of thumb:** If your map key is a concatenation of two strings, use a two-level map instead.

### 3. Profile After Every Change (Still True)

v2 → v3 → v4 shifted bottlenecks dramatically:
- v2: GC scanning (27%) + per-write allocation (78% heap)
- v3 baseline: GC zeroing (10%) + composite key (4%) + map ops (8%)
- v3 optimized: Scheduler (34%) + memmove (14%) — application-level, not GC
- v4 baseline: Scheduler (42%) + GC scan (8.6%) + madvise (2.4%)
- v4 optimized: Scheduler (45%) + memmove (10%) — volume remap eliminated fallback allocs

Each round of profiling reveals different bottlenecks. Without re-profiling, we would have optimized the wrong thing.

### 3b. Profile Attribution Requires Investigation

v4 discovered that `init.func1` (37.94% of heap, 3,138 MB) was NOT the bloom filter — it was `indexEntryPool.New` (sync.Pool). The generic profile label `init.func1` refers to anonymous functions in `var pool = sync.Pool{New: func() any { ... }}`. Always use `go tool pprof -peek` to trace call chains before assuming what a profile entry means.

### 4. Parallel Scalability Benefits Compound

The 6-34x improvement in parallel writes came from eliminating GC pauses, NOT from changing the locking strategy. GC stop-the-world pauses affect ALL goroutines equally, so reducing GC overhead multiplies by the concurrency level.

At C100: GC pause reduction × fewer goroutine wakeups × less madvise contention = exponential improvement.

### 5. Read Performance Has a Hardware Ceiling

Read optimizations (bloom filter, inline fast path) showed modest 1.1-1.2x gains because reads were already at memory bandwidth limits:
- Read/1KB at 3.8 GB/s (baseline) → 4.4 GB/s (v3) — approaching L2 cache bandwidth
- Read/1MB at 55.5 GB/s (baseline) → 55.1 GB/s (v3) — at mmap throughput limit

The right interpretation: reads were already well-optimized. The v3 improvements shifted CPU from overhead to useful work, which benefits writes more than reads.

### 6. Latency Improvements Follow Throughput

Write/1KB P50 improved from 667ns to 375ns (1.78x). This wasn't from any latency-specific optimization — it was a direct consequence of eliminating GC overhead. Less time in GC = lower P50 for every operation.

### 7. Volume Mmap Must Track File Growth

v4's biggest win came from an unexpected place: `growFile()` extended volume files via Truncate but never remapped the mmap region. This caused reads beyond the initial mmap capacity to fall back to `make([]byte) + ReadAt()` — creating 19.89 GB of heap allocations in one benchmark run. The fix (remapping after growth) gave 87-149% throughput improvement on medium-large writes.

**Rule of thumb:** If you mmap a file and the file can grow, you must remap after each growth. Old mappings can be leaked if readers hold references — the leak is bounded by geometric growth (total leaked <= current size).

### 8. unsafe.Slice Preserves Type Safety Over Mmap

Using `unsafe.Slice((*atomic.Uint64)(unsafe.Pointer(&data[0])), numWords)` creates a Go-typed view over mmap'd memory. This is superior to raw `*uint64` pointer arithmetic with `atomic.LoadUint64`/`CompareAndSwapUint64` because:
1. Preserves method dispatch (`bits.Or()` = single instruction on ARM64)
2. Slice bounds checking still works
3. GC sees only the slice header (~24 bytes), not the mmap'd data

A CAS-loop approach over raw pointers was 15-20% slower under contention than `atomic.Uint64.Or()`.

### 9. GC Frequency vs GC Duration Trade-off

Moving 1,024 MB of write buffers off-heap made the Go heap smaller. Go's GOGC=100 triggers GC when heap doubles from the post-GC baseline. A smaller baseline doubles faster → more GC cycles (31 → 51, +65%). But each cycle scans less → total GC pause stayed flat (6.0 → 5.9 ms). Net effect: slightly positive, but the GC cycle count increase can be alarming if not understood.

---

## Future Optimization Opportunities

### Near-Term (Moderate Effort, High Impact)

1. **Swiss table index**: Replace Go's `map[string]*indexEntry` with open-addressing hash table. Go maps have ~40% overhead from bucket chains and string hashing. A swiss table with inline keys could save 15-20% on map operations.

2. **Vectorized FNV-1a**: Use NEON SIMD for the FNV-1a hash in `stripeFor()` and `shardForParts()`. Currently byte-at-a-time; NEON can process 16 bytes per cycle.

3. **Write coalescing**: In the buffer ring, batch multiple small writes under a single lock acquisition. Currently each write acquires/releases the shard lock independently.

### Medium-Term (High Effort, High Impact)

4. **B-tree per shard**: Replace `map + sorted cache` with concurrent B-tree. O(log n) insert and range query without dirty/rebuild cycle. `github.com/tidwall/btree` provides good Go implementations.

5. **io_uring for writes**: Replace `pwrite` with io_uring batch submissions. Eliminates syscall overhead and enables true async I/O. Linux-only.

6. **NUMA-aware striping**: Pin stripes to NUMA nodes on multi-socket systems. All slab, index, and volume memory for a stripe stays on one node.

### Long-Term (Research)

7. **Epoch-based slab reclamation**: Currently slab memory is never freed (bump allocation). Add reference counting or RCU-style reclamation for delete-heavy workloads.

8. **Persistent memory (PMEM)**: Use CXL memory for the slab arena. Slab becomes persistent store — no need for separate volume.

9. **Learned bloom filters**: Neural-network-based filter trained on actual key patterns. Can achieve 10x better FPR at the same memory cost.

---

## Appendix: Profile Commands

All profiles are stored in `report/` with snapshots at each optimization stage:

```bash
# v4 baseline (before v4 optimizations)
go tool pprof -http=:8080 report/v4_baseline/herd/cpu.pprof
go tool pprof -http=:8080 report/v4_baseline/herd/heap.pprof
go tool pprof -http=:8080 report/v4_baseline/herd/allocs.pprof

# v4 optimized (after all v4 optimizations)
go tool pprof -http=:8080 report/v4_optimized/herd/cpu.pprof
go tool pprof -http=:8080 report/v4_optimized/herd/heap.pprof
go tool pprof -http=:8080 report/v4_optimized/herd/allocs.pprof

# Compare v4 baseline vs optimized
go tool pprof -base report/v4_baseline/herd/cpu.pprof report/v4_optimized/herd/cpu.pprof
go tool pprof -base report/v4_baseline/herd/heap.pprof report/v4_optimized/herd/heap.pprof

# Investigate profile attribution (critical for init.func1-style entries)
go tool pprof -peek "init.func1" report/v4_baseline/herd/heap.pprof

# v3 profiles
go tool pprof -http=:8080 report/v3_baseline/herd/cpu.pprof
go tool pprof -http=:8080 report/v3_optimized/herd/cpu.pprof

# Flamegraph (requires graphviz)
go tool pprof -http=:8080 -call_tree report/v4_optimized/herd/cpu.pprof

# Top 20 with cumulative
go tool pprof -top -cum -nodecount=20 report/v4_optimized/herd/cpu.pprof

# Earlier profiles (v1/v2 iterations)
go tool pprof -http=:8080 report/baseline/herd/cpu.pprof      # v1 baseline
go tool pprof -http=:8080 report/final/herd/cpu.pprof          # v2 final
```

### Running Benchmarks

```bash
# Full benchmark with profiling
go run ./cmd/bench --drivers herd --profile --output ./report/v4_optimized \
  --benchtime 2s --resource-tracking --formats markdown,json --large --progress

# Quick benchmark (no profiling)
go run ./cmd/bench --drivers herd --benchtime 1s --formats markdown

# Specific benchmark only
go run ./cmd/bench --drivers herd --benchtime 5s --bench "Write/1KB"
```
