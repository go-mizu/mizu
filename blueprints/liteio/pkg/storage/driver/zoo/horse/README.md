# Horse - In-Memory Indexed Single-Volume Storage

Durable single-volume-file storage with in-memory hash index. Combines Bitcask-style append-only logging with FASTER-inspired lock-free concurrent writes and Kreon-inspired mmap zero-copy reads.

## Architecture

```
Write Path:
  Key → Claim space in buffer ring (atomic)
    → Serialize record inline (19B header + payload)
    → Buffer full? → pwrite to volume, update atomic tail
    → Pool-allocate indexEntry → insert to shard

Read Path:
  Key → hash(bucket, key) → shard (1 of 256)
    → RLock + map lookup
    → Check write buffer first (may have unflushed data)
    → Zero-copy mmap read from volume
```

### Volume Format

Single `volume.dat` file with Bitcask-style append-only log:

```
[Header 64B: magic "HORS0001" + version + tail offset]
[Record: type(1) | crc32(4) | bucket_len(2) | key_len(2) |
         ct_len(2) | value_len(8) | payload...]
...
```

- **Preallocation**: 64GB sparse file (disk space consumed only on write)
- **Atomic tail**: `atomic.Int64` pointer for concurrent space claiming

### Index

- **256 shards**: FNV-1a hash distribution, cache-line padded
- **Per-shard**: `sync.RWMutex` + `map[string]*indexEntry`
- `indexEntry`: valueOffset, size, contentType, created, updated
- Index entries pool-allocated via `sync.Pool` (reduces GC pressure)

### Write Buffer Ring

- **4 buffers × 64MB** (configurable): Rotating ring with lock-free claim
- Background flusher goroutine processes frozen buffers
- Out-of-order swap handling with retry + `runtime.Gosched()`
- Values < 4KB: direct to mmap. Values ≥ 4KB: pwrite path.

### Read Strategy

| Object Size | Method | Characteristics |
|-------------|--------|-----------------|
| Any (unflushed) | Buffer ring scan | Check 4 active buffers |
| < 256KB | mmap zero-copy | Direct slice, TLB-cached |
| ≥ 256KB | pread | Kernel readahead, avoids TLB pressure |

## DSN

```
horse:///path/to/data
horse:///path/to/data?sync=none
horse:///path/to/data?sync=batch
horse:///path/to/data?prealloc=65536
```

| Param | Default | Description |
|-------|---------|-------------|
| `sync` | `none` | Durability: `none`, `batch`, `full` |
| `prealloc` | `65536` | Volume preallocation (MB, sparse) |

## Limitations

- **Memory scaling**: 300-500MB RSS at 1M objects (entire index in RAM)
- **Single volume file**: No automatic splitting or tiering
- **No compression**: Values stored raw
- **No cluster mode**: Single-node only
- **Bucket metadata not persisted**: Creation times lost on restart
- **Recovery requires full scan**: O(n) volume replay on startup
- **Static mmap region**: Doesn't grow mmap dynamically; large objects use pwrite
- **No rehashing**: Hash table doesn't resize for better cache locality
- **Directory stat uses List()**: Inefficient for deep hierarchies

## Enhancement Opportunities

1. **Dynamic mmap growth**: Remap region as volume grows instead of pwrite fallback
2. **Compression**: Snappy/zstd for small values
3. **Persistent bucket metadata**: Store creation times in volume header
4. **Bloom filter**: Fast negative lookups for non-existent keys
5. **Index persistence**: Periodic index snapshot to avoid full replay on restart
6. **TTL support**: Per-object expiration with background cleanup
7. **Rehashable table**: Adaptive resizing for better cache locality at scale

## Performance Profile

| Operation | Throughput | Notes |
|-----------|-----------|-------|
| Write 1KB | ~1M ops/sec | Buffer ring, sync=none |
| Parallel write (100 goroutines) | Linear scaling | Lock-free space claiming |
| Read | ~2.5M ops/sec | Zero-copy mmap |
| Stat | ~3M ops/sec | Hash lookup only |

**Memory**: 300-500MB at 1M objects (index + entry structs + map overhead).
**CRC overhead**: ~100ns/write (skipped in `sync=none` mode).
