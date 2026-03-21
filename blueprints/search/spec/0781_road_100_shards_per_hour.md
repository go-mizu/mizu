# 0781: Road to 100 Shards/Hour (HF Committed)

**Goal:** Achieve 100+ shards/hour of committed (pushed to HuggingFace) throughput
on server2 (6-core, 12 GB RAM, Ubuntu 24.04).

**Current state:** ~25-40 shards/hour (as of March 2026).

---

## 1. Bottleneck Analysis

### 1.1 Pipeline Stages (per shard)

| Stage | Current Duration | Bottleneck |
|-------|-----------------|------------|
| Download (.warc.gz from CC S3) | 30-120s | Network bandwidth |
| ScanGzipOffsets | 10-30s | Sequential gzip decompress (CPU) |
| RunPackParallel (HTML→MD) | 20-60s | CPU (gzip + tokenizer) |
| Parquet write (zstd) | 5-15s | CPU (zstd compression) |
| **Total per shard** | **65-225s** | **~2 min avg** |

### 1.2 HF Commit Overhead (per batch)

| Stage | Duration |
|-------|----------|
| Stats merge from HF | 5-15s |
| README regeneration | 1-2s |
| Python/xet upload (20 shards × 30 MB) | 60-300s |
| **Total per commit** | **~2-5 min** |

### 1.3 Current Throughput Math

```
6 concurrent sessions × 1 shard/2min = 180 shards/hour (pack rate)
But: commit rate = pack rate × (commit efficiency)
     commit efficiency ≈ 0.7 (rate limiting, retries, merge overhead)
     → 180 × 0.7 = 126 shards/hour theoretical max

Actual: ~25-40/hr because:
1. Sessions often downloading (idle CPU) — stagger helps
2. Rate limiting: 120s commit interval → 30 commits/hr × 20 batch = 600 max
3. Stalled sessions from OOM/errors consume slots
4. Download overlaps poorly with pack CPU
```

### 1.4 Key Insight

The bottleneck is NOT commit rate — it's **pack throughput per session**.
With 6 cores and proper pipeline overlap:
- Each session needs 1 core for packing (CPU-bound)
- Download can overlap with pack (network-bound)
- 6 sessions × 30 shards/hr/session = 180 shards/hr pack → 100+ commit

---

## 2. Optimization Plan

### 2.1 Eliminate ScanGzipOffsets (HIGH impact)

**Problem:** Every shard spends 10-30s doing a sequential gzip decompress scan
just to find gzip member boundaries. This is pure overhead.

**Solution:** Use Common Crawl's CDX index to get byte offsets directly. CC
provides `cc-index.paths.gz` with offset + length for every WARC record.
Skip ScanGzipOffsets entirely.

**Expected gain:** -15s/shard average → +25% per-session throughput.

**Implementation:**
- Parse CC CDX index for the shard's offset table
- Feed offsets directly to `RunPackParallel`
- Fall back to `ScanGzipOffsets` if CDX unavailable

### 2.2 Smarter Download Prefetching (MEDIUM impact)

**Problem:** Current prefetch downloads 1 shard ahead. When pack is fast,
the pipeline waits for download.

**Solution:** Download 2 shards ahead (double-buffered prefetch). While
packing shard N, download N+1 and N+2 concurrently.

**Expected gain:** Eliminates download wait in 80%+ of cases.

**Implementation:**
- Change `startPrefetchDownload` to support 2 concurrent downloads
- Throttle to avoid network contention with other sessions
- Track download bandwidth and adapt prefetch depth

### 2.3 Optimize ConvertUltraLight (MEDIUM impact)

**Problem:** HTML→Markdown tokenizer processes every byte sequentially.
Large HTML documents (>200 KB) take 50-100ms each.

**Solution:** Profile and optimize hot paths:
- Pre-allocate output buffer based on input size
- Skip known non-content tags earlier (script, style, svg)
- Use `[]byte` operations instead of string conversions where possible
- Consider SIMD-friendly byte scanning for tag boundaries

**Expected gain:** 10-20% faster pack → +5-10% per-session throughput.

### 2.4 Batch Parquet Writes (LOW impact)

**Problem:** Each shard creates a separate parquet file. The parquet writer
has per-file overhead (metadata, zstd dictionary init).

**Current approach is correct** — one file per shard allows independent commit
and watcher cleanup. No change needed.

### 2.5 Reduce Commit Interval (HIGH impact on commit rate)

**Problem:** 120s commit interval → max 30 commits/hr. With 20-shard batches,
that's 600 shards/hr capacity. But watcher also needs time for upload.

**Solution:** Reduce commit interval based on actual upload speed:
- Fast upload (xet warm cache): 30s for 20 shards → use 60s interval
- Slow upload (cold cache): 300s for 20 shards → keep 120s interval
- Adaptive: watcher tracks upload duration and adjusts interval

**Expected gain:** 30→60 commits/hr when upload is fast, doubling commit capacity.

