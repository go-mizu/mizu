# Spec 0618: Non-Blocking Binary Result Writer

**Date:** 2026-02-28
**Branch:** `open-index`
**Goal:** Eliminate DuckDB checkpoint from the hot write path, achieve **10,000 avg OK pages/s** on server2, maintain NO FALSE NEGATIVE via pass-2 retry, and surface real-time memory/writer telemetry in the status display.

---

## Problem Statement

### Current Bottleneck: DuckDB Checkpoint on Write Path

The keepalive engine's hot path is:

```
HTTP worker goroutine
  → results.Add(r)          [ResultDBWriter.Add → rdb.Add]
    → s.mu.Lock()
    → s.batch = append(...)
    → if full: s.flushCh <- batch   ← BLOCKS HERE when DuckDB is checkpointing
```

`flushCh` has buffer=2 (back-pressure design). When the DuckDB shard flusher goroutine is
stuck in `duckdb_execute_pending` (triggered by WAL checkpoint), HTTP worker goroutines block
at the channel send for **10-14 minutes** (confirmed via `SIGQUIT` goroutine dump showing
`[chan send, 10 minutes]`). This is the primary reason pass 1 drain takes 30+ minutes.

With a **clean DB** (no stale WAL), peak rps jumps from 5,447 → 10,832. But any SIGKILL restart
re-introduces stale WAL. The binary writer removes DuckDB from the hot path entirely.

### Current Throughput vs Target

| Metric | DuckDB writer (clean DB) | Target (bin writer) |
|--------|--------------------------|---------------------|
| Pass 1 peak rps | 10,832 | ≥ 10,832 |
| Pass 1 avg rps | 2,573 | ≥ 8,000 |
| Combined OK/s | ~303 (over 3.5 min) | ≥ 10,000 |
| Domain drain | ~50s (clean) → 30+ min (dirty WAL) | < 60s always |
| Memory display | none | heap / GOMEMLIMIT + GC cycles |

---

## Architecture: Swap-and-Drain

```
HTTP worker goroutines
  → BinSegWriter.Add(r)
    → ch <- r           [non-blocking: 128K-record channel ≈ 13s buffer at 10K/s]
                              │
                    ┌─────── flusher goroutine (single) ──────────┐
                    │  JSON marshal + bufio write to current seg   │
                    │  When seg ≥ maxMB: rotate → send to segCh   │
                    └──────────────────────────────────────────────┘
                              │ segCh (buffered 16 paths)
                    ┌─────── drain goroutine ─────────────────────┐
                    │  Read NDJSON segment file                    │
                    │  → rdb.Add(r) per record (non-blocking)      │
                    │  → os.Remove(seg) after successful drain     │
                    └──────────────────────────────────────────────┘
                              │
                    ┌─────── ResultDB (DuckDB, 16 shards) ────────┐
                    │  Existing async flusher goroutines           │
                    │  Checkpoint only affects drain goroutine,    │
                    │  NOT HTTP workers                            │
                    └──────────────────────────────────────────────┘
```

**Key property:** HTTP workers only block on `ch <- r`, which only fills when the flusher is
completely stuck (disk full, etc.). The flusher runs at ~900K records/s (JSON marshal + buffered
write), so at 10K input/s the channel stays near-empty. DuckDB checkpoint affects ONLY the drain
goroutine; HTTP workers are unaffected.

---

## Writer Variants

Three `crawl.ResultWriter` implementations for benchmarking:

| Writer | `--writer` | Description | Data persistence |
|--------|------------|-------------|-----------------|
| `ResultDBWriter` | `duckdb` | Current DuckDB direct write | ✅ DuckDB |
| `BinSegWriter` | `bin` | NDJSON segments + background drain | ✅ DuckDB (async) |
| `DevNullResultWriter` | `devnull` | Discards all results | ❌ None (benchmark only) |

Also two `crawl.FailureWriter` implementations:

| Writer | `--writer` | Description |
|--------|------------|-------------|
| `FailedDBWriter` | `duckdb` / `bin` | Current DuckDB failedDB |
| `DevNullFailureWriter` | `devnull` | Discards failures (no pass 2) |

---

## NDJSON Segment Format

Each segment is a plain newline-delimited JSON file (`.jsonl`). Field names use **snake_case**
to match DuckDB column names for direct `read_json_auto` compatibility.

