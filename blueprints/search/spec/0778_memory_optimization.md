# 0778: CC Pipeline Memory Optimization

**Goal:** Achieve 100 shards/hour by profiling real memory usage per session, eliminating waste, and rescheduling sessions based on actual (not estimated) memory budgets.

**Current state:** ~25 shards/hour with 4 sessions at 1.0 GB/session budget on a 12 GB / 6-core server.
**Target:** 100 shards/hour — requires either 4x throughput per session, 4x more sessions, or a combination.

---

## 1. Pipeline Architecture (per session)

Each pipeline session runs inside a `screen` session and executes:

```
Download (.warc.gz from CC S3)
  → ScanGzipOffsets (sequential, decompresses to find boundaries)
  → RunPackParallel (N workers × per-offset file descriptors)
      → processOneOffset: gzip decompress → WARC parse → HTML extract → ConvertLight
      → packWriteFile: sequential writer with 1 MB buffer, per-record gzip
  → exportWARCMdShardToParquet (sequential: read md.warc.gz → parquet with zstd)
  → cleanup raw WARC
```

Pipeline has prefetching: pack(N+1) runs in background while export(N) runs.

## 2. Memory Anatomy Per Session

### 2.1 Current memory model (estimated, pre-profiling)

| Component | Estimated MB | Notes |
|-----------|-------------|-------|
| Go runtime baseline | 15 | Go heap, stacks, GC metadata |
| gzip.NewReader (klauspost) | 1 per worker | Decompression window ~32 KB |
| WARC body accumulation | 0.5 per worker | io.ReadAll, max 512 KB + 8 KB |
| HTML DOM tree (x/net/html) | 2-10 per worker | Proportional to HTML complexity |
| ConvertLight DOM walking | 1-3 per worker | strings.Builder + DOM traversal |
| packResult strings in channel | 5-20 total | Workers×2 buffer, each ~10 KB markdown |
| Parquet writer (export phase) | 30-50 | 8 MB page buffer + zstd compressor |
| bufio write buffer | 1 | 1 MB for md.warc.gz output |
| File descriptors (processOneOffset) | 0.1 per worker | One fd per gzip member |
| GOGC=400 headroom | 4× live heap | GC runs at 4× live → large Sys |

### 2.2 Key unknowns (to be measured)

1. **Actual RSS per session** — `runtime.MemStats.Sys` ≠ OS RSS; need `/proc/self/status` VmRSS
2. **Peak vs steady-state** — does the peak during ScanGzipOffsets differ from parallel pack?
3. **GC overhead from GOGC=400** — 4× live heap means huge Sys; is this the bottleneck?
4. **Workers=NumCPU×4** — with 6 cores, that's 24 workers each holding ~10-15 MB DOM+HTML = 240-360 MB
5. **Parquet export vs pack overlap** — prefetch means both run simultaneously

---

## 3. Profiling Plan

### 3.1 Add pprof HTTP endpoint to pipeline sessions

Add `net/http/pprof` endpoint to pipeline sessions (port = 6060 + session_index). This allows live profiling with `go tool pprof http://localhost:606X/debug/pprof/heap`.

**File:** `cli/cc_publish.go` in `ccRunPipeline()`

### 3.2 Track real RSS (not Go Sys)

`runtime.MemStats.Sys` includes memory mapped but not yet used. Real RSS comes from:
- Linux: `/proc/self/status` → `VmRSS`
- macOS: `mach_task_basic_info`

Add `readRSSMB()` function that returns actual resident set size.

### 3.3 Report memory in pipeline output

Add RSS tracking at each pipeline stage:
- Before download
- After download (raw WARC in disk cache)
- During ScanGzipOffsets
- During RunPackParallel (peak)
- During export
- After cleanup

### 3.4 Emit per-session memory stats to stats.csv

Add columns: `peak_rss_mb`, `avg_rss_mb` so the scheduler can use real data for budgeting.

---

## 4. Optimization Attempts

### 4.1 Reduce worker count ✅ IMPLEMENTED

