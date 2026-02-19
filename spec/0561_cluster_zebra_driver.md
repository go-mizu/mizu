# 0561: Cluster Zebra Driver

## Research Summary

### Haystack (Facebook, OSDI 2010)

Facebook's photo storage system aggregates millions of objects into large append-only **volume files** (~100 GB), keeping a compact in-memory index (key → offset+size, ~32 bytes/object). This reduces photo reads from 3 disk ops (stat + open + read in POSIX) to **1 disk seek**.

Three-tier architecture: **Store** (volume machines), **Directory** (MySQL mapping logical volumes → machines), **Cache** (DHT for hot photos).

**Limitations:**
- 3x replication for ALL data regardless of temperature — wasteful for cold data
- Centralized MySQL directory — coordination bottleneck at scale
- Single volume file per machine — no I/O parallelism within a node
- Compaction requires copying entire 100 GB volume — I/O spikes
- No erasure coding option

### SeaweedFS (Open Source, Haystack-inspired)

Master server (Raft-based) assigns volume IDs, volume servers store data in Haystack-style volumes. Optional filer provides POSIX directory hierarchy.

**Limitations we address:**
- Master split-brain during leader election → volumes disappear temporarily
- NeedleMap RAM: 16 bytes/needle × millions = GBs per volume server
- Filer metadata consistency under high concurrency (10K+ connections)
- Compaction requires free space equal to volume size
- Single-threaded sequential appends per volume — write bottleneck

### f4 / Tectonic (Facebook, 2014/2021)

f4 introduced tiered storage: hot blobs stay in Haystack, warm blobs move to RS(10,4) erasure coding → 2.1x overhead vs 3.6x.

Tectonic unified blob and warehouse storage with **sharded metadata across 3 independent layers** (Name → File → Block), each hash-partitioned. Key insight: hash partitioning scatters metadata evenly, avoiding directory hotspots.

### Ambry (LinkedIn, SIGMOD 2016)

Uses **virtual partitions** instead of direct physical mapping: objects → logical partitions → physical machines. Adding capacity moves partitions, not individual objects. Multi-master writes with gossip replication (85% zero lag).

### CRUSH Algorithm (Ceph, SC 2006)

Deterministic placement function: any client independently computes data location from (object_id, cluster_map) — **no coordinator needed for reads**. Straw2 bucket algorithm provides minimal data movement on topology changes.

### Modern Optimizations Surveyed

| Technique | Benefit | Applicable? |
|-----------|---------|-------------|
| io_uring | 144% IOPS at depth 16, bypass mmap scalability wall at 8+ cores | Linux-only, Go support immature |
| Swiss tables (Go 1.24) | 60% faster maps, 70% less memory | Yes, built-in |
| xsync.Map (CLHT) | Obstruction-free reads, no mutex overhead | External dep |
| sendfile/splice | Zero-copy serving, 50% CPU reduction | Auto via io.Copy |
| Group commit | 100x throughput for durable writes (RocksDB benchmark) | Yes, for batch/full sync |
| MADV_HUGEPAGE | 2-2.9x speedup for large mmap regions | Yes, for volume mmap |
| Rendezvous hashing | O(N) deterministic placement, no coordinator | Yes, for cluster mode |
| Jump hash | O(log N) O(1) memory, but can't remove arbitrary nodes | For stripe selection |

## Design: Zebra

### Core Insight

Horse is already extremely fast single-node (1.3 GB/s write, 11 GB/s read for 1KB). Its bottlenecks are:

1. **Single volume file**: All writes contend on one buffer ring, one mmap region
2. **Index lock contention**: 256 shards with RWMutex still shows degradation at C50+
3. **Per-write bucket tracking**: BucketKeySet maintenance adds ~100-200ns per write
4. **Volume I/O for all sizes**: Even 256-byte values go through buffer ring + volume

Zebra's strategy: **eliminate these bottlenecks through striped isolation and inline caching**.

### Architecture: Embedded Mode

```
zebra:///path?stripes=8&sync=none&inline_kb=4

                    ┌─────────────────────────────┐
                    │         store                │
                    │  fnvHash(bucket, key) % N    │
                    └─────────┬───────────────────-┘
          ┌───────────┬───────┼───────┬────────────┐
          ▼           ▼       ▼       ▼            ▼
    ┌──────────┐┌──────────┐ ... ┌──────────┐┌──────────┐
    │ stripe 0 ││ stripe 1 │     │ stripe 6 ││ stripe 7 │
    │          ││          │     │          ││          │
    │ volume   ││ volume   │     │ volume   ││ volume   │
    │ bufRing  ││ bufRing  │     │ bufRing  ││ bufRing  │
    │ index    ││ index    │     │ index    ││ index    │
    └──────────┘└──────────┘     └──────────┘└──────────┘
    stripe_0.dat stripe_1.dat    stripe_6.dat stripe_7.dat
```

Each stripe is **completely independent**: own file, own mmap, own buffer ring, own index. Zero cross-stripe synchronization.

### Key Innovation 1: Inline Values

Values ≤ `inline_kb` (default 4KB) are stored **directly in the index entry's memory**:

```
Write (sync=none, size ≤ inlineMax):
  1. Copy value into index entry's inline []byte
  2. Insert into stripe's index
  3. Done. No volume I/O.

Read (inline value):
  1. Index lookup → entry with inline bytes
  2. Copy to reader
  3. Done. No volume I/O.
```

For sync=none mode, small objects **never touch the volume**. This eliminates:
- Buffer ring atomic operations (~50ns)
- memcpy to buffer ring (~20ns)
- Volume space management