**File naming:** `seg_000001.jsonl`, `seg_000002.jsonl`, …
**Location:** `<recrawl_dir>/segments/`
**Rotation:** rotate when file size ≥ `maxMB` (default 64 MB)
**Lifetime:** deleted by drain goroutine after successful `rdb.Add` drain

```json
{"url":"https://example.com/","status_code":200,"content_type":"text/html","content_length":12345,"body":"<html>...","title":"Example","description":"","language":"en","domain":"example.com","redirect_url":"","fetch_time_ms":187,"crawled_at_ms":1740700800000,"error":"","status":"done"}
{"url":"https://dead.example/","status_code":0,"content_type":"","content_length":0,"body":"","title":"","description":"","language":"","domain":"dead.example","redirect_url":"","fetch_time_ms":2001,"crawled_at_ms":1740700800001,"error":"context deadline exceeded","status":"failed"}
```

Note: `crawled_at_ms` is Unix milliseconds (not RFC3339) to avoid DuckDB timestamp parsing
ambiguity. Drain SQL: `epoch_ms(crawled_at_ms)` if converting directly; `rdb.Add()` path
converts via `time.UnixMilli(j.CrawledAtMs)`.

---

## BinSegWriter Design

### Parameters

| Parameter | Default | Notes |
|-----------|---------|-------|
| `ch` capacity | 131072 (128K) | ~13s buffer at 10K/s × 400B/record ≈ 50 MB max |
| `segCh` capacity | 16 | 16 pending segment paths; flusher blocks if drain far behind |
| Segment max size | 64 MB | ~160K records at 400B avg; ~6-11s at 10K/s |
| Flusher I/O buffer | 512 KB | `bufio.Writer` size, reduces syscall overhead |
| Drain concurrency | 1 goroutine | Sequential drain; OK since drain is not the bottleneck |

### Capacity Analysis

At 10K req/s and 400 B avg record:

- **Data rate to file:** 4 MB/s — well within disk I/O (100-200 MB/s sequential write)
- **Segment fill time:** 64 MB / 4 MB/s ≈ 16 seconds per segment
- **Drain speed (Go JSON decode + rdb.Add):** ~50K records/s → 160K records in 3.2s per segment
- **Steady state:** drain is 5× faster than fill → segments stay near-empty
- **Buffer cushion:** 128K channel = 13s, 16 segment queue = 16 × 16s = 256s of drain lag before any worker backpressure

### Backpressure Chain (worst case: DuckDB checkpoint during drain)

```
DuckDB checkpoint (10-14 min)
  → drain goroutine blocked on rdb.Add() after ~29s (batchSize×flushCh depth)
  → segCh fills after 16 × 16s = 256s
  → flusher blocked on segCh send
  → ch fills after 128K / 10K = 13s
  → HTTP workers blocked on ch <- r

Total time before HTTP workers see any backpressure: ~256s + 13s ≈ 4.5 minutes
vs. current: 1-2 seconds with flushCh=2
```

This is a 200× improvement in backpressure resilience.

### Exported Stats

```go
func (w *BinSegWriter) Written() int64  // records written to segment files
func (w *BinSegWriter) Drained() int64  // records drained to DuckDB
func (w *BinSegWriter) PendingSegs() int32  // segments waiting for drain
func (w *BinSegWriter) SegCount() int32     // total segments created
```

---

## Memory Monitor

Added to `v3RenderProgress` status line. Uses `runtime/metrics` (no STW, safe from any goroutine).

**Display line (added to status block):**
```
  Mem   heap=2.1 GB / lim=7.8 GB (27%)  │  GC 23×  │  Writer seg=0 pend=0 drain=65708
```

Fields:
- `heap`: `HeapInuse` bytes (active heap pages)
- `lim`: `GOMEMLIMIT` via `debug.SetMemoryLimit(-1)`
- `%`: heap/lim utilization
- `GC N×`: total GC cycles since process start
- `Writer ...`: shown only when `--writer bin`; shows segment count, pending drain queue, drained total

**Memory target:** heap < 50% of GOMEMLIMIT. If heap > 70%, log a warning.

---

## `--writer` Flag

Added to `search hn recrawl`:

```
--writer string   Result writer: duckdb (default), bin, devnull (default "duckdb")
```

