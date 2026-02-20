# Falcon Storage Driver - Research Notes

## Source Paper

**F2: Designing a Key-Value Store for Large Skewed Workloads**
VLDB 2025 -- evolved from Microsoft Research's FASTER system.

## Paper Architecture (F2)

### Core Concept: Two-Tier Hash-Indexed Key-Value Store

F2 physically separates hot and cold records into two tiers to exploit
workload skew. Most real workloads follow a Zipfian-like distribution where a
small fraction of keys receive the majority of operations. By keeping hot
records in memory and cold records on disk with a memory directory, F2
achieves close-to-in-memory performance for the hot path while storing
arbitrarily large data sets.

### Hybrid Log

The Hybrid Log is the central persistence abstraction. It is an append-only
log that is divided into two regions:

1. **Mutable in-memory tail**: Recently written records live here. They can
   be updated in-place with atomic CAS operations, avoiding write
   amplification for hot keys.
2. **Immutable on-disk prefix**: Older records that have been flushed from
   the tail. Once on disk, records are never mutated in place.

A `ReadOnlyAddress` boundary separates the two regions. Records above it are
mutable in memory; records at or below it are immutable on disk and require
read-modify-append to update.

### Hot Tier

- In-memory hash table mapping keys to records in the mutable tail of the
  Hybrid Log.
- Hash table uses bucket chaining. Each bucket is a 64-byte cache line
  containing 7 entries plus an overflow pointer.
- Records in the hot tier can be updated in-place (atomic CAS on the value
  payload) without appending a new log entry, avoiding write amplification.

### Cold Tier

- On-disk region of the Hybrid Log for records that have aged out of the hot
  tier.
- Uses 256-byte aligned disk chunks to match SSD page granularity.
- A separate in-memory directory (hash map) maps cold keys to their
  (file offset, size) on disk, so cold lookups require exactly one random
  disk I/O.
- Reads from cold tier go through a read cache for frequently accessed cold
  records that are not quite hot enough to warrant promotion.

### Conditional-Insert Primitive

F2 introduces a lock-free `ConditionalInsert` operation:

- Atomically inserts a record ONLY if the key does not already exist.
- Used for compaction: the compactor re-inserts a live record; if a
  concurrent writer already wrote a newer version, the conditional insert
  fails harmlessly.
- Eliminates the need for per-key locking during compaction.

### Lookup-Based Compaction

Traditional LSM compaction (like RocksDB) scans entire sorted runs. FASTER's
original scan-based compaction scanned the entire log, which consumed 25x
more memory for bookkeeping.

F2's lookup-based compaction instead:

1. Scans only the cold region being compacted.
2. For each record found, looks up the in-memory index to see if the record
   is still the latest version.
3. If it is the latest, uses `ConditionalInsert` to re-insert it into the
   hot log tail.
4. If it is stale, the record is simply dropped (space reclaimed).

This approach uses 25x less memory than FASTER's scan-based compaction and
touches only relevant entries.

### Epoch Protection

F2 uses epoch-based reclamation for lock-free thread coordination:

- A global monotonic epoch counter advances periodically.
- Each thread registers itself and tracks its current epoch.
- Resources (log pages, index entries) are protected: they cannot be freed
  until all threads have advanced past the epoch in which the resource became
  unreachable.
- This avoids fine-grained locking on the index and log tail.

### Read Cache

A separate in-memory cache for disk-resident records that are read
frequently. On a cold read, the record is optionally copied into the read
cache. Subsequent reads of the same key hit the cache instead of disk.

### Performance Characteristics

- Write amplification 1.3-1.7x lower than RocksDB across workloads.
- Up to 8x higher throughput than RocksDB on skewed workloads.
- Near-zero overhead on pure in-memory hot-path operations.
- Compaction runs concurrently without blocking foreground operations.

## Our Implementation Plan

### Overview

The falcon driver adapts the F2 two-tier architecture to the liteio storage
interface. It is a single-file Go package that compiles without external
dependencies beyond the standard library and the storage package.

### Hot Tier: Sharded Concurrent Hash Map

- Sharded by FNV-1a hash of the composite key.
- 256 shards, each protected by its own `sync.RWMutex`.
- Each shard is a `map[string]*hotEntry` where the composite key is
  `bucket + "\x00" + key`.
- `hotEntry` holds: `value []byte`, `contentType string`, `created int64`,
  `updated int64`, `size int64`.
- All writes go to the hot tier first (in-memory). This mirrors F2's mutable
  tail region of the Hybrid Log.

