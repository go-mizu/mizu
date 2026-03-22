# 0766 — Hyperdrive PostgreSQL Driver

Architecture spec for the Hyperdrive storage driver (`HyperdriveEngine`). This
driver connects Cloudflare Workers to a standard PostgreSQL database through
Cloudflare Hyperdrive, an edge TCP connection-pooling proxy. Blob storage uses
R2, identical to the D1 and DO drivers.

---

## 1. Overview

Hyperdrive is a Cloudflare service that sits between Workers and a PostgreSQL
database. It maintains warm TCP connection pools at the edge so that each Worker
request avoids the full TLS+auth handshake to the origin database. The driver
uses the `postgres` (postgresjs) library for SQL execution and delegates all
StorageEngine interface methods to `PgEngineBase`, a shared abstract class also
used by the Neon driver.

Key characteristics:

- **Connection pooling at the edge** — sub-millisecond connection acquisition
- **Query caching** — Hyperdrive can cache identical read queries at the edge
- **Automatic retry** — transient TCP errors are retried transparently
- **Any PostgreSQL provider** — Neon, Supabase, AWS RDS, self-hosted, etc.
- **Standard SQL** — uses `$1`-style parameterized queries via `sql.unsafe()`
- **`prepare: false`** — Hyperdrive's connection multiplexing does not support
  named prepared statements; all queries go through the simple/extended protocol
  without server-side statement caching

---

## 2. Architecture Diagram

```
                        Cloudflare Edge (PoP)
                ┌──────────────────────────────────┐
                │                                  │
  Client ──►   │  Worker                           │
  (HTTP)       │    │                              │
               │    ├── StorageEngine interface     │
               │    │     │                        │
               │    │     ├── SQL queries ──────────┼──► Hyperdrive ──TCP──► PostgreSQL
               │    │     │   (postgresjs)         │    (conn pool)        (Neon/Supabase/
               │    │     │                        │                        RDS/self-hosted)
               │    │     └── R2 blob ops ─────────┼──► R2 Bucket
               │    │         (put/get/delete)     │    (blob storage)
               │    │                              │
               │    └── Response                   │
                └──────────────────────────────────┘

  Data flow:
    write:  Worker ──PUT──► R2 (blob)  then  Worker ──SQL──► Hyperdrive ──► PG (metadata)
    read:   Worker ──SQL──► Hyperdrive ──► PG (metadata)  then  Worker ──GET──► R2 (blob)
    list:   Worker ──SQL──► Hyperdrive ──► PG (query only, no R2)
```

---

## 3. Hyperdrive Connection Model

### How Hyperdrive Works

Hyperdrive maintains a pool of authenticated TCP connections to the origin
PostgreSQL database at each Cloudflare edge PoP. When a Worker makes a database
request:

1. **Worker creates a `postgres` client** using `env.HYPERDRIVE.connectionString`.
   This connection string points to a Hyperdrive-generated hostname, not the
   origin database directly.

2. **Hyperdrive intercepts the TCP connection** at the edge. If a warm
   connection to the origin exists in the pool, it reuses it (sub-ms). If not,
   it opens a new one (full TLS handshake, ~50-200ms to the origin).

3. **SQL flows through the proxy** — Hyperdrive speaks the PostgreSQL wire
   protocol. It can inspect queries for caching purposes.

4. **Connection returns to the pool** after the Worker request completes.
   The `idle_timeout: 0` setting in the driver means the Worker-side client
   does not try to keep connections alive between requests (Workers are
   stateless).

### Driver Configuration

```typescript
// src/storage/hyperdrive_driver.ts

this.sql = postgres(config.connectionString, {
  max: 1,           // One connection per request — Hyperdrive handles pooling
  idle_timeout: 0,  // Workers don't persist between requests
  prepare: false,   // Hyperdrive doesn't support named prepared statements
});
```

**Why `max: 1`?** Each Worker invocation is a single request. Hyperdrive pools
connections across thousands of concurrent Workers. Having the Worker open
multiple connections would waste pool capacity.

**Why `prepare: false`?** Hyperdrive multiplexes many Worker connections onto
fewer origin connections. Named prepared statements are tied to a specific
PostgreSQL backend connection. Since Hyperdrive may route subsequent queries
to different backends, prepared statements would fail with
`prepared statement does not exist`. Disabling `prepare` forces the simple
query protocol or unnamed prepared statements, both of which work correctly
through a connection multiplexer.

**Why `idle_timeout: 0`?** Workers are ephemeral. There is no long-lived
process to keep connections alive. The connection is used for one request and
then discarded (Hyperdrive keeps the upstream TCP connection alive).