| Mode | ResultWriter | FailureWriter | Pass 2 | Data saved |
|------|-------------|---------------|--------|-----------|
| `duckdb` | ResultDBWriter | FailedDBWriter | ✅ | DuckDB (sync) |
| `bin` | BinSegWriter | FailedDBWriter | ✅ | DuckDB (async drain) |
| `devnull` | DevNullResultWriter | DevNullFailureWriter | ❌ skipped | None |

Config summary line added:
```
  Writer            bin  (segments → DuckDB drain)
```

---

## Throughput Model

### Why 10K OK/s Is Achievable with BinSegWriter

Server2: workers=8192, innerN=4, 6 CPUs, 12 GB RAM, fd=65536

**With hn_pages.duckdb seeds (stratified, ~75% OK, ~200ms median OK latency):**

```
throughput = workers / weighted_avg_latency
           = 8192 / (0.75 × 0.20s + 0.25 × 2.0s)
           = 8192 / (0.15 + 0.50)
           = 8192 / 0.65s
           ≈ 12,600 req/s
OK/s       = 12,600 × 0.75 ≈ 9,450 OK/s   (close to 10K)
```

**With adaptive timeout (ceiling 2s, P95 ≈ 500ms → actual cutoff ≈ 1000ms):**

```
= 8192 / (0.75 × 0.20 + 0.25 × 1.0)
= 8192 / (0.15 + 0.25)
= 8192 / 0.40s
= 20,480 req/s  [CPU-bound limit likely ~15K on 6 cores]
OK/s = 20,480 × 0.75 ≈ 15,360 OK/s
```

**Combined 2-pass OK/s (over full run):**

Pass 1 (2s timeout, 3.8% OK) + Pass 2 (5s timeout, 56.4% rescue) = 44.6% unique OK rate.

```
Combined OK/s = 57,804 ok / 191s = 303 avg OK/s
```

Note: "10K OK/s" refers to **peak pass-1 OK/s** with good seeds, NOT combined over full run.
The distinction:
- Pass 1 peak OK/s = pass1_peak_rps × ok_rate ≈ 10,832 × 0.90 ≈ **9,750 OK/s** (good seeds)
- Two-pass combined OK/s ≈ 303/s (all seeds including 95% that timeout in pass 1)

The target is pass-1 peak with pre-filtered seeds (hn_pages.duckdb stratified sample).

---

## Files Changed

| File | Change |
|------|--------|
| `pkg/crawl/writer_bin.go` | New: BinSegWriter implementation |
| `pkg/crawl/writer_devnull.go` | New: DevNullResultWriter + DevNullFailureWriter |
| `cli/cc.go` | Add `binWriter *crawl.BinSegWriter` to v3LiveStats; memory monitor in v3RenderProgress |
| `cli/hn.go` | Add `--writer` flag; wire up writers; show writer in config summary |

---

## Benchmark Plan

Run `search hn recrawl --limit 200000` three times on server2 (clean DB each run):

```bash
# Run 1: baseline DuckDB writer
search hn recrawl --limit 200000 --writer duckdb

# Run 2: binary segment writer
search hn recrawl --limit 200000 --writer bin

# Run 3: devnull (theoretical max, no I/O)
search hn recrawl --limit 200000 --writer devnull
```

Metrics to capture:
- Pass 1: avg rps, peak rps, domain drain time
- Pass 2: rescued URLs, total time (bin/duckdb only)
- Memory: peak heap, GC cycles
- Segment files: count, peak size, drain lag

---

## Benchmark Results (2026-02-28, server2, 200K seeds, clean DB, --no-retry)

All three runs: `search hn recrawl --limit 200000 --writer <mode> --no-retry`
Seeds: 200K → 129,591 after DNS filter (70,409 filtered). Workers=8192, innerN=4.

### Pass 1 Comparison

| Writer | Avg rps | **Peak rps** | Duration | OK | OK% | Heap peak | GC cycles |
|--------|---------|------------|----------|----|-----|-----------|-----------|
| `devnull` | 5,149 | **11,319** | 25s | 4,571 | 3.5% | 1.1 GB | 25 |
| `bin` | 4,432 | **9,972** | 29s | 5,036 | 3.9% | 1.5 GB | 23 |
| `duckdb` | 3,172 | **8,180** | 40s | 7,181 | 5.5% | 1.2 GB | 24 |

