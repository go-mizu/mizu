# Herd - 16-Stripe High-Performance Storage

High-performance striped object storage inspired by Facebook Haystack and SeaweedFS with fundamental limitations fixed. 16 independent stripes eliminate cross-partition contention for near-linear throughput scaling.

## Architecture

```
Write Path:
  Key → FNV-1a hash → Stripe (1 of 16)
    → Buffer ring claim (lock-free atomic)
    → Data buffered (≤8MB per buffer)
    → Background flush to volume (pwrite)
    → Index update (256-shard RWMutex map)
    → Bloom filter add (7× atomic OR)

Read Path:
  Key → FNV-1a hash → Stripe (1 of 16)
    → Bloom filter check → reject 99.9% negatives
    → Index shard lookup (RLock, O(1))
    → Inline (≤8KB)? → return slice, zero I/O
    → Small (<256KB)? → mmap zero-copy read
    → Large (≥256KB)? → pread with kernel readahead
    → Check unflushed buffer ring (4 buffers)
```

### Stripe Design

Each stripe is fully independent:
- **Volume**: Append-only mmap'd file (header 64B + records)
- **Index**: 256-shard hash table (RWMutex per shard)
- **Bloom filter**: Lock-free, 10 bits/item, 7 hashes → ~0.1% FPR
- **Buffer ring**: 4 × 8MB write buffers (atomic claim/release)

Total: 16 stripes × 256 shards = **4,096 independent lock domains**.

### Deployment Modes

| Mode | DSN | Description |
|------|-----|-------------|
| Single node | `herd:///path` | 16 local stripes |
| Multi-node embedded | `herd:///path?nodes=3` | N in-process stores, rendezvous hash |
| Static TCP cluster | `herd:///?peers=host:port,...` | Binary protocol, 256 conn pool/node |
| Gossip cluster | `herd:///?seeds=host:port,...` | SWIM via HashiCorp memberlist |

### Binary TCP Protocol

```
Request:  [magic 0x4844 (2B)] [op (1B)] [bodyLen (4B)] [body...]
Response: [status (1B)] [bodyLen (4B)] [body...]

Ops: Put(1) Get(2) Stat(3) Delete(4) List(5) Ping(6)
     InitMP(7) PartMP(8) CompleteMP(9) AbortMP(10) ListParts(11)
```

### Read Strategy (Size-Based)

| Object Size | Method | Characteristics |
|-------------|--------|-----------------|
| ≤8KB (inline) | Direct slice | Zero I/O, in-memory |
| <256KB | mmap | Zero-copy, TLB-cached |
| ≥256KB | pread | Kernel readahead, avoids TLB pressure |
| Unflushed | Buffer ring scan | Check all 4 active buffers |

### Bloom Filter

- **Lock-free**: Atomic OR for adds, plain reads for queries
- **10 bits/item, 7 hash functions**: ~0.1% false positive rate
- **Double FNV-1a hashing**: Independent hash functions
- Bits only set, never cleared (no delete support in filter)

## DSN

```
herd:///path/to/data?stripes=16&sync=none&inline_kb=8
herd:///path?nodes=3&replicas=1
herd:///?seeds=host1:7241,host2:7241&gossip_port=7241
herd:///?peers=host1:8080,host2:8080&replicas=1
```

| Param | Default | Description |
|-------|---------|-------------|
| `stripes` | `16` | Number of independent stripes |
| `sync` | `none` | Durability: `none`, `batch`, `full` |
| `inline_kb` | `8` | Max inline value size (KB) |
| `prealloc` | `1024` | Volume preallocation (MB) |
| `bufsize` | `8388608` | Buffer ring size per stripe (bytes) |
| `nodes` | - | Embedded multi-node count |
| `seeds` | - | Gossip seed peer addresses |
| `peers` | - | Static TCP peer addresses |
| `replicas` | `1` | Replication factor (cluster mode) |
| `gossip_port` | `7241` | SWIM gossip port |

## Limitations

- **No bucket-level isolation**: Buckets are virtual (in-memory maps only)
- **No automatic compaction**: Volume files grow forever; deleted space never reclaimed
- **No transactions**: Each operation independently atomic; no multi-key guarantees
- **No TTL/expiration**: Objects persist until explicitly deleted
- **No encryption**: Data at rest unencrypted; TCP protocol has no TLS
- **No access control**: TCP endpoints have no authentication
- **No streaming responses**: Objects buffered in memory for cluster forwarding
- **Bloom filter has no delete**: False positive rate increases with deletions
- **CRC optional**: `sync=none` disables checksums; no corruption detection
- **Recovery is forward-only**: Scans from header tail; no backward validation
- **Multipart parts in-memory**: No disk spillover for large uploads
- **Connection pool fixed at 256**: No adaptive sizing

## Enhancement Opportunities

1. **Compaction**: Background GC to merge volumes and reclaim deleted space
2. **Compression**: Optional zstd/gzip per-value or per-stripe
3. **Tiered storage**: Inline (memory) → volume (mmap) → cold archive
4. **WAL**: Write-ahead log before volume append for crash safety
5. **Replication**: Async forwarding to backup nodes
6. **Metrics export**: Prometheus endpoint for stripe health, buffer utilization
7. **Node weight**: Rendezvous weighted by capacity (currently uniform)
8. **Streaming**: Chunked transfer for large objects in cluster mode
9. **Adaptive inline threshold**: Per-bucket or workload-driven tuning
10. **Counting bloom filter**: Support delete operations

## Performance Profile

| Operation | Latency | Notes |
|-----------|---------|-------|
| Write (inline, sync=none) | ~10µs | No I/O, buffer only |
| Write (buffered, sync=batch) | 10-100µs | Async flush |
| Write (large, direct) | ms-scale | I/O bound |
| Read (bloom reject) | <1µs | 7 atomic reads |
| Read (inline hit) | <1µs | Direct memory slice |
| Read (mmap, <256KB) | 10-50µs | TLB + page cache |
| Read (pread, large) | 1-5ms | Kernel readahead |
| List (prefix) | O(n per stripe) | Scan + merge |

**Memory per stripe**: ~16MB baseline (8MB buffer ring + 8MB index/bloom).
**Per-object overhead**: 283 bytes (27B record header + 256B index entry).
**Throughput**: ~50K PUT/s single-node, ~150K PUT/s aggregate (3 nodes).