**Hypothesis:** `Workers = NumCPU × 4 = 24` workers each hold ~10-15 MB of DOM+HTML in flight. That's 240-360 MB per session just for in-flight conversion buffers.

**Change:** Cap workers at `min(NumCPU×4, 8)` for pack. ConvertLight is fast enough that 8 workers saturate I/O.

**Expected impact:** -200 MB per session → can run 2 more sessions → +50% throughput.

### 4.2 Pool gzip readers in processOneOffset ✅ IMPLEMENTED

**Current:** Each offset opens a new file + gzip.NewReader. The gzip reader allocates internal buffers.

**Change:** Use `sync.Pool` for `gzip.Reader` (call `gz.Reset(f)` instead of `gzip.NewReader(f)`).

**Expected impact:** Reduces GC pressure from short-lived gzip readers. Minor memory savings but significant GC time reduction.

### 4.3 Lower GOGC from 400 to 200 ✅ IMPLEMENTED

**Hypothesis:** GOGC=400 means GC triggers at 4× live heap. With 100 MB live heap, Go allocates up to 400 MB before GC. This multiplied by 4 sessions = 1.6 GB wasted on GC headroom.

**Change:** GOGC=200 (2× live heap). More frequent GC but 50% less peak memory.

**Expected impact:** -100-200 MB per session. More GC pauses but pack is I/O-bound so GC pauses are hidden.

### 4.4 Stream HTML body instead of io.ReadAll ✅ IMPLEMENTED

**Current:** `io.ReadAll(bodyReader)` allocates a fresh []byte for every WARC record (up to 520 KB).

**Change:** Use a `sync.Pool` of `bytes.Buffer` with `ReadFrom`. The buffer grows once and is reused across records.

**Expected impact:** Eliminates ~50,000 allocations per shard (one per WARC record). Major GC pressure reduction.

### 4.5 Reuse packItem.htmlBody via pool ✅ IMPLEMENTED

**Current:** `htmlBody` is a sub-slice of `bodyBytes` from io.ReadAll. After conversion, both are garbage.

**Change:** Pool the body buffer. After ConvertLight returns, return the buffer to the pool.

**Expected impact:** Combined with 4.4, eliminates nearly all large allocations in the hot path.

### 4.6 Pre-allocate strings.Builder in fastMarkdown ✅ IMPLEMENTED

**Current:** `b.Grow(4096)` — fixed 4 KB pre-allocation regardless of input size.

**Change:** `b.Grow(len(rawHTML) / 4)` — scale pre-allocation to ~25% of input size (typical markdown is 20-30% of HTML size).

**Expected impact:** Eliminates 2-3 reallocations per document in the Builder.

### 4.7 Reduce parquet PageBufferSize from 8 MB to 2 MB ✅ IMPLEMENTED

**Current:** `parquet.PageBufferSize(8*1024*1024)` — 8 MB per column per row group.

**Change:** Reduce to 2 MB. With 9 columns, that's 72 MB → 18 MB.

**Expected impact:** -54 MB per session during export phase.

### 4.8 Use GOMEMLIMIT for hard memory ceiling ⏳ PLANNED

**Current:** No hard memory limit. Go runtime can grow unbounded.

**Change:** Set `GOMEMLIMIT` per session based on budget. With `--ram-per-session 0.8`, set `GOMEMLIMIT=700MiB` (leaving 100 MB for OS overhead).

**Expected impact:** Go runtime aggressively GCs near the limit. Prevents OOM and makes memory usage predictable.

### 4.9 Sequential (non-parallel) pack mode for low-memory ⏳ PLANNED

**Current:** Always uses `RunPackParallel` with many workers.

**Change:** When `--ram-per-session < 0.5`, fall back to `RunPack` (sequential) with workers=2.

**Expected impact:** ~200 MB per session instead of 500+ MB. Slower per session but allows 2× more sessions.

### 4.10 Scheduler uses real RSS for budgeting ✅ IMPLEMENTED

**Current:** Budget is `available_ram / ram_per_session` where `ram_per_session` is a fixed estimate.

