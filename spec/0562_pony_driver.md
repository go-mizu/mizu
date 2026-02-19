# 0562: Pony — Memory-Constrained Object Storage Driver

**Status**: Implementing
**Target**: Total process memory < 100MB with competitive throughput vs Horse

## Problem Statement

Horse driver uses unlimited in-memory index (`map[string]*indexEntry`) and 256MB write buffers
(4 × 64MB ring). For 1M objects, the index alone consumes ~242MB:

| Component | Per Entry | 1M Entries |
|-----------|----------|------------|
| indexEntry struct | 48B | 46 MB |
| Go map overhead | ~62B | 59 MB |
| Composite key strings | ~50B | 48 MB |
| contentType strings | ~20B | 19 MB |
| bucketKeySet tracking | ~60B | 57 MB |
| **Total** | **~240B** | **229 MB** |

Plus 256MB write buffers = **485MB minimum** for Horse at 1M objects.

Pony achieves the same functionality under 100MB total process RSS by:
1. **On-disk hash table via mmap** — index lives in the filesystem page cache, not Go heap
2. **Small write buffers** — 2 × 4MB = 8MB (vs 4 × 64MB = 256MB)
3. **No large pool tiers** — cap pwrite buffers at 10MB
4. **Compact metadata** — content type read from volume on demand

## Research Survey

### Key-Value Separation (WiscKey, FAST'16)

Separates keys from values in the storage layer. Only `(key, location)` pairs enter
the index; values go to a separate log. This makes the index 50× smaller for 1KB+ values.

- **Index cost**: ~28 bytes/key (16B key + 12B metadata)
- **Write amplification**: ~1× (vs 10-30× for traditional LSM)
- **Relevance**: Pony's volume is already a value log; the index only stores offsets

*Source: Lu et al., "WiscKey: Separating Keys from Values in SSD-Conscious Storage", USENIX FAST 2016*

### SlimDB: 1.9 Bits Per Key (VLDB'17)

Three-level index using Entropy-Coded Tries achieves **1.9 bits per key**:
1. Prefix ECT compresses common key prefixes
2. Filter layer identifies candidate SSTables
3. Suffix ECT locates exact block

- For 1M keys: index = 237 KB (vs 229 MB for Go map)
- 4× faster insertions than LevelDB via Stepped-merge
- **Trade-off**: Semi-sorted data required, complex construction

*Source: Ren et al., CMU, 2017*

### Binary Fuse Filters (2022)

State-of-the-art probabilistic filter:
- **9 bits per key** at 0.39% false positive rate
- Within 13% of theoretical lower bound
- 2× faster construction than XOR filters
- For 1M keys: **1.07 MB** for negative lookup filter

Go library: `github.com/FastFilter/xorfilter`

*Source: Lemire et al., "Binary Fuse Filters", arXiv:2201.01174*

### Pogreb: On-Disk Hash Table (2018)

Production Go key-value store using on-disk linear hashing:
- **28 slots per 512-byte bucket** (one page per bucket read)
- Each slot: hash + key_size + value_size + 64-bit offset
- mmap'd index — memory determined by OS page cache, not Go allocation
- Average lookup: 2 I/O operations

*Source: Krylysov, github.com/akrylysov/pogreb*

### CompassDB: Two-Tier Perfect Hash (2024)

- **~6 bytes per key** average index cost
- O(1) lookup via two-tier perfect hash (global → file → local → offset)
- 2.5-4× throughput vs RocksDB
- 50-85% lower P99 latency

*Source: arXiv:2406.18099*

### F2/FASTER: Two-Level Hash Index (2025)

Microsoft's F2 uses a two-level hash spanning memory and disk:
- Hot records indexed in memory (level 1)
- Cold records indexed on disk (level 2)
- Epoch-based synchronization (negligible overhead)
- 2-12× throughput vs competing KV stores

*Source: VLDB 2025, arXiv:2305.01516*

### TurtleKV: Dynamic Memory Allocation (2025)

