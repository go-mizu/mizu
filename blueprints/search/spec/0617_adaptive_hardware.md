# Spec 0617: Adaptive Hardware Detection + Keepalive Throughput

**Date:** 2026-02-27
**Branch:** `open-index`
**Goal:** Auto-detect server hardware, fix fd limit for keepalive engine, auto-tune workers/innerN, and achieve 3,000+ avg OK pages/s on server2 with the keepalive engine.

---

## Server Hardware Profiles

| Metric | server1 (`tam@server`) | server2 (`root@server2`) |
|--------|----------------------|------------------------|
| CPUs | 4 × AMD EPYC @ 2794 MHz | 6 × AMD EPYC |
| RAM total | 5.8 GB | 11 GB |
| RAM available | **1.3 GB** (4.1 GB used) | **11 GB** (523 MB used) |
| Swap | none | none |
| fd soft (before) | 1,024 | 1,024 |
| fd hard | 1,048,576 | 1,048,576 |
| OS | Ubuntu 20.04 LTS | Ubuntu 24.04 LTS |
| Kernel | 5.4.0-105-generic | 6.8.0-100-generic |
| HN data | ✅ ready | ❌ needs seed copy |
| GOMEMLIMIT (wrapper) | 2 GB | 2 GB (wrong — too high for s1, too low for s2) |

---

## Baseline Benchmarks (pre-0617)

| Engine | Server | Workers | InnerN | Seeds | Avg RPS | OK/s | OK% | Duration |
|--------|--------|---------|--------|-------|---------|------|-----|---------|
| keepalive | server1 | 3,000 | 4 | 157K | ~2,480 | ~1,860 | 75% | ~63s |
| swarm | server1 | 300/drone | 8 | 1.27M | 694 | 594 | 85.9% | 10m13s |

---

## Root Cause Analysis

### 1. fd Soft Limit = 1,024 Blocks Keepalive at ~2,560 RPS

By Little's Law:
```
concurrent_connections = RPS × avg_latency_seconds
```

At 2,480 avg RPS and 400ms avg latency: `2480 × 0.4 = 992 concurrent connections ≈ 1,024`.
The fd limit is the exact ceiling. To exceed 2,560 RPS, we need `fd_soft > 1024`.

**`raiseRlimit(65536)` is called in SwarmEngine drone (`swarm_drone.go`) but NOT in `KeepAliveEngine.Run()`.** The main process crawl is still fd-limited.

### 2. GOMEMLIMIT Mismatch

- Server1 wrapper: `GOMEMLIMIT=2 GB`, but only 1.3 GB available → GC won't kick in until OOM
- Server2 wrapper: `GOMEMLIMIT=2 GB`, but 11 GB available → GC is overly aggressive, wastes RAM

### 3. No Hardware-Aware Tuning

Default `workers=1000, innerN=8` was a compromise. Server2 can safely use `workers=2730, innerN=12`. Server1 is memory-constrained to ~520 workers.

---

## Changes

### 1. `pkg/crawl/sysinfo.go` + `sysinfo_linux.go` + `sysinfo_other.go`

New `SysInfo` struct gathering:
- Hostname, OS, arch, kernel version
- CPU count, GOMAXPROCS, Go version
- MemTotal, MemAvailable (from `/proc/meminfo` on Linux)
- fd soft (before raise), fd soft (after raise attempt), fd hard
- GatheredAt timestamp

`LoadOrGatherSysInfo(cacheFile string, ttl time.Duration) SysInfo`:
- Loads from `~/.cache/search/sysinfo.json` if fresh (TTL: 30 min)
- Otherwise gathers live and saves to cache
- Always calls `raiseRlimit(65536)` regardless of cache hit
- Returns `SysInfo.FromCache = true` when loaded from cache

`(SysInfo).Table() string`: pretty-printed hardware table.

### 2. `pkg/crawl/autoconfig.go`

`AutoConfigKeepAlive(si SysInfo, fullBody bool) (Config, string)`:

```
innerN = clamp(CPUCount×2, 4, 16)
availKB = MemAvailableMB × 1024   (fallback: 2 GB if unknown)
bodyKB = 256 (full body) or 4 (status-only)

memExpectedKB = innerN × bodyKB / 4   # 25% saturation model
memWorstKB    = innerN × bodyKB

wMem = min(availKB×0.70 / memExpectedKB,
           availKB×0.80 / memWorstKB)  # soft & hard constraint

fdSoft = FdSoftAfter (after raise)
wFd   = fdSoft / (innerN × 2)         # safety factor 2×

workers = max(min(wMem, wFd, 10000), 200)
```

