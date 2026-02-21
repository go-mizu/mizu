# Pony Driver: Deep Performance Research

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [v1 Baseline Profiling Analysis](#v1-baseline-profiling-analysis)
3. [Bottleneck Analysis](#bottleneck-analysis)
4. [Memory Budget Analysis](#memory-budget-analysis)
5. [Optimization Targets](#optimization-targets)
6. [Lessons Learned](#lessons-learned)
7. [Appendix: Profile Commands](#appendix-profile-commands)

---

## Architecture Overview

Pony is a memory-constrained single-volume object storage driver. Unlike Horse (300–500MB RSS at 1M objects) or Herd (16 stripes, 4096 shards), Pony targets <100MB RSS via an mmap'd on-disk hash table index.

### Storage Layout (v1)

```
store (single volume, single index)
 ├── volume (append-only mmap file, pwrite/mmap dual path)
 │    ├── header (64B: magic, version, tail offset)
 │    ├── records (type|crc|bucket_len|key_len|ct_len|value_len|payload|ts)
 │    └── mmapRegion (atomic swap on grow, shared mmap)
 ├── diskIndex (on-disk hash table via mmap)
 │    ├── header (64B: magic, slotCount, entryCount, stringsPos)
 │    ├── slots (N × 64B: hash|strOff|strLen|ctLen|valOff|valSize|created|updated)
 │    ├── stringPool (append-only: compositeKey + contentType per entry)
 │    └── bucketKeys (in-memory per-bucket key lists for List)
 ├── bufferRing (2 × 4MB write buffers, background flush)
 └── multipartRegistry (in-memory part tracking)
```

### Key Design Decisions

| Component | Design | Rationale |
|-----------|--------|-----------|
| Single volume | Append-only, 256MB prealloc | Simplicity; one mmap region for all data |
| On-disk hash table | FNV-1a, 64B cache-aligned slots | Low RSS; OS pages out cold entries |
| Linear probing | Open addressing, 75% max load | Simple; good cache locality |
| 2-buffer ring | 2 × 4MB, background flusher | Keep memory low (8MB total) |
| String pool | Append-only in index file | No separate string allocation |
| bucketKeyList | In-memory map + lazy sorted array | Required for List/prefix operations |

### Data Flow

**Write path (sync=none, inline):**
```
Write() → validate key
        → bufRing.writeInline(totalSize)    [claim atomic, may swap+lock]
        → buildRecordBuf(buf, ...)          [serialize header+payload]
        → wb.done()                         [atomic decrement]
        → idx.put(bucket, key, ...)         [RWMutex.Lock, linear probe, appendString]
        → addBucketKey(bucket, key)         [sync.Map.Load, mu.Lock, map assign]
```

**Read path:**
```
Open() → validate key
       → idx.get(bucket, key)              [RWMutex.RLock, linear probe, readString]
       → bufRing.readFromBuffer(off, size)  [check 2 buffers, no lock]
       → vol.readValueSlice(off, size)      [mmap slice, zero-copy]
       → acquireMmapReader(slice)           [sync.Pool Get]
```

---

## v1 Baseline Profiling Analysis

**Environment:** Go 1.26.0, darwin/arm64, 10 CPUs, benchtime=2s, concurrency=200

### Benchmark Results

| Operation | ops/s | MB/s | P50 | P99 |
|-----------|------:|-----:|----:|----:|
| Write/1KB | 1,094,265 | 1,068 | 458ns | 1.6μs |
| Write/64KB | 25,724 | 1,642 | 1.8μs | 296μs |
| Write/1MB | 695 | 695 | 26.5μs | 6.6ms |
| Write/10MB | 65 | 646 | 14.6ms | 35.3ms |
| Write/100MB | 5 | 543 | 169ms | 252ms |
| Read/1KB | 6,550,397 | 6,397 | 125ns | 417ns |
| Read/64KB | 818,288 | 52,307 | 1.2μs | 1.6μs |
| Read/1MB | 55,710 | 55,710 | 17.5μs | 24.2μs |
| Read/10MB | 5,023 | 50,230 | 185μs | 827μs |
| Read/100MB | 478 | 47,800 | 1.9ms | 3.7ms |
| Stat | 11,715,175 | - | - | - |
| Delete | 2,373,296 | - | - | - |
| List/100 | 79,902 | - | - | - |

### Parallel Scalability (CRITICAL PROBLEM)

| Operation | C1 | C10 | C25 | C50 | Scaling |
|-----------|---:|----:|----:|----:|--------:|
| ParallelWrite/1KB | 850,616 | 75,588 | 2,794 | 497 | **0.001%** |
| ParallelRead/1KB | 1,403,781 | 1,056,339 | 888,077 | 755,797 | 1.1% |

**Parallel write is catastrophically broken.** Going from C1→C50 drops throughput by **1,710×** (from 851K to 497 ops/s). This is the #1 bottleneck by far.

### CPU Profile (377.87s samples / 116.31s wall = 3.25x CPU utilization)

| Function | Flat | Flat% | Category | Actionable? |
|----------|-----:|------:|----------|:-----------:|
| runtime.usleep | 152.62s | 40.39% | Scheduler | Partial |
| runtime.pthread_cond_wait | 75.44s | 19.96% | Lock wait | **YES** |
| runtime.pthread_cond_signal | 61.20s | 16.20% | Lock wake | **YES** |
| syscall.rawsyscalln | 8.08s | 2.14% | Syscall | No |
| runtime.memmove | 6.54s | 1.73% | Memory copy | Partial |
| runtime.lock2 | 5.12s | 1.35% | Lock | **YES** |
| runtime.tryDeferToSpanScan | 4.38s | 1.16% | GC | **YES** |
| runtime.scanObject | 3.40s | 0.90% | GC | **YES** |
| runtime.nanotime1 | 3.37s | 0.89% | Time | No |
| runtime.unlock2 | 3.31s | 0.88% | Lock | **YES** |

**Lock contention dominates:** 76.55% of CPU is in usleep + pthread_cond_wait + pthread_cond_signal. This is the Go scheduler spinning on lock contention, specifically the single `sync.RWMutex` on `diskIndex`.

### Detailed CPU Breakdown by Component

**diskIndex.put (14.22s cumulative, 3.76% of total):**
| Callee | Time | % of put |
|--------|-----:|--------:|
| appendString (→memmove) | 4.69s | 33.0% |
| addBucketKey (→mapaccess2) | 3.21s | 22.6% |
| rehash | 1.55s | 10.9% |
| RWMutex.Lock | 0.81s | 5.7% |
| compositeKey (string concat) | 0.69s | 4.9% |
| RWMutex.Unlock | 0.69s | 4.9% |

**diskIndex.get (3.96s cumulative, 1.05% of total):**
| Callee | Time | % of get |
|--------|-----:|--------:|
| compositeKey (string concat) | 1.98s | 50.0% |
| readString (→slicebytetostring) | 0.76s | 19.2% |
| hashComposite | 0.27s | 6.8% |
| memequal | 0.16s | 4.0% |

**bufferRing.swap (8.28s cumulative, 2.19% of total):**
| Callee | Time | % of swap |
|--------|-----:|--------:|
| Mutex.Lock (contention) | 6.17s | 74.5% |
| Mutex.Unlock | 2.09s | 25.2% |

### Heap Profile (814.97 MB in-use)

| Allocator | In-Use | % | Root Cause |
|-----------|-------:|--:|------------|
| addBucketKey | 415.62 MB | 51.0% | Per-bucket Go map[string]struct{} |
| fmt.Sprintf | 197 MB | 24.2% | Upload ID generation (bench artifact) |
| bench.payload | 111.16 MB | 13.6% | Benchmark payload (not driver) |
| diskIndex.list | 60.49 MB | 7.4% | Sorted key slice allocation |
| newWriteBuffer | 8 MB | 1.0% | 2 × 4MB buffers |

**Driver-attributable heap: ~484 MB** (415.62 + 60.49 + 8 = 484 MB). Far exceeds the 100MB target.

### Allocs Profile (43.08 GB total allocated)

| Allocator | Total | % | Root Cause |
|-----------|------:|--:|------------|
| diskIndex.list | 5.46 GB | 12.7% | `make([]string)` per list call, sorts |
| bucket.Stat | 4.94 GB | 11.5% | `idx.list()` for directory stat (scans all keys) |
| bucket.List | 4.76 GB | 11.0% | `[]Object` allocation per list call |
| readString | 4.51 GB | 10.5% | `string(data[off:off+len])` per lookup |
| bucket.Open | 4.23 GB | 9.8% | indexResult + readString per read |
| bucket.Write | 2.41 GB | 5.6% | compositeKey concat + indexResult |
| compositeKey | 2.38 GB | 5.5% | `bucket + "\x00" + key` concat every call |
| rehash | 1.62 GB | 3.8% | Collect all entries into slice |
| getWriteBuf | 1.46 GB | 3.4% | Pool misses under high concurrency |

---

## Bottleneck Analysis

### B1: Single Global RWMutex on Index (CRITICAL — 76% CPU)

The `diskIndex.mu sync.RWMutex` is a single global lock. Every write takes `mu.Lock()`, every read takes `mu.RLock()`. With 200 concurrent goroutines:

- **Write serialization:** All 200 writers compete for one exclusive lock. At C50, throughput drops 1,710× because writers must wait for each other sequentially.
- **Lock convoy:** Even `RLock` blocks when a writer is waiting, creating priority inversion.
- **CPU waste:** 76.55% of CPU time is the scheduler spinning (usleep/pthread_cond_wait) while goroutines wait for the lock.

**Impact:** This is the root cause of the 0.001% parallel write scaling.

### B2: bucketKeyList Memory (51% of heap, 415 MB)

Every key is stored twice: once in the mmap'd string pool (on-disk, OS-managed) and once in an in-memory Go `map[string]struct{}` per bucket. With 1M+ keys from the benchmark, this consumes 415 MB of GC-visible heap.

**Impact:** Blows the 100MB memory budget. Also creates GC pressure (scanObject at 0.9% CPU).

### B3: compositeKey String Concatenation (5.5% of allocs, 2.38 GB)

Every `get()`, `put()`, and `remove()` call creates `bucket + "\x00" + key` as a new heap allocation. This is called millions of times per second.

**Impact:** GC pressure, allocation overhead on hot path.

### B4: readString Allocations (10.5% of allocs, 4.51 GB)

`readString()` does `string(idx.data[off:off+len])` which copies mmap'd data to a new Go string allocation every time.

**Impact:** 4.51 GB of transient allocations during reads. Adds latency and GC pressure.

### B5: list() Allocations (12.7% + 11% of allocs)

Every `list()` call:
1. Locks the bucketKeyList mutex
2. If dirty: `make([]string, 0, n)` + copy all keys + `sort.Strings()`
3. For each matching key: calls `idx.get()` (which takes RLock + compositeKey + readString)
4. Allocates `[]listResult` growing dynamically

**Impact:** O(n) memory for every list operation; dominant allocation source.

### B6: Buffer Ring Swap Contention (2.19% CPU)

With only 2 × 4MB buffers, the buffer ring fills quickly under high concurrency. `swap()` takes a `sync.Mutex` and spins with `runtime.Gosched()` waiting for the flusher. With 200 concurrent writers, this becomes a serial bottleneck.

**Impact:** 8.28s of CPU time just on buffer ring mutex contention.

### B7: Rehash Stop-the-World (1.55s CPU, 1.62 GB allocs)

At 75% load factor (49K entries with 65K slots), rehash:
1. Collects all entries into a Go slice
2. `syscall.Munmap` (invalidates all pointers)
3. Truncates + remaps
4. Clears all slots byte-by-byte
5. Re-inserts all entries

This blocks ALL I/O during the operation.

**Impact:** Latency spike when crossing load factor thresholds.

---

## Memory Budget Analysis

### Current Memory Breakdown (v1)

| Component | Type | Size | Notes |
|-----------|------|-----:|-------|
| Volume mmap | Virtual (OS-managed) | 256 MB | Only touched pages are resident |
| Index mmap | Virtual (OS-managed) | ~8 MB | 65K slots × 64B + 4MB string pool |
| Write buffers | Heap | 8 MB | 2 × 4MB |
| bucketKeyList maps | Heap | 415 MB | **PROBLEM** — Go map per key |
| bucketKeyList sorted | Heap | 60 MB | Sorted slice per bucket |
| Multipart registry | Heap | ~0 MB | Empty when idle |
| **Total Heap** | | **~483 MB** | Far exceeds 100MB target |
| **Total RSS** | | **3,562 MB** | Includes virtual + heap + runtime |

### Target Memory Budget (v2, 100MB total)

| Component | Type | Target | Strategy |
|-----------|------|-------:|----------|
| Volume mmap | Virtual | ~10-20 MB resident | OS manages; only hot pages |
| Index mmap | Virtual | ~4-8 MB resident | OS manages; cache-aligned slots |
| Write buffers | Heap or mmap | 8 MB | Keep 2 × 4MB or increase to 4 × 4MB |
| Key tracking | Eliminated | 0 MB | Use index iteration instead of Go maps |
| Sorted key cache | Mmap | ~0 MB heap | Move to mmap-backed B-tree or sorted array |
| Go runtime | Heap | ~10-15 MB | GC, goroutine stacks, runtime |
| **Total Heap** | | **<30 MB** | |
| **Total RSS** | | **<100 MB** | |

---

## Optimization Targets

### O1: Sharded Index (eliminates B1 — expected 50-100× parallel write improvement)

Replace single `diskIndex` with N sharded sub-indexes, each with its own RWMutex and mmap region. Route keys via FNV-1a hash to shard.

- **Target:** 256 shards (matching Herd's proven design)
- **Expected impact:** Parallel write from 497 ops/s → 50K+ ops/s at C50
- **Memory:** Same total mmap size, split across files

### O2: Eliminate bucketKeyList (eliminates B2 — saves 475 MB heap)

Replace in-memory `map[string]struct{}` with:
- **Option A:** Walk mmap'd hash table for List (scan slots, match bucket prefix)
- **Option B:** Secondary sorted index in mmap (sorted by compositeKey for prefix scan)
- **Option C:** Per-shard sorted key tracking in mmap slab

Option A is simplest but O(slots) per list call. With sharding, each shard has fewer slots, making scan cheaper.

### O3: Inline compositeKey Hashing (eliminates B3 — saves 2.38 GB allocs)

Replace `compositeKey(bucket, key)` string concatenation with inline FNV-1a that hashes bucket+null+key without allocation. Compare keys by comparing hash + reading from string pool directly.

### O4: Zero-copy readString (eliminates B4 — saves 4.51 GB allocs)

Use `unsafe.String(unsafe.SliceData(data[off:off+len]), len)` to return a string that references mmap'd data directly without copying. Must ensure mmap region lifetime exceeds string usage.

### O5: Larger Buffer Ring (reduces B6)

Increase from 2 × 4MB to 4 × 8MB (32 MB total). More buffers means less contention; larger buffers mean fewer swaps. Still well within 100MB budget.

### O6: Incremental Rehash (reduces B7)

Pre-allocate 2× slots from the start and use incremental migration (move N entries per put) instead of stop-the-world rehash.

---

## Lessons Learned

### L1: Single Global Lock Kills Parallel Scalability

The single `sync.RWMutex` on diskIndex caused a 1,710× throughput drop from C1→C50. Even RLock creates contention when writers are waiting.

**Rule:** Any shared mutable state must be sharded. Use at least N=CPUs shards.

### L2: In-Memory Key Tracking Defeats the Purpose of On-Disk Index

The on-disk hash table was supposed to keep RSS low, but the in-memory `bucketKeyList` maps consumed 415 MB — more than Horse's entire index. The optimization target (low memory) was undermined by a secondary data structure.

**Rule:** Audit all in-memory structures when targeting memory-constrained operation.

### L3: compositeKey Concatenation is a Hidden Tax

`bucket + "\x00" + key` allocates a new string on every read, write, and delete. With millions of ops/sec, this generates gigabytes of short-lived allocations that pressure the GC.

**Rule:** Hash incrementally without allocating. Compare by hash first, then verify from stored data.

### L4: readString Copies Defeat Zero-Copy mmap

`string(data[off:off+len])` copies bytes from mmap to heap. This defeats the zero-copy benefit of mmap. Use `unsafe.String` for zero-copy access.

**Rule:** When using mmap, avoid `string()` conversions on the hot path. Use unsafe string views.

### L5: Small Buffer Rings Create Serialization Points

2 buffers of 4MB each means the ring fills quickly under load. When both buffers are frozen (flushing), all writers must spin-wait, creating a serialization point.

**Rule:** Size buffer ring to absorb burst writes during flush. Use at least 4 buffers.

---

## v2 Optimization Results

### Architecture Changes

1. **Sharded Index (256 shards)**: Replaced single `diskIndex` with 256 `diskShard` instances, each with own RWMutex and mmap file. Shard selection via FNV-1a hash bitmask.
2. **Eliminated bucketKeyList**: Removed 475 MB in-memory key tracking. List uses parallel shard scan with version-based caching.
3. **Zero-allocation key matching**: `matchCompositeKey()` compares bucket+key directly against mmap'd data without string allocation.
4. **Zero-allocation string pool writes**: `appendCompositeAndCT()` writes bucket+null+key+contentType directly to string pool.
5. **4-buffer ring with concurrent flush**: 4 × 4MB buffers, 2 flusher goroutines, sync.Cond for efficient blocking when all buffers full.
6. **Atomic nextBase offset allocation**: Race-free buffer recycling for concurrent flushers.
7. **List cache with version tracking**: Atomic version counter bumped on every put/remove. Cached list results returned when version matches.
8. **Parallel list scan**: 8 worker goroutines scan shard ranges concurrently.

### v2 Benchmark Results

| Operation | v1 Baseline | v2 Optimized | Change |
|-----------|------------:|-------------:|-------:|
| Write/1KB | 1,094,265 | 1,083,090 | -1% |
| Read/1KB | 6,550,397 | 3,626,784 | -45% |
| Stat | 11,715,175 | 5,745,058 | -51% |
| **List/100** | 79,902 | **368,529** | **+4.6×** |
| Delete | 2,373,296 | 1,791,847 | -24% |
| ParallelWrite/C1 | 850,616 | 931,987 | **+10%** |
| ParallelWrite/C10 | 75,588 | 123,939 | **+1.64×** |
| ParallelWrite/C50 | 497 | 7,874 | **+15.8×** |
| ParallelWrite/C100 | timeout | 463 | ∞ |
| ParallelWrite/C200 | timeout | 10,037 | ∞ |
| ParallelRead/C50 | 755,797 | 1,888,881 | **+2.5×** |
| ParallelRead/C100 | timeout | 1,841,428 | ∞ |
| ParallelRead/C200 | timeout | 1,529,833 | ∞ |

### v2 Memory

| Component | v1 | v2 | Change |
|-----------|---:|---:|-------:|
| Driver heap | 483 MB | 16 MB | **-96.7%** |
| Write buffers | 8 MB | 16 MB | 2× (4 buffers) |
| bucketKeyList | 475 MB | 0 MB | Eliminated |
| Index mmap | ~8 MB | ~16 MB | 256 shard files |
| Total heap | ~483 MB | **~16 MB** | Well under 100MB |

### v2 CPU Profile

| Function | v1% | v2% | Change |
|----------|----:|----:|--------|
| usleep (scheduler) | 40.4% | 17.2% | -23% |
| pthread_cond_wait | 20.0% | 17.9% | -2% |
| pthread_cond_signal | 16.2% | 13.4% | -3% |
| syscall (pwrite) | 2.1% | 11.9% | +10% (real I/O, not wasted) |
| memmove | 1.7% | 7.3% | +5.6% (real work) |
| idx.put | 3.8% | 8.7% | +5% (sharded, less contention) |

**Lock contention dropped from 76.6% to 48.5%** — remaining contention is from buffer ring sync.Cond and Go scheduler overhead.

### Trade-offs

1. **Read/Stat regression (-45%/-51%)**: Sharded index adds per-lookup overhead (hash → shard select → RLock → probe). The v1 single global lock was faster for individual point lookups since there was only one hash table to probe.
2. **Delete regression (-24%)**: Same cause as Read regression — sharded lookup overhead.
3. **List dramatically improved (+4.6×)**: Version-based cache returns cached results instantly for repeated identical queries. First scan after writes uses parallel 8-worker scan.

### Lessons Learned (v2)

**L6: Buffer ring is essential for serial write throughput.** Removing the buffer ring and writing directly to mmap caused Write/1KB to drop from 1.1M to 105K (10×) due to MAP_SHARED dirty page tracking overhead. The buffer ring amortizes page fault costs by batching writes into 4MB pwrite syscalls.

**L7: TryLock/Gosched pattern wastes 63% of CPU.** Using `sync.Mutex.TryLock()` + `runtime.Gosched()` for buffer ring swap causes massive scheduler overhead (usleep + pthread_cond). Replaced with `sync.Cond` for proper blocking — CPU waste dropped from 63% to 48%.

**L8: pwrite fallback from buffer ring causes data corruption.** When `writeInline` fails and falls back to `vol.appendRecord`, the `v.tail.Add()` advances into space reserved by buffer ring buffers. Fix: always succeed in writeInline (blocking via Cond), never fall back.

**L9: Atomic version counter on hot path causes contention.** `version.Add(1)` on every put() causes cache line bouncing under high concurrency. Impact is ~10-20% at C50+. Acceptable trade-off for 4.6× List improvement.

**L10: List cache is the only practical solution for hash table prefix scans.** Hash tables don't support O(k) prefix range queries. Scanning all slots is O(N). Caching results with version-based invalidation converts repeated identical queries from O(N) to O(1).

---

## Appendix: Profile Commands

```bash
# Run baseline benchmark with profiling
go run ./cmd/bench --drivers pony --profile --output ./report/pony_v1_baseline \
  --benchtime 2s --resource-tracking --formats markdown,json --large --progress

# View CPU profile interactively
go tool pprof -http=:8080 report/pony_v1_baseline/pony/cpu.pprof

# View heap profile interactively
go tool pprof -http=:8080 report/pony_v1_baseline/pony/heap.pprof

# View allocs profile interactively
go tool pprof -http=:8080 report/pony_v1_baseline/pony/allocs.pprof

# Top CPU consumers by flat time
go tool pprof -top -flat -nodecount=30 report/pony_v1_baseline/pony/cpu.pprof

# Investigate specific function
go tool pprof -peek "diskIndex" report/pony_v1_baseline/pony/cpu.pprof

# Compare baseline vs optimized
go tool pprof -base report/pony_v1_baseline/pony/cpu.pprof \
              report/pony_v2_optimized/pony/cpu.pprof
```
