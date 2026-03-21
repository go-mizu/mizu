# Self-Hosted PostgreSQL 18 for Storage Worker

**Status**: Implemented
**Date**: 2026-03-21
**Server**: server2 (Hetzner VPS, Germany)

---

## 1. Overview

### Why Self-Host PostgreSQL?

The Storage Worker supports multiple storage drivers (D1, Durable Objects, Neon Serverless, Hyperdrive). While managed services like Neon provide zero-ops convenience, self-hosting PostgreSQL offers distinct advantages:

- **Cost control**: A Hetzner VPS at fixed monthly cost is significantly cheaper than managed PostgreSQL at scale. No per-query or per-connection fees.
- **Full control**: Unrestricted access to `postgresql.conf`, extensions, replication, and system-level tuning. No vendor-imposed connection limits or feature gates.
- **Data sovereignty**: Data stays on a known server in a known jurisdiction (Germany, Hetzner Falkenstein/Nuremberg).
- **Learning and experimentation**: Direct access to PostgreSQL 18 features (SCRAM-SHA-256 default, improved JSON path queries, incremental backup improvements) without waiting for managed provider support.
- **Benchmarking baseline**: Establishes a reference point for comparing managed services against raw PostgreSQL performance.

### Use Cases

- **Development and staging**: A persistent PostgreSQL instance for integration testing against the real Hyperdrive pipeline.
- **Low-traffic production**: For workloads where European edge latency is acceptable and cost efficiency matters.
- **Disaster recovery**: An independent storage backend that does not depend on any single cloud provider.

---

## 2. Architecture

### Connection Flow

```
┌─────────────────────────────────┐
│     Cloudflare Edge Worker      │
│     (any global PoP)            │
└──────────────┬──────────────────┘
               │
               ▼
┌─────────────────────────────────┐
│     Cloudflare Hyperdrive       │
│  ┌───────────────────────────┐  │
│  │ Edge Connection Pool      │  │
│  │ Query Result Caching      │  │
│  │ Config: 91035394dca3...   │  │
│  └───────────────────────────┘  │
└──────────────┬──────────────────┘
               │ TCP over internet (TLS 1.3)
               ▼
┌─────────────────────────────────────────────┐
│  server2 — 185.209.229.109 (Hetzner, DE)   │
│                                             │
│  ┌───────────────────┐                      │
│  │ Port 5432 (direct)│◄─── Hyperdrive       │
│  │ PostgreSQL 18.3   │     connects here     │
│  │ SCRAM-SHA-256     │                      │
│  │ TLS 1.3 required  │                      │
│  └───────────────────┘                      │
│                                             │
│  ┌───────────────────┐                      │
│  │ Port 6432         │◄─── Optional pooled  │
│  │ PgBouncer         │     access            │
│  │ Transaction mode  │                      │
│  └────────┬──────────┘                      │
│           │ unix socket / localhost          │
│           ▼                                 │
│  ┌───────────────────┐                      │
│  │ PostgreSQL 18.3   │                      │
│  └───────────────────┘                      │
│                                             │
│  ┌───────────────────┐                      │
│  │ Backup (cron)     │                      │
│  │ Daily 03:00 UTC   │                      │
│  │ 30-day retention  │                      │
│  └───────────────────┘                      │
└─────────────────────────────────────────────┘
```

### Driver Selection

The Storage Worker selects the Hyperdrive engine when the `driver` parameter is set to `"hyperdrive"` and the `HYPERDRIVE` binding is available:

```typescript
const engine =
  driver === "hyperdrive" && c.env.HYPERDRIVE
    ? new HyperdriveEngine({ connectionString: c.env.HYPERDRIVE.connectionString, ...r2Config })
    : ...
```

The `HyperdriveEngine` extends `PgEngineBase`, which uses the `postgres` (postgresjs) library with `prepare: false` — a requirement for both Hyperdrive and PgBouncer transaction-mode pooling.

### Database Schema

The `PgEngineBase` auto-creates these tables on first use:

```sql
stg_files    (owner, path) PK    — file metadata (size, content_type, timestamps, etc.)
stg_events   (id BIGSERIAL) PK   — append-only event log for all mutations
stg_blobs    (addr, actor) PK    — blob reference counting for deduplication
stg_tx       (actor) PK          — per-actor transaction counter for consistency
```

---

## 3. Server Setup

### Host Details

| Property        | Value                          |
|-----------------|--------------------------------|
| Hostname        | server2                        |
| Provider        | Hetzner (Germany)              |
| OS              | Ubuntu 24.04 LTS               |
| RAM             | 12 GB                          |
| Disk            | 193 GB SSD                     |
| Public IP       | 185.209.229.109                |
| PostgreSQL      | 18.3 (February 2026 release)   |

### Docker Compose Structure

```
docker/postgres/
├── docker-compose.yaml   # 3 services + init-certs init container
├── pg_hba.conf           # Host-based authentication rules
├── init-certs.sh         # Legacy TLS cert script (replaced by init container)
└── .env                  # POSTGRES_PASSWORD (40-char random alphanumeric)
```

### docker-compose.yaml

The compose file defines four services:

**1. init-certs (init container)**
- Runs before `postgres` starts (dependency via `depends_on` with `service_completed_successfully`)
- Generates a self-signed TLS certificate with 10-year validity
- Writes `server.crt` and `server.key` to a shared Docker volume
- Sets file ownership to UID 999 (postgres user inside the container)
- Exits after certificate generation

**2. postgres**
- Image: `postgres:18` (official Docker image)
- Port: `5432:5432` (bound to all interfaces for Hyperdrive access)
- Mounts:
  - `pg_data` volume for data directory
  - `pg_certs` volume for TLS certificates (from init-certs)
  - `pg_hba.conf` as bind mount (read-only)
- Environment: `POSTGRES_DB=storage`, `POSTGRES_USER=storage`, `POSTGRES_PASSWORD` from `.env`
- Command-line overrides for all tuning parameters (see Section 5)
- SSL configuration: `ssl=on`, `ssl_cert_file`, `ssl_key_file` pointing to cert volume
- Restart policy: `unless-stopped`
- Health check: `pg_isready -U storage` every 10s

**3. pgbouncer**
- Image: `edoburu/pgbouncer` (lightweight PgBouncer image)
- Port: `6432:6432`
- Depends on: `postgres` (healthy)
- Transaction-mode pooling (see Section 6)
- Connects to postgres via Docker internal network

**4. backup**
- Image: `postgres:18` (reuses official image for `pg_dump`)
- No exposed ports
- Depends on: `postgres` (healthy)
- Runs a cron-like entrypoint that executes daily backup at 03:00 UTC
- Mounts `pg_backups` volume for dump storage

### Docker Volumes

| Volume       | Purpose                           |
|--------------|-----------------------------------|
| `pg_data`    | PostgreSQL data directory         |
| `pg_certs`   | TLS certificates (server.crt/key)|
| `pg_backups` | Daily pg_dump archives            |

### Starting the Stack

```bash
cd docker/postgres
docker compose up -d
```

All services start in order: `init-certs` generates certificates, then `postgres` starts with TLS, then `pgbouncer` and `backup` come up once postgres is healthy.

---

## 4. Security

### TLS Configuration