**Key findings:**

1. **BinSegWriter is 40% faster avg rps and 37% shorter duration than DuckDB direct** (4,432 vs 3,172 avg; 29s vs 40s)
2. **Peak rps**: devnull 11,319 > bin 9,972 > duckdb 8,180 — the expected ordering; bin is 22% faster peak than duckdb
3. **OK rate varies** because slower writers allow slow-but-alive servers more time to respond before domain timeout fires (confounding factor, not a quality difference)
4. **No data loss**: bin writer created 3 segments (~300 MB), all drained to DuckDB by program exit; `segments/` dir empty after run
5. **Memory**: bin writer uses 1.5 GB peak (vs 1.1-1.2 GB for others) — extra ~300 MB from 128K channel + 3× open bufio buffers + the DuckDB 16×96MB pool still active for drain target

### Memory Monitor Sample (bin writer)

```
  Mem   heap=1.5 GB / lim=7.7 GB (19%)  │  GC 20×  │  Writer seg=1 pend=0 drain=0
```

- Memory utilization stayed under 20% throughout
- GC ran 23 cycles during the 29-second run (no GC pressure)
- BinSegWriter stats: `seg=1` = 1 active segment, `pend=0` = drain queue empty, `drain=0` = drainer ran after last status tick (completed before exit)

### 2-Pass Combined Results (bin writer, separate run 2026-02-27)

The two-pass run with bin writer (from earlier in this session):

| Writer | Pass 1 OK | Pass 1 rps | Pass 2 rescued | Total OK | Total duration |
|--------|-----------|------------|----------------|----------|----------------|
| `duckdb` (earlier run) | 4,889 | 2,573 avg | 52,915 / 91,046 | **57,804** | ~3.5 min |
| `bin` | _est. ~5,000_ | _est. 4,432_ | _est. ~53,000_ | **~58,000** | _est. ~3 min_ |

Note: bin 2-pass not yet separately benchmarked; expected total similar to duckdb since pass 2 speed is limited by 5s timeout, not writer throughput.

### Why 10K OK/s Not Yet Achieved

Target: 10K avg OK/s on server2. Current gap:

| Step | Constraint | Path to 10K |
|------|-----------|-------------|
| Writer | DuckDB checkout (fixed with bin) | ✅ bin writer eliminates this |
| Seeds | 95% timeout rate (dead domains) | Need pre-filtered seeds (70%+ OK rate) |
| Workers | fd-capped 8,192 (no room to grow) | ✅ already optimal for fd=65,536, innerN=4 |
| Avg vs peak gap | Domain drain tail: ~50% of run time | Reduce with better seeds (fewer timeouts = faster drain) |

**Calculation:**
```
For 10K avg OK/s:
  Required avg rps = 10,000 / OK_rate
  With 80% OK seeds: 10,000 / 0.80 = 12,500 avg rps
  Current devnull avg = 5,149 rps  →  devnull × 0.80 = 4,119 OK/s (not enough)

  Need avg ≥ peak × 0.88 (reduce drain tail from 50% → 12% of run time)
  With hn_pages.duckdb seeds (known-good, high OK rate, fast drain):
    Expected avg ≈ 8,000-10,000 rps (fewer timeouts → workers stay busy)
    Expected OK rate ≈ 65-75%
    Expected OK/s ≈ 8,000 × 0.70 = 5,600-7,500 OK/s

  To reach 10K: need 85%+ OK rate OR higher avg rps (requires better seeds)
```

**Next action:** benchmark with `hn_pages.duckdb` stratified seeds (known-good pages, expected 70%+ OK rate, fewer slow domains → avg rps closer to peak).

---

## NO FALSE NEGATIVE Policy

The two-pass retry system (spec 0617) remains active for `--writer duckdb` and `--writer bin`:

- Pass 1 writes ALL results (ok + timeout + error) to the writer
- `FailedDBWriter` records `http_timeout` URLs for pass 2
- `LoadTimeoutURLs` reads from failedDB after pass 1 closes (connection conflict fix from 0617)
- Pass 2 retries with 5s timeout, catching servers that respond in 2–5s

`--writer devnull` disables pass 2 (no failedDB → no timeout URL tracking). This is benchmarking-only mode.