### Cold Tier: Hash-Indexed On-Disk File

- Single file `cold.dat` with a fixed 64-byte header followed by 256-byte
  aligned slots.
- File layout: `[header 64B] [slot0 256B] [slot1 256B] ... [slotN 256B]`
- Each 256B slot format:
  - `hash` (8B): FNV-1a 64-bit hash of composite key
  - `keyLen` (2B): length of composite key
  - `key` (variable): composite key bytes
  - `ctLen` (2B): content type length
  - `ct` (variable): content type string
  - `valLen` (8B): value length
  - `value` (variable): inline value bytes (if fits)
  - `created` (8B): creation timestamp (Unix nano)
  - `updated` (8B): update timestamp (Unix nano)
  - `flags` (1B): 0x01=occupied, 0x02=tombstone, 0x04=overflow
- Values too large to fit inline (>~200B after metadata) are stored in a
  separate `overflow.dat` file. The slot then contains an 8-byte offset and
  8-byte length pointing into the overflow file instead of inline value data.
- Collision resolution: linear probing across slots.
- Load factor maintained at ~0.7; the file is grown (doubled) when exceeded.

### Promotion (Cold to Hot)

When a key is read from the cold tier, the entry is copied into the hot tier.
This mirrors F2's read cache behavior. Hot tier access is O(1) in-memory
hash lookup; cold tier requires file I/O.

### Demotion (Hot to Cold)

- When the hot tier entry count exceeds the `hot_size` parameter (default
  1,048,576 entries), a background flush evicts the oldest ~25% of entries
  to the cold tier.
- Eviction uses a simple approximate-LRU: entries are sorted by their
  `updated` timestamp and the bottom quartile is flushed.
- During demotion, each entry is written to a slot in `cold.dat` (and
  `overflow.dat` if needed).
- After successful write to cold tier, the hot entry is removed.

### Write Path

1. Compute composite key: `bucket + "\x00" + key`.
2. Hash to select hot tier shard.
3. Write to hot tier (lock shard, insert/update entry, unlock).
4. If hot tier count exceeds threshold, trigger async demotion.

This is analogous to F2's approach of always writing to the mutable
in-memory tail.

### Read Path

1. Compute composite key.
2. Check hot tier shard (RLock). If found, return data. (Fast path.)
3. If not in hot tier, probe cold tier file via linear probing.
4. On cold hit, promote to hot tier and return data.
5. If neither tier has the key, return `ErrNotExist`.

### Delete Path

1. Remove from hot tier if present.
2. Mark tombstone in cold tier if present (set flag byte to 0x02).

### Bucket Management

- In-memory `map[string]time.Time` of bucket name to creation time.
- Protected by `sync.RWMutex`.
- No per-bucket files. The composite key `bucket + "\x00" + key` naturally
  partitions the namespace.

### File Layout on Disk

```
{root}/
  cold.dat       -- hash-indexed cold tier (256B aligned slots)
  overflow.dat   -- overflow storage for large values
```

### Epoch-Based Cleanup

For safe concurrent access during demotion and compaction, we use a
simplified epoch scheme:

- A global epoch counter increments on each demotion cycle.
- Readers snapshot the current epoch on entry.
- Slots in the cold file are not freed until all readers from the previous
  epoch have completed.
- In practice, this is implemented via a `sync.WaitGroup` per epoch, which
  is simpler than the full epoch framework in F2 but sufficient for our
  single-process use case.

### DSN Format

```
falcon:///path/to/data?sync=none&hot_size=1048576
```

Parameters:
- `sync`: `none` (default, no fsync), `batch` (periodic fsync), `full` (fsync every write)
- `hot_size`: maximum number of entries in the hot tier before demotion triggers (default 1048576)

### Differences from F2

1. **Single process only**: No distributed coordination. Epoch protection is
   simplified to sync.WaitGroup.
2. **No in-place update**: Hot tier entries are replaced entirely (Go map
   semantics) rather than CAS on a log record. This is acceptable because
   Go's GC handles memory reclamation.
3. **Linear probing vs. bucket chaining**: We use linear probing in the cold
   file for simplicity. F2 uses 64-byte cache-line-aligned bucket chains.
4. **No separate compaction thread**: Demotion doubles as compaction. Stale
   cold entries are overwritten when their slot is reused.
5. **Overflow file**: F2 stores all records inline in the Hybrid Log. We use
   a separate overflow file for values that do not fit in 256B slots, which
   simplifies the slot format.
