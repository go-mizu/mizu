# Kestrel Driver — Research & Design Document

> Based on **"From FASTER to F2"** (Kanellis & Chandramouli, PVLDB Vol 18, 2025, pp 4910–4923)
> and the C++ reference implementation: [microsoft/FASTER PR #922](https://github.com/microsoft/FASTER/pull/922)

## 1. Paper Summary

F2 is the evolution of Microsoft's FASTER key-value store, designed for **high-throughput
point operations on datasets much larger than available memory** with naturally skewed
(Zipfian) access patterns. The key innovation is a **two-tier record-oriented architecture**
that separates hot (frequently accessed) records from cold records, eliminating the
"death spiral" problem where garbage collection of cold data flushes hot data to disk.

### 1.1 The Death Spiral Problem

Original FASTER uses a single HybridLog. During GC/compaction, cold records at the log's
beginning are copied to the tail. This pushes hot (in-memory) records out to disk. When
compaction finishes, hot records re-enter the tail, evicting cold records again — triggering
another compaction cycle. The system becomes entirely occupied with background compaction
rather than serving requests.

### 1.2 Key Results (vs Baselines)

| System       | Avg Speedup |
|-------------|-------------|
| vs FASTER v1 | 2.1×        |
| vs LeanStore  | 2.0×        |
| vs KVell      | 11.9×       |
| vs SplinterDB | 4.6×        |
| vs RocksDB    | 11.8×       |

Best case: **22× over RocksDB** on read-heavy YCSB-C.

---

## 2. Architecture (from C++ Source: `cc/src/core/f2.h`)

### 2.1 Component Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                          F2Kv<K,V,D>                             │
│                                                                  │
│  ┌─────────────────────┐        ┌─────────────────────────────┐  │
│  │     Hot Store        │        │       Cold Store             │  │
│  │  (FasterKv instance) │        │    (FasterKv instance)       │  │
│  │                     │        │                             │  │
│  │  ┌───────────────┐  │        │  ┌───────────────────────┐  │  │
│  │  │ Hot-Log Index  │  │        │  │ Cold-Log Index         │  │  │
│  │  │ (MemHashIndex) │  │        │  │ (ColdIndex, 2-level)   │  │  │
│  │  │ In-memory      │  │        │  │ Small in-mem + on-disk │  │  │
│  │  │ 7 entries/     │  │        │  │ 8 entries/bucket       │  │  │
│  │  │ bucket + ovfl  │  │        │  │ No overflow chaining   │  │  │
│  │  └───────────────┘  │        │  └───────────────────────┘  │  │
│  │                     │        │                             │  │
│  │  ┌───────────────┐  │        │  ┌───────────────────────┐  │  │
│  │  │ Hot HybridLog  │  │        │  │ Cold HybridLog         │  │  │
│  │  │ ┌───────────┐ │  │        │  │ (Primarily on-disk)     │  │  │
│  │  │ │  Mutable  │ │  │        │  └───────────────────────┘  │  │
│  │  │ │ (in-mem)  │ │  │        │                             │  │
│  │  │ ├───────────┤ │  │        └─────────────────────────────┘  │
│  │  │ │ Read-Only │ │  │                                         │
│  │  │ │ (in-mem)  │ │  │        ┌─────────────────────────────┐  │
│  │  │ ├───────────┤ │  │        │      Read Cache              │  │
│  │  │ │  Stable   │ │  │        │  (In-memory HybridLog,       │  │
│  │  │ │ (on-disk) │ │  │        │   NullDisk, 2nd-chance FIFO) │  │
│  │  │ └───────────┘ │  │        └─────────────────────────────┘  │
│  │  └───────────────┘  │                                         │
│  └─────────────────────┘        ┌─────────────────────────────┐  │
│                                  │   Background Worker          │  │
│                                  │  - Hot→Cold compaction       │  │
│                                  │  - Cold→Cold compaction      │  │
│                                  │  - Checkpoint/Recovery       │  │
│                                  └─────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### 2.2 Two-Tier Hash Index

**Hot-Log Index** (`cc/src/index/hash_bucket.h`):
- Cache-line aligned (64 bytes) buckets
- 7 hash bucket entries (8 bytes each) + 1 overflow pointer (8 bytes)
- Each entry: `address(48 bits) | tag(14 bits) | reserved(1) | tentative(1)`
- In-memory only, points to records in the hot HybridLog

**Cold-Log Index** (`cc/src/index/cold_index.h`):
- Two-level structure: small in-memory hash → on-disk hash chunks
- 8 entries per bucket (no overflow pointer)
- Each entry: `address(48 bits) | tag(3 bits) | reserved(12 bits) | tentative(1)`
- Hash chunks: 256 bytes = 32 entries (4 buckets × 8 entries)
- For 250M cold keys: ~8M chunks, ~64 MiB in-memory overhead (<1 byte/key)

### 2.3 Record Format (`cc/src/core/record.h`)

```
┌──────────────────────────────────────────────────────┐
│ RecordInfo Header (8 bytes)                          │
│  previous_address : 48 bits  (hash chain link)       │
│  checkpoint_ver   : 13 bits  (CPR version)           │
│  invalid          :  1 bit   (record invalidated)    │
│  tombstone        :  1 bit   (deleted record)        │
│  final            :  1 bit   (checkpoint final)      │
├──────────────────────────────────────────────────────┤
│ [alignment padding to key alignment]                 │
├──────────────────────────────────────────────────────┤
│ Key (variable length, immutable once written)        │
├──────────────────────────────────────────────────────┤
│ [alignment padding to value alignment]               │
├──────────────────────────────────────────────────────┤
│ Value (variable length, mutable in mutable region)   │
└──────────────────────────────────────────────────────┘
```

### 2.4 Address Format (`cc/src/core/address.h`)

48-bit logical addresses:
- Bits 0–24 (25 bits): offset within page (32 MB pages)
- Bits 25–47 (23 bits): page index (~8M pages)
- Bit 47: read-cache flag
- Bits 48–63: reserved for hash table (tag + control)

### 2.5 HybridLog Regions (`cc/src/core/persistent_memory_malloc.h`)

```
               TAIL (append point)
                ↓
  ┌────────────────────────────┐
  │      MUTABLE REGION        │  90% of in-memory budget
  │  - In-place atomic updates │  - Thread-safe via CAS
  │  - New records appended    │  - No latches needed
  ├────────────────────────────┤
  │      READ-ONLY REGION      │  10% of in-memory budget
  │  - Immutable in memory     │  - Read-Copy-Update semantics
  │  - Being flushed to disk   │  - Pin-based reader protection
  ├────────────────────────────┤
  │      STABLE (ON-DISK)      │  Unbounded
  │  - Persistent storage      │  - I/O required for access
  │  - Compaction source       │
  └────────────────────────────┘
                ↑
              BEGIN (GC boundary)
```

---

## 3. Core Algorithms

### 3.1 Epoch Protection Framework (`cc/src/core/light_epoch.h`)

Lazy synchronization without fine-grained locks. Every thread that enters a protected
region stores the current global epoch in a per-thread slot. The safe-to-reclaim epoch
is `min(all active thread epochs) - 1`.

```
Thread entry:    table[thread_id].local_epoch = global_epoch.load()
Thread exit:     table[thread_id].local_epoch = UNPROTECTED (0)
Bump epoch:      global_epoch++, register (epoch, callback) in drain_list
Safe reclaim:    scan all slots, fire callbacks with trigger_epoch <= min_active - 1
```

Key constants from C++:
- Table size: 96 entries (max threads)
- Drain list: 256 action slots
- CAS-based lock pattern: `kFree` → `kLocked` → `epoch_value`

### 3.2 Read Operation (`f2.h` lines 302–404)

```
Read(key):
  1. hash = Hash(key)
  2. entry = hot_index.Lookup(hash)
  3. status = hot_store.Read(key, entry)
     - Walk hash chain from entry.address backward
     - If found in mutable/read-only region → return value
     - If found on disk → async I/O, return Pending
  4. If status == NotFound:
     status = cold_store.Read(key)
     - Look up cold index (may require disk I/O for hash chunk)
     - Walk cold log chain
  5. If status == Ok AND read_cache_enabled:
     read_cache.TryInsert(key, value)  // 2nd-chance FIFO
  6. Return status
```

### 3.3 Upsert Operation

```
Upsert(key, value):
  1. hash = Hash(key)
  2. entry = hot_index.FindOrCreate(hash)
  3. Append record to hot log tail (mutable region)
     - Set record.previous_address = entry.address
  4. CAS(entry.address, old_addr, new_tail_addr)
     - If CAS fails: mark record invalid, retry from step 2
  5. Return Ok
```

### 3.4 Read-Modify-Write (Algorithm 1 in paper, `f2.h` lines 416–485)

The most complex operation, spanning both stores:

```
RMW(key, modifier):
  Stage 1 — HOT_LOG_RMW:
    1. entry = hot_index.FindOrCreate(hash)
    2. start_addr = entry.address  (capture for conditional insert)
    3. status = hot_store.Rmw(key, modifier, do_not_create=true)
    4. If found in mutable region → in-place update, return Ok
    5. If found in read-only region → copy to tail with modification

  Stage 2 — COLD_LOG_READ (if not found in hot):
    6. old_value = cold_store.Read(key)
    7. new_value = modifier.apply(old_value)  // or initial value if not found

  Stage 3 — HOT_LOG_CONDITIONAL_INSERT:
    8. Validate start_addr still valid (not truncated by compaction)
    9. ConditionalInsert(key, new_value, expected=start_addr)
       - Check range (start_addr, TAIL] for matching key
       - If no match → write to tail, CAS index entry
       - If match found → abort (concurrent update won)
    10. If aborted → push to retry_rmw_requests queue
    11. CompleteRmwRetryRequests() retries with fresh index lookups
```

### 3.5 Conditional Insert (Foundational Primitive)

Ensures exactly one record per key is inserted during compaction/RMW:

```
ConditionalInsert(key, value, expected_entry):
  1. Save current index entry
  2. Walk hash chain from most-recent address in [START, TAIL]
  3. If matching key found → ABORT (newer version exists)
  4. If no match → write record to log tail
  5. CAS index entry (expected → new_tail)
  6. If CAS fails:
     - Mark written record invalid
     - Re-scan only NEW records (between old and new head)
     - Repeat until success or abort
```

### 3.6 Compaction (`cc/src/core/compact.h`)

**Hot→Cold Compaction** (when hot log exceeds budget):
```
CompactHotToCold(begin, until):
  1. Scan hot log range [begin, until] using circular buffer (3 × 32MB frames)
  2. Distribute records to N compaction threads (default 4)
  3. Each thread: ConditionalInsert record into cold log
  4. After all records processed:
     - Set hot_log.begin = until (truncate)
     - GC hot index entries pointing to truncated range
```

**Cold→Cold Compaction** (when cold log exceeds budget):
```
CompactColdInPlace(begin, until):
  Same algorithm but source=cold log, target=cold log tail
  Uses full CI variant (check for newer versions in cold log)
```

Configuration defaults from `cc/src/core/config.h`:
```
Hot:  check=250ms, trigger=80%, compact=20%, max=256MB, budget=1GB, threads=4
Cold: check=250ms, trigger=90%, compact=10%, max=1GB,   budget=8GB, threads=4
```

### 3.7 Read Cache (`cc/src/core/read_cache.h`)

- Separate in-memory HybridLog using NullDisk (never persists)
- Hash chains span both read cache and hot log
- At most one read-cache entry per key
- **Second-chance FIFO**: if record in read-only region is re-accessed, copy to tail

```
ReadCache.Read(key, address):
  1. If address.in_readcache():
     - Verify key match, not invalid, address >= safe_head
     - Return value
     - If in read-only region → CopyToTail (second chance)
  2. Else: Skip() to next non-RC address → return NotFound

ReadCache.TryInsert(key, value):
  1. Write record to RC tail
  2. CAS index entry to point to RC record
  3. Set rc bit on new record's previous_address

Eviction (when RC head advances):
  1. Collect valid records in evicted page range
  2. CAS index entries to skip RC records → point to hot log addresses
  3. Multi-threaded participation via atomic counter
```

### 3.8 False-Absence Anomaly Prevention

During concurrent cold→cold compaction, a read might miss a record migrating
between cold log positions. Solution:

```
Shared atomic: num_truncs (counts cold log truncations)

ColdRead(key):
  1. t1 = num_truncs.load()
  2. tail = cold_log.tail
  3. result = scan cold log for key
  4. t2 = num_truncs.load()
  5. If t1 != t2: re-traverse newly appended hash chain portion
```

---

## 4. Key Hash & Bucket Entry Details

### 4.1 KeyHash (`cc/src/core/key_hash.h`)

```cpp
struct KeyHash {
  uint64_t control_;

  // Lower bits select bucket
  size_t hash_table_index(uint64_t table_size) {
    return control_ & (table_size - 1);  // power-of-2 mask
  }

  // Upper 16 bits as tag for fast discrimination
  uint16_t tag() {
    return static_cast<uint16_t>(control_ >> 48);
  }
};
```

### 4.2 Hash Bucket Entry (8 bytes)

```
Bits 0-47:   address (48 bits) — logical log address
Bits 48-61:  tag (14 bits) — hash fingerprint for fast rejection
Bit 62:      reserved
Bit 63:      tentative — marks entry as being inserted (CAS coordination)
```

### 4.3 Hash Bucket (64 bytes, cache-line aligned)

```
Hot-Log Index:  7 entries × 8B = 56B + 8B overflow pointer = 64B
Cold-Log Index: 8 entries × 8B = 64B (no overflow, closed addressing)
```

---

## 5. Go Translation Strategy

### 5.1 What We Keep From F2

| F2 Concept | Kestrel Go Implementation |
|-----------|--------------------------|
| Two-tier (hot + cold) | Sharded in-memory hot map + on-disk cold log |
| Hash chain via previous_address | In-memory: Go map. Cold: append-only log with index |
| Epoch protection | Simplified: per-shard RWMutex + atomic epoch counter |
| Read cache (2nd-chance FIFO) | Dedicated read cache with clock/FIFO eviction |
| Conditional Insert | CAS-based upsert with version checking |
| Hot→Cold compaction | Background goroutine, batch flush |
| Cold→Cold compaction | Background goroutine, log rewrite |
| Record format (header + K + V) | Binary record: `[flags(1)][hash(8)][keyLen(2)][key][ctLen(2)][ct][valLen(4)][val][created(8)][updated(8)]` |
| Bloom filter for cold | Lock-free concurrent bloom filter |

### 5.2 What We Change for Go

| F2 C++ Design | Go Adaptation | Rationale |
|--------------|---------------|-----------|
| 48-bit packed addresses | 64-bit file offsets | Go has no 48-bit types; simpler addressing |
| Template metaprogramming | Interface + concrete types | Go doesn't have templates |
| Cache-line aligned buckets | Shard-based locking (256 shards) | Go scheduler not cache-line aware; sharding is idiomatic |
| per-thread epoch table | sync.Pool + atomic counters | Go goroutines ≠ OS threads; epoch table impractical |
| HybridLog pages (32MB) | Append-only log file + mmap for reads | Simpler, leverages OS page cache |
| FixedPageAddress overflow | Chained hash in cold file | Go maps handle overflow natively |
| 96-thread limit | Unbounded goroutines | Go runtime handles scheduling |
| NullDisk for read cache | In-memory circular buffer | No need for disk abstraction |

### 5.3 Kestrel Architecture

```
┌───────────────────────────────────────────────────────┐
│                    kestrel.store                       │
│                                                       │
│  ┌─────────────────────────────────────────────────┐  │
│  │ HOT TIER (in-memory, 256 shards)                │  │
│  │  Each shard:                                     │  │
│  │   - sync.RWMutex                                │  │
│  │   - map[string]*record (composite key → record) │  │
│  │   - Pending index ops (deferred bloom updates)  │  │
│  │   - LRU/clock eviction tracking                 │  │
│  └─────────────────────────────────────────────────┘  │
│                                                       │
│  ┌─────────────────────────────────────────────────┐  │
│  │ READ CACHE (in-memory, separate from hot tier)  │  │
│  │  - Fixed-size circular buffer (configurable)    │  │
│  │  - 2nd-chance FIFO eviction (clock algorithm)   │  │
│  │  - Populated on cold reads                      │  │
│  │  - At most 1 entry per key                      │  │
│  │  - Separate 64-shard map for O(1) lookup        │  │
│  └─────────────────────────────────────────────────┘  │
│                                                       │
│  ┌─────────────────────────────────────────────────┐  │
│  │ COLD TIER (on-disk, append-only log)            │  │
│  │  File layout:                                    │  │
│  │   [header 64B][record0][record1]...[recordN]    │  │
│  │                                                  │  │
│  │  Cold Index (in-memory directory):               │  │
│  │   map[uint64][]coldEntry  (hash → entries list) │  │
│  │   Each entry: {offset, keyHash, tag}            │  │
│  │                                                  │  │
│  │  Bloom filter for fast negative lookups          │  │
│  └─────────────────────────────────────────────────┘  │
│                                                       │
│  ┌─────────────────────────────────────────────────┐  │
│  │ BACKGROUND WORKERS                              │  │
│  │  - Hot→Cold compactor (triggered by size)       │  │
│  │  - Cold→Cold compactor (triggered by size)      │  │
│  │  - Index updater (deferred bloom+keyIdx)        │  │
│  └─────────────────────────────────────────────────┘  │
│                                                       │
│  ┌─────────────────────────────────────────────────┐  │
│  │ PER-BUCKET KEY INDEX                            │  │
│  │  - Segmented sorted keys for O(matching) List   │  │
│  │  - Same design as falcon driver                 │  │
│  └─────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────┘
```

---

## 6. Record Format (Cold Log)

Variable-length records appended to the cold log file:

```
Offset  Size  Field
──────  ────  ──────────────────────
0       1     flags (occupied|tombstone|invalid)
1       8     hash (FNV-1a 64-bit)
9       2     keyLen (uint16, composite key length)
11      var   key (composite: "bucket\x00key")
11+kl   2     contentTypeLen (uint16)
13+kl   var   contentType
13+kl+cl 4    valueLen (uint32)
17+kl+cl var  value (inline bytes)
...     8     created (unix nano)
...     8     updated (unix nano)

Total record size = 33 + keyLen + ctLen + valueLen
```

For values > 1MB, the value is stored in a separate overflow file and the record
stores an 8-byte offset + 4-byte length instead of inline bytes (flagged by
`flagOverflow` in the flags byte).

### 6.1 Cold Index Entry (in-memory)

```go
type coldEntry struct {
    offset    int64   // byte offset in cold log file
    size      int32   // total record size
    hash      uint64  // full hash for verification
    tombstone bool    // true if record is deleted
}
```

The cold index is a sharded map: `hash % numColdShards` selects the shard,
then linear scan of entries with matching hash. This is equivalent to F2's
hash chunk approach but using Go's native maps.

---

## 7. Optimization Strategy (Target: 2× Falcon)

### 7.1 Phase 1 — Core Performance (Match Falcon)

1. **Sharded hot tier** with 256 shards (same as falcon)
2. **Allocation-free lookups** using `unsafe.String` + stack buffers
3. **Entry pooling** via `sync.Pool` for hot entries
4. **Value chunk allocator** (4MB bump pointer, same as falcon)
5. **Deferred index updates** (batch bloom+keyIndex via channel)
6. **Zero-copy readers** via pooled `dataReader`

### 7.2 Phase 2 — Read Cache (Beat Falcon by ~1.3×)

The read cache is the biggest architectural advantage F2 has. Falcon doesn't have one.

1. **Clock-based eviction**: O(1) eviction vs falcon's no-cache design
2. **Dedicated cache shards** (64 shards, separate from hot tier)
3. **Population on cold reads**: every cold hit populates the cache
4. **At-most-one invariant**: CAS-based insertion prevents duplicates
5. **Expected improvement**: 19–27% on read-heavy workloads (per paper benchmarks)

### 7.3 Phase 3 — Cold Log Optimization (Beat Falcon by ~1.5×)

Falcon uses fixed 256-byte slots with linear probing — severely wasteful for
small objects and slow for large cold files.

1. **Append-only log** vs fixed-slot: no wasted space, no probing
2. **In-memory hash index** for cold log: O(1) lookup vs O(probe_length)
3. **Batch I/O**: coalesce reads with `preadv` when available
4. **Background compaction**: live records rewritten to new log, dead space reclaimed
5. **Expected improvement**: 50–100% for mixed workloads with cold data

### 7.4 Phase 4 — Concurrency & Memory (Beat Falcon by ~2×)

1. **Separate read/write paths**: reads never contend with writes (RWMutex per shard)
2. **Lock-free bloom filter**: atomic OR for adds, atomic AND for queries
3. **Epoch-inspired GC**: use atomic generation counter to batch-free old cold log entries
4. **Amortized compaction**: spread compaction work across ticks, not burst
5. **Cold file mmap for reads**: let OS manage page caching, zero-copy into userspace
6. **Write-ahead buffer**: batch cold writes into 1MB buffer, flush on threshold

### 7.5 Summary of Expected Improvements

| Benchmark | Falcon Bottleneck | Kestrel Advantage | Expected Gain |
|-----------|-------------------|-------------------|---------------|
| Write | Allocation, cold slot probing | Chunk allocator, append-only cold | 1.2–1.5× |
| Read (hot) | Map lookup (same) | Map lookup + read cache hit | 1.0–1.3× |
| Read (cold) | Linear probe cold file | In-memory index + mmap | 2.0–3.0× |
| ParallelRead | Lock contention | Read cache reduces cold path | 1.5–2.5× |
| ParallelWrite | Shard contention | Same sharding, faster cold | 1.2–1.5× |
| MixedWorkload | No caching, cold probe | Read cache + append log | 2.0–3.0× |
| Delete | Cold tombstone + probe | Append tombstone (O(1)) | 1.5–2.0× |
| List | Same key index | Same (negligible) | 1.0× |
| Scale (10K) | Cold file growth | Compaction keeps file tight | 1.5–2.0× |

---

## 8. Implementation Plan

### 8.1 File Structure

```
pkg/storage/driver/zoo/kestrel/
├── RESEARCH.md          — This document
├── storage.go           — Driver, store, bucket, all operations
└── (single file, same pattern as falcon)
```

### 8.2 Core Types

```go
// Record in hot tier (in-memory)
type hotRecord struct {
    value       []byte
    contentType string
    created     int64  // unix nano
    updated     int64  // unix nano
    size        int64
}

// Record in read cache
type cacheEntry struct {
    key         string // composite key
    value       []byte
    contentType string
    created     int64
    updated     int64
    size        int64
    accessed    atomic.Bool // second-chance bit
}

// Cold log index entry
type coldEntry struct {
    offset    int64   // file offset
    size      int32   // record size
    hash      uint64  // for verification
    tombstone bool
}

// Hot tier shard
type hotShard struct {
    mu      sync.RWMutex
    m       map[string]*hotRecord
    pending []indexOp
    _       [24]byte // padding
}

// Read cache shard
type cacheShard struct {
    mu      sync.RWMutex
    m       map[string]*cacheEntry
}

// Cold index shard
type coldShard struct {
    mu      sync.RWMutex
    entries map[uint64][]coldEntry // hash → entries
}
```

### 8.3 Operation Flow

**Write(key, value)**:
1. Compute shard = `hash(bucket, key) % 256`
2. Lock hot shard (write)
3. Create/update entry in hot map
4. Queue deferred bloom+keyIndex update
5. If hot tier exceeds budget → signal compactor

**Open(key) — Read**:
1. Check hot shard (read lock) → if found, return pooled reader
2. Check read cache (read lock) → if found, mark accessed, return
3. Check bloom filter → if negative, return NotFound
4. Check cold index → find offset → read from cold log
5. Insert into read cache (2nd-chance FIFO)
6. Return value

**Delete(key)**:
1. Remove from hot shard
2. Invalidate read cache entry
3. Append tombstone to cold log + update cold index
4. Queue keyIndex removal

**Hot→Cold Compaction** (background):
1. Select entries to evict (oldest by `updated` timestamp)
2. Batch-serialize records to write buffer
3. Append buffer to cold log file
4. Update cold index with new offsets
5. Remove entries from hot shards
6. Update bloom filter

**Cold→Cold Compaction** (background):
1. Scan cold log from beginning
2. Skip tombstones and entries with newer versions
3. Rewrite live records to new log file
4. Atomically swap old log → new log
5. Rebuild cold index

### 8.4 DSN Format

```
kestrel:///path/to/data?hot_size=1048576&cache_size=65536&cold_budget=1073741824
```

Parameters:
- `hot_size`: max hot entries before compaction trigger (default: 1M)
- `cache_size`: read cache capacity in entries (default: 64K)
- `cold_budget`: cold log size budget in bytes (default: 1GB)
- `sync`: sync mode — `none`, `data`, `full` (default: `none`)

---

## 9. Benchmark Plan

### 9.1 Baseline Comparison

Run `cmd/bench` with `--drivers falcon,kestrel` to get head-to-head comparison:

```bash
go run ./cmd/bench --drivers falcon,kestrel --benchtime 2s --quick
```

### 9.2 Target Metrics

| Benchmark | Falcon (baseline) | Kestrel Target |
|-----------|-------------------|----------------|
| Write 256B | X ops/s | ≥ 2X ops/s |
| Write 1KB | X ops/s | ≥ 2X ops/s |
| Read 256B | X ops/s | ≥ 2X ops/s |
| Read 1KB | X ops/s | ≥ 2X ops/s |
| ParallelRead C16 | X ops/s | ≥ 2X ops/s |
| ParallelWrite C16 | X ops/s | ≥ 2X ops/s |
| MixedWorkload ReadHeavy | X ops/s | ≥ 2X ops/s |
| MixedWorkload Balanced | X ops/s | ≥ 2X ops/s |
| Scale 10K | X ops/s | ≥ 2X ops/s |

### 9.3 Optimization Iteration

1. Implement base kestrel driver with all features
2. Run benchmark, identify bottlenecks with `--profile`
3. Optimize hot path (allocation, locking)
4. Optimize cold path (I/O, index lookup)
5. Tune read cache parameters
6. Re-benchmark until 2× achieved across all metrics

---

## 10. Key Lessons from FASTER C++ Source

1. **Epoch > Locks**: The epoch framework replaces most fine-grained locking with
   cooperative synchronization. In Go, we approximate this with generation counters
   and `sync.Pool` for safe reclamation.

2. **Conditional Insert is fundamental**: Both compaction and RMW correctness depend
   on CI. Our Go version uses CAS on atomic version counters per entry.

3. **Read cache has outsized impact**: 19–27% improvement for just caching cold reads.
   This is kestrel's primary advantage over falcon.

4. **Two-level cold index scales**: <1 byte/key overhead for 250M keys. Our Go version
   uses sharded maps which are comparably efficient.

5. **Append-only beats fixed-slot**: F2 uses append-only logs, not fixed-size slots.
   This eliminates falcon's linear probing overhead and wasted slot space.

6. **Compaction must be incremental**: F2 compacts 10–20% at a time with 250ms checks.
   Burst compaction causes latency spikes.

7. **Background work must not block foreground**: Compaction runs on separate threads
   with careful synchronization. In Go, we use separate goroutines with non-blocking
   channel signals.

---

## 11. References

1. Kanellis, K. & Chandramouli, B. "From FASTER to F2: Evolving Concurrent Key-Value
   Store Designs for Large Skewed Workloads." PVLDB 18(12): 4910–4923, 2025.
   https://www.vldb.org/pvldb/vol18/p4910-kanellis.pdf

2. Microsoft FASTER GitHub Repository.
   https://github.com/microsoft/FASTER

3. PR #922: [C++] F2 KV store.
   https://github.com/microsoft/FASTER/pull/922
   19,706 additions, 3,014 deletions, 81 files. Merged 2025-02-17.

4. Chandramouli, B. et al. "FASTER: A Concurrent Key-Value Store with In-Place Updates."
   SIGMOD 2018. (Original FASTER paper)

5. Key C++ source files referenced:
   - `cc/src/core/f2.h` — F2Kv top-level class
   - `cc/src/core/faster.h` — FasterKv core
   - `cc/src/core/record.h` — Record/RecordInfo layout
   - `cc/src/core/address.h` — 48-bit address scheme
   - `cc/src/core/light_epoch.h` — Epoch protection
   - `cc/src/core/persistent_memory_malloc.h` — HybridLog allocator
   - `cc/src/core/read_cache.h` — Read cache
   - `cc/src/core/compact.h` — Compaction
   - `cc/src/core/config.h` — Configuration defaults
   - `cc/src/index/hash_bucket.h` — Bucket entry format
   - `cc/src/index/hash_table.h` — Hash table
   - `cc/src/index/cold_index.h` — Two-level cold index
   - `cc/src/index/mem_index.h` — In-memory hash index
