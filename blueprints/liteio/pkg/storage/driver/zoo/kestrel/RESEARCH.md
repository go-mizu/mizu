# Kestrel Driver: Research & Implementation Guide

> Based on **"From FASTER to F2"** (Kanellis & Chandramouli, PVLDB Vol 18, 2025, pp 4910-4923)
> and the C++ reference implementation: [microsoft/FASTER PR #922](https://github.com/microsoft/FASTER/pull/922)
> C++ source at `$HOME/github/microsoft/FASTER/cc/src/`

---

## 1. Paper Summary

F2 is the evolution of Microsoft's FASTER key-value store. Key contribution: a
**two-tier record-oriented architecture** separating hot (frequently accessed) records
from cold records. This eliminates the "death spiral" where GC of cold data evicts
hot data to disk.

### 1.1 The Death Spiral Problem

Original FASTER uses a single HybridLog. During GC/compaction, cold records at the
log's beginning are copied to the tail. This pushes hot (in-memory) records to disk.
When compaction finishes, hot records re-enter, evicting cold records again - creating
an infinite compaction loop.

### 1.2 Key Results (Table 2 from paper)

| System      | Avg Speedup |
|-------------|-------------|
| vs FASTER v1 | 2.1x       |
| vs LeanStore | 2.0x       |
| vs KVell     | 11.9x      |
| vs SplinterDB | 4.6x      |
| vs RocksDB   | 11.8x      |

Best case: **22x over RocksDB** on read-heavy YCSB-C.

---

## 2. C++ Architecture (`cc/src/core/f2.h`)

### 2.1 Component Overview

```
F2Kv<K, V, D, HHI=MemHashIndex, CHI=ColdIndex>
 |
 +-- hot_store (FasterKv)        -- In-memory HybridLog + MemHashIndex
 |   +-- hash_index_             -- 7-entry buckets, 14-bit tags, overflow chain
 |   +-- hlog                    -- Mutable (90%) + ReadOnly (10%) + Stable (disk)
 |   +-- read_cache_             -- Optional 2nd-chance FIFO, NullDisk
 |
 +-- cold_store (FasterKv)       -- On-disk HybridLog + ColdIndex
 |   +-- hash_index_             -- 8-entry buckets, 3-bit tags, no overflow
 |   +-- hlog                    -- Small in-memory + large on-disk
 |
 +-- background_worker_thread_   -- Hot→Cold compaction, Cold→Cold compaction
 +-- retry_rmw_requests          -- Queue for failed ConditionalInsert retries
```

### 2.2 Data Flow

```
Write(key, value) ──────────────────> hot_store.Upsert()
                                      └─> Append record to hot log tail
                                      └─> CAS hash index entry

Read(key) ──> hot_store.Read() ──────> Found? Return
         └──> cold_store.Read() ─────> Found? Insert into read_cache, Return
         └──> NotFound

Delete(key) ────────────────────────> hot_store.Delete(force_tombstone=true)

RMW(key, modifier) ─> hot_store.Rmw() ─> Found? In-place update, Return
                  └──> cold_store.Read() ─> ConditionalInsert(modified) to hot log
                  └──> If CI fails: push to retry_rmw_requests queue
```

---

## 3. Core Data Structures (from C++ source)

### 3.1 Address (`cc/src/core/address.h`)

48-bit logical address, 8 bytes total:
```
Bits  0-24  (25 bits): offset within page (page size = 2^25 = 32 MB)
Bits 25-47  (23 bits): page index (max ~8M pages)
Bit     47:            read-cache flag (alternative layout)
Bits 48-63  (16 bits): reserved for hash table (tag + control)
```

Constants:
```cpp
kAddressBits   = 48
kOffsetBits    = 25      // 32 MB pages
kPageBits      = 23      // ~8M pages
kInvalidAddress = 1      // Not 0, to distinguish from empty hash bucket entry
kMaxAddress    = (1<<48)-1
kReadCacheMask = 1<<47
```

### 3.2 RecordInfo Header (`cc/src/core/record.h`)

8 bytes, packed bitfield:
```
Bits  0-47  (48 bits): previous_address  -- Hash chain link to older record
Bits 48-60  (13 bits): checkpoint_version -- CPR version number
Bit     61  ( 1 bit):  invalid           -- Record has been invalidated
Bit     62  ( 1 bit):  tombstone         -- Record is deleted
Bit     63  ( 1 bit):  final             -- Checkpoint final marker
```

Alternative layout (with read cache):
```
Bits  0-46  (47 bits): previous_address
Bit     47  ( 1 bit):  readcache         -- Points to read cache record
Bits 48-60  (13 bits): checkpoint_version
Bit     61:            invalid
Bit     62:            tombstone
Bit     63:            final
```

### 3.3 Record Layout (`cc/src/core/record.h`)

```
+----------------------------------------------+
| RecordInfo header (8 bytes)                   |
+----------------------------------------------+
| [alignment padding to alignof(key_t)]         |
+----------------------------------------------+
| Key (variable length, immutable)              |
+----------------------------------------------+
| [alignment padding to alignof(value_t)]       |
+----------------------------------------------+
| Value (variable length, mutable in mutable    |
|        region only)                           |
+----------------------------------------------+
| [alignment padding to alignof(RecordInfo)]    |
+----------------------------------------------+
```

Size calculation: `pad(pad(pad(8, alignof(K)) + keyLen, alignof(V)) + valLen, alignof(RecordInfo))`

### 3.4 Hash Bucket Entry (`cc/src/index/hash_bucket.h`)

**Hot-Log Index Entry** (8 bytes):
```
Bits  0-47  (48 bits): address    -- Logical address into hot log
Bits 48-61  (14 bits): tag        -- Hash fingerprint for fast rejection
Bit     62  ( 1 bit):  reserved
Bit     63  ( 1 bit):  tentative  -- CAS coordination flag
```

**Cold-Log Index Entry** (8 bytes):
```
Bits  0-47  (48 bits): address    -- Logical address into cold log
Bits 48-50  ( 3 bits): tag        -- Smaller tag (cold index uses chunk-level)
Bits 51-62  (12 bits): reserved   -- Used by cold index for in-chunk indexing
Bit     63  ( 1 bit):  tentative
```

### 3.5 Hash Bucket (`cc/src/index/hash_bucket.h`)

**Hot-Log Bucket** (64 bytes, cache-line aligned):
```
+--------------------------------------------------+
| entries[0..6]: 7 x AtomicHashBucketEntry (56 B)  |
| overflow_entry: AtomicHashBucketOverflowEntry (8B)|
+--------------------------------------------------+
```
- 7 entries per bucket + 1 overflow pointer
- Overflow chains to additional buckets allocated from FixedPageSize allocator
- `sizeof == Constants::kCacheLineBytes == 64`

**Cold-Log Bucket** (64 bytes, cache-line aligned):
```
+--------------------------------------------------+
| entries[0..7]: 8 x AtomicHashBucketEntry (64 B)  |
+--------------------------------------------------+
```
- 8 entries, NO overflow chaining (closed addressing)

### 3.6 KeyHash (`cc/src/core/key_hash.h`)

```cpp
struct KeyHash { uint64_t control_; };  // 8 bytes

// Tag extraction: upper 14 bits for hot index
tag = entry_.tag  // bits 48-61

// Bucket index: lower bits masked to table size
index = entry_.address & (table_size - 1)  // power-of-2 mask
```

For cold index, `IndexKeyHash` template adds in-chunk indexing:
- `kInChunkIndexBits` + `kInChunkTagBits` stored in reserved bits
- Used for two-level cold index hash chunks

---

## 4. HybridLog Allocator (`cc/src/core/persistent_memory_malloc.h`)

### 4.1 Region Model

```
                TAIL (append point, atomic bump pointer)
                 |
  +--------------+----------------------------------+
  |    MUTABLE REGION (90% of in-memory budget)     |
  |  - In-place atomic updates allowed              |
  |  - Thread-safe via CAS on RecordInfo            |
  +-------------------------------------------------+
  |    READ-ONLY REGION (10% of in-memory budget)   |
  |  - Immutable in memory                          |
  |  - Being flushed to disk asynchronously         |
  |  - Read-Copy-Update (RCU) for writes            |
  +-------------------------------------------------+
  |    STABLE (ON-DISK)                             |
  |  - Requires async I/O to access                 |
  |  - Compaction source                            |
  +-------------------------------------------------+
                 |
               BEGIN (GC/compaction boundary)
```

### 4.2 Key Addresses (atomic)

| Address | Meaning |
|---------|---------|
| `tail_page_offset_` | Next allocation point (atomic bump) |
| `read_only_address` | Boundary: mutable above, read-only below |
| `safe_read_only_address` | All threads have acknowledged read-only shift |
| `head_address` | In-memory boundary: above = in-memory, below = on disk |
| `begin_address` | GC boundary: below = truncated/compacted away |

### 4.3 Allocation

```cpp
// Atomic bump-pointer allocation:
Address Allocate(uint32_t record_size, uint32_t& page) {
  PageOffset tail = tail_page_offset_.fetch_add(record_size);
  page = tail.page();
  if (tail.offset() + record_size <= kPageSize) {
    return Address{page, tail.offset()};
  }
  // Page full - need new page
  return Address::kInvalidAddress;  // Caller retries with NewPage()
}
```

---

## 5. Epoch Protection (`cc/src/core/light_epoch.h`)

### 5.1 Thread Table

```cpp
struct alignas(64) Entry {  // Cache-line aligned (64 bytes)
  uint64_t local_current_epoch;     // Thread's view of global epoch
  uint32_t reentrant;               // Reentrance counter
  atomic<Phase> phase_finished;     // Phase acknowledgement
};
// Table size: Thread::kMaxNumThreads (96 entries)
```

### 5.2 Drain List (EpochAction)

```cpp
struct EpochAction {
  static constexpr uint64_t kFree = UINT64_MAX;
  static constexpr uint64_t kLocked = UINT64_MAX - 1;

  atomic<uint64_t> epoch;        // Trigger epoch (kFree = available slot)
  callback_t callback;           // Action to perform when safe
  IAsyncContext* context;        // Context for callback
};
// 256 action slots
```

### 5.3 Protocol

```
Thread entry:  table[thread_id].local_current_epoch = global_epoch.load()
Thread exit:   table[thread_id].local_current_epoch = kUnprotected (0)
Bump epoch:    global_epoch++, register (epoch, callback) in drain_list
Safe reclaim:  scan all slots, min_active = min(non-zero epochs)
               fire callbacks where trigger_epoch <= min_active - 1
```

---

## 6. Core Algorithms

### 6.1 Read (`f2.h` lines 302-404)

```
Read(key):
  1. hash = Hash(key)
  2. entry = hot_index.FindEntry(hash)
     → If Pending (cold index I/O): return Pending
     → If NotFound: skip to step 5
  3. address = entry.address
  4. status = hot_store.InternalRead(key, address)
     - Walk hash chain from address backward through hot log
     - If in mutable region: GetAtomic(record)  // concurrent-safe
     - If in read-only/immutable: Get(record)    // safe, no mutation
     - If tombstone found: return ABORTED (hot store read uses abort_if_tombstone)
     - If on disk: return RECORD_ON_DISK (async I/O)
     → If Ok or Aborted: return status
  5. status = cold_store.Read(key)  // no abort_if_tombstone
     → If NotFound: return NotFound
  6. If Ok AND read_cache enabled:
     - Verify hot log tail hasn't advanced past compaction boundary
     - TryInsert(key, value) into hot store's read cache
     - Abort status acceptable (concurrent race)
  7. Return status
```

### 6.2 Upsert (`f2.h` lines 408-412)

```
Upsert(key, value):
  → Delegates to hot_store.Upsert() directly
  → All writes go to hot log only

hot_store.InternalUpsert():
  1. entry = hash_index.FindOrCreateEntry(hash)
  2. address = read_cache.SkipAndInvalidate(entry)  // Skip RC entries
  3. Walk hash chain for matching key
  4. If found in MUTABLE region:
     - PutAtomic(record)  → In-place update, return SUCCESS
  5. If found in READ-ONLY or NOT found:
     - Allocate new record at tail (BlockAllocate)
     - Write RecordInfo{previous_address=old_address}
     - Copy key, write value
     - CAS hash index entry (old → new_address)
     - If CAS fails: mark record invalid, retry
```

### 6.3 RMW (Read-Modify-Write) (`f2.h` lines 414-485)

Three-stage operation spanning both stores:

```
Stage 1 - HOT_LOG_RMW:
  1. entry = hot_index.FindOrCreateEntry(hash)
  2. expected_address = entry.address (skip read cache entries)
  3. status = hot_store.Rmw(key, modifier, create_if_not_exists=false)
  4. If found in mutable: in-place RMW, return Ok
  5. If found in read-only: RCU (copy-modify to tail)

Stage 2 - COLD_LOG_READ (if not found in hot):
  6. old_value = cold_store.Read(key)
  7. new_value = modifier.apply(old_value)  // or initial if NotFound

Stage 3 - HOT_LOG_CONDITIONAL_INSERT:
  8. Validate expected_address still valid (not compacted)
  9. ConditionalInsert(key, new_value, expected=expected_address)
     - Scan (expected_address, TAIL] for matching key
     - If newer record found → ABORT (concurrent update won)
     - If no match → write to tail, CAS index entry
  10. If ABORTED: deep-copy context, push to retry_rmw_requests queue
  11. CompleteRmwRetryRequests() retries with fresh index lookups
```

### 6.4 Delete (`f2.h` lines 575-581)

```
Delete(key):
  → Delegates to hot_store.Delete(force_tombstone=true)
  → Ensures tombstone entry in hot log tail
  → Cold store records naturally obsoleted by hot tombstones
```

### 6.5 ConditionalInsert (`faster.h` lines 3968-4257)

Foundational primitive for compaction and RMW correctness:

```
ConditionalInsert(key, value, min_search_offset):
  1. entry = hash_index.FindOrCreateEntry(hash)
  2. address = read_cache.Skip(entry)  // Get hlog address
  3. Validate min_search_offset not truncated (begin_address check)
  4. Walk hash chain from address, searching down to min_search_offset:
     - If matching key found at address > min_search_offset: ABORT (newer exists)
     - If matching key found at address == min_search_offset: proceed (expected)
     - If matching key found on disk but > min_search_offset: async I/O
  5. No newer record found → create record at tail:
     - BlockAllocate(record_size)
     - Write RecordInfo{previous_address=expected_address}
     - Copy key + value
  6. CAS hash index entry (expected → new_address)
     - If CAS ok: SUCCESS
     - If CAS fails: mark record invalid, retry from step 1
```

---

## 7. Compaction (`cc/src/core/compact.h`, `f2.h` lines 788-961)

### 7.1 Configuration Defaults (`cc/src/core/config.h`)

```
Hot Store:
  check_interval:     250ms
  trigger_pct:        0.8   (trigger when hlog >= 80% of budget)
  compact_pct:        0.2   (compact 20% of total size)
  max_compacted_size: 256 MB
  hlog_size_budget:   1 GB
  num_threads:        4

Cold Store:
  check_interval:     250ms
  trigger_pct:        0.9   (trigger when hlog >= 90% of budget)
  compact_pct:        0.1   (compact 10% of total size)
  max_compacted_size: 1 GB
  hlog_size_budget:   8 GB
  num_threads:        4

HybridLog:
  mutable_fraction:   0.9   (90% mutable, 10% read-only)
  page_size:          32 MB (2^25 bytes)
```

### 7.2 Hot→Cold Compaction

```
CompactHotToCold(begin, until):
  1. Create ConcurrentLogPageIterator over [begin, until)
  2. Spawn num_threads workers
  3. Each worker:
     a. Get next record from concurrent iterator
     b. Skip tombstones (unless forwarding to cold)
     c. ConditionalInsert record into cold store
        - If newer version exists in hot: skip (ABORTED)
        - If ok: record now in cold log
     d. Track pending async I/Os
     e. Periodic CompletePending()
  4. After all records processed:
     - Set hot_log.begin = until (truncate)
     - GC hot index entries pointing to truncated range
```

### 7.3 Cold→Cold Compaction

Same algorithm, but source=cold log, target=cold log tail.
Uses full ConditionalInsert variant (check for newer versions in cold log).

---

## 8. Read Cache (`cc/src/core/read_cache.h`)

- Separate in-memory HybridLog using NullDisk (never persists)
- Hash chains span both read cache and hot log
- At most one read-cache entry per key
- **2nd-chance FIFO eviction**: if record in RC read-only region is re-accessed, copy to RC tail

```
ReadCache.Read(key, address):
  1. If address.in_readcache():
     - Verify key match, not invalid, address >= safe_head
     - Return value
     - If in RC read-only region: CopyToTail (2nd chance)
  2. Else: Skip() to next non-RC address → return NotFound

ReadCache.TryInsert(key, value):
  1. Allocate record in RC tail
  2. CAS hash index entry to point to RC record
  3. Set readcache bit on previous_address

Eviction (when RC head advances):
  1. Collect valid records in evicted page range
  2. CAS index entries to skip RC records → point to hlog addresses
  3. Multi-threaded participation via atomic counter
```

---

## 9. False-Absence Prevention (`f2.h`)

During concurrent cold→cold compaction, a read might miss a record migrating
between cold log positions:

```
Shared atomic: num_compaction_truncations_

ColdRead(key):
  1. t1 = num_compaction_truncations_.load()
  2. tail = cold_log.tail
  3. result = scan cold log for key
  4. t2 = num_compaction_truncations_.load()
  5. If t1 != t2: re-traverse newly appended hash chain portion
```

---

## 10. Go Translation Strategy

### 10.1 What We Keep From F2

| F2 C++ Concept | Kestrel Go Implementation |
|----------------|---------------------------|
| Two-tier (hot + cold) | Sharded in-memory hot map + mmap value arena |
| Cache-line aligned hash buckets | Go map (Swiss table) for O(1) lookup |
| Mmap'd HybridLog regions | Mmap'd value arena (bump-pointer allocator) |
| Epoch protection | Per-shard RWMutex (simpler, same effect for in-memory) |
| Record format (header + K + V) | `hotRecord{value []byte, ct string, ...}` |
| Lock-free allocation | `atomic.Int64.Add` on arena bump pointer |
| Background compaction | Not needed (fully in-memory, no disk tier) |
| Read cache | Not needed (all data in hot tier, no cold miss) |

### 10.2 What We Change For Go

| F2 C++ Design | Go Adaptation | Rationale |
|---------------|---------------|-----------|
| 48-bit packed addresses | Direct pointers / offsets | Go has no 48-bit types |
| Template metaprogramming | Concrete types | Go doesn't have templates |
| Cache-line aligned buckets | 256 sharded Go maps | Go's Swiss table is faster than manual buckets |
| Per-thread epoch table | sync.RWMutex per shard | Go goroutines != OS threads |
| HybridLog pages (32MB) | Mmap arena (128MB chunks) | Simpler bump allocator |
| 7-entry bucket scan | Go map lookup | Swiss table does same internally |
| Overflow chain | Go map handles growth | Automatic rehash |
| NullDisk read cache | Not applicable | Everything in memory |

### 10.3 Key Insight: Why Not Direct Port?

Go's built-in `map` uses **Swiss tables** (since Go 1.24) — the same data structure
FASTER's MemHashIndex implements manually. A custom hash table in Go cannot outperform
the runtime's implementation because:

1. Go's Swiss table uses SIMD-like tag matching (via `math/bits`)
2. It's deeply integrated with the GC (no write barriers for internal data)
3. It handles growth/rehash transparently
4. Inline assembly optimizations for common architectures

The real performance win from F2 is **separating bulk data from the GC heap**:
- Values stored in mmap'd arena → invisible to GC scanner
- Only map keys + pointers on Go heap → minimal GC work
- `debug.SetGCPercent(800)` further reduces GC frequency

---

## 11. Kestrel Architecture (v5 — Current)

```
256 Sharded Go Maps (Swiss table speed)
+--------------------------------------------+
| Shard 0: sync.RWMutex + map[string]*hotRecord |
| Shard 1: sync.RWMutex + map[string]*hotRecord |
| ...                                            |
| Shard 255: sync.RWMutex + map[string]*hotRecord|
+--------------------------------------------+
          | hotRecord.value points into |
Mmap Value Arena (GC-invisible, lock-free bump allocator)
+--------------------------------------------+
| Chunk 0: [128MB mmap MAP_ANON|MAP_PRIVATE] |
| Chunk 1: [128MB mmap region]               |
| ...   (CAS-linked list for chunk growth)   |
+--------------------------------------------+
          |
Background Index Loop (1ms tick)
+--------------------------------------------+
| Per-shard pending ops → keyIndex updates   |
+--------------------------------------------+
```

**Key optimizations:**
- Allocation-free lookups: `unsafeString(compositeKeyBuf(buf[:0], ...))` with `[256]byte` stack buffer
- FNV-1a shard selection: `shardForParts(bucket, key)` hashes without allocation
- Direct-to-arena writes: `io.ReadFull(src, arena.alloc(size))` — single copy
- Cache-line padding: `[24]byte` on shard struct prevents false sharing
- GC tuning: `debug.SetGCPercent(800)` since bulk data in mmap

### 11.1 File Structure

```
pkg/storage/driver/zoo/kestrel/
  RESEARCH.md     -- This document
  storage.go      -- Driver, store, shards, hot path (hotGet/hotPut/hotDelete)
  arena.go        -- Mmap value arena (bump-pointer allocator)
  bucket.go       -- storage.Bucket implementation (Write/Open/Stat/Delete/Copy/Move)
  multipart.go    -- Multipart upload support
  keyindex.go     -- Sorted key index for List operations
```

---

## 12. Benchmark Results (v5 vs Falcon Baseline)

**Overall: kestrel 26/40 wins (65%)**

| Category | Kestrel Wins | Notable |
|----------|-------------|---------|
| Read | 4/4 | Read/64KB +76% |
| ParallelRead | 3/3 | C50 +49% |
| ParallelWrite | 3/3 | C1 +47%, C50 +44% |
| Scale | 7/12 | Scale/Write/1000 3.3x, Scale/List/1 3.2x |
| Write | 2/4 | Write/64KB -47% (arena contention) |

### Resource Comparison

| Metric | Kestrel | Falcon | Ratio |
|--------|---------|--------|-------|
| Go Heap | 5.8 GB | 54.2 GB | **9.3x less** |
| GC Cycles | 14 | 6 | 2.3x more |
| Disk Usage | 0 MB | 1024 MB | **Zero disk** |

### Known Bottleneck: Write/64KB (-47%)

Kestrel's mmap arena uses a **shared atomic bump pointer** (`atomic.Int64.Add`)
that creates contention under concurrent medium-sized writes. Falcon uses **per-P
`sync.Pool` value chunks** with zero contention.

---

## 13. Optimization Targets for 5x

### 13.1 Arena Allocation Contention

**Problem:** Single atomic bump pointer bottleneck for concurrent writes.
**Solution:** Per-shard arena striping — each shard has its own arena region.
Each write locks the shard anyway, so arena allocation inside the lock is free.

### 13.2 Read Path Allocation

**Problem:** `compositeKey()` string concatenation allocates on every write.
**Current:** Fixed with `unsafeString` + stack buffer for reads, but writes still allocate.
**Solution:** Pool composite keys for write path too.

### 13.3 GC Pressure from Map Pointers

**Problem:** 256 maps × N entries × pointer to hotRecord = millions of GC-visible pointers.
**Solution:** Store records inline in a flat mmap'd slab instead of heap-allocated `*hotRecord`.
The map value becomes a 4-byte index into the slab instead of an 8-byte pointer.

### 13.4 Stat/Delete Hot Path

**Problem:** Stat and Delete paths do unnecessary work (composite key alloc, arena tracking).
**Solution:** Stat returns metadata only (no value access needed). Delete can skip arena tracking.

### 13.5 Large Value Writes (1MB, 10MB)

**Problem:** `io.ReadFull` into arena may block if reader is slow.
**Solution:** For known-size writes, pre-allocate arena space and use direct copy.
Already implemented — verify no regression.

---

## 14. Benchmark Tool Usage

### 14.1 Running Benchmarks

```bash
# Build bench CLI
go build -o /tmp/bench-cli ./cmd/bench/

# Quick benchmark (adaptive, ~500ms per test)
/tmp/bench-cli --drivers falcon,kestrel --quick --output /tmp/report --formats markdown,json

# Full benchmark (1s per test, more stable)
/tmp/bench-cli --drivers falcon,kestrel --output /tmp/report --formats markdown,json

# With profiling (CPU + heap pprof)
/tmp/bench-cli --drivers kestrel --profile --output /tmp/profile-report --formats markdown,json

# Cleanup data after each driver
/tmp/bench-cli --drivers falcon,kestrel --quick --output /tmp/report --cleanup-data
```

### 14.2 Profiling

```bash
# CPU profile analysis
go tool pprof -top -cum /tmp/profile-report/kestrel/cpu.pprof

# Heap profile analysis
go tool pprof -top /tmp/profile-report/kestrel/heap.pprof

# Interactive web UI
go tool pprof -http=:8080 /tmp/profile-report/kestrel/cpu.pprof

# Compare two profiles (before/after optimization)
go tool pprof -base /tmp/v1-profile/kestrel/cpu.pprof /tmp/v2-profile/kestrel/cpu.pprof
```

### 14.3 Benchmark Categories

| Category | What It Measures |
|----------|-----------------|
| Write/1KB..10MB | Single-writer throughput by object size |
| Read/1KB..10MB | Single-reader throughput by object size |
| ParallelWrite/C1..C50 | Multi-writer throughput by concurrency |
| ParallelRead/C1..C50 | Multi-reader throughput by concurrency |
| Delete | Single-thread delete throughput |
| Stat | Metadata-only lookup throughput |
| List/100 | List 100 objects throughput |
| Copy/1KB | Server-side copy throughput |
| MixedWorkload | Read+Write mixed ratios (90/10, 50/50, 10/90) |
| RangeRead | Partial reads from start/middle/end of 256KB object |
| Multipart | 15MB upload in 3 parts |
| EdgeCase | Empty objects, 256-byte keys, deeply nested paths |
| Scale/Write,Delete,List | Operations at 1/10/100/1000 pre-existing objects |

---

## 15. Optimization History (v1-v4)

### 15.1 v1 (Baseline)

Initial implementation using sharded Go maps. Architecture:
- 1024 sharded `map[string]hotRecord` (value type, not pointer)
- `sync.RWMutex` per shard
- Mmap arena for all value data
- Separate `pendingMu` mutex for index operations

Result: **kestrel 24/40 (60%)** — but unstable due to Go map + arena contention.

### 15.2 v3 (Per-P Value Allocation)

Key changes from v1:
1. **`allocValue` via sync.Pool chunks** (4MB each, per-P locality) replaces arena for values
2. **Pointer returns from `hotGet`** (`*hotRecord` instead of copying 5 fields)
3. **Embedded pending ops in shard** (single lock instead of separate `pendingMu`)
4. **Mmap arena only for composite key strings** (GC-invisible map keys)
5. **Stack-buffer composite keys for reads** (512B, allocation-free lookups)

Result: **kestrel 27/40 (68%)** — significant improvement on Write/1MB (+30%), Read/1KB (+14%).

### 15.3 v4 (Cache Isolation Fix)

**Critical finding**: Inline shard array `[1024]shard` caused 2-3x ParallelRead regression.

The problem: with 72-byte inline shards in a contiguous array, the store struct was ~73KB.
When kestrel ran after falcon, accumulated memory pressure caused cache thrashing across
the large contiguous shard array. Pointer-allocated shards avoid this by being individually
heap-allocated with better cache line isolation.

Key changes:
1. **256 pointer-allocated shards** (`[256]*shard` instead of `[1024]shard`)
2. **`compositeKey` (string concat)** for new map entries instead of mmap arena
3. **Removed mmap arena** entirely — composite keys use Go heap strings
4. **Reduced stack buffer to 256 bytes** (matching falcon)

Result: **kestrel 21-24/40 (52-60%)** — ParallelRead fixed from 2.3x slower to ~equal.

### 15.4 Key Lessons

| Lesson | Impact |
|--------|--------|
| Inline shard arrays cause false sharing / cache pressure | 2.3x ParallelRead regression |
| Per-P sync.Pool chunks beat mmap arena for values | +30% Write/1MB throughput |
| Pointer returns from hotGet beat field copies | +14% Read/1KB throughput |
| Embedded pending ops in shard save a mutex acquire | ~5% write path improvement |
| Go map + RWMutex sets a performance ceiling | Both drivers converge to ~equal |
| macOS ARM thermal throttling causes 50%+ variance | Second-to-run driver penalized |
| `debug.SetGCPercent(800)` reduces GC from ~20 to ~12 cycles | Measurable for ParallelRead |

### 15.5 Thermal Effects on macOS ARM

Back-to-back benchmark runs show massive variance due to CPU throttling:

| Run Order | Kestrel Wins | Notes |
|-----------|--------------|-------|
| Kestrel first | 24/40 (60%) | Cool CPU favors first driver |
| Falcon first | 18/40 (45%) | Hot CPU penalizes second driver |
| Average | ~21/40 (52%) | True performance is roughly equal |

Individual operations swing 20-50% depending on run order. This means:
- ParallelRead can show 5.0 GB/s (cold) vs 2.5 GB/s (hot) for the same driver
- MixedWorkload results are especially unreliable (high sustained load)
- Only isolated benchmarks with cooldown give stable numbers

### 15.6 Path to 5x: Custom Hash Table

Both kestrel and falcon use Go's built-in `map[string]*T`, which means:
- Same Swiss table implementation
- Same hash function overhead (~30ns per lookup)
- Same GC scanning for map internals
- Same lock contention patterns

To achieve 5x over falcon, kestrel would need to replace Go maps with a
FASTER-style flat hash table:
- 64-byte cache-line aligned buckets (7 entries per bucket)
- 14-bit tags for fast scan (1 cache line load checks 7 entries)
- Mmap'd record slab (48B fixed headers, GC-invisible)
- Lock-free reads via epoch protection
- Bump-pointer value arena (contiguous key+value storage)

This is the plan outlined in Section 2 (C++ Architecture) — a full reimplementation
of the storage layer replacing Go maps with FASTER's index design.

---

## 16. References

1. Kanellis, K. & Chandramouli, B. "From FASTER to F2: Evolving Concurrent Key-Value
   Store Designs for Large Skewed Workloads." PVLDB 18(12): 4910-4923, 2025.
   https://www.vldb.org/pvldb/vol18/p4910-kanellis.pdf

2. Microsoft FASTER GitHub Repository.
   https://github.com/microsoft/FASTER

3. PR #922: [C++] F2 KV store.
   https://github.com/microsoft/FASTER/pull/922
   19,706 additions, 3,014 deletions, 81 files.

4. Key C++ source files:
   - `cc/src/core/f2.h` (1,013 lines) — F2Kv top-level class
   - `cc/src/core/faster.h` (4,200+ lines) — FasterKv core with InternalRead/Upsert/Rmw/Delete
   - `cc/src/core/record.h` — RecordInfo (8B) + Record<K,V> layout
   - `cc/src/core/address.h` — 48-bit address: offset[25] | page[23]
   - `cc/src/core/light_epoch.h` — Epoch protection with 96-entry thread table
   - `cc/src/core/persistent_memory_malloc.h` — HybridLog allocator (32MB pages)
   - `cc/src/core/config.h` — Configuration defaults (hot/cold budgets, compaction)
   - `cc/src/core/compact.h` — Compaction with ConditionalInsert
   - `cc/src/core/read_cache.h` — 2nd-chance FIFO read cache
   - `cc/src/index/hash_bucket.h` — 64-byte cache-line buckets (7 or 8 entries)
   - `cc/src/index/hash_table.h` — Hash table with overflow bucket pool
   - `cc/src/index/mem_index.h` — In-memory hash index (MemHashIndex)
   - `cc/src/index/cold_index.h` — Two-level cold index (in-mem dir + on-disk chunks)
