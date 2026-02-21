# Falcon Storage Driver - Research Notes

## Source Paper

**"From FASTER to F2: Evolving Concurrent Key-Value Store Designs for Large
Skewed Workloads"**
Authors: Konstantinos Kanellis, Badrish Chandramouli, Ted Hart, Shivaram Venkataraman
Venue: PVLDB Vol 18, No 12, 2025
Lineage: Evolved from Microsoft Research's FASTER (SIGMOD 2018)

## F2 Architecture (Deep Analysis)

### 1. Epoch-Based Reclamation Framework

F2 inherits and extends FASTER's epoch-based synchronization. The framework has
three core components:

- **Global atomic counter E**: Monotonically increasing 32-bit integer
  representing the current epoch.
- **Thread-local epoch table**: Sharded table where each thread T maintains a
  cache-line-aligned entry holding its local epoch `E_t`.
- **Safe epoch E_s**: `min(E_t for all active threads T)`. When all thread-local
  epochs exceed some epoch C, C is considered synchronized and safe for deferred
  action execution.

#### Four Core Operations

1. **Acquire**: Thread T joins the epoch table, setting `E_t = E`.
2. **Refresh**: Thread T updates `E_t` to current E, recomputes `E_s`, and
   executes ready actions from the drain-list. Called every ~256 operations.
3. **BumpEpoch(action)**: Atomically increments E from c to c+1 and appends
   `<c, action>` to the drain-list.
4. **Release**: Removes thread T's entry from the epoch table.

#### Drain-List and Trigger Actions

The drain-list is a small fixed-size array of `<epoch, action>` pairs. When
`E_s` advances during Refresh, the system scans this array. For each entry where
`epoch < E_s`, the action fires exactly once (enforced via CAS). Used for:

- **Page flushing**: When TailAddress crosses a page boundary, a trigger fires
  async I/O to write the page to disk once all threads have advanced.
- **Circular buffer maintenance**: Memory-safe garbage collection of log pages.
- **Hash index scaling**: Safe resizing of the hash table.
- **Compaction frame advancement**: When a compaction frame is closed, a trigger
  fires to prefetch the next log page into the freed frame.

#### How It Differs from RCU and Hazard Pointers

| Aspect | F2 Epoch Protection | RCU | Hazard Pointers |
|--------|-------------------|-----|-----------------|
| Overhead | Minimal — mostly thread-local; global touched ~every 256 ops | Read-side near-zero; write-side waits for grace period | Every pointer access requires publishing a hazard pointer |
| Memory Bound | Unbounded — one slow thread prevents all reclamation | Unbounded (same weakness) | Bounded — unprotected objects freed immediately |
| Granularity | Epoch-granularity (batch) | Grace-period granularity | Per-object |
| Trigger Actions | Unique to F2 — arbitrary callbacks on epoch transitions | No equivalent | No equivalent |
| Cache Behavior | Excellent — thread-local epochs are cache-line aligned | Excellent read-side | Poor — scanning all hazard pointers causes contention |

The key innovation over standard EBR is **trigger actions**: the epoch framework
enables arbitrary deferred global actions (page flushes, index updates, compaction
coordination) to fire lazily when all threads have advanced.

### 2. Lock-Free ConditionalInsert

ConditionalInsert (CI) appends a record R to a target log's tail **if and only
if** no other record with a matching key exists within `[START, TAIL]` of a
source log. Two properties:

- **Freshness**: If CI succeeds, R is the most recent version for its key.
- **Absence**: If CI aborts, a newer record already exists.

#### Exact CAS Sequence

1. **Index Lookup**: Look up hash index entry for R's key. Store a copy of the
   current entry (address of most-recent record in this hash chain).
2. **Backward Traversal**: Follow hash chain backwards. Each record's header
   contains a pointer to the previous record. Issue disk I/O as needed.
3. **Liveness Check**: If matching key found → abort. If reaching end of chain
   or address < START → record is live, proceed.
4. **Write to Log Tail**: Append R at current TailAddress.
5. **CAS on Index Entry**: Atomic CAS expecting the entry value from step 1.
   New value points to just-written record.
6. **On CAS Failure** (concurrent insertion):
   - Invalidate the written log record (set invalidation bit in header).
   - Restart search, only examining newly-introduced records (between old and
     new index entry addresses).
   - If newer matching key found → abort.
   - Otherwise write new copy and retry CAS.
