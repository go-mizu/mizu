# spec/0623 — Adaptive Hardware Config + Expose All Tunable Constants

## Goal

Every meaningful tuning constant in the crawler should:
1. Have a **sane auto-adaptive default** derived from detected hardware (RAM, CPU count, fd limit)
2. Be **overridable via a CLI flag** for manual tuning
3. Be **reported in the Hardware Profile** printout so operators can see what was chosen and why

No more "edit source to change batch size" or "why is DuckDB using 2 GB?". The binary should
self-configure optimally on any hardware from a 2-core VPS to a 32-core server, and every
parameter should be one flag away from manual override.

---

## Baseline (2026-02-28, from code audit)

### Server Hardware

| Server | OS | RAM | CPUs | fd limit | GOMEMLIMIT |
|--------|----|----|------|----------|------------|
| server1 | Ubuntu 20.04 (Focal) | 5.9 GB | 4 | 65,536 | ~3.8 GB |
| server2 | Ubuntu 24.04 (Noble) | 11.7 GB | 6 | 65,536 | 7.7 GB |

### Current Auto-Configured

| Constant | server1 | server2 | Formula |
|----------|---------|---------|---------|
| workers (HN batch) | ~2,066 | 8,192 | `min(wMem, wFd, 10000)` |
| innerN (conns/domain) | 8 | 4 | `clamp(CPUs×2, 4, fdSoft/(2×workers))` |
| GOMEMLIMIT | 3.8 GB | 7.7 GB | `availMB × 0.75` |
| AutoBatchDomains | ~2,156 | 4,843 | `availMB×1024/3/bodyKB/avgURLs` |

### Currently Hard-Coded (NOT adaptive, NOT CLI-configurable)

| Constant | Value | File | Line | Impact |
|----------|-------|------|------|--------|
| DuckDB `memory_limit` per shard | `256MB` | resultdb.go | 99 | 8×256=2 GB total, excessive on small RAM |
| DuckDB `checkpoint_threshold` | `4MB` | resultdb.go | 120 | Fine as-is; may need tuning with larger pool |
| DuckDB `threads` per shard | `1` | resultdb.go | 106 | Intentional; do not expose |
| ResultDB shard count | `8` | resultdb.go | 17 | Fixed; could scale with CPU count |
| ResultDB flush queue | `cap=2` | resultdb.go | 68 | Fine; controls write backpressure |
| FailedDB URL channel cap | `100,000` | faileddb.go | (flush) | Fine for now |
| BinSeg rotation size | `64 MB` | writer_bin.go | 32 | Small on large-RAM servers |
| BinSeg flush buffer | `512 KB` | writer_bin.go | 35 | Fine |
| BinChan back-pressure threshold | `0.90` | writer_bin.go | 191 | Fine |
| BinChan back-pressure pause | `100ms` | writer_bin.go | 207 | Fine |
| Worker max cap (AutoConfigKeepAlive) | `10,000` | autoconfig.go | 75 | Unnecessarily low; fd is real cap |
| Worker min floor | `200` | autoconfig.go | 75 | Fine |
| AutoWorkersFull max cap | `16,384` | autoconfig.go | 111 | Fine |
| minInnerN | `4` | autoconfig.go | 45 | Fine |
| Memory fallback (no sysinfo) | `2 GB` | autoconfig.go | 32 | Fine |
| Pass 2 workers formula | `max(workers/2, 200)` | hn.go | 1226 | Unnecessary halving; slow pass 2 |
| Pass 2 domain timeout | `retryTimeoutMs×3` | hn.go | 1230 | OK default, but worth exposing |
| HN recrawl batch size (DB writes) | `100` | hn.go flag default | 692 | Fine |
| HN recrawl DNS workers | `1000` | hn.go flag default | 697 | Fine |
| HN recrawl DNS timeout | `1500ms` | hn.go flag default | 698 | Fine |
| HN recrawl pass 1 timeout | `1000ms` | hn.go flag default | 690 | Fine |
| HN recrawl pass 2 timeout | `5000ms` | hn.go flag default | 700 | Fine |
| CC recrawl workers | `500` | cc.go flag default | — | Not adaptive to hardware |
| CC recrawl timeout | `5000ms` | cc.go flag default | — | Fixed; no adaptive |
| CC recrawl domain-fail-threshold | `-1` (engine default=3) | cc.go | — | Fixed |
| CC recrawl: no pass 2 | — | cc.go | — | False negatives never rescued |
| Adaptive histogram edges | `[100,250,500,1000,2000,3500,5000,10000]ms` | keepalive.go | 33 | Fine |
| Domain work queue cap | `min(domains, 4096)` | keepalive.go | 112 | Fine |

---

## Component 1: Adaptive DuckDB memory_limit

### Problem