### Query Execution

All queries go through `sql.unsafe(text, params)`:

```typescript
protected async query<T>(text: string, params?: any[]): Promise<T[]> {
  const rows = await this.sql.unsafe(text, params || []);
  return rows as unknown as T[];
}
```

`unsafe()` accepts raw SQL strings with `$1`-style positional parameters. This
is necessary because `PgEngineBase` builds dynamic SQL (e.g., conditional WHERE
clauses in `search()` and `log()`). The postgresjs tagged template API
(`sql\`...\``) cannot express dynamic query construction cleanly.

### Transactions

Transactions use `sql.begin()` from postgresjs, which issues
`BEGIN`/`COMMIT`/`ROLLBACK` over a single connection:

```typescript
protected async transaction<R>(fn: (q: QueryFn) => Promise<R>): Promise<R> {
  return this.sql.begin(async (tx) => {
    const q: QueryFn = async <T>(text: string, params?: any[]): Promise<T[]> => {
      const rows = await tx.unsafe(text, params || []);
      return rows as unknown as T[];
    };
    return fn(q);
  });
}
```

Hyperdrive pins the connection for the duration of a transaction — all queries
within `sql.begin()` are guaranteed to hit the same PostgreSQL backend.

---

## 4. PostgreSQL Schema

The `PgEngineBase` creates these tables on first access (idempotent
`CREATE TABLE IF NOT EXISTS`):

```sql
-- File metadata (one row per file per owner)
CREATE TABLE IF NOT EXISTS stg_files (
  owner      TEXT    NOT NULL,
  path       TEXT    NOT NULL,
  name       TEXT    NOT NULL,
  size       BIGINT  NOT NULL DEFAULT 0,
  type       TEXT    NOT NULL DEFAULT 'application/octet-stream',
  addr       TEXT,                    -- SHA-256 content address (R2 blob key)
  tx         INTEGER,                 -- Transaction number
  tx_time    BIGINT,                  -- Transaction timestamp (epoch ms)
  updated_at BIGINT  NOT NULL,
  PRIMARY KEY (owner, path)
);
CREATE INDEX IF NOT EXISTS idx_stg_files_name    ON stg_files(owner, lower(name));
CREATE INDEX IF NOT EXISTS idx_stg_files_updated ON stg_files(owner, updated_at DESC);

-- Event log (append-only audit trail)
CREATE TABLE IF NOT EXISTS stg_events (
  id     BIGSERIAL PRIMARY KEY,
  tx     INTEGER   NOT NULL,
  actor  TEXT      NOT NULL,
  action TEXT      NOT NULL CHECK(action IN ('write','move','delete')),
  path   TEXT      NOT NULL,
  addr   TEXT,
  size   BIGINT    NOT NULL DEFAULT 0,
  type   TEXT,
  meta   TEXT,                        -- JSON metadata (e.g., {"from": "old/path"})
  msg    TEXT,                        -- Human-readable commit message
  ts     BIGINT    NOT NULL           -- Epoch ms
);
CREATE INDEX IF NOT EXISTS idx_stg_events_actor_tx ON stg_events(actor, tx DESC);
CREATE INDEX IF NOT EXISTS idx_stg_events_path     ON stg_events(actor, path, tx DESC);

-- Blob reference counting (for R2 GC)
CREATE TABLE IF NOT EXISTS stg_blobs (
  addr       TEXT    NOT NULL,
  actor      TEXT    NOT NULL,
  size       BIGINT  NOT NULL,
  ref_count  INTEGER NOT NULL DEFAULT 1,
  created_at BIGINT  NOT NULL,
  PRIMARY KEY (addr, actor)
);

-- Per-actor monotonic transaction counter
CREATE TABLE IF NOT EXISTS stg_tx (
  actor   TEXT PRIMARY KEY,
  next_tx INTEGER NOT NULL DEFAULT 1
);
```

### Schema Design Decisions

- **`stg_` prefix**: Avoids name collisions when sharing a database with other
  applications (common with Neon/Supabase shared databases).

- **`owner` column instead of table sharding**: Unlike the D1 driver (which
  creates per-actor tables like `f_{shard}`), PostgreSQL uses a standard
  multi-tenant `owner` column with composite indexes. PostgreSQL's query planner
  handles partition pruning efficiently with B-tree indexes.

- **`BIGSERIAL` for event IDs**: PostgreSQL sequences provide gap-free,
  monotonically increasing IDs without the locking overhead of a counter table.

- **`ILIKE` for search**: PostgreSQL's case-insensitive LIKE enables
  case-insensitive file search without a separate collation or function index
  (though `lower(name)` is indexed for exact prefix matches).