Dynamic memory split between page cache (reads) and write buffers (writes):
- Up to 8× write throughput of RocksDB
- Up to 5× read throughput
- Memory dynamically reallocated based on workload

*Source: Astolfi et al., Tufts/MathWorks 2025*

### Go 1.24 Swiss Tables (2025)

Go 1.24's built-in map uses Swiss Tables:
- ~70% memory reduction for large maps (Datadog: 930 → 217 MiB)
- 87.5% load factor (vs ~65% previously)
- Still fundamentally O(N) memory for N keys

*Source: Go Blog, Datadog Engineering Blog*

## Architecture

### Design Choice: mmap'd On-Disk Hash Table

Selected approach combines ideas from Pogreb (on-disk hash), WiscKey (key-value separation),
and Kreon (mmap-based I/O):

```
                    ┌──────────────┐
                    │   Go Heap    │
                    │  ~35 MB      │
                    │              │
                    │ bucketKeys   │ ← per-bucket sorted key lists for List ops
                    │ writeBufRing │ ← 2 × 4MB write buffers
                    │ Go runtime   │ ← goroutines, GC, net/http
                    └──────────────┘
                           │
                    ┌──────┴───────────────────┐
                    │    mmap (OS-managed)     │
                    │    ~0 MB RSS             │
                    │    (only hot pages)      │
                    │                          │
                    │  volume.dat  (256 MB)    │ ← append-only data log
                    │  index.dat   (~12 MB)    │ ← on-disk hash table
                    └──────────────────────────┘
```

### Memory Budget (100MB Total)

| Component | Budget | Notes |
|-----------|--------|-------|
| Go runtime + GC | ~10 MB | Goroutines, stacks, runtime |
| Write buffers (2 × 4MB) | 8 MB | Ring buffer for async flush |
| Per-bucket key lists | ~2 MB | Sorted keys for List (50K objects) |
| Multipart upload parts | ~5 MB | Temporary, during upload only |
| Benchmark runner overhead | ~10 MB | Payloads, metrics, progress |
| **Go heap subtotal** | **~35 MB** | |
| mmap: volume.dat | OS-managed | Only touched pages in RSS |
| mmap: index.dat | OS-managed | Only hot slots in RSS |
| **Total process RSS** | **< 50 MB typical** | Well under 100MB |

### File Layout

**volume.dat** — Append-only data log (same format as Horse):
```
[Header 64B] [Record₁] [Record₂] ... [Recordₙ]

Header:
  magic:    "PONY0001"    (8 bytes)
  version:  1             (4 bytes)
  flags:    0             (4 bytes)
  tail:     uint64        (8 bytes) — next write offset
  _padding: 40 bytes

Record:
  type:        byte       — 1=put, 2=delete
  crc32:       uint32     — checksum (skipped in sync=none)
  bucket_len:  uint16
  bucket:      [N]byte
  key_len:     uint16
  key:         [N]byte
  ct_len:      uint16
  contentType: [N]byte
  value_len:   uint64
  value:       [N]byte
  timestamp:   int64      — UnixNano
```

**index.dat** — mmap'd open-addressing hash table:
```
[Header 64B] [Slot₀ 64B] [Slot₁ 64B] ... [Slotₙ 64B] [StringPool...]

Header (64 bytes):
  magic:       "PONYIDX\0"  (8 bytes)
  version:     uint32       (4 bytes)
  _flags:      uint32       (4 bytes)
  slot_count:  uint64       (8 bytes) — power of 2
  entry_count: uint64       (8 bytes)
  strings_pos: uint64       (8 bytes) — next write offset in string pool
  _padding:    24 bytes

Slot (64 bytes, cache-line aligned):
  hash:     uint64   — FNV-1a of composite key. 0=empty, 1=tombstone
  str_off:  uint64   — offset in string pool for compositeKey+contentType
  str_len:  uint32   — composite key length
  ct_len:   uint16   — content type length (starts at str_off+str_len)
  _pad:     uint16
  val_off:  int64    — value offset in volume.dat
  val_size: int64    — value size in bytes
  created:  int64    — UnixNano
  updated:  int64    — UnixNano
  _pad2:    8 bytes

String Pool (append-only):
  [compositeKey₁][contentType₁] [compositeKey₂][contentType₂] ...
```

