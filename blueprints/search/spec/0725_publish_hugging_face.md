# 0725 — Publish Open Index to Hugging Face

**Date:** 2026-03-13
**Status:** In progress — 85/100,000 shards committed (58.8/h)
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
| Overall HTML → Parquet | **98.9%** | |

CC-MAIN-2026-08 structure: **100 segments × 1,000 WARC files each** = 100,000 total.
File indices 0–99,999 map directly into `warc.paths.gz`.

---

## Baseline (as of 2026-03-13 08:30 UTC)

| Metric | Value |
|---|---|
| Shards committed | **85** |
| Sessions running | 7 (server2, files 0–999) |
| Throughput | **58.8 shards/h** (1,411/day) |
| ETA: current sessions (0–999) | **~15.6h** |
| ETA: all 100k at this rate | **~70.8 days** |

### Pipeline stage timings (avg, current binary)

| Stage | Avg | Notes |
|---|---|---|
| Download WARC from S3 | ~183s | combined dl+pack in old binary; ~74s pure dl in new |
| Pack HTML → Markdown | ~74s | combined in old format |
| Export Parquet | ~33s | |
| Publish to HF | **44s** | dominant bottleneck after download |
| **Total (serial)** | **~334s** | old binary; new binary separates dl/conv |

> Note: old binary lumped download+convert timing. New binary (from this session) separates them accurately as `dur_download_s` + `dur_convert_s`.

---

## Bottleneck Analysis

### #1 — Download (74s real, 50% of wall time)
- 7 sessions share server2 bandwidth (~35 MB/s total, ~5 MB/s per session)
- 800 MB WARC / 5 MB/s = ~160s — current 74s avg suggests not all sessions download simultaneously
- **Fix**: More servers with dedicated bandwidth per segment

### #2 — HF Publish (44s, 30% of wall time)
- LFS upload of ~28 MB at only ~0.64 MB/s
- Each commit has handshake + retry overhead (fixed cost per commit)
- **Fix: `--commit-batch N`** — batch N parquets per HF commit, amortize fixed overhead
  - Batch-10: ~80s per 10 shards → 8s amortized = **5× speedup on publish step**
  - Estimated wall-time gain: 44s → 8s per shard = **~22% faster overall**

### #3 — Pack/Export (107s combined, ~32% of wall time)
- Light engine at ~290 pages/s; export I/O-bound
- Could overlap with download of next shard (prefetch)
- **Fix: prefetch download** — download N+1 while packing N
  - Effective per-shard: max(74, 107) + 8 = 115s vs current 334s → **2.9× speedup**

---

## Time Estimates

| Scenario | Per shard | Shards/h (×7) | ETA 100k |
|---|---|---|---|
| **Baseline** (current) | ~334s | 75/h | 70.8 days |
| + batch commits (×10) | ~298s | 85/h | 62 days |
| + batch + prefetch | ~115s | 219/h | 19 days |
| 2 servers + batch + prefetch | ~115s | 438/h | 9.5 days |
| **<1 day target** | ~82s | 4,167/h | requires **14 servers** |

---

## 90-Iteration Improvement Strategy (files 1001–10,000)

Test 90 batches of 100 files each with the new binary. Use each batch to measure improvement, tune parameters, and update this spec.

### Iteration structure
```
Files 1001–10,000 = 9,000 files
Batch size: 100 files per iteration
Iterations: 90 total
```

### Iteration schedule

| Iterations | Files | Focus |
|---|---|---|
| 1–9 | 1001–1900 | Measure batch commit speedup (--commit-batch 1, 5, 10, 20) |
| 10–18 | 1901–2800 | Tune prefetch / session count |
| 19–27 | 2801–3700 | Optimize pack workers / light engine |
| 28–36 | 3701–4600 | Measure network saturation ceiling |
| 37–45 | 4601–5500 | Multi-server coordination (add server1) |
| 46–54 | 5501–6400 | Segment-based sharding |
| 55–63 | 6401–7300 | HF commit batching at 50 |
| 64–72 | 7301–8200 | Adaptive batch size (by parquet size) |
| 73–81 | 8201–9100 | Final tuning |
| 82–90 | 9101–10,000 | Stable production configuration |

### Iteration 1 — batch commit baseline
```bash
screen -S iter01
export HF_TOKEN=...
search cc publish --pipeline --cleanup --commit-batch 1 --file 1001-1100
# measure: shards/h, publish time avg
```

### Iteration 2 — batch-10
```bash
search cc publish --pipeline --cleanup --commit-batch 10 --file 1101-1200
```

### Iteration 3 — batch-20
```bash
search cc publish --pipeline --cleanup --commit-batch 20 --file 1201-1300
```

---

## Optimization Roadmap

### ✅ Implemented
- `--commit-batch N`: batch N parquets per HF commit (merged 2026-03-13)
- Charts: `totals_chart.png` (HTML vs MD vs Parquet total with % labels)
- Charts: `size_chart.png` renamed to "HTML vs Markdown" per shard
- `search cc pull`: reverse pipeline (HF → local parquet → md.warc.gz)
- Aggressive cleanup: raw WARC + md.warc.gz + local parquet all deleted after commit

### 🔴 Next: Prefetch download
While session packs shard N, download shard N+1 in background goroutine.
- Overlap 74s download with 107s pack+export
- Effective per-shard: max(74, 107) + pub = no wait for download
- Implementation: goroutine in `ccRunPipelineWithCommits` that pre-downloads

### 🟡 More sessions per server
- Current: 7 sessions using 35 MB/s total (~5 MB/s each)
- With prefetch, download is hidden → can increase to 10–14 sessions
- Monitor `sar -n DEV 1` to watch bandwidth ceiling

### 🔵 Multi-server segmentation
- Divide 100 CC segments across servers
- Segment 0 (files 0–999): server2 (in progress)
- Segments 1–49 (files 1000–49999): server2 second pass
- Segments 50–99 (files 50000–99999): server1 or additional servers

---

## Current Sessions (server2, 2026-03-13)

| Session | File range | Status |
|---|---|---|
| s37_100 | 0037–0099 | running |
| s101_250 | 0101–0249 | running |
| s251_400 | 0251–0399 | running |
| s401_550 | 0401–0549 | running |
| s551_700 | 0551–0699 | running |
| s701_850 | 0701–0849 | running |
| s851_1000 | 0851–0999 | running |

After these finish (~15.6h): files 0–36 still need to be run (0–36).
Then iteration sessions take over for 1001–10,000.
