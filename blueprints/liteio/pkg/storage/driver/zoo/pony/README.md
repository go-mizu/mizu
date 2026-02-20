# Pony - Memory-Constrained Single-Volume Storage

Same append-only volume as Horse but with an on-disk mmap'd hash table index instead of an in-memory Go map. Keeps total process RSS under 100MB even at 1M+ objects — 3-5x lower than Horse.

## Architecture

```
Write Path:
  Key → Buffer ring claim (atomic, 2 × 4MB)
    → Serialize record to buffer
    → Flush to volume on full
    → Update on-disk hash table (linear probing)
    → Append composite key to string pool

Read Path:
  Key → FNV-1a hash → slot lookup in mmap'd hash table
    → Linear probe until match or empty slot
    → Read content-type from string pool
    → Zero-copy volume read (mmap or pread)
```

### On-Disk Hash Table

```
Index File Layout:
  [Header 64B: magic "PONY0001" + slotCount + entryCount + stringsPos]
  [Slot 0: 64B cache-aligned]
  [Slot 1: 64B cache-aligned]
  ...
  [Slot N-1: 64B cache-aligned]
  [String Pool: append-only composite keys + content types]

diskSlot (64B = 1 cache line):
  hash(8B) | strOff(8B) | strLen(4B) | ctLen(2B) | pad(2B) |
  valOff(8B) | valSize(8B) | created(8B) | updated(8B) | pad(8B)
```

- **Linear probing**: Open addressing with tombstone deletion (hash=1)
- **Load factor cap**: 75% → automatic rehash to 2× slot count
- **String pool**: Append-only, stores composite keys and content types
- **Default slots**: 65,536 (power-of-2 for fast modulo)

### Horse vs Pony Comparison

| Aspect | Horse | Pony |
|--------|-------|------|
| Index location | RAM (Go map) | On-disk (mmap'd hash table) |
| Memory @ 1M objects | 300-500MB | <100MB |
| Prealloc default | 64GB | 256MB |
| Buffer ring | 4 × 64MB | 2 × 4MB |
| Index persistence | No (full replay) | Yes (survives restart) |
| Recovery speed | O(n) volume scan | O(1) header load |
| Hash collision | Chain-free map | Linear probing |

### Index Operations

- **Lookup**: Hash → slot → linear probe until match or empty
- **Insert**: Find empty slot or tombstone, write 64B entry
- **Delete**: Set hash field to 1 (tombstone marker)
- **Rehash**: Triggered at 75% load factor
  1. Collect all live entries
  2. Munmap current file
  3. Rewrite with 2× slots
  4. Remmap

## DSN

```
pony:///path/to/data
pony:///path/to/data?sync=none
pony:///path/to/data?prealloc=256&slots=65536
pony:///path/to/data?bufsize=4194304
```

| Param | Default | Description |
|-------|---------|-------------|
| `sync` | `none` | Durability: `none`, `batch`, `full` |
| `prealloc` | `256` | Volume preallocation (MB) |
| `slots` | `65536` | Initial hash table slot count |
| `bufsize` | `4194304` | Index read buffer size (bytes) |

## Limitations

- **Smaller default buffers**: 4MB vs Horse's 64MB (more frequent flushes)
- **Rehash is expensive**: Full munmap → rewrite all entries → remmap (blocks all I/O)
- **String pool fragmentation**: Append-only, no garbage collection for deleted keys
- **Linear probing clustering**: At high load factor, probe chains lengthen significantly
- **No in-memory key cache**: List operations rebuild sorted list from scratch every time
- **Single-node only**: No cluster or replication support
- **No compression or encryption**
- **File growth on rehash**: Doubles slot count, could hit OS file limits at extreme scale

## Enhancement Opportunities

1. **Robin Hood hashing**: Reduce probe chain length variance
2. **Cuckoo hashing**: Better worst-case lookup performance
3. **String pool compaction**: Background GC for deleted key space
4. **In-memory sorted key cache**: LRU of frequently-listed prefixes
5. **Incremental rehashing**: Spread rehash work over time instead of stop-the-world
6. **Bloom filter overlay**: Fast negative lookups before disk probe
7. **Tiered index**: Hot entries cached in RAM, cold on disk

## Performance Profile

| Operation | Expected Performance | Notes |
|-----------|---------------------|-------|
| Write | Similar to Horse | Volume operations identical |
| Read | Similar to Horse | Volume path same; index adds 1 mmap lookup |
| Index lookup | O(1) average | O(n) worst case on probe clustering |
| Memory @ 1M objects | <100MB | 64B per slot + string pool |
| Recovery | O(1) | Load header + rebuild key list (no volume scan) |

**Memory budget**: 64B/slot × 65,536 default = 4MB initial index. Grows 2× on rehash.