`memory_limit='256MB'` is fixed regardless of server RAM. On server1 (5.9 GB, 8 shards):
8 × 256 MB = 2 GB for DuckDB alone, leaving only 1.8 GB for Go heap + workers.
On a hypothetical 2-core VPS with 2 GB RAM this causes OOM.

### Formula

```
duckMemPerShardMB = max(availMB × 0.15 / shardCount, 64)
```

At 30% of available RAM reserved for DuckDB total:
- server1 (5,900 MB avail, 8 shards): `5900 × 0.15 / 8 = 110 MB` per shard
- server2 (10,300 MB avail, 8 shards): `10300 × 0.15 / 8 = 193 MB` per shard
- 2 GB VPS (1,500 MB avail, 8 shards): `1500 × 0.15 / 8 = 28 → clamped to 64 MB`

### checkpoint_threshold

Scale with pool size: `max(duckMemPerShardMB / 40, 4)` MB.
- At 64 MB pool: 64/40 = 1.6 → 4 MB (minimum)
- At 110 MB pool: 110/40 = 2.75 → 4 MB (minimum)
- At 256 MB pool: 256/40 = 6.4 → 6 MB

### Changes

**File:** `pkg/archived/recrawler/resultdb.go`

```go
// NewResultDB signature extended:
func NewResultDB(dir string, shardCount, batchSize, duckMemPerShardMB int) (*ResultDB, error)

// In initResultSchema, take memMB as parameter:
func initResultSchema(db *sql.DB, memMB int) error {
    ckptMB := max(memMB/40, 4)
    db.Exec(fmt.Sprintf("SET memory_limit='%dMB'", memMB))
    db.Exec(fmt.Sprintf("SET checkpoint_threshold='%dMB'", ckptMB))
    // threads=1, preserve_insertion_order=false unchanged
}
```

**CLI flags (both `hn recrawl` and `cc recrawl`):**

```
--db-mem-mb int   DuckDB memory per shard in MB (0 = auto: 15% avail RAM / shards)
--db-shards int   ResultDB shard count (0 = auto: min(max(CPUs,4),16), default 8)
```

---

## Component 2: Adaptive ResultDB Shard Count

### Formula

```
shardCount = clamp(CPUCount*2, 4, 16)   # proportional to parallelism
```

- server1 (4 CPUs): `clamp(8, 4, 16) = 8` (same as current)
- server2 (6 CPUs): `clamp(12, 4, 16) = 12`
- 2-core VPS: `clamp(4, 4, 16) = 4`
- 32-core server: `clamp(64, 4, 16) = 16`

Fewer shards on small machines = less DuckDB overhead.

**File:** `pkg/archived/recrawler/resultdb.go` — add `AutoShardCount(si SysInfo) int`

**File:** `pkg/archived/recrawler/resultdb.go` + callers in `cli/hn.go`, `cli/cc.go`

---

## Component 3: Remove Artificial Worker Cap in AutoConfigKeepAlive

### Problem

`workers = max(min(wMem, wFd, 10000), 200)` — the 10,000 cap is arbitrary. On a server with
fd=65536 and innerN=4, the real cap is `65536/8 = 8,192` anyway. But on a server with fd=1M
and 64 GB RAM, we'd be artificially capped at 10,000 when hardware supports 80,000+.

### Change

Remove the `10000` upper bound, let fd and RAM be the real constraints:

```go
// Before:
workers := max(min(wMem, wFd, 10000), 200)

// After:
workers := max(min(wMem, wFd), 200)
```

**CLI flag to override:**

```
--workers-max int   Cap worker count (0 = no cap beyond fd/RAM limits)
```

---

## Component 4: Pass 2 Workers — Remove Artificial Halving

### Problem

`retryCfg.Workers = max(workers/2, 200)` — no rationale for halving. Pass 2 has slow domains
that take longer per URL, so it benefits from the SAME or MORE workers as pass 1.
Halving workers means slower pass 2 = longer total runtime.

### Change

Use the same worker count as pass 1 for pass 2:

```go
// Before:
retryCfg.Workers = max(workers/2, 200)

// After:
retryCfg.Workers = workers  // same capacity; slow domains need more concurrency, not less
```

**CLI flag:**

```
--pass2-workers int   Override pass 2 worker count (0 = same as pass 1)
```

---

## Component 5: CC Recrawl — Add Pass 2 (False Negative Recovery)

### Problem

The CC recrawl (`search cc recrawl`) has **no pass 2**. All `http_timeout`,
`domain_http_timeout_killed`, and `dns_timeout` URLs are permanently lost.

From the p:0 run: `⌛ 8,770 domain-killed (11.3%)` — 11% of processed URLs were domain-killed
and never retried. At 200K seeds with 69K processed, that's a significant false negative rate.

### Change

Mirror the HN recrawl pass 2 pattern in `runCCRecrawlV3`:

```go
// After pass 1 engine.Run():
if retryTimeoutMs > 0 && !noRetry {
    retrySeeds, _ := recrawler.LoadRetryURLs(failedDBPath)
    if len(retrySeeds) > 0 {
        fmt.Printf("Pass 2: %d urls → retrying\n", len(retrySeeds))
        retryCfg.Timeout = time.Duration(retryTimeoutMs) * time.Millisecond
        retryCfg.Workers = cfg.Workers
        retryCfg.DomainFailThreshold = 0
        retryCfg.DomainTimeout = time.Duration(retryTimeoutMs*3) * time.Millisecond
        eng2.Run(ctx, retrySeeds, dnsCache, retryCfg, pw, fw)
    }
}
```

**CLI flags to add to `cc recrawl`:**

```
--retry-timeout int   Pass-2 timeout in ms (0 = disabled, default 10000)
--no-retry            Skip pass-2 retry
```

Note: CC recrawl default pass 2 timeout is longer (10s) than HN (5s) because CC pages are
larger and more bandwidth-intensive.

---

## Component 6: CC Recrawl — Auto-Configure Workers

### Problem

CC recrawl defaults to `--workers 500` regardless of hardware. On server2 (11.7 GB, fd=65536)
it should be using 8,192+ workers like the HN recrawl.

### Change

When `--workers` is ≤0 or not specified, apply `AutoConfigKeepAlive` for CC recrawl too:

```go
if workers <= 0 || autoWorkers {
    si, _ := crawl.LoadOrGatherSysInfo(sysInfoPath, 30*time.Minute)
    cfg, reason := crawl.AutoConfigKeepAlive(si, !statusOnly)
    workers = cfg.Workers
    maxConns = cfg.MaxConnsPerDomain
    fmt.Printf("  Auto-config: %s\n", reason)
}
```

**CLI flag change:** default `--workers -1` (auto, same as HN recrawl)

---

## Component 7: BinSeg Rotation Size — Adaptive

### Current: 64 MB hardcoded

On server2 with 10 GB available, 64 MB segments mean very frequent rotation. On server1 with
5.9 GB, 64 MB is fine. Make it scale:

```go
func AutoBinSegMB(availMB int) int {
    mb := availMB / 64  // ~1.5% of available RAM per segment
    return clamp(mb, 32, 256)  // min 32 MB, max 256 MB
}
```

- server1 (5,900 MB): 5900/64 = 92 MB
- server2 (10,300 MB): 10300/64 = 160 MB
- 2 GB VPS (1,500 MB): 1500/64 = 23 → clamped 32 MB

**CLI flag:**

```
--seg-size-mb int   Binary segment rotation threshold in MB (0 = auto)
```

---

## Component 8: Print All Auto-Configured Values in Hardware Profile

### Current

Only workers/innerN/GOMEMLIMIT shown in Hardware Profile.

### Change

Print ALL auto-configured constants in the Hardware Profile section:

```
Hardware Profile
  ┌─ Hardware Profile ──────────────────────────────────────────────
  │  Hostname       vmi3112167
  │  OS             linux/amd64  │  kernel 6.8.0-100-generic
  │  CPUs           6  │  GOMAXPROCS 6  │  go1.26.0
  │  RAM total      11.7 GB
  │  RAM avail      10.3 GB (88% free)
  │  fd soft        65,536 → 65,536  │  fd hard  65,536
  └─────────────────────────────────────────────────────────────────
  GOMEMLIMIT     7.7 GB (auto-set from avail RAM)

  Auto-config:  workers=8192  innerN=4  (fd-capped (65536÷8))
  DB config:    shards=8  mem/shard=193MB  ckpt=4MB
  Bin writer:   seg=160MB  chan=4096  (from 10,300 MB avail)
  Pass 2:       workers=8192  timeout=5s  domain_timeout=15s
```

---

## Component 9: `--auto-config` Dry-Run Flag

Add a flag to print what would be auto-configured without actually running:

```bash
search hn recrawl --auto-config
# Prints hardware profile + all auto-configured values, then exits
```

Useful for operators to verify configuration before a long crawl.

---

## File Change Summary

| File | Change |
|------|--------|
| `pkg/archived/recrawler/resultdb.go` | Add `duckMemPerShardMB` param; adaptive memory_limit; checkpoint_threshold |
| `pkg/archived/recrawler/resultdb.go` | Add `AutoShardCount(si SysInfo) int` |
| `pkg/crawl/autoconfig.go` | Remove 10,000 worker cap; add `AutoBinSegMB`; add `AutoDuckMemMB` |
| `cli/hn.go` | Wire `--db-mem-mb`, `--db-shards`, `--pass2-workers`, `--seg-size-mb` flags |
| `cli/hn.go` | Pass 2 workers = same as pass 1 (remove halving) |
| `cli/hn.go` | Print all auto-configured values in Hardware Profile |
| `cli/cc.go` | Auto-configure workers via `AutoConfigKeepAlive`; `--workers -1` default |
| `cli/cc.go` | Add pass 2: `--retry-timeout 10000`, `--no-retry` |
| `cli/cc.go` | Wire same `--db-mem-mb`, `--db-shards` flags |
| `Makefile` | Update `bench-chunk` to reflect new defaults |

