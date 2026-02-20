# Jaguar Storage Driver - Research

## Paper Architecture: Jungle (HotStorage 2019, eBay)

Jungle is a key-value storage engine designed to solve write amplification in
LSM-tree based stores. It was published at HotStorage 2019 by Jung-Sang Ahn
et al. at eBay and is openly available on GitHub
(github.com/eBay/Jungle).

### Core Idea

Traditional LSM-trees (LevelDB, RocksDB) store each level as a set of
immutable sorted files (SSTables). Compaction rewrites entire files, producing
significant write amplification (10-30x in RocksDB). Jungle replaces SSTables
with **copy-on-write B+-trees** at each level.

### Architecture Components

1. **MemTable**: Lock-free skip list that absorbs all incoming writes. Backed by
   a Write-Ahead Log (WAL) for durability. Once the memtable exceeds a
   configured size, it is flushed to the first disk level.

2. **WAL (Write-Ahead Log)**: Append-only log file recording every write before
   it enters the memtable. On crash recovery, the WAL is replayed to rebuild
   the in-memory state. This ensures durability without requiring the memtable
   to be on disk at all times.

3. **CoW B+-tree Levels**: Instead of creating immutable sorted files at each
   LSM level, Jungle stores data in an append-only copy-on-write B+-tree per
   level. When a node is modified, a new copy is appended to the file and the
   parent pointer is updated. Old nodes become garbage that can be reclaimed.

4. **Merge / Compaction**: When flushing from a higher level to a lower one,
   Jungle only appends new or changed records to the target CoW B+-tree. This
   avoids rewriting the entire level, reducing write amplification by 4-5x
   compared to RocksDB.

5. **Persistent Zero-Cost Snapshots**: Because the tree file is append-only and
   old nodes are never overwritten in-place, a snapshot is simply a recording
   of the current root node offset and WAL sequence number. Readers holding a
   snapshot see a consistent, immutable view at no additional cost.

6. **MVCC**: Writers never block readers. Readers acquire a snapshot (root offset)
   and traverse a frozen tree structure while writers append to the end of the
   file. This gives full multi-version concurrency control.

### Performance Claims (from paper)

- 4-5x less write amplification than RocksDB on write-heavy workloads.
- Comparable read latency due to B+-tree point lookups (O(log N) per level).
- Snapshots are "free" -- no file duplication, no checkpoint cost.
- Smooth compaction throughput because merges are incremental appends.

---

## Our Implementation Plan (Jaguar)

Jaguar is a faithful but simplified implementation of the Jungle concepts,
adapted to the liteio storage driver interface.

### File Layout

```
{root}/
  wal.log       -- write-ahead log (append-only)
  level1.tree   -- CoW B+-tree (append-only)
  meta.json     -- metadata (root offset, WAL position, bucket map)
```

### Write-Ahead Log (WAL)

- Append-only binary log file (`wal.log`).
- Every write and delete is appended to the WAL before touching the memtable.
- Entry format:

  ```
  [type 1B] [keyLen 2B] [key ...] [ctLen 2B] [ct ...] [valLen 8B] [value ...] [ts 8B]
  ```

  - type: `0x01` = put, `0x02` = delete
  - keyLen: uint16 big-endian, length of composite key (bucket + "\x00" + key)
  - key: composite key bytes
  - ctLen: uint16 big-endian, length of content type string
  - ct: content type bytes
  - valLen: uint64 big-endian, length of value
  - value: raw value bytes
  - ts: int64 big-endian, UnixNano timestamp

- On recovery: replay WAL from beginning to rebuild memtable state.
- After a successful flush to the CoW B+-tree, the WAL is truncated.

### MemTable

- In-memory sorted data structure: a Go `sync.RWMutex`-protected sorted slice
  of entries (keys kept in sorted order via binary search insertion).
- Each entry stores: composite key, value bytes, content type, timestamp, and
  a tombstone flag for deletes.