**Change:** Read actual RSS of running pipeline sessions via `/proc/{pid}/status`. Compute real per-session memory. Adjust budget dynamically.

**Expected impact:** Eliminates over/under-provisioning. Can safely pack more sessions when real usage is lower than estimated.

---

## 5. Benchmarks

### 5.1 Baseline (pre-optimization)

```
Server: 12 GB RAM, 6 cores (Ubuntu 24.04)
Sessions: 4 concurrent
RAM per session (estimated): 1.0 GB
Throughput: 25 shards/hour [csv]
GOGC: 400
Workers per session: 24 (NumCPU×4)
```

### 5.2 After each optimization

Results will be recorded here as each optimization is applied and measured:

| # | Optimization | Sessions | Peak RSS/session | Shards/hour | Notes |
|---|-------------|----------|-----------------|-------------|-------|
| 0 | Baseline | 4 | TBD (measure) | 25 | Pre-optimization |
| 1 | Cap workers at 8 | TBD | TBD | TBD | |
| 2 | Pool gzip readers | TBD | TBD | TBD | |
| 3 | GOGC 400→200 | TBD | TBD | TBD | |
| 4 | Pool body buffers | TBD | TBD | TBD | |
| 5 | Scale Builder.Grow | TBD | TBD | TBD | |
| 6 | Parquet 8→2 MB | TBD | TBD | TBD | |
| 7 | GOMEMLIMIT | TBD | TBD | TBD | |
| 8 | Real RSS budgeting | TBD | TBD | TBD | |

### 5.3 Theoretical maximum

With real RSS of ~300 MB per session (optimized):
- 12 GB server, 2 GB for OS+arctic = 10 GB usable
- 10 GB / 0.3 GB = 33 sessions (CPU-capped at 9 = 6 cores × 1.5)
- At 9 sessions × 12 shards/hour/session = 108 shards/hour ✅

With real RSS of ~500 MB per session:
- 10 GB / 0.5 GB = 20 sessions (CPU-capped at 9)
- At 9 sessions × 12 shards/hour/session = 108 shards/hour ✅

**Critical insight:** On this hardware, CPU (9 sessions cap) is the bottleneck, not RAM. Memory optimization allows packing up to 9 sessions, but per-session throughput (~12 shards/hour) is the real limit. To reach 100 shards/hour, we need 9+ sessions each doing 11+ shards/hour.

---

## 6. Implementation Order

1. **Add RSS tracking** — measure actual baseline before changing anything
2. **Cap workers at 8** — biggest memory win, lowest risk
3. **Pool body buffers** — biggest GC pressure reduction
4. **GOGC 400→200** — reduce peak memory
5. **Parquet buffer reduction** — easy win during export
6. **GOMEMLIMIT** — safety net
7. **Real RSS in scheduler** — smart budgeting
8. **Measure and iterate** — fill in benchmark table

---

## 7. Files Modified

| File | Changes |
|------|---------|
| `pkg/warc_md/pack.go` | Body buffer pool, worker cap |
| `pkg/warc_md/pack_parallel.go` | Gzip reader pool, worker cap, RSS tracking |
| `pkg/warc_md/types.go` | readRSSMB() function |
| `pkg/markdown/fastmd.go` | Scaled Builder.Grow |
| `cli/cc_warc_export.go` | Reduced PageBufferSize |
| `cli/cc_publish.go` | pprof endpoint, RSS stats in CSV |
| `cli/cc_publish_schedule.go` | Real RSS-based budgeting |
| `cli/cc_publish_pipeline.go` | RSS columns in stats.csv |

---

## 8. Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Lower GOGC increases GC pauses | Pack is I/O-bound; GC pauses hidden by disk/network wait |
| Buffer pools leak/grow | Pools have bounded size; buffers capped at MaxBodySize |
| Fewer workers reduces throughput | ConvertLight is ~1ms; 8 workers still far exceed I/O throughput |
| GOMEMLIMIT causes OOM-like behavior | Set 90% of budget; Go soft limit degrades gracefully |
| Real RSS sampling is approximate | Sample every 2s; use p95 not max for budgeting |
