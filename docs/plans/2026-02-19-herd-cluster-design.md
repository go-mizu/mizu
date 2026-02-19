# Herd Enhanced Cluster Design

## Problem

Herd's embedded single-node mode already beats horse on writes (59.6x at 64KB, 645x at ParallelWrite C100) but horse wins reads (3.7x at 1KB) and many scale operations. The TCP cluster mode (herd3net) adds 15-20µs overhead per operation, losing the small-object advantage.

Target: 10x horse performance across benchmarks, with truly dynamic cluster membership.

## Architecture: Two-Mode Cluster

### Mode 1: Embedded Multi-Node (`nodes=N`)

Like bee's pattern, create N independent herd stores in one process with zero-overhead function calls.

```
DSN: herd:///tmp/data?nodes=3&stripes=16&sync=none&inline_kb=8
```

- `multiNodeStore` wraps N `*store` instances
- Each node: `{root}/node_{i}/` directory, 16 stripes each
- 3 nodes × 16 stripes = 48 independent partitions
- Rendezvous hashing (FNV-1a) routes to correct node
- Zero TCP, zero serialization, zero buffer allocation

Why this achieves 10x:
- 3× write bandwidth (independent volumes, no contention)
- 3× read parallelism (independent mmap pools, better page cache)
- Fewer keys per node = smaller indexes = faster lookups
- All inline caching benefits preserved

### Mode 2: TCP Cluster with Gossip Membership

```
DSN: herd:///?seeds=127.0.0.1:7241,127.0.0.1:7242&data_port=9241
```

- HashiCorp memberlist (SWIM protocol) for dynamic membership
- Binary TCP data protocol (existing HD magic 0x4844)
- Nodes auto-discover via seed nodes, join/leave dynamically
- Client routing table auto-updates on membership changes

## Gossip Membership (HashiCorp memberlist)

### Why memberlist

| Property | memberlist | etcd/consul | Custom gossip |
|----------|-----------|-------------|---------------|
| Dependencies | 1 library | Full cluster | From scratch |
| Failure detection | O(log N) | Raft consensus | Manual |
| Scalability | 10K+ nodes | ~1K | Unknown |
| Complexity | Low | High | Very high |
| Battle-tested | Yes (Consul, Nomad, Serf) | Yes | No |

### SWIM Protocol Essentials

1. **Ping**: Each node pings a random peer every T interval
2. **Ping-req**: On timeout, ask K random nodes to ping the suspect
3. **Suspect→Dead**: If no ack after S intervals, mark dead
4. **Piggybacking**: Membership changes ride on existing protocol messages (zero extra bandwidth)

Failure detection: O(log N) messages to detect a failure across N nodes.

### Node Metadata

Each node broadcasts via memberlist:
- `data_addr`: TCP address for HD protocol (e.g., "127.0.0.1:9241")
- `status`: "ready" | "draining" | "joining"
- `weight`: Relative capacity weight (default 100)

### Event Handling

```
NodeJoin  → add to routing table, open TCP connection pool
NodeLeave → remove from routing table, close connections
NodeUpdate → update metadata (e.g., status change)
```

The client (`clusterStore`) subscribes to membership events and atomically swaps its node list. In-flight operations complete on old routing, new operations use updated routing.

## DSN Format

```
# Embedded single-node (existing):
herd:///path/to/data?stripes=16&sync=none&inline_kb=8

# Embedded multi-node (new):
herd:///path/to/data?nodes=3&stripes=16&sync=none&inline_kb=8

# TCP cluster with static peers (existing):
herd:///?peers=127.0.0.1:9241,127.0.0.1:9242&replicas=1

# TCP cluster with gossip (new):
herd:///?seeds=127.0.0.1:7241&data_port=9241&gossip_port=7241
```

Decision logic in `driver.Open()`:
1. `nodes=N` → `openMultiNode()` (embedded multi-node)
2. `seeds=` → `openGossipCluster()` (TCP + memberlist)
3. `peers=` → `openCluster()` (TCP, static peers, existing)
4. Otherwise → `openEmbedded()` (single embedded store)

## cmd/herd Updates

```
# S3 server (default):
herd -listen :9230

# TCP node server (existing):
herd -node -listen :9241

# TCP node server with gossip (new):
herd -node -listen :9241 -seeds 127.0.0.1:7241 -gossip-port 7241

# Embedded multi-node S3 server:
herd -listen :9230 -nodes 3
```

## Benchmark Configs

```go
{Name: "herd3", DSN: "herd:///tmp/herd3-bench?nodes=3&stripes=16&sync=none&inline_kb=8&prealloc=1024&bufsize=8388608"}
{Name: "herd5", DSN: "herd:///tmp/herd5-bench?nodes=5&stripes=16&sync=none&inline_kb=8&prealloc=1024&bufsize=8388608"}
```

## Scale Testing

Test dynamic membership with gossip:
1. Start 1 node → verify CRUD operations
2. Join 2nd node → verify key redistribution
3. Join 3 more nodes (total 5) → verify load spread
4. Remove 3 nodes (back to 2) → verify graceful drain
5. All operations remain available throughout

## Key Design Decisions

1. **No master SPOF**: Client-side consistent hashing, no coordinator
2. **No rebalancing on join/leave**: New keys route to new topology, old data stays (eventual consistency via background repair if needed)
3. **Embedded mode uses same routing as cluster**: Code path tested in both modes
4. **memberlist metadata is minimal**: Just addr + status + weight (< 100 bytes)
5. **Graceful drain**: Node sets status="draining" → clients stop routing new writes → node leaves after drain timeout
