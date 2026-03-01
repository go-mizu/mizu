# Rust Crawler: Road to 5,000 avg RPS — Bin Writer Focus

> **Date:** 2026-03-01
> **Baseline (post-v0.6.0):** 1,684 avg RPS, 6,067 peak (200K seeds, binary writer, --no-retry, server2)

## Benchmark Results (post-v0.6.0, server2)

| Writer   | Avg RPS | Peak RPS | Workers | Duration | Drain    |
|----------|---------|----------|---------|----------|----------|
| devnull  | 1,708   | 5,660    | 3,004   | 1m57s    | —        |
| **binary** | **1,684** | **6,067** | **3,004** | **1m58s** | **96.3s** |

**Critical finding:** Binary writer adds **zero overhead** vs devnull. The single-threaded bincode flusher + crossbeam channel keeps up with 6,067 peak RPS without any back-pressure. The crawl phase bottleneck is entirely the engine, not the writer.

---

## 1. Root Cause Analysis

### 1.1 Domain-Batch Architecture (Primary Bottleneck — 3.6× avg/peak gap)

**The gap:** avg=1,684 / peak=6,067 = 27.7%. Workers are active only 28% of the time.

**Little's Law quantification:**
- avg_RPS × avg_latency = effective_concurrency
- avg_latency = 88% × 50ms (fast-fail) + 12% × 300ms (ok) = **80ms**
- effective_concurrency = 1,684 × 0.080 = **134 concurrent** out of 3,004 workers
- Idle workers = 3,004 − 134 = **2,870 (95.5% idle)**

**Root cause:** Workers pick up a `DomainBatch` (all URLs for one domain), spawn inner_n=4 fetch tasks, wait for ALL inner tasks to complete, then pick the next domain. Between domain batches, workers are idle. Single-URL domains (70% of HN) release workers after one timeout (~1s), then the worker waits for the next batch from the channel.

**The fix:** Flat URL task queue with per-domain semaphores. Workers process one URL at a time and immediately pick up the next URL when done. Per-domain `tokio::sync::Semaphore` (capacity=inner_n) limits concurrency per domain without blocking the worker.

**Expected improvement:** avg approaches peak → 5,000–5,500 avg RPS (+3–3.3×).

### 1.2 Drain Phase (Secondary — 96.3s for 200K records)

The drain after crawl is single-threaded and sequential:
- Reads one segment file into memory (~200K records for 200K seeds at 0% ok, 0 body)
- Iterates records, routes to shard batches by URL hash
- On each full batch (5,000 records): DuckDB INSERT
- Sequence: 40 DuckDB INSERT batches × ~2.4s/batch = 96s

**Bottleneck:** DuckDB INSERT batches are slow (~2.4s/5K rows for WAL + checkpoint overhead) and run sequentially. With 8 shards, parallel insertion gives theoretical 8× speedup.

**The fix:** Rayon-parallel drain — read all segments, partition by shard, then insert into all 8 shards simultaneously. Expected: 96s → ~14s.

### 1.3 Workers Under-Count (Contributing — workers=3,004 vs optimal 8,192)

`auto_config` uses `available_mb` (snapshot at startup, variable) and the old inner_n multiplication formula designed for domain-batch architecture (which buffers inner_n bodies simultaneously). With flat URL queue, each worker holds one body at a time:
- Old formula: `avail_kb * 80% / (inner_n * body_kb)` = 10GB × 80% / (4 × 256KB) = 8,192
- Current result: workers=3,004 (available_mb was ~3.8GB at startup)

**Fix:** Use `mem_total_mb` (stable, not variable) and update formula for flat queue: `total_kb * 75% / body_kb`. For server2 (12GB): 12 × 1024² × 75% / 256 = 37,748 → capped at 16,000.

More workers = more concurrent fetches = higher peak, not just higher avg. With flat queue and 8,192 workers, peak could reach 8,000–10,000 RPS.

### 1.4 Binary Writer write() Mutex (Negligible)

