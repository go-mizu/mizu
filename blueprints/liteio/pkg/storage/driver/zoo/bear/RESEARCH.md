# Bear Storage Driver — Research Notes

## Paper: "B-Trees Are Back" (SIGMOD 2025)

### Core Thesis

A comprehensive engineering study demonstrating that well-engineered pageable B-trees
compete with in-memory data structures — even when the dataset fits entirely in RAM.
The paper systematically evaluates six B-tree node layout optimizations for variable-sized
records (keys of arbitrary length) and shows that thoughtful page layout engineering
eliminates the gap between disk-oriented and memory-optimized structures.

### Key Insight: Mmap as the Unifying Strategy

The paper uses memory-mapped files (`mmap`) as the I/O substrate. This gives:

- **In-memory fast path**: When data fits in RAM, mmap'd pages stay resident — no
  syscall overhead on reads. The OS page cache handles everything transparently.
- **Graceful degradation**: As data grows beyond RAM, the OS pages out cold data.
  The B-tree never needs to change its code path — `mmap` handles the transition.
- **4KB page alignment**: Pages are 4KB (the OS page size on x86/ARM), ensuring each
  B-tree node maps exactly to one virtual memory page. No partial page loads.

### The Six Node Layout Optimizations

The paper evaluates these in order of increasing sophistication:

#### 1. Sorted Inline

The baseline. Entries (key-value pairs) are stored sorted contiguously within the page.
Binary search finds the target key. Simple but expensive for inserts: shifting N/2 entries
on average per insertion.

```
[header] [entry0] [entry1] [entry2] ... [entryN] [free space]
```

#### 2. Indirection Slots

A fixed-size slot array at the front of the page maps logical position to physical byte
offset within the page. Entries are written from the end of the page backwards, and the
slot array grows forward. Insert only requires shifting 2-byte slot pointers (not the full
variable-length entries). This is the key enabler for variable-sized records.

```
[header] [slot0(2B)] [slot1(2B)] ... [slotN(2B)] ... free ... [entryN] ... [entry1] [entry0]
```

- Binary search operates on the slot array
- Insert: append entry at end, shift slot pointers
- Delete: remove slot, entry becomes garbage (compacted lazily)

#### 3. Prefix Truncation (Inner Nodes Only)

Inner nodes don't need full keys — they only need enough of the key to distinguish left
subtree from right subtree. Storing only the shortest distinguishing prefix reduces inner
node size dramatically, allowing higher fanout and shallower trees.