7. **Repeat** until CAS succeeds or abort.

ABA problem is handled implicitly: the log is append-only with monotonically
increasing addresses, so index entry values can only increase. No reversal
possible.

#### Variants for Compaction

- **Hot→Cold**: CI follows hash chain across entire hot log address range. Cold
  log records are older by design, freshness naturally satisfied.
- **Cold→Cold**: Both source and target are the same log. CI safely compacts
  live records to cold log tail even with concurrent appends.

### 3. Hybrid Log Address Space

The HybridLog unifies a 2^48 logical address space into three regions:

```
[BEGIN] ......... [HeadAddress] ........... [SafeReadOnlyAddress] ........... [TAIL]
   |--- STABLE ---|--- READ-ONLY (in-memory) ---|--- MUTABLE (in-memory) ---|
      (on-disk)
```

- **Mutable Region**: Records updated in-place via atomic CAS. Default: 90% of
  in-memory budget (`MutableFraction = 0.9`).
- **Read-Only Region**: Records immutable. Updates trigger read-copy-update —
  new copy appended to mutable tail, index CAS-updated.
- **Stable Region**: On-disk. Accessing requires async disk I/O.

#### Key Addresses

- **BEGIN**: First valid log address.
- **HeadAddress**: Oldest in-memory address (disk/memory boundary).
- **SafeReadOnlyAddress**: All threads have acknowledged this boundary.
- **ReadOnlyAddress**: Latest announced boundary (some threads may lag).
- **TAIL**: Next write position.

#### How ReadOnlyAddress Moves

1. Tail advances past mutable budget → atomically update ReadOnlyAddress.
2. `BumpEpoch` with trigger action.
3. When epoch safe → `SafeReadOnlyAddress = ReadOnlyAddress`.
4. Guarantees no thread still performing in-place updates on now-read-only pages.

#### Circular Buffer

- Fixed-size pages (default 2^25 = 32 MiB).
- Total buffer: `2^(MemorySizeBits - PageSizeBits)` pages.
- Pages recycled in a ring: `page_index = logical_page_number % num_pages`.
- When TailAddress crosses a page boundary, oldest in-memory page is flushed.

### 4. Hash Index Structure

#### 64-Byte Cache-Line-Aligned Buckets

Each bucket = 64 bytes (one CPU cache line):
- 7 entries × 8 bytes = 56 bytes
- 1 overflow pointer × 8 bytes = 8 bytes

#### 8-Byte Entry Format

| Bits | Field | Purpose |
|------|-------|---------|
| 48 | Address | Logical address of most-recent record in hash chain |
| 15 | Tag | Hash tag bits for quick rejection without dereferencing |
| 1 | Tentative | Flag for two-phase insertion protocol |

The **hash tag** provides quick rejection: check tag bits before following pointer
to fetch actual record and compare keys. Avoids cache misses on non-matching
entries.

#### Two-Phase Insertion Protocol (Tentative Bit)

1. Find empty slot, write tag with **tentative bit set**.
2. Re-scan bucket for duplicate tags (detecting concurrent insertions).
3. If unique → **clear tentative bit** (finalize).
4. If duplicate → set record Invalid, return RETRY_LATER.

#### Overflow Chain

When all 7 entries occupied: allocate overflow bucket from page-granularity
allocator, link via the 8th slot. Creates a chain of 64-byte buckets.

### 5. Read Cache

A separate HybridLog instance with **only mutable and read-only regions** (no
disk backing). Serves read-hot, write-cold records.

**Constraint**: At most one read-cache record per hash chain.

#### Promotion: Disk → Read Cache

When Read finds a record on disk:
1. Copy to read-cache tail (mutable region).
2. Update hash index to point to read-cache copy.
3. Future reads serve from cache.

#### Second-Chance FIFO Eviction

- Records inserted at tail, age toward head.
- When a read-only record is accessed again → copied to tail (second chance).
- Ensures most read-hot records are never evicted.

**Important**: F2 does NOT use bloom filters. The paper explicitly notes bloom
filters as overhead in LSM-tree systems like RocksDB. F2 avoids this via its
tiered log design and read cache.

### 6. Lookup-Based Compaction

F2 replaces FASTER's scan-based compaction with multi-threaded lookup-based
compaction built on ConditionalInsert.

#### Two-Phase Process