The `write()` method holds a `Mutex<Option<Sender>>` on every call, blocking ALL workers while the channel send completes. At 1,684 RPS × ~100ns/send = 0.017% mutex occupancy. **Not a bottleneck** — confirmed by binary ≈ devnull benchmark. Not worth refactoring.

---

## 2. Implementation Plan

### Priority 1: Flat URL Task Queue (HIGH IMPACT — 3× avg RPS)

**Architecture change:** Replace domain-batch workers with flat URL workers.

```
Current:  Seeds → group_by_domain → DomainBatch channel → N workers
          (worker: pop batch, spawn inner_n tasks, wait for ALL)

New:      Seeds → group_by_domain → DomainEntry map → flat URL channel → N workers
          (worker: pop 1 URL, acquire domain semaphore, fetch, release, loop)
```

**Key components:**
- `DomainEntry`: per-domain state with `tokio::sync::Semaphore` (capacity=inner_n), abandoned flag, ok/timeout counters
- `Arc<DashMap<String, Arc<DomainEntry>>>`: lock-free concurrent domain state map
- Flat `async_channel::bounded<(SeedURL, Arc<DomainEntry>)>`: URL queue (cap = workers×4)
- Workers: `while let Ok((url, entry)) = rx.recv() { process_one_url(...).await }`
- No domain timeout needed (workers never "stuck" on a domain batch)

**New dep:** `dashmap = "6"` in `crawler-lib/Cargo.toml`

### Priority 2: Parallel Drain (MEDIUM IMPACT — 7× drain speedup)

Replace sequential segment→shard→DuckDB drain with parallel-by-shard drain using rayon.

**Architecture:**
```
Current: read seg → route to shards → INSERT shard[0] → INSERT shard[1] → ... → INSERT shard[7]

New:     read segs (sequential, I/O) → partition by shard → rayon::par_iter → INSERT all shards ∥
```

Each rayon thread opens its own DuckDB connection (not Sync) and inserts all records for its shard. Sequential read ensures I/O doesn't saturate; parallel insert saturates all CPU cores.

**New dep:** `rayon = "1"` in `crawler-lib/Cargo.toml`

### Priority 3: Workers Auto-Config for Flat Queue

Update `auto_config()` to use `mem_total_mb` (stable) and flat-queue formula (1 body/worker):
- `workers = clamp(total_kb * 75% / body_kb, 200, 16_000)`
- For server2 (12GB total): 12×1024²×75%/256 = 37,748 → capped to 16,000

---

## 3. Benchmark Targets

| Scenario            | Before  | Target  | Notes                           |
|---------------------|---------|---------|----------------------------------|
| 200K seeds, binary  | 1,684   | 5,000+  | Phase 1 only                    |
| Drain 200K records  | 96.3s   | ~14s    | Phase 2 parallel drain          |
| Peak RPS            | 6,067   | 8,000+  | Phase 3 more workers            |

---

## 4. Key Lessons

- **Binary writer is free:** 1,684 avg RPS with binary ≈ 1,708 with devnull. No writer optimization needed for the crawl path.
- **Drain is the post-crawl bottleneck:** 96.3s for 200K records; DuckDB INSERT serial throughput ~2,076 records/s. Parallelism across shards is the fix.
- **Little's Law reveals waste:** 1,684 avg × 80ms latency = 134 effective concurrent. 2,870 of 3,004 workers are idle at any moment. Flat URL queue eliminates idle gaps.
- **Peak already 6,067 RPS:** We don't need to increase peak throughput. We need avg to sustain at peak levels. The flat queue achieves this by keeping workers always busy.
- **Domain timeout not needed with flat queue:** Per-request timeout (1s) + dead_probe=3 + stall_ratio=5 provide fast abandonment without a separate domain-level timeout.
- **workers=3,004 is enough:** At avg_latency=80ms, we need 5,000×0.080=400 concurrent. Even 3,004 workers provide enough headroom. Increasing to 8,192 helps peak but not avg (both exceed the 400 needed).