- **Schema is checked once per isolate**: A module-level `schemaReady` flag
  avoids running `CREATE TABLE IF NOT EXISTS` on every request. Reset between
  tests via `resetSchemaFlag()`.

### Content Addressing

Files are stored in R2 using SHA-256 content addressing:

```
R2 key: blobs/{actor}/{hash[0:2]}/{hash[2:4]}/{hash}

Example: blobs/user@example.com/a1/b2/a1b2c3d4e5f6...  (full 64-char hex)
```

The two-level directory fan-out (`hash[0:2]/hash[2:4]`) prevents R2 prefix
hotspots. Identical file contents within an actor share the same R2 object
(dedup via `bucket.head()` before PUT). The `stg_blobs` table tracks reference
counts; blobs with `ref_count = 0` are candidates for garbage collection.

---

## 5. How It Differs from D1/DO/Neon Drivers

| Aspect                    | D1Engine              | DOEngine              | HyperdriveEngine       | NeonEngine              |
|---------------------------|-----------------------|-----------------------|------------------------|-------------------------|
| **Database**              | Cloudflare D1 (SQLite)| Per-actor DO SQLite   | Any PostgreSQL via Hyperdrive | Neon PostgreSQL    |
| **Connection model**      | D1 binding (no TCP)   | RPC to DO (no TCP)    | TCP via edge proxy     | HTTP + WebSocket        |
| **Connection pooling**    | N/A (D1 managed)      | N/A (DO local)        | Hyperdrive edge pool   | None (per-request)      |
| **SQL dialect**           | SQLite                | SQLite                | PostgreSQL             | PostgreSQL              |
| **SQL library**           | D1 binding API        | DO `ctx.storage.sql`  | `postgres` (postgresjs)| `@neondatabase/serverless` |
| **Transactions**          | `db.batch()`          | `transactionSync()`   | `sql.begin()`          | Pool client BEGIN/COMMIT|
| **Schema approach**       | Per-actor table shards| Per-DO unsharded      | Shared tables + `owner`| Shared tables + `owner` |
| **Isolation**             | Table-level           | Process-level         | Row-level (owner col)  | Row-level (owner col)   |
| **Prepared statements**   | N/A (D1 API)          | N/A (DO sync SQL)     | Disabled (`prepare: false`) | N/A (HTTP mode)    |
| **Query caching**         | D1 edge replicas      | None (single region)  | Hyperdrive query cache  | None                   |
| **Read latency**          | ~0.2ms (edge replica) | ~10-30ms (DO region)  | ~1-5ms (cache hit) / ~20-80ms (miss) | ~30-100ms (HTTP) |
| **Write latency**         | ~10-50ms (to primary) | ~5-30ms (to DO)       | ~20-80ms (to origin PG)| ~30-100ms (WebSocket)   |
| **Provider lock-in**      | Cloudflare only       | Cloudflare only       | Any PostgreSQL         | Neon only               |
| **Base class**            | Standalone            | Standalone            | `PgEngineBase`         | `PgEngineBase`          |

### Hyperdrive vs Neon Driver (both use PgEngineBase)

Both extend `PgEngineBase` and share identical SQL logic. The difference is
only in how they connect to PostgreSQL:

| Aspect              | HyperdriveEngine                    | NeonEngine                          |
|---------------------|-------------------------------------|-------------------------------------|
| **Reads**           | TCP via Hyperdrive proxy            | HTTP (one request per query, zero conn overhead) |
| **Transactions**    | `sql.begin()` over TCP              | WebSocket Pool + `BEGIN`/`COMMIT`   |
| **Pooling**         | Hyperdrive edge pool (managed)      | Application-level `Pool` (per-isolate) |
| **Query caching**   | Built-in (Hyperdrive)               | None                                |
| **Provider**        | Any PostgreSQL                      | Neon only                           |
| **TCP dependency**  | Yes (Hyperdrive provides TCP)       | No (HTTP + WebSocket only)          |
| **Prepared stmts**  | Disabled                            | N/A (HTTP mode)                     |

---

## 6. Configuration

### Environment Variables

```
STORAGE_DRIVER=hyperdrive          # Select the Hyperdrive driver
```

The `HYPERDRIVE` binding is set in `wrangler.toml`, not as an env var.

### wrangler.toml

```toml
[[hyperdrive]]
binding = "HYPERDRIVE"
id = "<hyperdrive-config-id>"      # From `wrangler hyperdrive create`
```

### Creating a Hyperdrive Configuration

