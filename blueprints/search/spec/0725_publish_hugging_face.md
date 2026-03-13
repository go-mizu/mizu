# 0725 — Publish Open Index to Hugging Face

**Date:** 2026-03-13
**Status:** In progress — 66/100,000 shards committed
**Repo:** `open-index/cc-main-2026-08`
**Total scale:** CC-MAIN-2026-08 = **100,000 WARC files** × ~20k pages = ~2 billion pages

---

## Dataset Scale

| Metric | Per shard | × 100,000 shards |
|---|---|---|
| Raw WARC size | ~800 MB | ~80 TB |
| Rows (documents) | ~19,700 | ~2 billion |
| HTML bytes | ~2.5 GB | ~2.2 TB |
| Markdown (.md.warc.gz) | ~88 MB | ~8.2 TB |
| **Final parquet on HF** | **~28 MB** | **~2.6 TB** |
| HTML → Markdown compression | 96.5% | |
| Markdown → Parquet compression | 67.9% | |
| Overall compression | **98.9%** | |

CC structure: **100 segments × 1,000 WARC files each** = 100,000 total.
File indices 0–99,999 map directly into `warc.paths.gz`.

---

## Pipeline Stages (per shard)

```
download WARC from S3       ~74s   ← DOMINANT BOTTLENECK (network)
pack HTML → md.warc.gz      ~14s   ← CPU (light engine, ~290 pages/s)
export → parquet            ~18s   ← CPU + disk
commit to Hugging Face      ~42s   ← network (LFS upload 28 MB @ ~0.7 MB/s)
─────────────────────────────────
total per shard             ~148s  (2.5 min)
```

---

## Bottleneck Analysis

### #1 — Download (74s, 50% of time)
- Server2 downloads at ~35 MB/s total across all sessions
- Per session: ~5 MB/s when 7 sessions run concurrently
- Root cause: S3 throughput from this server + parallel session contention
- **Fix**: More servers (horizontal scale) or CDN/closer S3 region

### #2 — HF Publish (42s, 28%)
- LFS upload of 28 MB parquet: only ~0.7 MB/s → suspiciously slow
- Likely: HF rate limiting or LFS handshake overhead per commit
- **Fix: batch N parquets per commit** → amortize fixed overhead
  - Batch-10: ~80s for 10 parquets → 8s amortized = **5× speedup on publish**

### #3 — Pack + Export (32s, 22%)
- Light engine: fast enough, not a bottleneck
- Could overlap with download of next shard (prefetch)

---

## Time Estimates

### Current state (7 sessions, 1 server)
```
100,000 shards × 148s / 7 sessions = ~596 hours = 24.8 days
```

### With batch commits (N=10 per commit, amortized pub ≈ 8s)
```
per-shard: 74 + 14 + 18 + 8 = 114s
100,000 × 114s / 7 = ~453h = 18.9 days   (+24% faster)
```

### With batch + prefetch (overlap download with pack/export)
```
effective per-shard: max(74, 32) + 8 = 82s
100,000 × 82s / 7 = ~326h = 13.6 days   (+45% faster)
```

### To complete in <1 day (86,400s)
```
Need: 100,000 × 82s / 86,400 = ~95 parallel sessions
With 7 sessions per server: need ~14 servers
```

### Realistic 2-server target (server1 + server2)
```
2 servers × 7 sessions × optimized (82s): ~163 hours → 6.8 days
```

**Conclusion: 100k files in <1 day requires ~14 servers each running 7 sessions.**
Practical near-term target with 2 servers: complete in **~7 days**.
With 10+ servers: achievable in **<1 day**.

---

## Optimization Roadmap

### ✅ Done
- Aggressive cleanup (delete raw WARC + md.warc.gz + parquet after commit)
- Charts only on last shard (no per-shard commit conflicts)
- `search cc pull` for recovery from HF

### 🔴 High impact — Batch HF commits
**Goal**: commit N parquets in one HF operation instead of one-per-shard.
- `--commit-batch N` flag (default 1 = current behavior, recommend 10)
- Collect parquets, upload all as LFS in one commit
- Saves ~34s per shard amortized → biggest single code win
- **Estimated gain: 1.3–1.5× throughput**

### 🟡 Medium impact — Prefetch next WARC during pack/export
**Goal**: overlap 74s download with 32s pack+export of previous shard.
- Per-session goroutine: while pack(N) runs, download(N+1) in background
- Effective per-shard time: max(74, 14+18) + pub ≈ 82s
- **Estimated gain: 1.3× throughput** (limited by download being dominant)

### 🟡 Medium impact — More sessions per server
- Current: 7 sessions, each downloading sequentially
- Network at 35 MB/s total — adding sessions may not help unless sessions
  are in non-download phases (pack/export/publish) simultaneously
- Safe to try 10–12 sessions; monitor `sar -n DEV`

### 🟢 Low impact — Parallel pack workers within session
- Light engine: already fast (14s), not bottleneck

### 🔵 Architectural — Multi-server coordination
- Divide 100 segments across servers (1 segment = 1000 files per server)
- Server1: segments 0–49 (files 0–49999)
- Server2: segments 50–99 (files 50000–99999)
- Each server: 7 sessions × 50000/7 = ~7143 files
- With optimizations (82s): 50000 × 82s / 7 = ~165h = 6.9 days per server
- Need 7 more servers for <1 day

---

## Current Sessions (server2, 2026-03-13)

| Session | Files | Current shard | Status |
|---|---|---|---|
| s37_100 | 0037–0099 | 00042 | packing |
| s101_250 | 0101–0249 | ~00101 | running |
| s251_400 | 0251–0399 | 00255 | packing |
| s401_550 | 0401–0549 | running | |
| s551_700 | 0551–0699 | 00554 | exporting |
| s701_850 | 0701–0849 | 00705 | packing |
| s851_1000 | 0851–0999 | 00855 | packing |

Only segment 0 (files 0–999) is being processed. 99 segments remain untouched.

---

## Implementation Plan

### Step 1: Batch commits (code change)
Add `--commit-batch N` to `search cc publish --pipeline`:
- Buffer parquet paths after export
- When buffer reaches N (or last shard), do one HF commit with all N parquets
- README/stats.csv/charts included in final batch commit only

### Step 2: Test batch commits
```bash
search cc publish --pipeline --cleanup --file 1001-1020 --commit-batch 10
```

### Step 3: Expand to remaining segments on server2
Cover files 1000–99999 across multiple screen sessions per segment boundary.

### Step 4: Recruit more servers
Coordinate server1 + additional servers for segments 1–99.
