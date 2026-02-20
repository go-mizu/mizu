# Zebra - Tectonic-Style Striped Storage

High-performance striped storage inspired by Facebook Tectonic. Multiple independent stripes eliminate cross-partition contention with optional inline value caching for small objects.

## Architecture

```
Write Path:
  Key → FNV-1a hash → Stripe (1 of 8)
    → Buffer ring claim (lock-free atomic)
    → Data buffered (≤32MB per buffer)
    → Background flush to volume (pwrite)
    → Index update (64-shard RWMutex map)
    → Inline cache insert (if ≤4KB)

Read Path:
  Key → FNV-1a hash → Stripe (1 of 8)
    → Index shard lookup (RLock, O(1))
    → Inline (≤4KB)? → return slice, zero I/O
    → Check write buffer first (unflushed data)
    → mmap zero-copy read from stripe volume
```

### Stripe Design

Each stripe is independent:
- **Volume**: Append-only mmap'd file (same format as Horse)
- **Index**: 64-shard hash table (RWMutex per shard)
- **Buffer ring**: 4 × 32MB write buffers
- **Inline caching**: Values ≤4KB (default) stored directly in index entry

Total: 8 stripes × 64 shards = **512 independent lock domains**.

### Inline Value Caching

Values smaller than `inline_kb` threshold are stored directly in `indexEntry.inline`:
- Zero volume I/O for small object reads
- Memory trade-off: each inline entry adds to heap size
- Especially effective for metadata-heavy workloads (configs, manifests, small assets)

### Deployment Modes

| Mode | DSN | Description |
|------|-----|-------------|
| Embedded | `zebra:///path?stripes=8` | Local stripes in one process |
| Cluster | `zebra:///?peers=host:port,...` | Remote nodes via binary TCP |

### Binary TCP Protocol

```
Request:  [magic 0x5A42 "ZB" (2B)] [op (1B)] [bodyLen (4B)] [body...]
Response: [status (1B)] [bodyLen (4B)] [body...]

Ops: Put(1) Get(2) Stat(3) Delete(4) List(5) Ping(6)
Status: OK(0) NotFound(1) Error(2)
```

### Cluster Architecture

- **Rendezvous hashing**: `Score(nodeAddr + key)` determines owning node
- **Replica count**: Write to top N scoring nodes (default 1)
- **Connection pool**: 64 connections per remote node
- **List fan-out**: Query all nodes, merge results

## DSN

```
zebra:///path/to/data
zebra:///path?stripes=8&sync=none&inline_kb=4
zebra:///?peers=host1:8080,host2:8080&replicas=1
```

| Param | Default | Description |
|-------|---------|-------------|
| `stripes` | `8` | Number of independent stripes |
| `sync` | `none` | Durability: `none`, `batch`, `full` |
| `inline_kb` | `4` | Max inline value size (KB) |
| `prealloc` | `1024` | Volume preallocation per stripe (MB) |
| `bufsize` | `32MB` | Buffer ring size per stripe |
| `peers` | - | TCP peer addresses for cluster |
| `replicas` | `1` | Replication factor (cluster mode) |

## Herd vs Zebra Comparison

| Aspect | Zebra | Herd |
|--------|-------|------|
| Default stripes | 8 | 16 |
| Shards per index | 64 | 256 |
| Lock domains | 512 | 4,096 |
| Bloom filter | No | Yes (~0.1% FPR) |
| Inline default | 4KB | 8KB |
| Buffer per stripe | 32MB | 8MB |
| Gossip support | No | Yes (memberlist) |
| Multipart support | Yes (embedded) | Yes |

## Limitations

- **No bloom filter**: Every negative lookup hits the index (vs Herd's 99.9% rejection)
- **No automatic rebalancing**: Manual stripe redistribution on node changes
- **Replica consistency**: Last-write-wins only; no conflict resolution
- **List fans out to all nodes**: Latency = slowest node × result merge
- **No cross-stripe transactions**: Operations isolated per stripe
- **TCP overhead**: 7-byte header per remote operation
- **Inline threshold is global**: Can't vary per-bucket or per-value type
- **No compaction**: Volume files grow forever
- **No gossip**: Static peer list only (no dynamic membership)
- **Fewer shards than Herd**: Higher per-shard contention under extreme concurrency

## Enhancement Opportunities

1. **Bloom filters**: Add per-stripe bloom for fast negative lookups
2. **Consistent hashing**: Replace rendezvous with hash ring + vnodes
3. **Gossip protocol**: Dynamic membership (like Herd's memberlist integration)
4. **Replica quorum reads**: Tunable consistency (like Bee)
5. **Compression**: Per-value or per-stripe compression
6. **Adaptive inline threshold**: Per-bucket or workload-driven tuning
7. **Vector clocks / CRDT**: Proper conflict resolution
8. **Background replication**: Async forwarding to backup nodes
9. **Compaction**: GC pass to reclaim deleted space

## Performance Profile

| Operation | Latency | Notes |
|-----------|---------|-------|
| Write (inline, sync=none) | ~10µs | No I/O, buffer only |
| Write (buffered) | 10-100µs | Async flush |
| Read (inline hit) | <1µs | Direct memory slice |
| Read (mmap) | 10-50µs | Zero-copy |
| Read (pread, large) | 1-5ms | Kernel readahead |
| Read (cluster, TCP) | 2-10ms | Network + remote lookup |
| List (cluster) | O(n × m) | n = nodes, m = items/node |

**Memory per stripe**: ~40MB baseline (32MB buffer ring + 8MB index).
**Total embedded**: 8 stripes × 40MB = ~320MB baseline.