**Phase 1 — Copying:**
1. Sequentially scan fixed range `[BEGIN, UNTIL]` of source log (~10% of total).
2. For each key, attempt ConditionalInsert to target log.
3. CI's backward traversal determines liveness — only live records copied.
4. Stale records (newer versions elsewhere) naturally skipped.

**Phase 2 — Truncation:**
1. Atomically set `BEGIN = UNTIL`.
2. Invalidate hash index entries pointing to addresses < BEGIN via CAS.
3. Use `num_truncs` counter to detect/handle false-absence anomaly.

#### Multi-Threaded Execution

Active frontier maintained in 3 in-memory frames (~96 MiB total):
1. Populate frames with initial log pages.
2. Each thread does fetch-and-add to claim next record in current frame.
3. Each thread independently issues ConditionalInsert.
4. When frame exhausted → close + register epoch trigger action.
5. When trigger fires → async I/O prefetches next page into recycled frame.

#### Memory Comparison

| Metric | F2 Lookup-Based | FASTER Scan-Based |
|--------|----------------|-------------------|
| Memory | 120 MiB (3 frames) | 3 GiB (full hash table) |
| Threading | Multi-threaded (4+) | Single-threaded |
| Speed | 5.2x faster (4 threads) | Baseline |
| Memory efficiency | 25x less | Baseline |

### 7. Memory Management

#### Default Budget (3 GiB total, 10% of 30 GiB dataset)

| Component | Budget | Purpose |
|-----------|--------|---------|
| Hot-log hash index | 512 MiB (~4M buckets) | In-memory index for write-hot records |
| Hot-log in-memory region | ~1.75 GiB | Mutable + read-only regions |
| Read cache | 512 MiB | Cache for read-hot write-cold records |
| Cold-log in-memory region | 64 MiB | Small in-memory portion |
| Cold-log index | 64 MiB | Two-level cold index |

#### Cold-Log Index Memory Efficiency

Two-level index with 256-byte hash chunks (32 entries × 8 bytes):
- 250M cold keys → ~8M chunks → ~64 MiB in-memory index
- Per-cold-key overhead: ~1 byte
- Savings vs flat index: ~1.75 GiB for 250M keys, ~6 GiB for 1B keys

### 8. Performance (Paper Results)

#### Throughput (16 cores, 10% memory budget, Zipfian skew)

| vs. Baseline | YCSB-A (50R/50W) | YCSB-B (95R/5W) |
|-------------|-------------------|-------------------|
| F2 vs RocksDB | 11.75x | - |
| F2 vs SplinterDB | 4.52x | - |
| F2 vs KVell | 10.56x | - |
| F2 vs LeanStore | 2.04x | 1.1-1.4x |
| F2 vs FASTER | 2.38x | 1.5x |

#### I/O Amplification

| System | Read-Amp (A) | Read-Amp (B) | Write-Amp (A) | Write-Amp (B) |
|--------|-------------|-------------|---------------|---------------|
| F2 | 6.41 | 5.50 | 1.23 | 1.77 |
| FASTER | 7.23 | 5.03 | 2.62 | 1.21 |
| RocksDB | 5.28 | 2.64 | 5.28 | 2.64 |

Key: F2 writes 1.7x fewer bytes to disk than best competitor due to in-place
updates in the mutable region.

#### Memory Budget Sensitivity

At 750 MiB (2.5% of 30 GiB dataset):
- YCSB-A: 36% of peak throughput (1.73x better than LeanStore)
- YCSB-B: 83% of peak throughput (2.14x better than LeanStore)

#### Thread Scaling

Linear from 1 to 6 threads. Flattens at 10-12 threads (NVMe SSD bandwidth
saturation, not CPU contention).

## Falcon Implementation

### How We Map F2 Concepts

| F2 Concept | Falcon Implementation |
|------------|----------------------|
| HybridLog mutable tail | Sharded in-memory hash map (256 shards, sync.RWMutex) |
| HybridLog read-only/stable | Cold file (cold.dat) with 256-byte aligned slots |
| 64-byte cache-line buckets | Go map (runtime hash table with bucket chaining) |
| Hash tag for quick rejection | FNV-1a hash stored in cold slot header |
| ReadOnlyAddress boundary | hotMaxBytes threshold triggers background demote |
| Epoch protection | Dedicated demote goroutine + flushMu + atomic counters |
| Read cache | Automatic hot promotion on cold read |
| ConditionalInsert | Not needed — Go map semantics handle updates atomically |
| Lookup-based compaction | Shard-local eviction (no global sort/collect) |
| In-place update in mutable region | Direct map write for existing keys |
| Page-based circular buffer | Per-shard maps (GC handles memory recycling) |