Expected write time for 1KB inline: ~250-350ns (vs horse's ~727ns) = **2-3x improvement**.

### Key Innovation 2: Striped Parallelism

At concurrency C with N stripes, average contention per stripe = C/N.
- Horse at C50: 50 goroutines contend on 1 buffer ring → degraded throughput
- Zebra at C50, 8 stripes: ~6 goroutines per stripe → near-zero contention

### Key Innovation 3: Lazy Key Listing

Horse maintains per-bucket `bucketKeySet` with per-segment maps, updated on every write. This adds ~100-200ns overhead to every mutation.

Zebra defers key tracking: `List()` scans relevant index shards and sorts results on demand. For typical list sizes (100-1000 objects), the scan cost is negligible.

### Architecture: Cluster Mode

```
zebra:///?peers=host1:9601,host2:9602,host3:9603&replicas=1

Client
  │ rendezvousHash(bucket+key, nodeList) → target node
  │
  ├──TCP───▶ node1:9601 (zebra engine, 8 stripes)
  ├──TCP───▶ node2:9602 (zebra engine, 8 stripes)
  └──TCP───▶ node3:9603 (zebra engine, 8 stripes)
```

**Binary TCP protocol** (not HTTP — bee's HTTP overhead was 80-90µs/op):
```
Request:  [2B magic: 0x5A42] [1B op] [4B body_len] [body]
Response: [1B status] [4B body_len] [body]
```

Overhead per request: ~17 bytes vs HTTP's ~300+ bytes.
Expected latency: ~15-25µs (loopback TCP) vs bee's ~80-90µs (HTTP).

**Rendezvous hashing** for placement: every client independently computes target node from (key, node_list). No coordinator. No directory service.

### DSN Parameters

**Embedded mode:** `zebra:///path?stripes=8&sync=none&inline_kb=4&prealloc=1024&bufsize=16777216`

| Parameter | Default | Description |
|-----------|---------|-------------|
| stripes | 8 | Number of independent stripes |
| sync | none | Sync mode: none, batch, full |
| inline_kb | 4 | Inline threshold in KB (0 to disable) |
| prealloc | 1024 | Preallocate per stripe in MB |
| bufsize | 16777216 | Buffer ring size per stripe in bytes |

**Cluster mode:** `zebra:///?peers=host:port,...&replicas=1&w=1&r=1`

| Parameter | Default | Description |
|-----------|---------|-------------|
| peers | required | Comma-separated node addresses |
| replicas | 1 | Number of replicas per object |
| w | 1 | Write quorum |
| r | 1 | Read quorum |

### Performance Targets

| Benchmark | Horse | Zebra Target | Strategy |
|-----------|-------|-------------|----------|
| Write/1KB | 1,342 MB/s | 3,000+ MB/s | Inline (skip volume I/O) |
| Read/1KB | 10,994 MB/s | 14,000+ MB/s | Inline (skip mmap) |
| Stat | 20M ops/s | 25M+ ops/s | Same (pure index) |
| Delete | 2.63M ops/s | 5M+ ops/s | Inline (skip volume tombstone) |
| ParallelWrite C50 | 40 MB/s | 200+ MB/s | Stripe isolation |
| ParallelRead C50 | 5.1 MB/s | 20+ MB/s | Stripe isolation |

### Volume Format

Same record format as horse for compatibility:
```
Header (64 bytes): magic("ZEBRA001") + version(1) + tail(int64) + reserved
Record: type(1) + crc(4) + bucketLen(2) + keyLen(2) + ctLen(2) + valueLen(8) + timestamp(8) + bucket + key + contentType + value
```

### Cluster Wire Protocol

Operations:
- `PUT (0x01)`: [2B bucket_len][bucket][2B key_len][key][2B ct_len][ct][8B timestamp][8B size][value]
- `GET (0x02)`: [2B bucket_len][bucket][2B key_len][key][8B offset][8B length]
- `STAT (0x03)`: [2B bucket_len][bucket][2B key_len][key]
- `DELETE (0x04)`: [2B bucket_len][bucket][2B key_len][key]
- `LIST (0x05)`: [2B bucket_len][bucket][2B prefix_len][prefix][1B recursive]
- `PING (0x06)`: empty body

Response status: 0=OK, 1=NOT_FOUND, 2=ERROR

GET response: [8B size][2B ct_len][ct][8B created][8B updated][value]
STAT response: [8B size][2B ct_len][ct][8B created][8B updated]

## Implementation Plan

### Phase 1: Core Driver (Embedded Mode)
1. `volume.go` — Append-only volume with mmap (adapted from horse, per-stripe)
2. `writebuf.go` — 4-buffer ring per stripe
3. `index.go` — 64-shard index with inline value support
4. `storage.go` — Driver registration, store, stripe routing, bucket ops
5. `multipart.go` — Multipart upload support

### Phase 2: Cluster Mode
6. `cluster.go` — TCP binary protocol, remote node client, server
7. `cmd/zebra/main.go` — Standalone node binary

### Phase 3: Benchmark
8. Update `bench/config.go` with zebra configs
9. Update `server/server.go` with zebra import
10. Benchmark script and results

## References

- Beaver et al., "Finding a Needle in Haystack," OSDI 2010
- Muralidhar et al., "f4: Facebook's Warm BLOB Storage System," OSDI 2014
- Pan et al., "Facebook's Tectonic Filesystem," FAST 2021
- Weil et al., "CRUSH: Controlled, Scalable, Decentralized Placement," SC 2006
- Ambry: LinkedIn's Scalable Geo-Distributed Object Store, SIGMOD 2016
- SeaweedFS Architecture, github.com/seaweedfs/seaweedfs
- Axboe, "Efficient IO with io_uring," kernel.dk/io_uring.pdf
