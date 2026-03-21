# 0764 — Storage Driver Architecture & Benchmark

Two storage drivers implement the same `StorageEngine` interface. This document
compares their architecture, performance, cost, and scalability — and explains
when to use each.

---

## 1. Driver Overview

### D1Engine (`d1_driver.ts`)

Uses Cloudflare D1 (managed SQLite) with per-actor table sharding. All actors
share one D1 database, but each actor gets isolated tables (`f_{shard}`,
`e_{shard}`, `b_{shard}`) where `shard = sha256(actor).slice(0, 16)`.

```
Worker  ──►  D1 (shared database)
               ├── shards (actor registry)
               ├── f_a1b2c3d4e5f67890 (actor A files)
               ├── e_a1b2c3d4e5f67890 (actor A events)
               ├── b_a1b2c3d4e5f67890 (actor A blobs)
               ├── f_9f8e7d6c5b4a3210 (actor B files)
               └── ...
        ──►  R2 (shared blob store)
               └── blobs/{actor}/{hash[0:2]}/{hash[2:4]}/{hash}
```

### DOEngine (`do_driver.ts`)

Uses Cloudflare Durable Objects — each actor gets a dedicated DO instance with
its own SQLite database. The DO handles metadata; the Worker handles R2 directly.

```
Worker  ──RPC──►  StorageDO[actor A]  (own SQLite)
        ──RPC──►  StorageDO[actor B]  (own SQLite)
        ──────►  R2 (shared blob store)
```

---

## 2. Architecture Comparison

| Aspect                  | D1Engine                           | DOEngine                          |
|-------------------------|------------------------------------|-----------------------------------|
| **Database**            | Shared D1 (one SQLite for all)     | Per-actor DO SQLite (isolated)    |
| **Isolation**           | Table sharding (`f_{shard}`)       | Process-level (separate DO)       |
| **SQL execution**       | Async (network hop to D1)          | Sync within DO (local SQLite)     |
| **Transactions**        | `db.batch()` (D1 batch = SQLite tx)| `transactionSync()` (real ACID)   |
| **Write contention**    | D1 single-writer (global)          | Per-actor (zero cross-actor)      |
| **Read replicas**       | D1 edge replicas (auto)            | None (DO is single-region)        |
| **Schema migration**    | D1 migrations + lazy shard creation| Auto-created on first DO access   |
| **Cross-actor queries** | Possible (query shared tables)     | Impossible (separate databases)   |
| **Actor deletion**      | Delete shard tables                | Delete the DO                     |
| **Max database size**   | 10 GB (shared across all actors)   | 1 GB per DO (per actor)           |

### D1 Per-Actor Sharding

The D1 driver computes a deterministic shard from the actor name:

```
shard = sha256(actor).slice(0, 16)   # 16 hex chars = 64-bit hash
```

On first access, `ensureShard()`:
1. Checks the `shards` registry table
2. Creates `f_{shard}`, `e_{shard}`, `b_{shard}` tables if new
3. Migrates data from legacy shared tables (if any)
4. Caches the shard locally for the request lifetime

This eliminates `WHERE owner = ?` from all queries — the table IS the scope.

### DO Actor Isolation

Each DO instance has its own SQLite with simple, unsharded tables:

```sql
files  (path PK, name, size, type, addr, tx, tx_time, updated_at)
events (id PK, tx, action, path, addr, size, type, meta, msg, ts)
blobs  (addr PK, size, ref_count, created_at)
meta   (key PK, value)  -- stores next_tx counter
```

The `DOEngine` adapter splits work:
- **Metadata operations** (write record, list, search, stats) → RPC to DO
- **Blob operations** (R2 put, get, delete, presign) → directly in Worker

This avoids routing large blobs through the DO's network hop.

---

## 3. Benchmark Results

Measured locally via Miniflare (Cloudflare Workers local emulator). Both drivers
run against the same local R2 emulator and local SQLite. Absolute numbers reflect
local performance, not production. Relative comparisons are meaningful.

### Latency (lower is better)

| Operation | D1 avg (ms) | DO avg (ms) | D1 p95 (ms) | DO p95 (ms) |
|-----------|-------------|-------------|-------------|-------------|
| write     | 1.50        | 1.10        | 2.00        | 2.00        |
| head      | 0.17        | 0.28        | 1.00        | 1.00        |
| list      | 0.26        | 0.40        | 1.00        | 1.00        |
| search    | 0.24        | 0.42        | 1.00        | 1.00        |
| stats     | 0.15        | 0.28        | 1.00        | 1.00        |
| delete    | 0.57        | 0.37        | 1.00        | 1.00        |
| move      | 0.70        | 0.40        | 1.00        | 1.00        |