Expected results:

| Server | innerN | wMem | wFd | **workers** | Worst-case mem |
|--------|--------|------|-----|-------------|----------------|
| server2 | 12 | 10,266 | 2,730 | **2,730** | 8.4 GB (of 11 GB) |
| server1 | 8 | 1,820 | 4,096 | **520** | 1.04 GB (of 1.3 GB avail) |

The limiting factor: server2 is **fd-capped** (65536÷24); server1 is **memory-capped**.

### 3. `pkg/crawl/keepalive.go`

Add `raiseRlimit(65536)` at the top of `KeepAliveEngine.Run()` as a safety net (idempotent, called even if sysinfo wasn't gathered).

### 4. `cli/hn.go`

- `--workers` default: `1000` → `-1` (auto-detect from hardware)
- `--max-conns-per-domain` default: `8` → `-1` (auto-detect from hardware)
- In `runHNRecrawlV3`:
  1. Call `LoadOrGatherSysInfo(cacheFile, 30m)`
  2. Print `SysInfo.Table()`
  3. If `workers <= 0`: call `AutoConfigKeepAlive`, apply result
  4. If `workers > 0` but `maxConns <= 0`: auto-set innerN = clamp(CPUs×2, 4, 16)
  5. Call `debug.SetMemoryLimit(MemAvailableMB × 1024² × 0.75)` — overrides wrapper's 2 GB

### 5. `Makefile`

- New `seed-copy` target: copies HN seed files from server1 → server2 via SCP+SSH
- `remote-hn-recrawl-swarm` updated workers: 300 → auto (remove hardcoded flag)

---

## Expected Throughput Model

### Server2 (workers=2730, innerN=12, fd=65536)

| Ceiling | Calc | Value |
|---------|------|-------|
| fd limit | 65536 ÷ 1 | 65,536 conns |
| active connections | 2730 × min(12, avg_urls_per_domain) | ~8,190 |
| throughput (400ms avg) | 8190 ÷ 0.4 | ~20,475 RPS theoretical |
| network ceiling (1 Gbps) | 128 MB/s ÷ 50 KB/page | ~2,560 pages/s |
| CPU ceiling (6 cores, 5ms/parse) | 6 × 1000 ÷ 5 | 1,200 parses/s/core × 6 = 7,200/s |
| **Expected** | network-limited | **2,500–3,500 OK/s** |

With HN data (25% fail rate without body): effective OK page transfer rate may be higher than network ceiling suggests.

### Server1 (workers=520, innerN=8, fd=65536)

| Ceiling | Value |
|---------|-------|
| fd limit (raised) | not limiting |
| active connections | 520 × 8 = 4,160 |
| throughput (400ms avg) | 10,400 RPS theoretical |
| network ceiling | ~2,560 pages/s |
| **Expected** | **1,500–2,500 OK/s** (was 1,860 before fix) |

---

## Benchmark Results

_Updated after each run._

### Post-0617 Benchmarks

| Engine | Server | Workers | InnerN | Seeds | Avg RPS | **OK/s** | OK% | Duration | GOMEMLIMIT |
|--------|--------|---------|--------|-------|---------|----------|-----|---------|-----------|
| keepalive | server2 | 2,730 (auto) | 12 (auto) | TBD | TBD | **TBD** | TBD | TBD | auto |
| keepalive | server1 | 520 (auto) | 8 (auto) | TBD | TBD | TBD | TBD | TBD | auto |
| keepalive | server2 | 2,730 (auto) | 12 (auto) | TBD | TBD | TBD | TBD | TBD | auto |

_Goal: server2 ≥ 3,000 avg OK/s._

---

## Files Changed

| File | Change |
|------|--------|
| `pkg/crawl/sysinfo.go` | New: `SysInfo` struct, `GatherSysInfo`, `LoadOrGatherSysInfo`, cache I/O, `Table()` |
| `pkg/crawl/sysinfo_linux.go` | New: Linux /proc/meminfo, /proc/version, fd via syscall |
| `pkg/crawl/sysinfo_other.go` | New: non-Linux stub |
| `pkg/crawl/autoconfig.go` | New: `AutoConfigKeepAlive` formula |
| `pkg/crawl/keepalive.go` | Add `raiseRlimit(65536)` at top of `Run()` |
| `cli/hn.go` | `--workers`/`--max-conns-per-domain` default → -1 (auto); inject sysinfo + GOMEMLIMIT |
| `Makefile` | Add `seed-copy` target; fix `remote-hn-recrawl-swarm` defaults |
