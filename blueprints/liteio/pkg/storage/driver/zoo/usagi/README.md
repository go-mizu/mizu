# Usagi - Log-Structured Append-Only Storage

Write-optimized storage driver using append-only segment files with in-memory indexing. Inspired by bitcask with sharded segments for parallel writes.

## Architecture

```
Write Path:
  Key → FNV-1a hash → Shard selection
    → Shard writer lock
    → Append record [Header(36B) | Key | ContentType | Data]
    → Optional fsync
    → Index update (256-shard hash map)
    → Small cache insert (if ≤64KB)

Read Path:
  Key → Index lookup (O(1) hash)
    → Small cache hit? → return copy
    → Segment reader pool → io.SectionReader → return

Recovery:
  manifest.usagi → partial replay from last checkpoint
  No manifest → full sequential scan of all segments
```

### Record Format

```
[Magic(4B "USAG")] [Version(1B)] [Op(1B)] [Flags(2B)]
[KeyLen(4B)] [CTLen(2B)] [Reserved(2B)] [DataLen(8B)]
[UpdatedUnixNs(8B)] [CRC32(4B)]
[Key...] [ContentType...] [Data...]
```

36-byte fixed header. CRC32 Castagnoli checksum. Little-endian encoding.

### Segment Storage

- Files: `segment-{SHARD}-{NNNNNN}.usg` (zero-padded segment ID)
- Default size: 64MB per segment (auto-rotate on threshold)
- Default shards: `GOMAXPROCS` (parallel writes to different shards)
- Per-shard writer mutex serializes writes to same shard

### Index

- **256 hash shards**, each with RWMutex-protected map
- Entry: shard, segmentID, offset, size, contentType, updated, checksum
- Cached sorted key list with atomic version counter (invalidated on mutation)
- Optional **prefix index**: trie for fast prefix listing

### Small Object Cache

- LRU eviction via `container/list`
- Default: 32MB total, 64KB per-item max
- Returns copies on Get (prevents external mutation)

### Manifest Checkpointing

- V2 JSON format: index snapshot + last segment positions per shard
- Written every 30s (configurable)
- Enables partial replay on recovery (skip already-indexed segments)

## DSN

```
usagi:///path/to/data
usagi:///path/to/data?nofsync=true&segment_size_mb=128&segment_shards=8
```

| Param | Default | Description |
|-------|---------|-------------|
| `bucket` | `default` | Default bucket name |
| `nofsync` | `false` | Skip fsync (benchmarking only) |
| `segment_size_mb` | `64` | Segment file rotation threshold |
| `segment_shards` | `GOMAXPROCS` | Number of parallel write shards |
| `manifest_interval_s` | `30` | Checkpoint frequency (seconds) |
| `small_cache_mb` | `32` | Small object cache capacity |
| `small_cache_max_kb` | `64` | Max object size for caching |

## Disk Layout

```
root/
└── bucket-name/
    ├── manifest.usagi           # JSON checkpoint
    ├── .usagi-segments/
    │   ├── segment-0-000001.usg
    │   ├── segment-0-000002.usg
    │   ├── segment-1-000001.usg
    │   └── ...
    └── .usagi-multipart/
        └── {uploadID}/
            ├── part-000001
            └── ...
```

## Limitations

- **No segment compaction/GC**: Old segments with deleted entries never reclaimed
- **In-memory index**: Must rebuild on crash if manifest missing (slow for large datasets)
- **Single-threaded per shard**: Writes to same hash shard serialize
- **No data compression**: Segments stored uncompressed
- **No encryption at rest**
- **2GB object size limit** (`1<<31` check)
- **Multipart parts buffered to disk**: Sequential assembly, no parallel merge
- **Manifest-only recovery**: Corrupted manifest forces full replay
- **Segment reader pooling**: File handles may become stale on external modification

## Enhancement Opportunities

1. **Segment compaction**: Merge old segments, remove deleted records, reclaim disk
2. **Compression**: Optional zstd/lz4 per-record or per-segment
3. **Parallel index rebuild**: Multiple goroutines scanning segments concurrently
4. **Batch writes**: Accumulate records, flush atomically
5. **Bloom filters**: Fast negative lookups before index scan
6. **TTL support**: Auto-delete expired objects
7. **Streaming manifest writes**: Non-blocking checkpoint during heavy writes

## Performance Profile

| Operation | Expected Latency | Notes |
|-----------|-----------------|-------|
| Write (small, with fsync) | 100µs-1ms | 1 fsync per write |
| Write (nofsync) | 1-10µs | Buffered append only |
| Read (cache hit) | <1µs | Map lookup + memcpy |
| Read (segment, SSD) | 10-100µs | Pooled reader + SectionReader |
| List (prefix, indexed) | O(log k) | Binary search in prefix index |
| List (full scan) | O(n log n) | Sort all keys |
| Recovery (with manifest) | O(delta) | Partial replay from checkpoint |
| Recovery (no manifest) | O(total) | Full segment scan |
