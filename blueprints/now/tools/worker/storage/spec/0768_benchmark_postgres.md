# 0768 — PostgreSQL Driver Benchmark & Comparison

Benchmarks comparing all three PostgreSQL-capable drivers (Neon Serverless,
Hyperdrive, D1) from deployed Cloudflare Workers on the edge. Includes local
benchmarks for reference.

---

## 1. Test Environments

### Databases

| Database | Provider | Region | Connection |
|----------|----------|--------|------------|
| D1 | Cloudflare D1 (SQLite) | SIN (Singapore) | D1 binding (edge-native) |
| Neon SIN | Neon Serverless (PostgreSQL) | ap-southeast-1 (Singapore) | HTTP + WebSocket |
| Neon EU | Neon Serverless (PostgreSQL) | eu-central-1 (Frankfurt) | HTTP + WebSocket |
| Self-hosted PG | PostgreSQL 18.3 (Docker) | Germany (Hetzner VPS) | TCP via Hyperdrive |

### Remote Test Runner

- **Environment**: Deployed Cloudflare Workers
- **Endpoint**: `GET /benchmark?driver=d1|neon|hyperdrive` (DEV_MODE only)
- **Auth**: `X-Benchmark-Key` header
- **Edge locations observed**: SIN (Singapore), HKG (Hong Kong)

### Local Test Runner

- **Environment**: Miniflare (Cloudflare Workers local emulator)
- **Location**: Local machine → remote databases
- **R2**: Local emulator

---

## 2. Remote Benchmark Results (Edge-Deployed)

Measured from deployed Cloudflare Workers hitting real databases.
These numbers represent actual production performance.

### D1 — Remote (Edge → D1 in SIN)

**Run 1** — Edge: SIN (colocated with D1)

| Operation | Avg (ms) | p50 (ms) | p95 (ms) | ops/sec |
|-----------|----------|----------|----------|---------|
| write     | 380      | 371      | 429      | 2.6     |
| head      | **9**    | 9        | 13       | 111.1   |
| list      | **9**    | 9        | 13       | 106.7   |
| search    | **9**    | 10       | 11       | 108.1   |
| stats     | **9**    | 8        | 12       | 115.9   |
| move      | 33       | 32       | 34       | 30.6    |
| delete    | 34       | 35       | 37       | 29.4    |

**Run 2** — Edge: HKG (~35ms hop to D1 in SIN)

| Operation | Avg (ms) | p50 (ms) | p95 (ms) | ops/sec |
|-----------|----------|----------|----------|---------|
| write     | 531      | 530      | 537      | 1.9     |
| head      | 48       | 49       | 52       | 20.7    |
| list      | 49       | 50       | 52       | 20.3    |
| search    | 49       | 50       | 55       | 20.4    |
| stats     | 51       | 51       | 62       | 19.6    |
| move      | 198      | 200      | 201      | 5.1     |
| delete    | 206      | 209      | 219      | 4.9     |

### Neon Serverless — Remote (Edge → Neon in SIN)

**Run 1** — Edge: SIN (colocated with Neon)

| Operation | Avg (ms) | p50 (ms) | p95 (ms) | ops/sec |
|-----------|----------|----------|----------|---------|
| write     | 434      | 397      | 576      | 2.3     |
| head      | **7**    | 7        | 9        | 150.9   |
| list      | **5**    | 4        | 7        | 205.1   |
| search    | **6**    | 6        | 6        | 181.8   |
| stats     | **5**    | 5        | 7        | 186.0   |
| move      | **13**   | 13       | 13       | 76.9    |
| delete    | **14**   | 14       | 16       | 71.4    |

**Run 2** — Edge: SIN (colocated)

| Operation | Avg (ms) | p50 (ms) | p95 (ms) | ops/sec |
|-----------|----------|----------|----------|---------|
| write     | 384      | 386      | 435      | 2.6     |
| head      | **6**    | 5        | 11       | 181.8   |
| list      | **4**    | 5        | 5        | 228.6   |
| search    | 11       | 5        | 56       | 89.9    |
| stats     | **5**    | 5        | 6        | 216.2   |
| move      | **14**   | 14       | 16       | 71.4    |
| delete    | 25       | 30       | 42       | 40.8    |