Example: keys "application" and "approach" can be separated by just "app" (or even "ao"
if the left subtree's max key is known).

#### 4. Head/Key Optimization

Store the first 4 bytes of each key directly in the slot structure. During binary search,
compare these 4 bytes first. If they differ from the search key's first 4 bytes, we can
skip the full key comparison entirely. Since most keys diverge in the first few bytes,
this eliminates the majority of cache misses from chasing pointers to the actual key data.

```
Slot: [keyHead(4B)] [offset(2B)]    // 6 bytes per slot
```

Binary search: compare `searchKeyHead` vs `slot.keyHead`. Only dereference
`slot.offset` to compare the full key when heads match.

#### 5. Hint-Based Search

Store key "hints" (hash or prefix bytes) in a separate contiguous array, independent
of the slot array. This array is scanned linearly or with SIMD to find candidate positions.
Because the hints are contiguous, the CPU can prefetch them efficiently — one cache line
(64 bytes) holds 16 four-byte hints, covering 16 keys in a single fetch.

#### 6. Hybrid Layouts

Select the best layout per node based on the characteristics of its records. Nodes with
short, fixed-size keys might use sorted inline. Nodes with long, variable-size keys use
indirection slots + head optimization. The page header encodes which layout is in use.

### Performance Results (from paper)

- On in-memory workloads, the optimized B-tree matches or beats red-black trees,
  ART (Adaptive Radix Tree), and hash maps for point lookups.
- Prefix truncation + head optimization gives 2-3x improvement over naive sorted layout.
- The indirection slot approach has < 5% overhead vs. sorted inline for reads, but
  10-50x faster for inserts (no data movement).
- Hint-based search shows diminishing returns for nodes with < 50 entries but helps
  significantly for high-fanout inner nodes.

---

## Our Implementation Plan

### Architecture

Single mmap'd file (`btree.dat`) containing 4KB pages organized as a B-tree.
All keys and values live within the B-tree pages — no separate data files.

### File Layout

```
{root}/
  btree.dat    — mmap'd B-tree (4KB pages)
```

### Page Format

#### Page 0: File Header

```
[magic "BEAR0001" (8B)]
[rootPage    (4B)]     // page ID of root node
[pageCount   (4B)]     // total allocated pages
[height      (4B)]     // tree height
[entryCount  (8B)]     // total key-value entries
[freeHead    (4B)]     // head of free page list (0 = none)
[pad to 4096B]
```

#### Inner Node Page (4KB)

```
[type(1B)=0x01] [count(2B)] [freeOffset(2B)]
[slotArray: count+1 children × {childPage(4B)} followed by count × {keyHead(4B), keyOffset(2B)}]
... free space ...
[key data packed from page end backwards]
```

Inner nodes store N keys and N+1 child page pointers. Keys serve as separators:
all keys in `child[i]` are < `key[i]` and all keys in `child[i+1]` are >= `key[i]`.

#### Leaf Node Page (4KB)

```
[type(1B)=0x02] [count(2B)] [freeOffset(2B)] [nextLeaf(4B)] [prevLeaf(4B)]
[slotArray: count × {keyHead(4B), entryOffset(2B)}]
... free space ...
[entries packed from page end backwards]
```

Leaf entry format:
```
[keyLen(2B)] [key bytes] [ctLen(2B)] [contentType bytes] [valLen(8B)] [value bytes] [created(8B)] [updated(8B)]
```

Leaf nodes are linked via `nextLeaf`/`prevLeaf` for efficient range scans.

### Optimizations Implemented

1. **Indirection slots**: Variable-length entries packed from end of page; fixed-size
   slots at front. Insert shifts only slot pointers (6 bytes each), not data.

2. **Head optimization**: First 4 bytes of each key stored in the slot. Binary search
   compares heads first, avoiding cache-miss-inducing pointer chases for ~90% of
   comparisons.

3. **Mmap**: Transparent OS paging via `syscall.Mmap` with `MAP_SHARED`. Small datasets
   stay fully resident; large datasets page gracefully.

4. **Page allocator**: New pages appended at end of file. Deleted pages tracked via a
   free list (singly linked through the page's first 4 bytes).

### Key Design: Composite Keys for Bucket/Key Management

All bucket+key pairs are stored in a single B-tree using composite keys:
`bucket + "\x00" + key`. The null byte separator ensures correct lexicographic ordering
(bucket names cannot contain null bytes). This avoids needing a separate B-tree per bucket.

Bucket metadata (creation time) is stored in an in-memory map backed by a special
key prefix `\x00bucket\x00{name}` in the B-tree.

### Operations

- **Write**: Root-to-leaf traversal, find target leaf, insert entry. If leaf full, split
  (allocate new page, redistribute half of entries, push separator to parent).
- **Read**: Root-to-leaf traversal using head comparison, binary search in leaf, read entry.
- **Delete**: Find entry in leaf, remove slot. If page drops below minimum fill, merge
  or redistribute with sibling.
- **List**: Find first leaf matching prefix, scan linked list of leaves.
- **Split**: Allocate new page, move upper half of entries to new page, insert median
  key as separator in parent. If parent also full, split recursively.

### Sync Modes

Configurable via DSN query parameter `sync=`:
- `none`: No explicit sync. Relies on OS writeback (fastest, data loss risk on crash).
- `msync`: Call `msync(MS_SYNC)` after mutations (safe, slower).
- Default is `none` for performance.
