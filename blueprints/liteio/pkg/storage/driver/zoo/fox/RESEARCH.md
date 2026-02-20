# Fox Storage Driver - Research Notes

## Paper: Bf-Tree (VLDB 2024)

### Core Architecture

The Bf-Tree is a B-tree variant designed for modern storage hardware that
separates the hot write path from the cold on-disk storage path. The key
innovation is the **mini-page** concept: small, variable-length per-leaf
write buffers that live in a circular buffer pool in memory.

### B-tree Structure

- **Inner nodes** reside entirely in memory as sorted arrays of
  (separator key, child pointer) pairs. The branching factor is high
  (~100) to keep the tree shallow.
- **Leaf pages** are fixed-size (4 KB) pages stored on disk. Each leaf
  page contains sorted key-value entries and is the unit of I/O.
- The tree is traversed from root to leaf to find the target leaf for
  any given key. Inner node traversal uses binary search over the
  separator keys at each level.

### Mini-Pages

Mini-pages are the central contribution. Each leaf in the B-tree may have
an associated mini-page in the buffer pool. A mini-page serves three
roles simultaneously:

1. **Record cache** -- recently read records are present in the
   mini-page, avoiding a disk read on the next access.
2. **Write buffer** -- writes are absorbed into the mini-page without
   touching the disk page, batching many small writes into one eventual
   flush.
3. **Gap cache** -- the mini-page can record negative lookups (key
   ranges known to be absent), avoiding unnecessary disk reads.

Mini-pages grow dynamically from a minimum of 64 bytes to a maximum of
4096 bytes as more writes accumulate for that leaf. This adaptive sizing
means cold leaves consume almost no buffer pool space while hot leaves
get room to buffer many writes.

### Circular Buffer Pool

All mini-pages are allocated from a single circular buffer pool of
configurable size (default 16 MB). The pool uses LRU eviction: when the
pool is full and a new mini-page needs space, the least-recently-used
mini-page is evicted. On eviction, the dirty entries in the mini-page
are merged into the corresponding disk leaf page.

### Write Path

1. Traverse inner nodes to find the target leaf.
2. If a mini-page exists for this leaf, insert the record into it.
3. If no mini-page exists, allocate one from the buffer pool (evicting
   the LRU mini-page if necessary).
4. If the mini-page reaches 4096 bytes, flush it to the disk leaf page.
5. If the disk leaf page overflows after the flush, split the leaf and
   update the inner node.

### Read Path

1. Traverse inner nodes to find the target leaf.
2. Check the mini-page cache for the key.
3. If not found in the mini-page, read the disk leaf page and binary
   search for the key.

### Performance Characteristics

Compared to baselines measured in the paper:

- **2.5x faster scans** than RocksDB, because data is sorted in leaf
  pages (no LSM compaction overhead, no overlapping levels).
- **6x faster writes** than a standard B-tree, because writes are
  buffered in mini-pages and flushed in bulk.
- **2x faster point lookups** than RocksDB, because there is no bloom
  filter false-positive overhead and the single-level index is shallow.

---

## Our Implementation Plan

### Overview

We implement a simplified Bf-Tree as the `fox` storage driver. The driver
stores all data in a single directory with two files (`pages.dat` and
`meta.json`) plus an in-memory B-tree index with a mini-page buffer pool.

### B-tree Inner Nodes

- In-memory sorted arrays of `(separatorKey, childPageID)` pairs.
- The separator key is a composite `bucket + "\x00" + key` string.
- Binary search at each inner node to find the child pointer.
- Branching factor of ~100 keys per inner node (adjustable).

### Leaf Pages

- Fixed 4 KB pages stored sequentially in `pages.dat`.
- Page format:
  ```
  [pageHdr 16B]
    count    uint16  -- number of entries
    freeOff  uint16  -- offset of free space within page
    flags    uint16  -- page flags (0 = normal, 1 = overflow)
    nextPage uint32  -- overflow page ID (0 = none)
    pad      [6]byte -- reserved
  [entries...]
    keyLen   uint16
    key      [keyLen]byte
    ctLen    uint16
    ct       [ctLen]byte  -- content type
    valLen   uint32       -- 0xFFFFFFFF means tombstone
    value    [valLen]byte
    created  int64        -- unix nano
    updated  int64        -- unix nano
  ```
- Entries within a page are sorted by key for binary search on read.

### Mini-Page Buffer Pool

- In-memory LRU cache of per-leaf write buffers, configurable via the
  `pool_size` DSN parameter (default 16 MB).
- Each mini-page is a dynamic byte slice that starts at 64 bytes and
  grows up to 4096 bytes.
- On eviction, the mini-page's buffered entries are merged into the
  on-disk leaf page.
- The pool is protected by a mutex; individual mini-pages are not
  shared across goroutines (the store-level RWMutex serializes access
  to the tree).

### Write Path (Implementation)

1. Acquire store write lock.
2. Compose composite key: `bucket + "\x00" + key`.
3. Traverse B-tree inner nodes (binary search at each level) to find
   the target leaf page ID.
4. Look up or allocate a mini-page for that leaf.
5. Insert the entry into the mini-page.
6. If the mini-page exceeds 4096 bytes, flush it: merge all mini-page
   entries with the on-disk leaf page, write the merged page back, and
   split if the merged result exceeds 4096 bytes.
7. Release lock.

### Read Path (Implementation)

1. Acquire store read lock.
2. Compose composite key.
3. Traverse B-tree to the target leaf.
4. Check the mini-page for the key (linear scan of buffered entries).
5. If not found, read the leaf page from `pages.dat` and binary search.
6. Release lock.
7. Return the value (or ErrNotExist / tombstone).

### Delete Path

- Insert a tombstone entry (valLen = 0xFFFFFFFF) into the mini-page.
- On flush, tombstones remove the corresponding entry from the disk
  page.

### Meta File

- `meta.json` stores the root page ID, total page count, and tree
  height. It is rewritten on Close() and loaded on Open() for crash
  recovery of the tree structure.

### File Layout

```
{root}/
  pages.dat    -- 4 KB aligned leaf pages, sequentially allocated
  meta.json    -- B-tree metadata (root, height, page count)
```

### DSN Format

```
fox:///path/to/data?sync=none&page_size=4096&pool_size=16777216
```

Parameters:
- `sync` -- `none` (default, no fsync), `batch`, or `full`
- `page_size` -- leaf page size in bytes (default 4096)
- `pool_size` -- mini-page buffer pool size in bytes (default 16 MB)