### Throughput (higher is better)

| Operation | D1 ops/sec | DO ops/sec |
|-----------|-----------|-----------|
| write     | 667       | 909       |
| head      | 5,882     | 3,571     |
| list      | 3,846     | 2,500     |
| search    | 4,167     | 2,381     |
| stats     | 6,667     | 3,571     |
| delete    | 1,765     | 2,727     |
| move      | 1,429     | 2,500     |

### Analysis

**Local Miniflare numbers are deceptive.** Both drivers hit local SQLite with
~0ms network latency. In production:

- **D1 reads** are served from edge replicas (~0.2ms from nearest PoP).
  D1 writes go to the primary region (~10-50ms depending on distance).
- **DO operations** always go to the DO's region (~5-30ms for the RPC hop,
  but SQL itself is synchronous and local to the DO once there).

The key production differences:

1. **Write latency:** DO wins for write-heavy actors. D1 writes serialize
   globally (one writer). DO writes serialize per-actor (zero contention).
2. **Read latency:** D1 wins. Edge replicas mean reads hit the nearest PoP.
   DO reads always route to the DO's region.
3. **Write throughput:** D1 caps at ~100-500 writes/sec globally. DO has
   no global cap — each actor gets its own write throughput.

---

## 4. Cost Comparison

### D1 Pricing (as of March 2026)

| Resource          | Free tier              | Paid                          |
|-------------------|------------------------|-------------------------------|
| Rows read         | 5M/month               | $0.001 per million            |
| Rows written      | 100K/month             | $1.00 per million             |
| Storage           | 5 GB                   | $0.75/GB/month                |
| Databases         | Unlimited              | Unlimited                     |

### Durable Objects Pricing

| Resource          | Free tier              | Paid                          |
|-------------------|------------------------|-------------------------------|
| Requests          | 1M/month (included)    | $0.15 per million             |
| Duration (wall)   | 400K GB-s/month        | $12.50 per million GB-s       |
| Storage (SQLite)  | 1 GB included          | $0.20/GB/month                |

### Per-Operation Cost

| Operation    | D1 cost          | DO cost           | Winner |
|--------------|------------------|-------------------|--------|
| write        | ~$0.000003       | ~$0.00015         | D1     |
| read (meta)  | ~$0.000001       | ~$0.00015         | D1     |
| list         | ~$0.000001       | ~$0.00015         | D1     |
| R2 PUT       | ~$0.0000044      | ~$0.0000044       | Tie    |
| R2 GET       | ~$0.00000036     | ~$0.00000036      | Tie    |

**D1 is ~50x cheaper per SQL operation.** The DO request charge ($0.15/M)
dominates. For a personal storage use case (50 writes/day, 500 reads/day),
monthly SQL cost is:

- D1: ~$0.005 (effectively free tier)
- DO: ~$0.003 (effectively free tier)

Both fall within free tiers for light use. At scale (10K writes/day):

- D1: ~$0.03/month
- DO: ~$0.50/month

### Storage Cost

| Driver | Per-actor overhead        | 1 GB total cost/month |
|--------|---------------------------|-----------------------|
| D1     | Shared (no per-actor cost)| $0.75                 |
| DO     | Each DO uses its own 1 GB | $0.20/GB/month        |

DO SQLite storage is cheaper per GB ($0.20 vs $0.75), but each DO has a 1 GB
cap. For actors with >1 GB of metadata, D1 is the only option (10 GB cap).

---

## 5. Scalability

### D1 Scaling Limits

| Dimension          | Limit                              | Mitigation              |
|--------------------|------------------------------------|-------------------------|
| Database size      | 10 GB                              | Shard to multiple D1 DBs|
| Write throughput   | ~100-500 writes/sec (global)       | Move hot actors to DO   |
| Read throughput    | Unlimited (edge replicas)          | N/A                     |
| Tables per DB      | ~10,000 (SQLite limit)             | Multiple D1 databases   |
| Actors per DB      | ~3,000 (3 tables per shard)        | Multiple D1 databases   |
| Row count          | ~66M files at 10 GB                | Per-actor sharding helps|

### DO Scaling Limits