### Hot Tier: Sharded Concurrent Hash Map

- 256 shards, FNV-1a hash of composite key `bucket + "\x00" + key`.
- Each shard: `sync.RWMutex` + `map[string]*hotEntry`.
- `hotEntry` struct pooled via `sync.Pool` to reduce GC pressure.
- Allocation-free lookups: `[256]byte` stack buffer + `unsafe.String` for
  map lookups without heap allocation.
- Byte-capped eviction: `hotBytes atomic.Int64` tracks total value bytes.
  When exceeding `hotMaxBytes` (1 GB), signals background demote goroutine.

### Cold Tier: Hash-Indexed On-Disk File

- `cold.dat`: 64-byte header + 256-byte aligned slots.
- Slot format: hash(8B) + keyLen(2B) + key + ctLen(2B) + ct + valLen(8B) +
  value + created(8B) + updated(8B) + flags(1B).
- Values > ~200B stored in `overflow.dat` (8-byte offset + 8-byte length).
- Collision: linear probing. Load factor < 0.7.
- Initial capacity: 4M slots (~1 GB file) to avoid grow events.
- In-memory cold directory: O(1) map lookup instead of linear probing.

### Bloom Filter (Divergence from F2)

F2 explicitly avoids bloom filters. Our falcon uses a lock-free concurrent
bloom filter (10 bits/item, 7 hash functions, atomic OR for adds) because:

1. Our cold tier uses linear probing (not HybridLog), so negative lookups
   require probing the cold file without a bloom filter.
2. The bloom filter eliminates cold I/O for keys that only exist in the hot
   tier, which is the common case for recently-written data.
3. At 10 bits/item, the false positive rate is < 1%, and adds use lock-free
   atomic ORs.

### Per-Bucket Key Index

Segmented sorted keys for O(matching) List operations:
- `sync.Map` for bucket lookup, `sync.RWMutex` per bucket.
- Optimistic RLock for key existence check (skip write lock when key exists).
- This replaces F2's full log scan for range queries.

### Demotion Strategy

Background demote goroutine (dedicated goroutine, never blocks writers):

1. Writers signal `demoteCh` when `hotBytes > hotMaxBytes`.
2. Goroutine processes signal: collect entries from shards (brief per-shard
   locks), then write to cold file (no shard locks held).
3. Two-phase to minimize hot-path contention: collect phase holds each shard
   lock briefly, write phase holds only cold.mu.
4. Target: evict down to `hotMaxBytes/2`.

### Zero-Copy Reads

`Open()` returns a pooled `dataReader` wrapping a `bytes.Reader` over the hot
entry's value slice directly. No copy. Reader returned to pool via `Close()`.

### Differences from F2

1. **Single process**: No distributed coordination. Epoch → flushMu + atomics.
2. **No in-place update on cold**: Hot tier entries replaced entirely (Go map).
   F2 does CAS on log records in the mutable region.
3. **Linear probing vs. bucket chaining**: Cold file uses linear probing. F2
   uses 64-byte cache-line-aligned bucket chains.
4. **No separate compaction**: Demotion overwrites cold slots directly.
5. **Overflow file**: F2 stores all records inline in HybridLog. We use a
   separate overflow file for values > ~200B.
6. **Bloom filter**: F2 avoids bloom filters; we add one for cold lookup
   elimination since our cold tier lacks F2's two-level index.
7. **Background demote goroutine**: F2 uses epoch trigger actions for page
   flushing. We use a dedicated goroutine signaled via channel.
8. **Go GC dependency**: F2 manages its own memory via circular buffer page
   recycling. We rely on Go's garbage collector for hot tier memory, using
   byte-capped eviction to keep heap manageable.

## References

- F2 paper: https://arxiv.org/abs/2305.01516 (PVLDB Vol 18, 2025)
- FASTER paper: https://www.microsoft.com/en-us/research/uploads/prod/2018/03/faster-sigmod18.pdf (SIGMOD 2018)
- FASTER GitHub: https://github.com/microsoft/FASTER
- FASTER docs: https://microsoft.github.io/FASTER/docs/fasterkv-basics/
- Epoch protection (VLDB Journal): https://link.springer.com/article/10.1007/s00778-024-00859-8