- When total memtable size exceeds `memtable_size` (default 4 MB), a flush to
  the Level 1 CoW B+-tree is triggered.

### CoW B+-tree (Level 1)

- Single file (`level1.tree`) with append-only semantics.
- **File header** (24 bytes at offset 0):
  - Magic: 8 bytes (`"JAGUAR01"`)
  - Root offset: int64 big-endian (offset of current root node)
  - Node count: int64 big-endian (total nodes ever written)

- **Node format**: Each node is variable-length, appended to the end of file.
  - `[type 1B]`: 0x01 = leaf, 0x02 = inner
  - `[count 2B]`: number of entries in this node (uint16 big-endian)
  - Leaf entry: `keyLen(2B) | key | ctLen(2B) | ct | valLen(8B) | value | created(8B) | updated(8B)`
  - Inner entry: `keyLen(2B) | key | childOffset(8B)`
  - Inner nodes also have a trailing `rightmostChildOffset(8B)` after all entries.

- **Copy-on-Write**: When a leaf node is modified (insert, update, delete), a
  new copy of the leaf is appended to the file. Then, a new copy of its parent
  inner node (with the updated child pointer) is appended, and so on up to a
  new root. The old nodes remain in the file as historical data (enabling
  snapshots).

- **Read path**: Start at the root offset from the header. Binary search inner
  nodes to find the correct child. Descend to leaf. Binary search leaf for key.

- **Flush from memtable**: All memtable entries are merged into the tree by
  walking the tree from root to leaves, creating new leaf nodes as needed and
  propagating changes upward via CoW.

### Meta File

- `meta.json`: JSON file recording:
  - Current root offset in the tree file.
  - Total node count.
  - WAL position (byte offset of last flushed entry).
  - Bucket map: `map[string]string` of bucket name to creation timestamp.

- Updated atomically (write to temp file, rename) after each flush.

### Composite Keys

- All keys stored in the WAL and tree use a composite format:
  `bucket + "\x00" + objectKey`
- This allows a single tree to hold data for multiple buckets.
- Bucket listing is done from the in-memory bucket map.

### DSN Format

```
jaguar:///path/to/data?sync=none&memtable_size=4194304&wal=true
```

- `sync`: `none` (no fsync, fastest), `batch` (periodic fsync), `full` (fsync every write). Default: `none`.
- `memtable_size`: memtable flush threshold in bytes. Default: `4194304` (4 MB).
- `wal`: `true` or `false`. When false, WAL is skipped for maximum write speed (data loss risk). Default: `true`.

### Interface Coverage

Jaguar implements:
- `storage.Storage`: Bucket, Buckets, CreateBucket, DeleteBucket, Features, Close
- `storage.Bucket`: Name, Info, Features, Write, Open, Stat, Delete, Copy, Move, List, SignedURL
- `storage.HasDirectories`: Directory() returning storage.Directory
- `storage.HasMultipart`: InitMultipart, UploadPart, CopyPart, ListParts, CompleteMultipart, AbortMultipart
- `storage.BucketIter`, `storage.ObjectIter` iterators
- `storage.Directory`: Bucket, Path, Info, List, Delete, Move

### Simplifications vs. Full Jungle

1. Single level only (Level 1). The paper describes multiple levels with
   tiered compaction; we use just one CoW B+-tree file for simplicity.
2. No background compaction threads. Flush is synchronous.
3. Snapshot API not exposed externally (though the mechanism is present).
4. No bloom filters on the tree (the in-memory index serves that role).
5. The memtable uses a mutex-protected sorted slice rather than a lock-free
   skip list. This is simpler and sufficient for our access patterns.

### Performance Expectations

- Writes: One WAL append + one memtable insert. Bulk flushes amortize B+-tree
  I/O across many keys.
- Reads: Check memtable first (fast in-memory binary search), then B+-tree
  (O(log N) with disk seeks, partially mitigated by OS page cache).
- Write amplification: ~1x for WAL, ~1-2x for CoW tree appends. Much lower
  than traditional LSM compaction.