**Run 3** — Edge: HKG (~35ms hop to Neon in SIN)

| Operation | Avg (ms) | p50 (ms) | p95 (ms) | ops/sec |
|-----------|----------|----------|----------|---------|
| write     | 602      | 595      | 636      | 1.7     |
| head      | 38       | 37       | 43       | 26.1    |
| list      | 38       | 37       | 44       | 26.1    |
| search    | 37       | 37       | 44       | 26.8    |
| stats     | 39       | 37       | 43       | 25.9    |
| move      | 239      | 237      | 244      | 4.2     |
| delete    | 235      | 234      | 239      | 4.3     |

### Neon EU — Remote (Edge → Neon in eu-central-1 Frankfurt)

**Run 1** — Edge: FRA (colocated with Neon EU)

| Operation | Avg (ms) | p50 (ms) | p95 (ms) | ops/sec |
|-----------|----------|----------|----------|---------|
| write     | 1363     | 1287     | 1768     | 0.7     |
| write_meta| **22**   | 22       | 27       | 45.0    |
| head      | **7**    | 5        | 26       | 137.9   |
| list      | **5**    | 5        | 5        | 205.1   |
| search    | **5**    | 5        | 5        | 210.5   |
| stats     | **5**    | 5        | 5        | 216.2   |
| move      | **18**   | 18       | 19       | 54.5    |
| delete    | **19**   | 19       | 20       | 53.3    |

**Run 2** — Edge: SIN (~175ms hop to Neon EU in Frankfurt)

| Operation | Avg (ms) | p50 (ms) | p95 (ms) | ops/sec |
|-----------|----------|----------|----------|---------|
| write     | 1557     | 1550     | 1569     | 0.6     |
| write_meta| 1223     | 1221     | 1227     | 0.8     |
| head      | 177      | 177      | 177      | 5.7     |
| list      | 176      | 176      | 177      | 5.7     |
| search    | 176      | 177      | 177      | 5.7     |
| stats     | 176      | 176      | 177      | 5.7     |
| move      | 1223     | 1222     | 1224     | 0.8     |
| delete    | 1228     | 1231     | 1240     | 0.8     |

### Hyperdrive — Remote (Edge → Self-hosted PG 18 in Germany)

**Run 1** — Edge: CDG (Paris, ~15ms to PG in Germany) — **colocated**

| Operation | Avg (ms) | p50 (ms) | p95 (ms) | ops/sec |
|-----------|----------|----------|----------|---------|
| write     | 1822     | 1884     | 1902     | 0.5     |
| head      | **31**   | 30       | 40       | 32.1    |
| list      | **31**   | 32       | 34       | 32.4    |
| search    | **31**   | 29       | 41       | 32.3    |
| stats     | **31**   | 29       | 35       | 32.5    |
| move      | **180**  | 179      | 191      | 5.6     |
| delete    | **194**  | 197      | 208      | 5.2     |

**Run 2** — Edge: HKG (~200ms hop to PG in Germany) — **cross-continent**

| Operation | Avg (ms) | p50 (ms) | p95 (ms) | ops/sec |
|-----------|----------|----------|----------|---------|
| write     | 2778     | 2731     | 2906     | 0.4     |
| head      | 390      | 390      | 393      | 2.6     |
| list      | 391      | 391      | 399      | 2.6     |
| search    | 397      | 398      | 411      | 2.5     |
| stats     | 392      | 391      | 401      | 2.6     |
| move      | 2357     | 2359     | 2364     | 0.4     |
| delete    | 2328     | 2326     | 2341     | 0.4     |

### All Drivers from Europe (FRA/CDG Edge)

Fairest comparison — all drivers tested from European edge nodes.

| Operation | D1 (FRA→SIN) | Neon SIN (CDG→SIN) | Neon EU (FRA→FRA) | Hyperdrive (CDG→DE) |
|-----------|-------------|-------------------|--------------------|---------------------|
| **head**  | 173ms       | 165ms             | **7ms**            | 15ms                |
| **list**  | 176ms       | 165ms             | **5ms**            | 16ms                |
| **search**| 187ms       | 167ms             | **5ms**            | 21ms                |
| **stats** | 196ms       | 171ms             | **5ms**            | 21ms                |
| **write** | 1,767ms     | 2,706ms           | **1,363ms**        | 1,474ms             |
| **move**  | 760ms       | 1,073ms           | **18ms**           | 145ms               |
| **delete**| 716ms       | 1,076ms           | **19ms**           | 129ms               |

