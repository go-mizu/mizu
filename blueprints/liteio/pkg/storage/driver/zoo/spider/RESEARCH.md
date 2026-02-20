# Spider Storage Driver - Research Notes

## Paper Architecture: SplinterDB + Maplets (SIGMOD 2023)

### STBe-tree (Splinter Tree Be-tree)

SplinterDB introduces a "tree-of-trees" architecture where a **trunk tree**
backbone points to collections of **B-tree branches** at each level. This is
fundamentally different from LSM-trees (LevelDB/RocksDB) in that levels contain
multiple independent sorted runs rather than a single sorted table.

### Trunk Tree

The trunk tree is the backbone data structure. Each node in the trunk tree
contains pointers to branch sorted runs at its level. The trunk tree itself
is not a traditional B-tree; it is a routing structure that tracks which
branches exist at which level. When data flushes down from one level to the
next, the trunk tree is updated to reflect the new branch placement.

Key insight: the trunk tree is metadata-only. It does not store key-value
pairs directly. Instead, it stores references to branch files and their
key ranges.

### Branch Trees (Sorted Runs)

At each level, data is organized into multiple sorted runs (branches). Each
branch is a self-contained sorted sequence of key-value pairs, conceptually
similar to an SSTable in LSM-tree parlance. Branches at the same level may
have overlapping key ranges because they were flushed independently.

### Maplets

Maplets are compact per-branch filters that map keys to approximate values.
They extend the concept of Bloom filters from "is key present?" to
"is key present AND what is its approximate value?". This is sometimes
described as a key-value Bloom filter.

Maplets allow the read path to skip not just branches that definitely do not
contain a key (standard Bloom filter behavior), but also branches where the
key's value is known to be superseded by a newer branch. This dramatically
reduces read amplification in write-heavy workloads.

### Size-Tiered Compaction

When a level accumulates too many sorted runs (branches), SplinterDB merges
them into a single sorted run at the next level. This is size-tiered
compaction: rather than merging within a level (leveled compaction, as in
RocksDB default), branches are promoted downward.

Benefits:
- 9x insert throughput vs RocksDB
- 89% better query throughput
- 2x lower write amplification

### Read Path

Reads check all levels top-down. At each level, the maplets (or Bloom
filters in our simplified implementation) allow skipping branches that
definitely do not contain the key. The first match found at the highest
(most recent) level wins, because newer writes are always at higher levels.

### Write Path

Writes go to an in-memory buffer (memtable). When the memtable is full,
it is flushed to Level 1 as a new sorted run. If Level 1 now has too many
runs, they are compacted into Level 2, and so on.

---

## Our Implementation Plan

We implement a simplified but faithful version of the SplinterDB architecture.

### Level Structure

- **Level 0 (Memtable)**: In-memory sorted buffer protected by sync.RWMutex.
  All writes go here first. When it exceeds `memtable_size` bytes, it is
  flushed to Level 1 as a new on-disk sorted run file.

- **Level 1+**: On-disk sorted run files (SST-like). Each level can hold
  multiple sorted runs (branches). When a level accumulates more than 4 runs,
  all runs at that level are merged into a single run at the next level
  (size-tiered compaction).

### Sorted Run File Format (.sst)

Each run is a file named `L{level}_R{run}.sst` containing sorted key-value
pairs with a Bloom filter appended at the end.

```
File layout:
[Header: 32 bytes]
  magic     [4]byte   = "SPDR"
  version   uint32    = 1
  count     uint64    = number of entries
  minKeyLen uint16
  minKey    [minKeyLen]byte
  maxKeyLen uint16
  maxKey    [maxKeyLen]byte
  (padding to 32 bytes from start of magic)

  Actually, we use a simpler variable-size header:
  magic(4) | version(4) | count(8) | minKeyLen(2) | minKey | maxKeyLen(2) | maxKey

[Entries: repeated]
  keyLen(2) | key | ctLen(2) | contentType | valLen(8) | value | created(8) | updated(8)

[Bloom filter]
  bloomLen(4) | bloomBits(bloomLen bytes) | bloomNumHash(1)
```

- Tombstones are encoded as valLen = 0xFFFFFFFFFFFFFFFF (max uint64).
- Entries are sorted by composite key: `bucket + "\x00" + objectKey`.

### Bloom Filters

Each sorted run has a Bloom filter stored at the end of the SST file. We use
a double-hashing scheme with FNV-1a, targeting approximately 1% false positive
rate (10 bits per item, 7 hash functions).

On the read path, the Bloom filter is checked before performing a binary search
within the run. This avoids expensive disk I/O for keys that are definitely not
in the run.

### Level Manifest

A `manifest.json` file in the data directory tracks which runs exist at which
level. This is loaded at startup and updated atomically (write temp, rename)
after each flush or compaction.

```json
{
  "levels": {
    "1": ["L1_R0.sst", "L1_R1.sst"],
    "2": ["L2_R0.sst"]
  },
  "next_run_id": 5
}
```

### Compaction

When level L has more than 4 runs (configurable), all runs are merged into
one run at level L+1. The merge is a k-way merge of sorted sequences.
Tombstones are preserved during compaction unless the run being created is
at the maximum level (in which case tombstones can be dropped).

### File Layout

```
{root}/
  manifest.json
  L1_R0.sst
  L1_R1.sst
  L1_R2.sst
  L2_R0.sst
  ...
```

### Read Path

1. Check memtable (Level 0) - most recent data
2. For each level (1 to max), for each run (newest to oldest):
   a. Check Bloom filter - skip if definitely absent
   b. Binary search within run for the key
   c. If found, check for tombstone
3. First non-tombstone match wins (most recent level takes precedence)

### Write Path

1. Insert into memtable (protected by mutex)
2. If memtable exceeds `memtable_size`:
   a. Freeze current memtable
   b. Flush to new sorted run at Level 1
   c. Create new empty memtable
   d. If Level 1 now has >4 runs, trigger compaction to Level 2
   e. Cascade compaction checks to higher levels

### Delete Path

Insert a tombstone marker (valLen = 0xFFFFFFFFFFFFFFFF) into the memtable.
The tombstone propagates through flush and compaction like any other entry.

### Bucket/Key Management

Composite key format: `bucket + "\x00" + objectKey`. This allows all
bucket operations to map to a flat key-value store while maintaining
sort order within a bucket.

Bucket metadata (creation time) is tracked in an in-memory map with RWMutex,
persisted as part of the manifest.

### DSN Format

```
spider:///path/to/data?sync=none&levels=4&memtable_size=4194304
```

Parameters:
- `sync`: "none" (default, fast), "flush" (fsync after flush), "full" (fsync every write)
- `levels`: maximum number of disk levels (default 4)
- `memtable_size`: memtable size threshold in bytes before flush (default 4MB)
- `runs_per_level`: maximum runs per level before compaction (default 4)
