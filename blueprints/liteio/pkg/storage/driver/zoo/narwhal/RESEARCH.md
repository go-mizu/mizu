# Narwhal Storage Driver -- Research Notes

## Paper Architecture: Nos/Nostor (OSDI 2025)

### Stripeless Erasure Coding

Traditional erasure coding divides data into fixed-size **stripes**, each encoded independently. A stripe consists of k data chunks and m parity chunks. Every write must coordinate across all k+m nodes in the stripe to maintain consistency. This per-stripe coordination introduces significant overhead: write amplification, tail latency from the slowest node in the stripe, and complex recovery when a node fails mid-stripe.

**Nos eliminates the stripe abstraction entirely.** Instead of organizing data into stripes, each node independently replicates its data to a set of backup nodes. The backup nodes XOR incoming replicas into local parity volumes. No cross-node synchronization is required during writes.

### SBIBD-Based Affinity

The assignment of which nodes back up which other nodes is determined by **Symmetric Balanced Incomplete Block Design (SBIBD)**, a mathematical construct from combinatorial design theory.

Key properties of SBIBD assignment:
- Every pair of nodes shares **exactly one** backup block -- this guarantees optimal recovery because any single node failure can be reconstructed from exactly one other node's parity.
- Backup responsibility is **evenly distributed** across all nodes -- no single node becomes a bottleneck.
- The design is parameterized by (v, k, lambda) where v = number of nodes, k = block size, lambda = 1 (every pair shares one block).

### Write Path (Nos)

1. Client writes data to the **primary node** for that key.
2. The primary node **independently replicates** to its SBIBD-assigned backup nodes.
3. Each backup node **XORs the incoming data** into its local parity volume.
4. No per-write coordination barrier -- parity is computed locally.

### Recovery Path (Nos)

When a node fails:
1. Identify all backup nodes that hold parity data containing the failed node's contributions.
2. For each parity block, XOR out the surviving nodes' contributions to isolate the failed node's data.
3. Reconstruct the failed node's data and place it on a replacement node.
4. Because every pair of nodes shares exactly one SBIBD block, recovery reads are spread evenly.

### Performance Results

- **1.61x to 2.60x** throughput improvement over traditional stripe-based erasure coding (Reed-Solomon, LRC).
- The throughput gain comes from eliminating the write barrier -- traditional EC must wait for all k+m chunks in a stripe to be written before acknowledging; Nos acknowledges after the primary write + local XOR.
- Recovery is faster because parity is already distributed by SBIBD -- no need to read an entire stripe.

### Key Insight

**Parity is computed locally by XOR, with no cross-node synchronization for writes.** The mathematical SBIBD structure guarantees that despite the lack of explicit stripes, the system still achieves the same fault tolerance properties as traditional EC, while removing the coordination overhead that limits throughput.

---

## Our Implementation Plan (Single-Node Simulation)

### Approach

We simulate the Nos architecture within a single process. Instead of N physical nodes, we use N data volume files. A parity volume file stores XOR parity computed across fixed-size blocks from all data volumes.

### Data Volumes

- N independent append-only files: `vol_0.dat`, `vol_1.dat`, ..., `vol_{N-1}.dat`
- Each volume has a 32-byte header: magic bytes (8B), version (4B), flags (4B), tail offset (8B), reserved (8B)
- Records are appended sequentially: `keyLen(2B) | key | ctLen(2B) | contentType | valLen(8B) | value | created(8B) | updated(8B)`
- Volume assignment: `FNV-1a(compositeKey) % N` -- deterministic, no coordination needed
- Default N=4 (configurable via `stripes` query parameter)

### Parity Volume

- Single file: `parity.dat`
- Stores XOR parity of fixed-size blocks (64KB default) across all data volumes
- `parity_block[i] = vol_0_block[i] XOR vol_1_block[i] XOR ... XOR vol_{N-1}_block[i]`
- On each write, the new bytes are XOR'd into the corresponding parity block position
- On read failure or corruption, reconstruct from parity + other volumes

### In-Memory Indexes

- **Per-volume index**: `map[compositeKey] -> {offset, length}` -- rebuilt on startup by replaying the volume
- **Global index**: `compositeKey -> volumeID` -- routes reads to the correct volume
- **Bucket map**: `map[string]time.Time` -- tracks bucket creation times

### Write Path

1. Compute `compositeKey = bucket + "\x00" + key`
2. Hash key to select volume: `volumeID = FNV-1a(compositeKey) % N`
3. Append record to `vol_{volumeID}.dat`
4. XOR value bytes into parity at the corresponding block offset
5. Update in-memory index

### Read Path

1. Lookup compositeKey in global index to find volumeID
2. Lookup compositeKey in per-volume index to find {offset, length}
3. Read data from `vol_{volumeID}.dat`
4. If read fails or data is corrupt, reconstruct from parity + other volumes

### Delete

1. Mark deleted in the in-memory index (remove from maps)
2. Append a tombstone record to the volume (keyLen, key, valLen=0, special flag)
3. Parity is not updated on delete (acceptable for our simulation)

### Recovery on Startup

1. For each volume file, replay all records sequentially
2. Rebuild per-volume index and global index
3. Skip tombstoned entries

### File Layout

```
{root}/
  vol_0.dat    -- data volume 0
  vol_1.dat    -- data volume 1
  vol_2.dat    -- data volume 2
  vol_3.dat    -- data volume 3
  parity.dat   -- XOR parity volume
  meta.json    -- volume tails, entry counts
```

### Differences from Paper

| Aspect | Nos Paper | Our Implementation |
|--------|-----------|-------------------|
| Scope | Distributed cluster | Single-node simulation |
| Nodes | Physical machines | Volume files |
| Network | Real network replication | File I/O |
| SBIBD | Full block design | Simplified: round-robin by hash |
| Recovery | Cross-node reconstruction | Cross-volume XOR reconstruction |
| Parity | Per-backup-node | Single parity file |

### DSN Format

```
narwhal:///path/to/data?sync=none&stripes=4
```

- `sync`: `none` (default, no fsync), `batch`, `full`
- `stripes`: number of data volumes (default 4)