---

## Expected Outcomes

| Metric | Before | After |
|--------|--------|-------|
| DuckDB RSS on server1 | 2 GB (8×256) | ~880 MB (8×110) |
| DuckDB RSS on server2 | 2 GB (8×256) | ~1.5 GB (8×193) |
| CC recrawl workers | 500 (fixed) | 8,192 (auto, server2) |
| CC recrawl pass 2 | none | rescues http_timeout + domain-killed |
| Pass 2 wall time | workers/2 = slower | workers = 2× faster pass 2 |
| Operator visibility | workers+innerN only | all auto-config printed |
| Small VPS (2 GB) | OOM risk | 4 shards × 64 MB = 256 MB DuckDB |

---

## Performance Comparison: server1 vs server2 (CC recrawl p:0, --limit 200000)

| Metric | server1 (5.9 GB, 4 CPU) | server2 (11.7 GB, 6 CPU) |
|--------|--------------------------|--------------------------|
| Workers (auto) | 2,028 | 8,192 |
| innerN | 8 | 4 |
| DB shards | 8 | 12 |
| DuckDB mem/shard | 95 MB | 139 MB |
| Total DuckDB RSS | ~760 MB (was 2 GB) | ~1.67 GB (was 2 GB) |
| Seeds (p:0) | 105,285 | 200,000 |
| Pass 1 ok rate | 40.1% | 78.1% |
| Pass 1 avg RPS | 804 | ~1,000 |
| Pass 1 peak RPS | 5,235 | 2,508 |
| Pass 1 elapsed | 52s | ~34s |
| Heap / GOMEMLIMIT | 2.7 GB / 3.7 GB (72%) | 5.2–6.0 GB / 8.2 GB (64–74%) |
| Pass 2 rescued | 17,936 | 31,597 |
| Pass 2 retried | 27,726 | 40,381 |
| Pass 2 avg RPS | 710 | 1,145 |
| Pass 2 elapsed | 39s | 35s |
| OOM | None ✓ | None ✓ |

**Key observations:**
- Server1 memory dropped from 107-123% of limit (OOM) to 72% after `LoadRetryURLsSince` fix
- Server2: 12 shards vs 8 (scales with 6 CPUs), 139 MB/shard vs 256 MB — total DuckDB RSS reduced 18%
- Server1: 8 shards, 95 MB/shard vs 256 MB — total DuckDB RSS reduced 63%
- Pass 2 rescued 17,936/17,011 on server1 (+105% bonus URLs), 31,597/30,928 on server2 (+102%)
- `LoadRetryURLsSince` fix critical: without it, pass 2 loaded 767K stale URLs → OOM on server1

**Fix applied:** `LoadRetryURLsSince(failedDBPath, start)` (commit `96decd3c`) filters by `detected_at >= run_start_time` to exclude stale entries from prior runs on same parquet file.

---

## Status

- [x] `resultdb.go`: adaptive `memory_limit` + `checkpoint_threshold`; `duckMemPerShardMB` param
- [x] `autoconfig.go`: remove 10K worker cap; add `AutoShardCount`, `AutoDuckMemPerShard`, `AutoBinSegMB`, `AutoBinChanCap`
- [x] `hn.go`: wire new flags; pass 2 workers = pass 1 workers; enhanced Hardware Profile; `--auto-config` dry-run
- [x] `cc.go`: auto-configure workers (default -1=auto); add pass 2; wire `--db-mem-mb`, `--db-shards`; adaptive shard count + DuckDB mem
- [x] `swarm_drone.go`: updated `NewResultDB` call with `duckMemPerShardMB=0`
- [x] `faileddb.go`: added `LoadRetryURLsSince(dbPath, since)` to prevent stale cross-run retries
- [x] `cc.go`: use `LoadRetryURLsSince(failedDBPath, start)` in pass 2
- [x] Deployed and verified on both servers — no OOM
- [x] Verified CC recrawl pass 2 rescues domain-killed URLs (17,936 on server1, 31,597 on server2)
- [x] Verified DuckDB RSS reduction: server1 760 MB (was 2 GB, -63%), server2 1.67 GB (was 2 GB, -18%)
- [ ] Run `bench-chunk` on server2 to validate HN recrawl no regression

**Implementation commits:** `bd8fc424` (spec/0623 impl) + `96decd3c` (LoadRetryURLsSince fix)