```bash
# Create a Hyperdrive config pointing to your PostgreSQL database
wrangler hyperdrive create my-storage-db \
  --connection-string="postgres://user:pass@host:5432/dbname?sslmode=require"

# Output includes the Hyperdrive config ID — put this in wrangler.toml
```

### R2 Configuration (same for all drivers)

```toml
[[r2_buckets]]
binding = "BUCKET"
bucket_name = "storage-files"
```

Optional presigned URL support (for direct client uploads):

```
R2_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com
R2_ACCESS_KEY_ID=<key>
R2_SECRET_ACCESS_KEY=<secret>
R2_BUCKET_NAME=storage-files
```

### Driver Selection in Code

```typescript
// src/index.ts — engine middleware
const driver = c.env.STORAGE_DRIVER;
const engine =
  driver === "do" && c.env.STORAGE_DO
    ? new DOEngine({ ns: c.env.STORAGE_DO!, ...r2Config })
    : driver === "hyperdrive" && c.env.HYPERDRIVE
      ? new HyperdriveEngine({
          connectionString: c.env.HYPERDRIVE.connectionString,
          ...r2Config,
        })
      : driver === "neon" && c.env.POSTGRES_DSN
        ? new NeonEngine({ connectionString: c.env.POSTGRES_DSN, ...r2Config })
        : new D1Engine({ db: c.env.DB, ...r2Config });
```

The `HYPERDRIVE` binding exposes a `.connectionString` property at runtime.
This is a Hyperdrive-generated connection string (pointing to the edge proxy),
not the origin database connection string.

---

## 7. Pricing

### Hyperdrive Pricing (as of March 2026)

| Resource               | Cost                                |
|------------------------|-------------------------------------|
| Hyperdrive service     | **$0** (included with Workers Paid) |
| Connection pooling     | **$0** (no per-connection charge)   |
| Query caching          | **$0** (no per-cache-hit charge)    |
| Bandwidth (Hyperdrive) | **$0** (included in Workers egress) |

Hyperdrive itself is free. You pay only for:

1. **Workers requests** — standard Workers pricing ($0.30/M requests on Paid plan)
2. **PostgreSQL hosting** — from your chosen provider
3. **R2 storage** — for blob data

### PostgreSQL Provider Costs (representative)

| Provider        | Free tier              | Paid (starter)                      |
|-----------------|------------------------|-------------------------------------|
| **Neon**        | 0.5 GB, 190 hours/mo   | $19/mo (10 GB, 300 hours)           |
| **Supabase**    | 500 MB, 2 projects     | $25/mo (8 GB, unlimited)            |
| **AWS RDS**     | 750 hrs t3.micro/yr    | ~$15/mo (db.t4g.micro)             |
| **Self-hosted** | N/A                    | Server cost                         |

### R2 Pricing (same for all drivers)

| Resource          | Free tier           | Paid                     |
|-------------------|---------------------|--------------------------|
| Storage           | 10 GB/month         | $0.015/GB/month          |
| Class A (PUT)     | 1M/month            | $4.50 per million        |
| Class B (GET)     | 10M/month           | $0.36 per million        |
| Egress            | Free                | Free                     |

### Cost Comparison: Hyperdrive vs D1 (per SQL operation)

| Operation | D1 cost       | Hyperdrive cost (PG provider) | Notes                          |
|-----------|---------------|-------------------------------|--------------------------------|
| Read      | ~$0.000001    | Provider-dependent            | D1 charges per row read        |
| Write     | ~$0.000003    | Provider-dependent            | D1 charges per row written     |
| List      | ~$0.000001    | Provider-dependent            | D1 charges per row scanned     |

D1 is significantly cheaper for SQL operations because Cloudflare subsidizes
D1 pricing. With Hyperdrive, the SQL cost shifts entirely to the PostgreSQL
provider. However, Hyperdrive query caching can reduce origin load for
read-heavy workloads (cached reads cost $0 at the PostgreSQL provider).

---

## 8. When to Use

### Use Hyperdrive When

- **You already have a PostgreSQL database** — Neon, Supabase, RDS, or
  self-hosted. Hyperdrive plugs in without changing your database.

- **You want provider portability** — unlike D1 (Cloudflare-only) or the Neon
  driver (Neon-only), Hyperdrive works with any PostgreSQL. Switch from Neon to
  Supabase to self-hosted by changing the connection string.

- **You need PostgreSQL features** — full-text search (`tsvector`), JSONB
  operators, CTEs, window functions, advanced indexing (GIN, GiST, BRIN) that
  SQLite/D1 cannot provide.

- **You need cross-actor queries** — like D1, the shared-table design supports
  `SELECT ... WHERE owner IN (...)` for admin dashboards and analytics. DO
  cannot do this.