**Implementation:**
- Track `lastUploadDuration` in watcher
- If `lastUploadDuration < commitInterval/2`, reduce interval by 25%
- If `lastUploadDuration > commitInterval`, increase interval by 25%
- Min interval: 45s (safety margin for HF 128/hr limit with 2 servers)

### 2.6 Larger Batch Sizes (MEDIUM impact on commit rate)

**Problem:** Max batch = 20 shards per commit. Each commit has fixed overhead
(stats merge, README regen, xet handshake). More shards/commit = less overhead.

**Solution:** When upload is fast and many shards pending:
- Increase batch to 40 shards when >30 pending
- Decrease to 10 when upload was slow (>5 min)

**Expected gain:** 30-50% reduction in per-shard commit overhead.

### 2.7 Parallel Offset Processing Tuning (MEDIUM impact)

**Problem:** `Workers = NumCPU×4 = 24` but capped at 8 (from 0778). On 6 cores,
8 workers may leave CPU underutilized during I/O waits.

**Solution:** Profile actual CPU utilization during pack:
- If load average < cores × 0.8, increase workers to 12
- If load average > cores × 2, decrease workers to 6
- Each pipeline session adapts independently via Redis hardware state

**Expected gain:** 10-30% better CPU utilization during pack phase.

### 2.8 Session Overlap: Download During Export (LOW impact)

**Problem:** Export phase writes parquet (CPU: zstd). Download phase reads
network (I/O). These don't compete but currently run sequentially within a session.

**Current implementation already overlaps** via prefetch. No change needed.

---

## 3. Implementation Roadmap

### Phase 1: Measure (Day 1)
- [ ] Deploy Redis monitoring (spec 0780)
- [ ] Instrument per-stage timing in Redis sorted sets
- [ ] Run for 1 hour, collect baseline data from Redis Insight
- [ ] Identify actual bottleneck distribution (download vs pack vs commit)

### Phase 2: Quick Wins (Day 2-3)
- [ ] Reduce commit interval to 60s (when upload < 30s)
- [ ] Increase batch size to 40 (when >30 pending)
- [ ] Double-buffered download prefetch

### Phase 3: CDX Index Integration (Day 4-5)
- [ ] Parse CC CDX index for byte offsets
- [ ] Skip ScanGzipOffsets when CDX available
- [ ] Benchmark: expect 15-20s savings per shard

### Phase 4: ConvertUltraLight Optimization (Day 6-7)
- [ ] Profile hot paths with pprof
- [ ] Optimize tag scanning and buffer management
- [ ] Benchmark: expect 10-20% pack speed improvement

### Phase 5: Adaptive Tuning (Ongoing)
- [ ] Worker count adapts to load average (from Redis hw state)
- [ ] Commit interval adapts to upload speed
- [ ] Prefetch depth adapts to download bandwidth

---

## 4. Target Throughput Model

After optimizations:

```
Pack rate per session:
  Download:    30s (overlapped with previous shard's export)
  GzipOffset:  0s  (CDX index, skipped)
  PackParallel: 30s (optimized tokenizer, 12 workers)
  ParquetWrite: 10s
  Total visible: ~40s/shard (download overlapped) = 90 shards/hr/session

With 6 sessions (CPU-limited):
  Pack rate:   6 × 90 = 540 shards/hr

Commit rate (watcher):
  Interval:    60s (adaptive)
  Batch size:  30 (adaptive)
  Commits/hr:  60
  Shards/hr:   60 × 30 = 1,800 capacity >> 540 pack rate

Bottleneck: pack rate = 540 shards/hr >> 100 target ✅

Conservative estimate (50% efficiency loss):
  540 × 0.5 = 270 shards/hr committed >> 100 target ✅
```

Even with significant efficiency losses, 100 shards/hr is achievable with
6 optimized sessions on 6 cores.

---

## 5. Monitoring Dashboard (Redis Insight)

Key metrics to track in Redis Insight:

| Metric | Redis Key | Chart |
|--------|-----------|-------|
| Pack rate (shards/hr) | `cc:*:rate:packed` | Time series |
| Commit rate (shards/hr) | `cc:*:rate:committed` | Time series |
| Download rate | `cc:*:rate:downloaded` | Time series |
| Pending queue depth | `LLEN cc:*:watcher:pending` | Gauge |
| Active sessions | `SCARD cc:*:pipelines` | Gauge |
| Total committed | `SCARD cc:*:committed` | Counter |
| Per-session RSS | `cc:*:sessions:rss` | Sorted set |
| Load average | `cc:*:hw` → load_avg_1 | Time series |

---

## 6. Success Criteria

- [ ] Pack rate: 200+ shards/hr sustained
- [ ] Commit rate: 100+ shards/hr sustained (measured over 1 hour)
- [ ] No OOM kills during 24-hour run
- [ ] Redis memory usage < 100 MB
- [ ] All rates visible in Redis Insight real-time
- [ ] Graceful degradation: pipeline works without Redis