| Dimension          | Limit                              | Mitigation              |
|--------------------|------------------------------------|-------------------------|
| Storage per DO     | 1 GB (SQLite)                      | Hard limit per actor    |
| Write throughput   | Unlimited per-actor (local SQLite) | N/A                     |
| Concurrent DOs     | Unlimited                          | N/A                     |
| DO instances       | Auto-scaled                        | N/A                     |
| Request duration   | 30 sec (default), 15 min (paid)    | Break into chunks       |
| Memory per DO      | 128 MB                             | Stream large operations |

### When to Use Each

| Scenario                              | Recommended Driver |
|---------------------------------------|--------------------|
| **Multi-tenant, light usage**         | D1 (cheapest)      |
| **Few actors, heavy writes**          | DO (no contention) |
| **Read-heavy workloads**              | D1 (edge replicas) |
| **Actor isolation required**          | DO (process-level) |
| **Cross-actor analytics**             | D1 (shared DB)     |
| **Compliance (GDPR actor deletion)**  | DO (delete the DO) |
| **Large metadata (>1 GB/actor)**      | D1 (10 GB cap)     |
| **Real-time sync per actor**          | DO (WebSocket)     |

---

## 6. Consistency & Durability

### D1

- **Writes**: Serializable (single-writer SQLite). `db.batch()` = SQLite tx.
- **Reads**: Strong consistency on primary; eventual (~100ms lag) on edge
  replicas.
- **Durability**: D1 manages replication and backups.

### DO

- **Writes**: Serializable within the DO (local SQLite). Zero contention with
  other actors.
- **Reads**: Strong consistency (single instance, no replicas).
- **Durability**: Cloudflare replicates DO storage across data centers.
- **Liveness**: DO is single-homed. If the host goes down, Cloudflare migrates
  the DO to another host (seconds of downtime).

### Cross-Service (both drivers)

R2 blob writes happen before SQL metadata commits. If SQL fails after R2 PUT,
an orphaned blob exists until GC. This is safe — the blob is unreferenced
and inert.

---

## 7. Migration Between Drivers

The `StorageEngine` interface makes drivers interchangeable. Switching requires:

1. Set `STORAGE_DRIVER=do` (or `d1`) in wrangler.toml or env vars.
2. For D1→DO: each actor's first DO access auto-creates its schema. No data
   migration needed — new writes go to the DO, old data remains in D1.
3. For DO→D1: export each DO's SQLite and import into D1 shards.

A hybrid approach is possible: keep existing actors on D1, route new or
high-throughput actors to DO. The engine selection is per-request, controlled
by the middleware in `index.ts`.

---

## 8. Driver Selection in Code

```typescript
// src/index.ts — engine middleware
const useDoDriver = c.env.STORAGE_DRIVER === "do" && c.env.STORAGE_DO;
const engine = useDoDriver
  ? new DOEngine({
      ns: c.env.STORAGE_DO!,
      bucket: c.env.BUCKET,
      ...r2Config,
    })
  : new D1Engine({
      db: c.env.DB,
      bucket: c.env.BUCKET,
      ...r2Config,
    });
c.set("engine", engine);
```

Future: per-actor routing based on a config table or feature flag:

```typescript
// Hypothetical: route hot actors to DO, others to D1
const driver = isHotActor(actor) ? doEngine : d1Engine;
```

---

## 9. Trade-Off Summary

```
                    D1Engine                DOEngine
                    ────────                ────────
Cost per op:        ~50x cheaper            Higher (DO request charge)
Write throughput:   Limited (shared writer) Unlimited (per-actor)
Read latency:       ~0.2ms (edge replicas)  ~10-30ms (single region)
Write latency:      ~10-50ms (to primary)   ~5-30ms (to DO region)
Isolation:          Table-level sharding    Process-level (strongest)
Max metadata:       10 GB (shared)          1 GB (per actor)
Cross-actor query:  Yes                     No
Actor deletion:     DROP tables             Delete DO
Schema migration:   D1 migrations           Auto on first access
Complexity:         Medium (shard mgmt)     Low (DO handles everything)
```

**Default recommendation: D1Engine.** It's cheaper, reads are faster (edge),
and the 10 GB shared DB handles thousands of actors. Switch to DOEngine when:

- A single actor generates >100 writes/second (D1 contention).
- Process-level isolation is required (compliance, security).
- You need WebSocket push from the DO (real-time sync).
- Account deletion must be instantaneous and complete.