- **Certificate type**: Self-signed, generated by the init-certs container
- **Validity**: 10 years (3650 days)
- **Protocol**: TLS 1.3 (PostgreSQL 18 default minimum)
- **Key**: RSA 2048-bit (sufficient for self-signed; Let's Encrypt is unnecessary since Hyperdrive connects by IP, not hostname)

The self-signed certificate is acceptable because:
1. Hyperdrive connects with `sslmode=require`, which encrypts the connection but does not verify the certificate authority.
2. The connection is point-to-point from Cloudflare's infrastructure to a known IP.
3. No browser or end-user ever connects directly to PostgreSQL.

### Authentication: SCRAM-SHA-256

PostgreSQL 18 defaults to SCRAM-SHA-256 (MD5 was deprecated in PG16 and removed as default in PG18). SCRAM-SHA-256 advantages:
- Password is never sent in plaintext, even during authentication
- Server stores a salted, iterated hash — not the password itself
- Resistant to replay attacks via channel binding
- Mutual authentication: client verifies server knows the password too

### pg_hba.conf

```
# TYPE  DATABASE  USER     ADDRESS        METHOD

# Local connections (unix socket) — trust for maintenance
local   all       all                     trust

# Loopback — password auth
host    all       all      127.0.0.1/32   scram-sha-256
host    all       all      ::1/128        scram-sha-256

# External connections — SSL required, SCRAM auth
hostssl all       all      0.0.0.0/0      scram-sha-256
hostssl all       all      ::/0           scram-sha-256

# Reject all non-SSL external connections
hostnossl all     all      0.0.0.0/0      reject
hostnossl all     all      ::/0           reject
```

Key rules:
- All external connections MUST use SSL (`hostssl`). Non-SSL connections from external IPs are explicitly rejected.
- Authentication is SCRAM-SHA-256 for all remote connections.
- Local (unix socket) connections use `trust` for docker-internal maintenance (pg_dump from backup container, PgBouncer health checks).

### Network Security

- **Firewall**: UFW on server2, ports 5432 and 6432 open (Cloudflare Hyperdrive connects from various IPs, so IP allowlisting is impractical; SCRAM + TLS provide the security layer).
- **Password strength**: 40-character random alphanumeric password stored in `.env` file (not committed to git).
- **No public pgAdmin**: Database management is via SSH tunnel only.

---

## 5. PostgreSQL Tuning

All parameters are optimized for the 12GB RAM Hetzner VPS with SSD storage. They are passed as command-line flags in `docker-compose.yaml` to avoid managing a separate `postgresql.conf`.

### Memory Parameters

| Parameter              | Value  | Rationale                                                                                       |
|------------------------|--------|-------------------------------------------------------------------------------------------------|
| `shared_buffers`       | 2 GB   | ~16% of RAM. Standard recommendation is 25%, but 2GB leaves headroom for PgBouncer and backup.  |
| `effective_cache_size` | 6 GB   | Hint to query planner: total memory available for caching (shared_buffers + OS page cache).      |
| `maintenance_work_mem` | 512 MB | Memory for VACUUM, CREATE INDEX, ALTER TABLE. 512MB speeds up maintenance on large tables.       |
| `work_mem`             | 32 MB  | Per-sort/hash operation memory. With max 100 connections, worst case is 3.2GB (acceptable).      |
| `wal_buffers`          | 64 MB  | WAL write buffer. Default auto-tuning picks ~64MB at shared_buffers=2GB; explicitly set for clarity. |

### Connection Parameters

| Parameter          | Value | Rationale                                                                |
|--------------------|-------|--------------------------------------------------------------------------|
| `max_connections`  | 100   | Direct connections. PgBouncer multiplexes 200 client connections to ~20. |

### Write-Ahead Log (WAL)

| Parameter                      | Value | Rationale                                                    |
|--------------------------------|-------|--------------------------------------------------------------|
| `checkpoint_completion_target` | 0.9   | Spread checkpoint writes over 90% of the checkpoint interval. Reduces I/O spikes. |

### Query Planner

| Parameter                | Value | Rationale                                                                |
|--------------------------|-------|--------------------------------------------------------------------------|
| `random_page_cost`       | 1.1   | SSD storage makes random reads nearly as fast as sequential. Default 4.0 is for spinning disks. |
| `effective_io_concurrency`| 200  | SSD can handle many concurrent I/O requests. Default 1 is for spinning disks. |

### Timeout Protection

| Parameter                              | Value  | Rationale                                                        |
|----------------------------------------|--------|------------------------------------------------------------------|
| `statement_timeout`                    | 30 s   | Kill queries running longer than 30 seconds. Prevents runaway queries from consuming resources. |
| `idle_in_transaction_session_timeout`  | 60 s   | Kill sessions idle inside a transaction for over 60 seconds. Prevents lock holding. |

### Logging

| Parameter                    | Value   | Rationale                                                  |
|------------------------------|---------|-----------------------------------------------------------|
| `log_min_duration_statement` | 1000 ms | Log any query taking longer than 1 second. Enables slow query analysis without overwhelming logs. |

### Why These Values?

The tuning follows the PGTune methodology for a "web application" workload:
- Moderate `work_mem` (32MB) because the Storage Worker runs simple CRUD queries, not analytical workloads
- Conservative `shared_buffers` (2GB instead of 3GB) because the server also runs PgBouncer and the backup container
- Aggressive SSD tuning (`random_page_cost=1.1`, `effective_io_concurrency=200`) because Hetzner VPS uses NVMe SSDs
- Strict timeouts to protect against connection leaks from edge workers

---

## 6. PgBouncer

### Why PgBouncer?

1. **Hyperdrive requirement**: Cloudflare Hyperdrive opens and closes connections rapidly from edge locations worldwide. Without pooling, PostgreSQL would spend excessive time on connection setup/teardown (TLS handshake, authentication, fork).
2. **Transaction mode**: Hyperdrive (and the `postgres` library with `prepare: false`) does not use prepared statements across transactions, making transaction-mode pooling safe.
3. **Connection multiplication**: 200 concurrent edge connections are multiplexed to 20 PostgreSQL backend connections, staying well within `max_connections=100`.

### Configuration

| Parameter                | Value         | Rationale                                                   |
|--------------------------|---------------|-------------------------------------------------------------|
| `pool_mode`              | `transaction` | Connections returned to pool after each transaction completes. Required by Hyperdrive. |
| `max_client_conn`        | 200           | Maximum simultaneous client connections PgBouncer accepts.   |
| `default_pool_size`      | 20            | Backend connections per user/database pair.                  |
| `min_pool_size`          | 5             | Keep at least 5 connections warm to avoid cold-start latency.|
| `reserve_pool_size`      | 5             | Extra connections available during traffic spikes.           |
| `server_tls_sslmode`     | `prefer`      | PgBouncer connects to PostgreSQL over the Docker network; TLS optional for internal traffic. |

### Transaction Mode Implications

In transaction mode, PgBouncer reassigns backend connections between transactions. This means:
- **No session-level state**: `SET` commands, temporary tables, and advisory locks do not persist between transactions.
- **No prepared statements**: The `postgres` library must use `prepare: false` (already configured in `HyperdriveEngine`).
- **No LISTEN/NOTIFY**: Notification channels require session-mode pooling.

These restrictions are acceptable because the Storage Worker uses simple, stateless transactions exclusively.

### Port Allocation

| Port | Service    | Use Case                                    |
|------|------------|---------------------------------------------|
| 5432 | PostgreSQL | Direct SSL connection (Hyperdrive connects here) |
| 6432 | PgBouncer  | Pooled connection (optional, for manual access)  |

Hyperdrive connects to port 5432 directly. PgBouncer on 6432 is available for manual access or future use if Hyperdrive is configured to use pooled connections.

---

## 7. Backup Strategy

### Daily Automated Backups

| Property        | Value                                       |
|-----------------|---------------------------------------------|
| Schedule        | Daily at 03:00 UTC                          |
| Tool            | `pg_dump` (logical backup)                  |
| Format          | Custom format with compression (`-Fc -Z6`)  |
| Retention       | 30 days                                     |
| Storage         | `pg_backups` Docker volume                  |
| Auto-prune      | Backups older than 30 days deleted after each dump |

### Backup Process

The backup container runs a loop:
1. Sleep until 03:00 UTC
2. Execute: `pg_dump -h postgres -U storage -Fc -Z6 storage > /backups/storage_$(date +%Y%m%d_%H%M%S).dump`
3. Delete files in `/backups/` older than 30 days: `find /backups -name "*.dump" -mtime +30 -delete`
4. Log success/failure
5. Return to step 1

### Restoration

To restore from a backup:

```bash
# List available backups
docker compose exec backup ls -la /backups/

# Restore a specific backup (drops and recreates objects)
docker compose exec -T backup pg_restore \
  -h postgres -U storage -d storage \
  --clean --if-exists \
  /backups/storage_20260321_030000.dump
```

### Limitations

- **Logical backup only**: `pg_dump` captures a snapshot at a point in time. Transactions committed between the last backup and a failure are lost (up to ~24 hours of data).
- **No WAL archiving**: For point-in-time recovery (PITR), WAL archiving to object storage would be needed. Not implemented because the Storage Worker can reconstruct state from R2 blobs if needed.
- **No off-site backup**: Backups are stored on the same server. For production, backups should be replicated to R2 or another location.

---

## 8. Cloudflare Hyperdrive Setup

### Creating the Hyperdrive Configuration

```bash
npx wrangler hyperdrive create storage-pg \
  --connection-string="postgres://storage:PASSWORD@185.209.229.109:5432/storage?sslmode=require"
```

This returns a Hyperdrive configuration ID: `91035394dca3474bb2d2a27a661eda4a`.

### wrangler.toml Binding

```toml
[[hyperdrive]]
binding = "HYPERDRIVE"
id = "91035394dca3474bb2d2a27a661eda4a"
```

The `HYPERDRIVE` binding exposes a `connectionString` property at runtime that the worker uses to connect. Hyperdrive transparently:
1. Maintains a persistent connection pool from Cloudflare's edge to the origin database
2. Caches read query results at the edge (with automatic invalidation)
3. Routes through Cloudflare's backbone network for reduced latency to the origin

### Connection String Anatomy

```
postgres://storage:PASSWORD@185.209.229.109:5432/storage?sslmode=require
         └──user──┘└─pass─┘└────────host────────┘└port┘└──db──┘└──tls──┘
```

- **User/Password**: The `storage` role with SCRAM-SHA-256 authentication
- **Host**: Direct public IP (no DNS) since the VPS has a static IP
- **sslmode=require**: Encrypt the connection; do not verify the certificate (self-signed)
- **Database**: `storage` — dedicated database for the Storage Worker

### Runtime Usage

```typescript
// In the Hyperdrive engine constructor
import postgres from "postgres";

const sql = postgres(env.HYPERDRIVE.connectionString, {
  prepare: false,  // Required for transaction-mode pooling
});
```

The `prepare: false` option is mandatory because:
1. Hyperdrive uses connection pooling internally
2. PgBouncer runs in transaction mode
3. Prepared statements are per-connection state that does not survive pool reassignment

---

## 9. Benchmark Results

### Test Configuration

- **Edge location**: HKG (Hong Kong)
- **Origin**: server2 (Germany)
- **Geographic distance**: ~9,200 km
- **Estimated TCP round-trip**: ~200ms
- **Driver**: Hyperdrive with `postgres` library

### Results: Hyperdrive to Self-Hosted PG (HKG to Germany)

| Operation | Avg (ms) | p50 (ms) | p95 (ms) | ops/sec |
|-----------|----------|----------|----------|---------|
| write     | 2778     | 2731     | 2906     | 0.4     |
| head      | 390      | 390      | 393      | 2.6     |
| list      | 391      | 391      | 399      | 2.6     |
| search    | 397      | 398      | 411      | 2.5     |
| stats     | 392      | 391      | 401      | 2.6     |
| move      | 2357     | 2359     | 2364     | 0.4     |
| delete    | 2328     | 2326     | 2341     | 0.4     |

### Latency Analysis

**Read operations (head, list, search, stats): ~390ms**

Each read operation requires approximately 1 TCP round trip:
- TCP + TLS round trip: ~200ms (HKG to Germany)
- Query execution: ~1-5ms
- Cloudflare internal routing: ~10-20ms
- Serialization overhead: ~5ms
- Total: ~390ms

This matches the theoretical minimum for a single round trip between HKG and Germany.

**Write operations (write, move, delete): ~2300-2800ms**

Write operations execute within a transaction and involve multiple sequential queries:
- `BEGIN`
- Check/read existing data
- `INSERT`/`UPDATE`/`DELETE` on `stg_files`
- `INSERT` into `stg_events`
- `INSERT`/`UPDATE` on `stg_blobs` (reference counting)
- `UPDATE` on `stg_tx` (transaction counter)
- `COMMIT`

Each query requires a round trip (~200ms), and a typical write transaction has 7 sequential queries: 7 x 200ms = 1400ms for network alone, plus query execution and overhead, totaling ~2500ms.

### Comparison: Neon from SIN (Singapore, Colocated)

| Operation | Self-hosted (HKG→DE) | Neon (SIN→SIN) | Factor |
|-----------|----------------------|-----------------|--------|
| Reads     | ~390ms               | 4-7ms           | ~70x   |
| Writes    | ~2500ms              | 384-434ms       | ~6x    |

### Key Takeaway

**Database proximity to the edge matters enormously.** The Hyperdrive driver itself works correctly and efficiently. The latency is purely geographic:

- **From a European edge (e.g., FRA, AMS)**: Reads would be ~10-30ms, writes ~100-200ms. Comparable to Neon in the same region.
- **From Hong Kong**: 200ms per round trip is an immutable physical constraint (speed of light through fiber).

Hyperdrive's query caching mitigates read latency for repeated queries, but write latency is irreducible without reducing the number of round trips (e.g., batching queries into a single `DO $$ ... $$` block).

---

## 10. Comparison: Self-Hosted vs Neon vs Managed

### Decision Matrix

| Factor                | Self-Hosted PG      | Neon Serverless      | Managed (RDS, Cloud SQL) |
|-----------------------|---------------------|----------------------|--------------------------|
| **Monthly cost**      | ~$10-20 (VPS)       | Free tier, then usage-based | $50-200+         |
| **Ops burden**        | High (you manage everything) | Zero           | Low (provider manages)   |
| **Latency (colocated)** | 10-30ms reads    | 4-7ms reads          | 10-30ms reads            |
| **Latency (cross-region)** | 200ms+ per RT  | Same                 | Same                     |
| **Scaling**           | Manual (bigger VPS) | Automatic            | Semi-automatic           |
| **Extensions**        | Any                 | Limited set          | Provider-dependent       |
| **Backup/PITR**       | DIY                 | Built-in             | Built-in                 |
| **High availability** | DIY (replication)   | Built-in             | Built-in                 |
| **Connection pooling**| DIY (PgBouncer)     | Built-in             | Some include it          |
| **Compliance/audit**  | Full control        | Provider-dependent   | Provider-dependent       |

### When to Use Each

**Self-hosted PostgreSQL** is best when:
- Cost is a primary concern and you have ops capability
- You need full PostgreSQL extension access (PostGIS, pgvector, etc.)
- You want complete control over data location and configuration
- You are running in a region where managed providers have limited presence
- You need a persistent development/staging database

**Neon Serverless** is best when:
- Zero operational overhead is required
- Scale-to-zero is valuable (low-traffic or bursty workloads)
- You want built-in branching for development workflows
- The free tier covers your usage
- You can deploy in a Neon-supported region close to your edge

**Managed PostgreSQL (RDS, Cloud SQL, etc.)** is best when:
- You need production-grade HA with automated failover
- Compliance requirements mandate a specific cloud provider
- You want managed backups, PITR, and monitoring
- Budget allows for the managed service premium

---

## 11. Operations

### Monitoring

**PostgreSQL built-in statistics:**

```sql
-- Active connections
SELECT count(*) FROM pg_stat_activity WHERE state = 'active';

-- Long-running queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active' AND now() - pg_stat_activity.query_start > interval '5 seconds';

-- Table sizes
SELECT relname, pg_size_pretty(pg_total_relation_size(relid))
FROM pg_catalog.pg_statio_user_tables ORDER BY pg_total_relation_size(relid) DESC;

-- Cache hit ratio (should be >99%)
SELECT
  sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) AS cache_hit_ratio
FROM pg_statio_user_tables;

-- Slow queries (from log)
-- Queries exceeding 1000ms are logged to PostgreSQL's stderr/Docker logs
docker compose logs postgres | grep "duration:"
```

**Docker-level monitoring:**

```bash
# Container health and resource usage
docker compose ps
docker stats postgres pgbouncer backup

# PostgreSQL logs
docker compose logs -f postgres --since 1h

# PgBouncer stats
docker compose exec pgbouncer psql -p 6432 -U storage pgbouncer -c "SHOW STATS;"
docker compose exec pgbouncer psql -p 6432 -U storage pgbouncer -c "SHOW POOLS;"
```

### Upgrades

**Minor version upgrade (e.g., 18.3 to 18.4):**

```bash
cd docker/postgres
docker compose pull postgres   # Pull new image
docker compose up -d postgres  # Restart with new image (data volume persists)
```

Minor upgrades are binary-compatible and require no data migration.

**Major version upgrade (e.g., 18 to 19):**

1. Take a full backup: `docker compose exec postgres pg_dumpall -U storage > /tmp/full_backup.sql`
2. Stop all services: `docker compose down`
3. Update image tag in `docker-compose.yaml` to `postgres:19`
4. Remove the data volume: `docker volume rm postgres_pg_data`
5. Start fresh: `docker compose up -d`
6. Restore: `docker compose exec -T postgres psql -U storage < /tmp/full_backup.sql`

Alternatively, use `pg_upgrade` for in-place major upgrades (faster for large databases).

### Scaling

**Vertical scaling (bigger VPS):**
1. Snapshot the Hetzner VPS
2. Create a new, larger VPS from the snapshot
3. Update the Hyperdrive connection string to the new IP
4. Adjust tuning parameters for new RAM (e.g., `shared_buffers`, `effective_cache_size`)

**Read replicas (horizontal read scaling):**
1. Set up streaming replication to a second VPS
2. Create a second Hyperdrive config pointing to the replica
3. Route read-only queries to the replica Hyperdrive binding

**Connection scaling:**
- Increase PgBouncer `max_client_conn` (can handle thousands)
- Increase `default_pool_size` if backend connections are saturated
- PostgreSQL `max_connections` rarely needs to exceed 100 with PgBouncer in front

### Troubleshooting

**Connection refused:**
```bash
# Check if PostgreSQL is running
docker compose ps
# Check if port is open from outside
nc -zv 185.209.229.109 5432
# Check UFW firewall
sudo ufw status | grep 5432
# Check PostgreSQL logs for auth failures
docker compose logs postgres | grep "FATAL"
```

**Authentication failures:**
```bash
# Verify pg_hba.conf is mounted correctly
docker compose exec postgres cat /etc/postgresql/pg_hba.conf
# Test SCRAM authentication locally
docker compose exec postgres psql -U storage -h 127.0.0.1 -c "SELECT 1;"
# Check for password mismatch
docker compose exec postgres psql -U storage -c "SELECT rolname, rolpassword IS NOT NULL FROM pg_authid;"
```

**High latency:**
```bash
# Check if it's network or query latency
docker compose exec postgres psql -U storage -c "EXPLAIN ANALYZE SELECT * FROM stg_files LIMIT 10;"
# Check for lock contention
docker compose exec postgres psql -U storage -c "SELECT * FROM pg_locks WHERE NOT granted;"
# Check PgBouncer pool saturation
docker compose exec pgbouncer psql -p 6432 -U storage pgbouncer -c "SHOW POOLS;"
```

**Disk space:**
```bash
# Check disk usage
df -h
# Check PostgreSQL data size
docker compose exec postgres psql -U storage -c "SELECT pg_size_pretty(pg_database_size('storage'));"
# Check backup volume
docker compose exec backup du -sh /backups/
# Force VACUUM if bloat is suspected
docker compose exec postgres psql -U storage -c "VACUUM (VERBOSE, ANALYZE) stg_files;"
```

**Hyperdrive not connecting:**
```bash
# Verify Hyperdrive config
npx wrangler hyperdrive get 91035394dca3474bb2d2a27a661eda4a
# Test connection string directly (from a machine with psql)
psql "postgres://storage:PASSWORD@185.209.229.109:5432/storage?sslmode=require"
# Check Hyperdrive logs in Workers dashboard
# Hyperdrive errors appear in the worker's console.log output
```
