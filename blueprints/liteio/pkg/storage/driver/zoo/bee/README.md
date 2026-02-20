# Bee - Distributed Quorum-Replicated Storage

Distributed, sharded, replicated cluster storage inspired by DynamoDB and Riak. Uses Haystack-style append-only logs with rendezvous hashing and quorum semantics for tunable consistency.

## Architecture

```
Write (quorum=W):
  Key → Rendezvous hash → select R replica nodes
    → Fan-out parallel writes to W sync targets
    → Collect W successes (quorum met)
    → Async: enqueue remaining replicas for repair
    → Cache result in gateway LRU

Read (r=1, eventual consistent):
  Key → Rendezvous hash → try replicas sequentially
    → First success returned
    → Trigger read-repair on missing replicas

Read (r>1, strong consistent):
  Key → Rendezvous hash → parallel reads from R replicas
    → Wait for quorum → return latest timestamp version
    → Repair stale/missing replicas
```

### Node Engine

Each node maintains:
- **Append-only log file**: Header (16B magic "BEELOG01") + records (27B fixed header + payload)
- **In-memory index**: `map[string]*nodeEntry` keyed by composite `"bucket\x00key"`
- **Dual mutex**: `appendMu` (serializes log writes) + `idxMu` (RWMutex for index)
- **Inline caching**: Values ≤64KB (default) stored directly in index entry

### Cluster Coordination

- **Rendezvous hashing**: FNV-1a with 3× avalanche mixing for uniform distribution
- **Replication factor R**: Default 3 (configurable)
- **Write quorum W**: Default ⌊R/2⌋+1 (configurable)
- **Read quorum r**: Default 1 (configurable)
- **Repair workers**: 4 async workers with 4096-task queue, 3 retries with exponential backoff

### Deployment Modes

| Mode | DSN | Description |
|------|-----|-------------|
| Embedded N-node | `bee:///path?nodes=3&replicas=3` | N in-process nodes |
| Network cluster | `bee:///?peers=http://host:port,...` | Remote HTTP nodes |
| Turbo (all RAM) | `bee:///path?mode=turbo` | Gateway-side in-memory store |

### Turbo Mode

Triple in-memory maps for zero-disk reads:
- `turboData`: composite key → full entry + data
- `turboBuckets`: bucket → keys
- `turboParents`: bucket → parent-prefix → keys (for directory listing)

### HTTP Node Protocol

| Endpoint | Methods | Description |
|----------|---------|-------------|
| `/v1/ping` | GET | Health check |
| `/v1/object` | GET/PUT/HEAD/DELETE | Object CRUD |
| `/v1/list` | GET | List objects |
| `/v1/buckets` | GET | List bucket names |
| `/v1/bucket` | DELETE | Delete bucket |

## DSN

```
bee:///path?nodes=3&replicas=3&w=2&r=1&sync=batch
bee:///?peers=http://host1:9401,http://host2:9402&replicas=3
```

| Param | Default | Description |
|-------|---------|-------------|
| `nodes` | `3` | Number of embedded nodes |
| `replicas` | `3` | Replication factor |
| `w` | `⌊R/2⌋+1` | Write quorum |
| `r` | `1` | Read quorum |
| `sync` | `none` | Durability: `none`, `batch` (10ms), `full` |
| `inline_kb` | `64` | Max inline value size (KB) |
| `repair` | `true` | Enable background repair |
| `repair_workers` | `4` | Repair worker count |
| `repair_max_kb` | `8192` | Max push-repair size (KB) |
| `cache_mb` | `0` | Gateway LRU cache capacity |
| `cache_obj_kb` | `64` | Per-object cache limit |
| `cache_ttl_ms` | `1000` | Cache TTL |
| `mode` | `` | `turbo` for all-RAM mode |

## Limitations

- **No log compaction**: Append-only logs grow unbounded; no GC for deleted entries
- **Timestamp-only conflict resolution**: No vector clocks; out-of-order writes cause silent conflicts
- **No cluster rebalancing**: Adding/removing nodes moves ~66% of keys (no gradual handoff)
- **Single gateway**: All traffic through store coordinator (not peer-to-peer gossip)
- **No authentication/encryption**: HTTP node protocol sends data in plaintext
- **Unlimited conns per host**: Remote transport has no per-host connection cap
- **Read cache has no invalidation**: Only TTL/LRU eviction; direct node writes leave cache stale
- **Multipart uploads in-memory**: Large uploads can OOM; lost on gateway restart
- **Inline cache grows unbounded**: No LRU eviction within index entries
- **Batch sync hardcoded at 10ms**: No adaptive batching

## Enhancement Opportunities

1. **Log compaction/tiering**: LSM-style merge of old log entries
2. **Vector clocks**: Per-node causality tracking for true conflict detection
3. **Consistent hashing**: Replace rendezvous with hash ring + vnodes for smooth scaling
4. **Peer-to-peer gossip**: Remove gateway SPOF, enable true distributed cluster
5. **TLS**: Encrypt node-to-node and client-to-node communication
6. **Adaptive quorum**: Reduce quorum in degraded mode
7. **Bloom filters**: Per-node filters to eliminate read amplification on misses
8. **Compression**: Pluggable zstd/snappy per bucket
9. **TTL support**: Auto-delete expired objects
10. **Incremental repair**: Merkle tree-based delta sync instead of full object copy

## Performance Profile

| Operation | Latency (quorum=1) | Latency (quorum=2/3) |
|-----------|--------------------|-----------------------|
| Write (SSD) | ~1-5ms | ~5-20ms (p95 of replicas) |
| Write (partial fail) | ~50-100ms | Same + async repair |
| Read (turbo) | <1ms | N/A |
| Read (cache hit) | ~0.5ms | N/A |
| Read (r=1, disk) | 1-5ms | 10-30ms (p75 of replicas) |
| Repair task | 10-30ms (amortized) | Per retry: +10ms backoff |

**Memory per node**: ~500 bytes/key (entry + map + inline). 1M keys ≈ 500MB.
