# Ant Storage Driver -- SMART ART (OSDI 2023)

## Paper Architecture: SMART -- Adaptive Radix Tree

The SMART paper (OSDI 2023) introduces the first Adaptive Radix Tree (ART) designed
for disaggregated memory with hybrid concurrency. The core data structure is the
Adaptive Radix Tree, which provides O(key_length) lookup by decomposing keys
byte-by-byte and adapting internal node representation based on occupancy.

### Adaptive Radix Tree (ART) Node Types

ART uses four node types that dynamically adapt as children are added or removed:

- **Node4**: Holds up to 4 children. Uses a 4-byte key array and 4 child pointers.
  Lookup is a simple linear scan over 4 entries. This is the most compact
  representation, used when a node has very few children.

- **Node16**: Holds up to 16 children. Uses a 16-byte sorted key array and 16
  child pointers. Lookup uses sorted binary search or, on x86, SIMD comparison
  (SSE2 `_mm_cmpeq_epi8`) to find the matching key byte in a single instruction.

- **Node48**: Holds up to 48 children. Uses a 256-byte child index array (one
  byte per possible key byte value, mapping to a slot index 0..47) and 48 child
  pointers. Lookup is O(1): index the 256-byte array by the key byte to get the
  slot, then dereference the pointer at that slot.

- **Node256**: Holds up to 256 children. Uses a direct 256-slot pointer array.
  Lookup is O(1) by directly indexing the array with the key byte. This is the
  most space-expensive but fastest representation.

### Key Decomposition and Traversal

Keys are decomposed byte-by-byte for traversal. Each level of the tree processes
exactly one byte of the key. For an N-byte key, the tree has at most N levels.
This gives O(key_length) lookup complexity, independent of tree size, compared to
O(tree_height * fanout) for B+-trees where tree height grows with data volume.

### Path Compression

When a sequence of nodes each has only a single child, ART applies path
compression: the single-child chain is collapsed into a stored prefix on the
node. This avoids creating inner nodes that would each hold only one child.
During lookup, the compressed prefix bytes are compared in bulk rather than
traversing one level per byte.

### Lazy Expansion

Inner nodes are not created until actually needed. When inserting a key that
shares a prefix with an existing leaf but diverges at some byte position, ART
creates the minimum number of inner nodes to distinguish the two keys. This
keeps the tree shallow and memory-efficient.

### Node Growth and Shrink

Nodes grow when capacity is exceeded:
- Node4 (full at 4) -> promote to Node16
- Node16 (full at 16) -> promote to Node48
- Node48 (full at 48) -> promote to Node256

Nodes shrink when occupancy drops:
- Node256 (at 48 or below) -> demote to Node48
- Node48 (at 16 or below) -> demote to Node16
- Node16 (at 4 or below) -> demote to Node4

### Performance Results from Paper

- 6.1x higher write throughput versus B+-trees
- 2.8x higher read throughput versus B+-trees
- First ART implementation for disaggregated memory with hybrid concurrency
  (optimistic reads with versioned locks for writes)

## Our Implementation Plan

### Full ART with All 4 Node Types

We implement the complete ART data structure with Node4, Node16, Node48, and
Node256. Keys are composite strings formed as `bucket + "\x00" + key` and
traversed byte-by-byte through the tree.

### Persistence Strategy

The driver uses an append-only value log combined with an in-memory ART index:

1. **Value log file** (`values.dat`): Append-only file for value data. Each
   entry at a given offset has the format:
   `ctLen(2B) | contentType | valLen(8B) | value | created(8B) | updated(8B)`
   Leaf nodes in the ART store the offset and size to locate the value.

2. **ART tree** (in-memory): The full ART with four node types. Leaf nodes
   contain `{valueOffset, valueSize, contentType, created, updated}`.

3. **WAL file** (`wal.log`): Write-ahead log for crash recovery. Each WAL
   entry has format:
   `[op(1B)] [keyLen(2B)] [key] [valOffset(8B)] [valSize(8B)] [ts(8B)]`
   On recovery, the WAL is replayed to rebuild the ART.

4. **Snapshot file** (`tree.snap`): Periodic ART serialization using DFS
   traversal for faster recovery:
   `[nodeType(1B)] [prefix...] [children...]`

### File Layout

```
{root}/
  values.dat   -- append-only value log
  wal.log      -- write-ahead log
  tree.snap    -- periodic ART snapshot (optional, faster recovery)
```

### Write Path

1. Append value data to `values.dat`, record offset and size.
2. Append operation record to WAL.
3. Insert composite key into ART, growing nodes as needed.

### Read Path

1. Traverse ART byte-by-byte using the composite key.
2. Find leaf node, read `valueOffset` and `valueSize`.
3. Read value data from `values.dat` at the recorded offset.

### Delete Path

1. Mark leaf as deleted in ART (soft delete).
2. Append delete record to WAL.
3. Value data remains in `values.dat` (reclaimed on compaction, not implemented).

### Bucket and Key Management

- Composite key: `bucket + "\x00" + key`
- Bucket metadata: in-memory `map[string]time.Time` with `sync.RWMutex`
- Bucket operations (create, delete, list) managed via the bucket map
- Directory support via prefix scanning on the ART

### Interface Compliance

The driver implements:
- `storage.Storage` (Bucket, Buckets, CreateBucket, DeleteBucket, Features, Close)
- `storage.Bucket` (Name, Info, Features, Write, Open, Stat, Delete, Copy, Move, List, SignedURL)
- `storage.HasDirectories` (Directory returning storage.Directory)
- `storage.HasMultipart` (InitMultipart, UploadPart, CopyPart, ListParts, CompleteMultipart, AbortMultipart)
- `storage.BucketIter` and `storage.ObjectIter` for iteration
- `storage.Directory` (Bucket, Path, Info, List, Delete, Move)