Neon EU (Frankfurt) is the **fastest driver from Europe** across all operations.
Reads at 5ms and move/delete at 18-19ms beat Hyperdrive (15-21ms / 129-145ms)
thanks to Neon's HTTP mode (zero connection overhead, optimal same-region routing).

### Write Meta — SQL-Only (No R2 Overhead)

The `write_meta` benchmark runs the same 4-5 SQL statements as a full write
(SELECT existing, INSERT tx, INSERT event, UPSERT file, UPSERT blob) but
**skips R2 head + R2 put**. This isolates pure database transaction latency
from blob storage overhead.

#### From Asia (HKG/SIN Edge)

| Driver | write (full) | write_meta (SQL only) | R2 overhead |
|--------|-------------|----------------------|-------------|
| D1 (HKG→SIN) | 598ms | **194ms** | ~404ms (68%) |
| Neon (HKG→SIN) | 726ms | **274ms** | ~452ms (62%) |
| Hyperdrive (HKG→DE) | 2,749ms | **2,399ms** | ~350ms (13%) |

From Asia, R2 adds 350-450ms to writes. For D1 and Neon, R2 is the majority
of write cost (62-68%). For Hyperdrive cross-continent, the ~2,400ms SQL
transaction dominates (each of 5 queries pays ~200ms RTT to Germany).

#### From Europe (FRA/CDG Edge)

| Driver | write (full) | write_meta (SQL only) | R2 overhead |
|--------|-------------|----------------------|-------------|
| D1 (FRA→SIN) | 2,028ms | **760ms** | ~1,268ms (63%) |
| Neon SIN (FRA→SIN) | 2,409ms | **1,123ms** | ~1,286ms (53%) |
| Neon EU (FRA→FRA) | 1,363ms | **22ms** | ~1,341ms (98%) |
| Hyperdrive (CDG→DE) | 1,474ms | **124ms** | ~1,350ms (92%) |

Neon EU write_meta at **22ms** is the fastest — 5.6x faster than Hyperdrive
(124ms). Both are colocated in Europe, but Neon's HTTP mode avoids TCP
connection overhead. R2 adds ~1,300-1,350ms from Europe for all drivers.

#### write_meta Summary

```
                    D1          Neon SIN     Neon EU      Hyperdrive
                    ──          ────────     ───────      ──────────
From Asia (HKG):    194ms       274ms        1,223ms      2,399ms
From Europe (FRA):  760ms       1,123ms      22ms ★★      124ms ★

★  = Hyperdrive colocated: 6x faster than D1 from EU
★★ = Neon EU colocated: 5.6x faster than Hyperdrive, 35x faster than D1 from EU
```

**Key insight:** Neon EU colocated (22ms) is the fastest SQL write engine,
followed by Hyperdrive (124ms). The R2 PUT operation (~1,300ms from EU) is
the dominant bottleneck — accounting for 92-98% of full write time for
colocated drivers. Moving the R2 bucket closer to European users (or using
regional R2 hints) would dramatically improve write performance.

---

## 3. Analysis

### Geographic Proximity Is Everything

The benchmark conclusively proves that **database proximity to the edge
determines performance**, not the driver type:

```
Hyperdrive colocated (CDG→DE): 31ms reads   ← fastest from Europe
Neon colocated (SIN→SIN):     5ms reads     ← fastest overall (Asia)
D1 colocated (SIN→SIN):       9ms reads
Hyperdrive colocated (CDG→DE): 31ms reads
D1 cross-region (HKG→SIN):    49ms reads   (5.4x slower than colocated)
Neon cross-region (HKG→SIN):  38ms reads   (7.6x slower than colocated)
Hyperdrive cross-continent:   390ms reads  (12.6x slower than colocated)
```

### Neon EU: Fastest from Europe

Neon EU (Frankfurt) achieves **5ms reads** from FRA edge — matching Neon
SIN's colocated performance. Move/delete at 18-19ms and write_meta at
22ms are the fastest of any driver from Europe.