### Operations

**Write(bucket, key, value)**:
1. Write record to volume (via write buffer ring or direct pwrite)
2. Lock index (WLock)
3. Compute hash = FNV-1a(bucket + "\x00" + key)
4. Linear probe to find empty/tombstone slot
5. Append compositeKey + contentType to string pool
6. Write slot (hash, strOff, strLen, ctLen, valOff, valSize, timestamps)
7. Increment entry count
8. Update per-bucket key list
9. Unlock
10. If load > 75%, trigger async grow

**Read(bucket, key)**:
1. Lock index (RLock)
2. Compute hash, linear probe to find slot with matching hash
3. Verify key by reading compositeKey from string pool
4. Read content type from string pool
5. Unlock
6. Read value from volume (mmap zero-copy or write buffer ring)

**Stat(bucket, key)**:
1. Same as Read steps 1-5
2. Return metadata without reading value

**Delete(bucket, key)**:
1. Lock index (WLock)
2. Find slot, mark as tombstone (hash=1)
3. Decrement entry count
4. Update per-bucket key list
5. Unlock
6. Append delete record to volume

**List(bucket, prefix)**:
1. Use in-memory per-bucket sorted key list
2. Binary search for prefix start
3. Iterate forward while prefix matches
4. For each matching key, read metadata from hash table
5. Return sorted results

### Hash Table Growth

When entry_count / slot_count > 75%:
1. Create new index file with 2× slots
2. Take exclusive lock
3. Rehash all non-tombstone entries from old to new
4. munmap old file, mmap new file
5. Release lock

Growth doubles capacity, so it happens O(log N) times total.

### Recovery

On startup, if index.dat is missing or corrupted:
1. Scan volume.dat from header to tail
2. For each put record: insert into hash table
3. For each delete record: mark as tombstone
4. Persist rebuilt index to index.dat

This reuses Horse's proven recovery algorithm.

## DSN Format

```
pony:///path/to/data
pony:///path/to/data?sync=none           # default: no sync overhead
pony:///path/to/data?sync=batch          # 10ms group commit
pony:///path/to/data?bufsize=4194304     # write buffer size (default 4MB)
pony:///path/to/data?prealloc=256        # volume prealloc in MB (default 256)
pony:///path/to/data?slots=65536         # initial hash table slots (default 64K)
```

## Expected Performance vs Horse

| Operation | Horse | Pony (expected) | Ratio |
|-----------|-------|-----------------|-------|
| Write/1KB | ~1.4 GB/s | ~1.0 GB/s | 0.7× (smaller buffers) |
| Read/1KB | ~9.4 GB/s | ~8.0 GB/s | 0.85× (mmap read same) |
| Stat | ~15.4M ops/s | ~5-8M ops/s | 0.5× (mmap index) |
| List/100 | ~135K ops/s | ~100K ops/s | 0.75× (same key lists) |
| Delete | ~2.0M ops/s | ~1.5M ops/s | 0.75× (mmap write) |
| ParallelWrite C200 | ~varies | ~comparable | ~1× (write buffer dominated) |
| Memory (50K objs) | ~300 MB | **< 50 MB** | **6×** better |
| Memory (1M objs) | ~500 MB | **< 80 MB** | **6×** better |

## Implementation Files

```
pkg/storage/driver/zoo/pony/
├── storage.go    — Driver, store, bucket, iterators, directory support
├── volume.go     — Append-only volume (Horse format, smaller defaults)
├── index.go      — mmap'd on-disk hash table
├── writebuf.go   — Small write buffer ring (2×4MB)
└── multipart.go  — Multipart upload support
```