- **You need query caching** — Hyperdrive caches identical read queries at the
  edge. For read-heavy public content (e.g., a file-sharing app), this reduces
  both latency and origin database load.

- **You have an existing PostgreSQL schema** — the `stg_` prefix tables can
  coexist with other application tables in the same database.

### Do Not Use When

- **Cost is the primary concern** — D1 is dramatically cheaper per SQL
  operation. For small/personal projects, D1 is nearly free.

- **You need sub-millisecond reads** — D1 edge replicas serve reads from the
  nearest PoP (~0.2ms). Hyperdrive cache hits are ~1-5ms; misses go to origin
  (~20-80ms).

- **You need per-actor isolation** — Hyperdrive uses row-level isolation via
  the `owner` column. For compliance requirements demanding process-level
  isolation, use DO.

- **You need offline/edge-local writes** — D1 edge replicas and DO provide
  Cloudflare-managed durability. With Hyperdrive, your PostgreSQL origin is a
  single point of failure (mitigated by the provider's own HA).

---

## 9. Trade-offs vs D1/DO/Neon Drivers

```
                    D1Engine          DOEngine          HyperdriveEngine    NeonEngine
                    ────────          ────────          ────────────────    ──────────
Cost (SQL):         Cheapest          Moderate          Provider-dependent  Provider-dependent
                    (~$0.001/M rows)  ($0.15/M reqs)    (PG hosting cost)   (Neon pricing)

Read latency:       ~0.2ms (edge)     ~10-30ms          ~1-5ms (cache)      ~30-100ms (HTTP)
                                      (DO region)       ~20-80ms (miss)

Write latency:      ~10-50ms          ~5-30ms           ~20-80ms            ~30-100ms
                    (to D1 primary)   (to DO region)    (to PG origin)      (WebSocket)

Write throughput:   ~100-500/s        Unlimited/actor   PG-dependent        PG-dependent
                    (global cap)                        (~1000-10K/s)       (~1000-10K/s)

Query caching:      Edge replicas     None              Hyperdrive cache    None
                    (eventual)                          (automatic)

Provider lock-in:   Cloudflare        Cloudflare        None (any PG)       Neon only

Max database:       10 GB             1 GB/actor        PG-dependent        Neon plan limit
                                                        (TB+ possible)

Isolation:          Table sharding    Process-level     Row-level (owner)   Row-level (owner)

Cross-actor query:  Yes               No                Yes                 Yes

Transactions:       db.batch()        transactionSync() sql.begin()         Pool + BEGIN
                    (batch = tx)      (real ACID)       (real ACID)         (real ACID)

SQL dialect:        SQLite            SQLite            PostgreSQL          PostgreSQL

Complexity:         Medium (shards)   Low (DO managed)  Low (standard PG)   Low (standard PG)

Best for:           Multi-tenant      Actor isolation   Existing PG infra   Neon-native apps
                    light workloads   heavy writes      portability needs   serverless PG
```

### Decision Matrix

| If you need...                        | Choose           | Reason                               |
|---------------------------------------|------------------|--------------------------------------|
| Lowest cost                           | D1               | Subsidized pricing, generous free tier|
| Fastest reads                         | D1               | Edge replicas, ~0.2ms                |
| Fastest writes per actor              | DO               | Local SQLite, zero contention        |
| PostgreSQL compatibility              | Hyperdrive/Neon  | Full PG dialect and features         |
| Provider portability                  | Hyperdrive       | Works with any PG provider           |
| Edge query caching                    | Hyperdrive       | Built-in, zero config                |
| No TCP dependency (HTTP-only Workers) | Neon             | HTTP for reads, WebSocket for writes |
| Unlimited database size               | Hyperdrive       | PG providers support TB+             |
| Process-level actor isolation         | DO               | Each actor gets its own DO + SQLite  |
| Cross-actor analytics                 | D1 / Hyperdrive  | Shared database, standard queries    |
| Real-time sync (WebSocket push)       | DO               | DO supports WebSocket hibernation    |

### Migration Path

All four drivers implement the same `StorageEngine` interface. Switching
requires:

1. Change `STORAGE_DRIVER` env var (e.g., `d1` to `hyperdrive`).
2. Add the `[[hyperdrive]]` binding to `wrangler.toml`.
3. Run the PostgreSQL schema (auto-created on first request via `ensureSchema()`).
4. Migrate existing data from D1/DO (export + import) or start fresh.

The engine selection happens per-request in the middleware. A hybrid deployment
is possible: route specific actors or paths to different drivers based on
configuration or feature flags.