Hyperdrive (CDG→DE) is a close second with 15-21ms reads but slower
writes (124ms write_meta vs 22ms). The difference is Neon's HTTP mode
vs Hyperdrive's TCP — HTTP has zero connection overhead per query.

### Neon Colocated: Fastest from Asia

When Neon runs colocated with the Worker (both in SIN), it achieves the
**fastest reads of any driver**: 4-7ms avg, up to 228 ops/sec. This beats
D1's 9ms reads because Neon's HTTP SQL proxy has zero connection overhead
and routes optimally within the same region.

### Cross-Region Comparison (HKG→SIN, ~35ms RTT)

| Operation | D1    | Neon HTTP | Winner |
|-----------|-------|-----------|--------|
| head      | 49ms  | 38ms      | Neon (**22% faster**) |
| list      | 50ms  | 38ms      | Neon (**24% faster**) |
| search    | 50ms  | 37ms      | Neon (**26% faster**) |
| write     | 531ms | 602ms     | D1 (**12% faster**) |
| move      | 198ms | 239ms     | D1 (**17% faster**) |
| delete    | 206ms | 235ms     | D1 (**12% faster**) |

Neon wins reads (HTTP is lighter than D1's protocol overhead). D1 wins
writes (fewer round trips per transaction).

---

## 4. Local Benchmark Results (Miniflare)

Measured from local Miniflare → remote Neon (ap-southeast-1). D1/DO
numbers are local SQLite with ~0ms network latency.

### Neon Serverless — Local (Miniflare → Neon SIN)

| Operation | Avg (ms) | p50 (ms) | p95 (ms) | ops/sec |
|-----------|----------|----------|----------|---------|
| write     | 311      | 312      | 316      | 3.2     |
| head      | 52       | 52       | 54       | 19.3    |
| list      | 51       | 51       | 53       | 19.5    |
| search    | 53       | 52       | 63       | 19.0    |
| stats     | 52       | 52       | 54       | 19.2    |
| delete    | 362      | 363      | 384      | 2.8     |
| move      | 359      | 361      | 362      | 2.8     |

---

## 5. Cross-Driver Comparison (Measured)

All four drivers implement the same `StorageEngine` interface. Numbers
below are from actual remote benchmarks (not estimates).

### Read Latency (lower is better)

| Operation | D1 (SIN→SIN) | D1 (HKG→SIN) | Neon (SIN→SIN) | Neon (HKG→SIN) | Hyperdrive (HKG→DE) |
|-----------|-------------|--------------|----------------|----------------|---------------------|
| head      | 9ms         | 49ms         | **5-7ms**      | 38ms           | 390ms               |
| list      | 9ms         | 50ms         | **4-5ms**      | 38ms           | 391ms               |
| search    | 9ms         | 50ms         | **6ms**        | 37ms           | 397ms               |
| stats     | 9ms         | 51ms         | **5ms**        | 39ms           | 392ms               |

### Write Latency (lower is better)

| Operation | D1 (SIN→SIN) | D1 (HKG→SIN) | Neon (SIN→SIN) | Neon (HKG→SIN) | Hyperdrive (HKG→DE) |
|-----------|-------------|--------------|----------------|----------------|---------------------|
| write     | 380ms       | 531ms        | **384ms**      | 602ms          | 2778ms              |
| move      | **33ms**    | 198ms        | **13ms**       | 239ms          | 2357ms              |
| delete    | **34ms**    | 206ms        | **14ms**       | 235ms          | 2328ms              |

### Throughput (ops/sec, higher is better)

| Operation | D1 (SIN) | D1 (HKG) | Neon (SIN) | Neon (HKG) | HD (HKG→DE) |
|-----------|----------|----------|------------|------------|-------------|
| reads     | 110      | 20       | **200+**   | 26         | 2.6         |
| writes    | 2.6      | 1.9      | 2.5        | 1.7        | 0.4         |
| move/del  | 30       | 5        | **71**     | 4          | 0.4         |

---

## 6. Cost Comparison

### Per-Operation Cost

| Operation    | D1 cost      | DO cost       | Neon cost      | Self-hosted PG   |
|--------------|--------------|---------------|----------------|------------------|
| write (SQL)  | ~$0.000003   | ~$0.00015     | ~$0.000004     | Fixed (VPS cost) |
| read (SQL)   | ~$0.000001   | ~$0.00015     | ~$0.000004     | Fixed (VPS cost) |
| R2 PUT       | ~$0.0000044  | ~$0.0000044   | ~$0.0000044    | ~$0.0000044      |
| R2 GET       | ~$0.00000036 | ~$0.00000036  | ~$0.00000036   | ~$0.00000036     |

### Monthly Cost at Scale

| Usage Level          | D1      | DO      | Neon Free | Neon Pro | Self-hosted PG |
|----------------------|---------|---------|-----------|----------|----------------|
| Light (50w/500r day) | Free    | Free    | Free      | ~$0.50   | ~$5/mo VPS     |
| Medium (1K/10K day)  | ~$0.10  | ~$1.50  | Free *    | ~$5      | ~$5/mo VPS     |
| Heavy (10K/100K day) | ~$1.00  | ~$15    | Over free | ~$30     | ~$5/mo VPS     |

Self-hosted is cheapest at scale — fixed VPS cost regardless of query volume.
Hyperdrive itself is **free** (no per-query charges from Cloudflare).

---

## 7. Architectural Trade-offs

```
                D1         DO         Neon HTTP    Hyperdrive (self-hosted)
                ────       ──         ─────────    ────────────────────────
Read latency:   Best*      Good       Best**       Depends on location
                (edge)     (DO hop)   (HTTP hop)   (TCP to your server)

Write latency:  Medium     Best       Medium       Depends on location
                (primary)  (local)    (WS+TX)      (TCP+TX to your server)

Write scale:    Limited    Best       Good         Good
                (shared)   (per-act)  (PG pool)    (PG pool)

Isolation:      Table      Process    Row-level    Row-level
                sharding   (DO)       (WHERE)      (WHERE)

Max storage:    10 GB      1 GB/act   0.5 GB free  Unlimited (your disk)
                (D1 cap)   (DO cap)   Unlimited $

Cross-actor:    Yes        No         Yes          Yes
queries         (shared)   (separate) (shared)     (shared)

Vendor lock:    High       High       Medium       Lowest
                (CF D1)    (CF DO)    (Neon API)   (standard PG)

Data control:   None       None       Limited      Full
                                      (Neon ToS)   (your server)
```

\* D1 reads fastest when colocated (9ms), comparable to Neon cross-region
\** Neon reads fastest when colocated (4-7ms), even faster than D1

---

## 8. When to Use Each Driver

| Scenario | Recommended |
|----------|-------------|
| **Maximum read speed (colocated)** | Neon Serverless |
| **Lowest cost at scale** | Hyperdrive + self-hosted PG |
| **Data sovereignty / compliance** | Hyperdrive + self-hosted PG |
| **Already on Neon** | Neon Serverless |
| **Serverless, no infra to manage** | D1 or Neon |
| **Write-heavy (colocated)** | Neon or D1 |
| **Per-actor isolation** | Durable Objects |
| **Standard SQL ecosystem** | Neon or Hyperdrive |
| **Maximum portability** | Hyperdrive + self-hosted PG |
| **Minimum configuration** | D1 (default, zero config) |

### PostgreSQL vs Cloudflare-native (D1/DO)

| Use PostgreSQL when... | Stay on D1/DO when... |
|------------------------|-----------------------|
| Need standard SQL ecosystem | Want zero-config default |
| Cross-actor analytics/reporting | Read latency critical (D1 edge replicas) |
| Existing Postgres infrastructure | Process-level isolation (DO) |
| Complex queries (JOIN, window functions) | Per-actor WebSocket |
| Unlimited storage per actor | < 3,000 actors total |
| Data sovereignty requirements | Already invested in CF |
| Want to avoid vendor lock-in | Want cheapest option (free tier) |

---

## 9. Benchmark Methodology

### Remote Benchmark Endpoint

```
GET /benchmark?driver=d1|neon|hyperdrive
Header: X-Benchmark-Key: <BENCHMARK_KEY>
```

Only available when `DEV_MODE=1` is set on the deployed Worker.

The endpoint:
1. Creates a unique actor `__benchmark_{driver}_{timestamp}`
2. Runs 8 operations with individual timing per iteration
3. Returns JSON with `{ driver, region, timestamp, results }`
4. Cleans up all benchmark data after completion

**Iterations per operation:**
- write: 5 (with 1 warmup)
- write_meta: 5 (SQL-only, no R2 — isolates DB transaction latency)
- read ops (head, list, search, stats): 8 each (with 1 warmup)
- move: 3 (no warmup, needs pre-created files)
- delete: 4 (no warmup, deletes files from write phase)

**Subrequest limit:** Cloudflare Workers allow 1000 subrequests per
invocation. Neon HTTP mode uses one subrequest per query.

### Driver-Specific Notes

**D1:** Reads use edge replicas when colocated. Writes go to the primary.

**Neon:** Reads use `neon().query()` (HTTP mode, zero connection overhead).
Writes use `Pool` (WebSocket, interactive transactions). The
`channel_binding=require` param is stripped from DSN for Pool connections.

**Hyperdrive:** Uses `postgres` library with `prepare: false` (required by
Hyperdrive's connection multiplexing). Connects to self-hosted PG 18.3 in
Germany via Hyperdrive's edge TCP pool (config ID: `91035394dca3474bb2d2a27a661eda4a`).

---

## 10. PgEngineBase Shared Architecture

Both Neon and Hyperdrive drivers extend `PgEngineBase`:

```
PgEngineBase (abstract)
├── PostgreSQL schema management (stg_files, stg_events, stg_blobs, stg_tx)
├── StorageEngine interface implementation (all 15 methods)
├── R2 blob operations (content-addressing, presign, multipart)
├── query<T>(text, params) → abstract (subclass provides)
└── transaction<R>(fn) → abstract (subclass provides)

HyperdriveEngine extends PgEngineBase
├── query() → sql.unsafe(text, params)  [postgres lib, TCP]
└── transaction() → sql.begin(fn)       [postgres lib, TCP]

NeonEngine extends PgEngineBase
├── query() → httpSql.query(text, params)  [neon HTTP, fetch()]
└── transaction() → pool.connect() + BEGIN/COMMIT  [neon WS]
```

### PostgreSQL Schema

```sql
stg_files    (owner, path) PK — file metadata projection
stg_events   (id BIGSERIAL) PK — append-only event log
stg_blobs    (addr, actor) PK — blob reference counting
stg_tx       (actor) PK — per-actor transaction counter
```

---

## 11. Summary

```
                     D1 (SIN→SIN)  Neon (SIN→SIN)  Neon EU (FRA→FRA)  Hyperdrive (CDG→DE)
                     ────────────  ──────────────  ─────────────────  ───────────────────
Read latency:        9ms           4-7ms ★         5ms ★★             15-21ms
Write latency:       380ms         384ms           1,363ms            1,474ms
Write meta (SQL):    194ms         274ms           22ms ★★★           124ms
Move/Delete:         33ms          13ms ★          18ms ★★            129-145ms
Transport:           D1 binding    HTTP + WS       HTTP + WS          TCP (Hyperdrive)
Database region:     SIN           SIN             Frankfurt (EU)     Germany (Hetzner)
Edge region:         SIN           SIN             FRA (Frankfurt)    CDG (Paris)

★   = fastest when colocated in Asia
★★  = fastest from Europe (reads + move/delete)
★★★ = fastest SQL transaction overall (22ms — 5.6x faster than Hyperdrive)
```

**Key takeaways:**

1. **Colocate your database with your users.** Neon SIN + Workers SIN =
   4ms reads. Neon EU + Workers FRA = 5ms reads. D1 SIN from EU = 173ms.

2. **Neon EU dominates from Europe.** 5ms reads, 22ms write_meta, 18ms
   move — faster than Hyperdrive (15ms reads, 124ms write_meta, 145ms move)
   thanks to Neon's HTTP mode (zero connection overhead).

3. **R2 is the write bottleneck.** Pure SQL transactions (write_meta) take
   22-274ms colocated. R2 adds 350-1,350ms depending on region. For colocated
   EU drivers, R2 accounts for 92-98% of full write time.

4. **No single "fastest" driver.** The optimal choice depends on where
   your users are. For Asian users: Neon SIN or D1. For European users:
   Neon EU. For data sovereignty: Hyperdrive + self-hosted PG. For global
   users: deploy databases in multiple regions.
