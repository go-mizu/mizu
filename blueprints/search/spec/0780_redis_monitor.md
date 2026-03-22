# 0780: Redis-Backed State Management for CC Pipeline

**Goal:** Rewrite cc_watcher, cc_scheduler, and cc_pipeline to use Redis as the
single source of truth for all state, progress, and metrics — while gracefully
degrading to file-based state when Redis is unavailable.

---

## 1. Architecture Overview

```
┌─────────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│  cc_pipeline (×6)   │     │    cc_scheduler      │     │    cc_watcher        │
│  (screen sessions)  │     │  (long-running loop) │     │  (HF commit loop)   │
│                     │     │                      │     │                      │
│  Writes:            │     │  Reads:              │     │  Reads:              │
│  - shard progress   │     │  - all pipeline state│     │  - pending parquets  │
│  - download status  │     │  - rates/counters    │     │  Writes:             │
│  - pack status      │     │  - watcher status    │     │  - commit status     │
│  - peak RSS         │     │  Writes:             │     │  - committed set     │
│                     │     │  - session mgmt      │     │  - cleanup events    │
└────────┬────────────┘     └────────┬─────────────┘     └────────┬────────────┘
         │                           │                             │
         └───────────────────────────┼─────────────────────────────┘
                                     │
                              ┌──────▼──────┐
                              │    Redis     │
                              │  (6379)      │
                              │              │
                              │  Keys:       │
                              │  cc:*        │
                              └──────┬───────┘
                                     │
                              ┌──────▼──────┐
                              │ Redis Insight│
                              │  (5540)      │
                              └─────────────┘
```

---

## 2. Redis Key Schema

All keys use prefix `cc:{crawlID}:` for namespace isolation.

### 2.1 Committed Shards (replaces stats.csv for tracking)

```
cc:{crawl}:committed          SET of file indices (int strings)
                              SADD on each HF commit, SISMEMBER for lookups
                              Replaces: stats.csv scan + ccLoadCommittedSet()

cc:{crawl}:stats:{fileIdx}    HASH with per-shard stats
                              Fields: rows, html_bytes, md_bytes, parquet_bytes,
                                      created_at, dur_download_s, dur_convert_s,
                                      dur_export_s, dur_publish_s, peak_rss_mb
                              Replaces: per-row in stats.csv
```

### 2.2 Pipeline Session State (new — not possible with files)

```
cc:{crawl}:pipeline:{sessionID}    HASH with session state
                                   Fields: status (downloading|packing|exporting|idle|done),
                                           shard (current file index),
                                           started_at, progress (0-100),
                                           input_records, output_records, errors,
                                           read_bytes, write_bytes, peak_rss_mb,
                                           last_heartbeat
                                   TTL: 5 min (auto-expire dead sessions)
                                   Replaces: pgrep pattern matching

cc:{crawl}:pipelines              SET of active session IDs
                                  SADD on start, SREM on completion
                                  TTL per member via sorted set score = timestamp
```

### 2.3 Watcher State (replaces watcher_status.json)

```
cc:{crawl}:watcher                HASH with watcher state
                                  Fields: commit_number, message, commit_url,
                                          shards_in_commit, total_committed,
                                          timestamp, status (idle|uploading|rate_limited)
                                  Replaces: watcher_status.json

cc:{crawl}:watcher:pending        LIST of pending parquet paths (FIFO queue)
                                  RPUSH by pipeline, LPOP by watcher
                                  Replaces: os.ReadDir polling for .parquet files
```

### 2.4 Rate Tracking (replaces sliding window in scheduler memory)

```
cc:{crawl}:rate:packed            SORTED SET (score=unix timestamp, member=count)
                                  ZADD on each new parquet
                                  ZRANGEBYSCORE for window queries
                                  Replaces: rateSnapshot[] in scheduler

cc:{crawl}:rate:committed         SORTED SET (score=unix timestamp, member=count)
                                  ZADD on each HF commit
                                  Replaces: session-local commit counter

cc:{crawl}:rate:downloaded        SORTED SET (score=unix timestamp, member=count)
                                  ZADD on each download completion
                                  Replaces: countDownloading() file scan

cc:{crawl}:counters               HASH with atomic counters
                                  Fields: total_packed, total_committed,
                                          total_downloaded, total_errors,
                                          session_packed, session_committed
                                  HINCRBY for atomic increment
```

### 2.5 Log Stream (new — real-time log aggregation)

```
cc:{crawl}:log                    STREAM (Redis Streams)
                                  XADD by all components with structured fields:
                                    source (pipeline|watcher|scheduler),
                                    level (info|warn|error),
                                    message, shard, session
                                  XREAD by scheduler for aggregated view
                                  XTRIM MAXLEN ~1000 to bound memory
                                  Replaces: per-screen tail -f
```

### 2.6 Resource Tracking

```
cc:{crawl}:hw                     HASH with latest hardware snapshot
                                  Fields: cpu_cores, ram_total_gb, ram_avail_gb,
                                          disk_free_gb, load_avg_1, load_avg_5
                                  Updated each scheduler round

cc:{crawl}:sessions:rss           SORTED SET (score=rss_mb, member=session_id)
                                  Updated by each pipeline session
                                  Used by scheduler for memory-aware scaling
```

---

## 3. Code Changes

### 3.1 New File: `cli/cc_redis.go`

Central Redis connection and helper functions:

```go
// ccRedis wraps redis.Client with pipeline-specific helpers.
// All methods are no-ops when rdb is nil (graceful degradation).
type ccRedis struct {
    rdb    *redis.Client
    crawl  string
}

func newCCRedis(crawlID string) *ccRedis {
    return &ccRedis{rdb: ccRedisClient(), crawl: crawlID}
}

func (r *ccRedis) Available(ctx context.Context) bool
func (r *ccRedis) key(parts ...string) string  // "cc:{crawl}:parts..."

// Committed set
func (r *ccRedis) AddCommitted(ctx context.Context, fileIdx int) error
func (r *ccRedis) IsCommitted(ctx context.Context, fileIdx int) bool
func (r *ccRedis) CommittedCount(ctx context.Context) int
func (r *ccRedis) CommittedSet(ctx context.Context) map[int]bool

// Shard stats
func (r *ccRedis) SetShardStats(ctx context.Context, fileIdx int, stats ccShardStats) error
func (r *ccRedis) GetShardStats(ctx context.Context, fileIdx int) (ccShardStats, bool)
func (r *ccRedis) AllShardStats(ctx context.Context) []ccShardStats

// Pipeline session
func (r *ccRedis) RegisterPipeline(ctx context.Context, sessionID string) error
func (r *ccRedis) UpdatePipeline(ctx context.Context, sessionID string, fields map[string]interface{}) error
func (r *ccRedis) HeartbeatPipeline(ctx context.Context, sessionID string) error
func (r *ccRedis) UnregisterPipeline(ctx context.Context, sessionID string) error
func (r *ccRedis) ActivePipelines(ctx context.Context) []string

// Watcher
func (r *ccRedis) SetWatcherStatus(ctx context.Context, status ccWatcherStatus) error
func (r *ccRedis) GetWatcherStatus(ctx context.Context) (ccWatcherStatus, bool)
func (r *ccRedis) PushPendingParquet(ctx context.Context, path string) error
func (r *ccRedis) PopPendingParquets(ctx context.Context, max int) []string

// Rate tracking
func (r *ccRedis) RecordPacked(ctx context.Context) error     // ZADD + HINCRBY
func (r *ccRedis) RecordCommitted(ctx context.Context, n int) error
func (r *ccRedis) RecordDownloaded(ctx context.Context) error
func (r *ccRedis) PackRate(ctx context.Context, window time.Duration) float64   // shards/hour
func (r *ccRedis) CommitRate(ctx context.Context, window time.Duration) float64
func (r *ccRedis) DownloadRate(ctx context.Context, window time.Duration) float64
func (r *ccRedis) Counters(ctx context.Context) map[string]int64

// Logging
func (r *ccRedis) Log(ctx context.Context, source, level, msg string, fields ...string) error
func (r *ccRedis) ReadLogs(ctx context.Context, since string, count int) []ccLogEntry

// Hardware
func (r *ccRedis) SetHardware(ctx context.Context, hw arctic.HardwareProfile) error
func (r *ccRedis) SetSessionRSS(ctx context.Context, sessionID string, rssMB float64) error
```

### 3.2 Modified: `cli/cc_publish_pipeline.go` (ccRunPipeline)

```go
func ccRunPipeline(ctx context.Context, ...) error {
    rds := newCCRedis(crawlID)
    sessionID := fmt.Sprintf("pipe_%s_%d", hostname(), os.Getpid())

    if rds.Available(ctx) {
        rds.RegisterPipeline(ctx, sessionID)
        defer rds.UnregisterPipeline(ctx, sessionID)

        // Heartbeat goroutine: update last_heartbeat every 30s
        go func() { ... }()
    }

    for i, idx := range indices {
        // Check committed via Redis first, fall back to stats.csv
        if rds.Available(ctx) && rds.IsCommitted(ctx, idx) {
            continue
        }

        // Update pipeline state in Redis
        rds.UpdatePipeline(ctx, sessionID, map[string]interface{}{
            "status": "downloading", "shard": idx,
        })

        // ... download ...
        rds.RecordDownloaded(ctx)
        rds.UpdatePipeline(ctx, sessionID, map[string]interface{}{
            "status": "packing", "shard": idx,
        })

        // ... pack → parquet ...
        rds.RecordPacked(ctx)
        rds.PushPendingParquet(ctx, parquetPath) // notify watcher

        // Update RSS
        rds.SetSessionRSS(ctx, sessionID, readRSSMB())
        rds.UpdatePipeline(ctx, sessionID, map[string]interface{}{
            "status": "idle", "peak_rss_mb": readRSSMB(),
        })
    }
}
```

### 3.3 Modified: `cli/cc_publish_watcher.go` (ccRunWatcher)

```go
func ccRunWatcher(ctx context.Context, ...) error {
    rds := newCCRedis(crawlID)

    // Option A: if Redis available, use BRPOP on pending queue (event-driven)
    // Option B: fall back to os.ReadDir polling

    flush := func() {
        var newFiles []ccUncommittedParquet
        if rds.Available(ctx) {
            // Pop from Redis queue instead of scanning directory
            paths := rds.PopPendingParquets(ctx, maxBatchSize)
            for _, p := range paths { ... }
        } else {
            newFiles = ccFindUncommittedParquets(dataDir, crawlID, committed)
        }

        // ... commit to HF ...

        // After successful commit:
        for _, f := range uploadedFiles {
            rds.AddCommitted(ctx, f.fileIdx)
            rds.SetShardStats(ctx, f.fileIdx, stats)
        }
        rds.RecordCommitted(ctx, len(uploadedFiles))
        rds.SetWatcherStatus(ctx, ccWatcherStatus{...})
        rds.Log(ctx, "watcher", "info", commitMsg)
    }
}
```

### 3.4 Modified: `cli/cc_publish_schedule.go` (runCCSchedule)

```go
func runCCSchedule(ctx context.Context, cfg ccScheduleConfig) error {
    rds := newCCRedis(cfg.CrawlID)

    for {
        round++

        if rds.Available(ctx) {
            // Committed count from Redis (O(1) instead of reading stats.csv)
            totalCommitted = rds.CommittedCount(ctx)

            // Rates from Redis sorted sets (15-min window)
            packRate = rds.PackRate(ctx, 15*time.Minute)
            commitRate = rds.CommitRate(ctx, 15*time.Minute)
            downloadRate = rds.DownloadRate(ctx, 15*time.Minute)

            // Watcher status from Redis hash
            watcherStatus, _ = rds.GetWatcherStatus(ctx)

            // Pipeline sessions from Redis (no pgrep needed)
            activeSessions = rds.ActivePipelines(ctx)

            // Hardware snapshot
            rds.SetHardware(ctx, hw)

            // Pending count from Redis counter (no readdir)
            pending = int(rds.Counters(ctx)["total_packed"]) - totalCommitted
        } else {
            // Fall back to file-based state (existing code)
            ...
        }
    }
}
```

---

## 4. Graceful Degradation

**Critical design principle:** Redis is optional. All components MUST work
without Redis, falling back to the existing file-based state management.

```go
// Every Redis call follows this pattern:
func (r *ccRedis) AddCommitted(ctx context.Context, fileIdx int) error {
    if r.rdb == nil {
        return nil // no-op when Redis unavailable
    }
    return r.rdb.SAdd(ctx, r.key("committed"), fileIdx).Err()
}
```

Detection at startup:
```go
rds := newCCRedis(crawlID)
if rds.Available(ctx) {
    fmt.Printf("  Redis    %s\n", successStyle.Render("connected"))
    // Seed Redis from stats.csv if Redis is empty (first run)
    if rds.CommittedCount(ctx) == 0 {
        seedRedisFromCSV(ctx, rds, statsCSV, crawlID)
    }
} else {
    fmt.Printf("  Redis    %s (using file-based state)\n", warningStyle.Render("not available"))
}
```

---

## 5. Data Flow Changes

### Before (file-based):
```
Pipeline → creates .parquet file
                ↓ (os.ReadDir polling, 10s)
Watcher  → finds .parquet → pushes to HF → writes stats.csv + watcher_status.json
                                                    ↓ (45s polling)
Scheduler → reads stats.csv + watcher_status.json → displays progress
```

### After (Redis):
```
Pipeline → creates .parquet → RPUSH cc:*:watcher:pending + ZADD cc:*:rate:packed
                ↓ (Redis BRPOP, instant)
Watcher  → pops from queue → pushes to HF → SADD cc:*:committed + HSET cc:*:watcher
                                                    ↓ (Redis read, instant)
Scheduler → reads all state from Redis → displays real-time progress
```

Key improvements:
- **Latency**: 45s polling → instant Redis reads
- **Accuracy**: file counting → atomic counters
- **Visibility**: pgrep → registered session hashes with heartbeats
- **Debugging**: tail log files → Redis Streams + Insight

---

## 6. Migration Strategy

1. **Phase 1**: Add `cc_redis.go` with all helpers. No behavior changes yet.
2. **Phase 2**: Pipeline writes to Redis AND files (dual-write).
3. **Phase 3**: Scheduler reads from Redis when available, falls back to files.
4. **Phase 4**: Watcher uses Redis queue when available, falls back to readdir.
5. **Phase 5**: Remove file-based rate tracking from scheduler (Redis is primary).

Each phase is independently deployable. Rolling restart is safe.

---

## 7. Files Modified

| File | Changes |
|------|---------|
| `cli/cc_redis.go` | New: Redis client, helpers, key schema |
| `cli/cc_publish_pipeline.go` | Register/heartbeat/update pipeline state |
| `cli/cc_publish_watcher.go` | Redis queue + committed set + watcher status |
| `cli/cc_publish_schedule.go` | Read rates/state from Redis, remove file counting |
| `cli/cc_publish.go` | Redis init at startup, seed from CSV |
| `go.mod` | Add `github.com/redis/go-redis/v9` |

---

## 8. Redis Memory Estimate

| Key Pattern | Count | Size Each | Total |
|-------------|-------|-----------|-------|
| cc:*:committed | 1 SET | 100K members × 8 bytes | ~1 MB |
| cc:*:stats:* | 100K HASHes | ~200 bytes | ~20 MB |
| cc:*:pipeline:* | ~6 HASHes | ~500 bytes | ~3 KB |
| cc:*:rate:* | 3 ZSETs | ~10K entries | ~300 KB |
| cc:*:log | 1 STREAM | ~1000 entries | ~200 KB |
| cc:*:watcher | 1 HASH | ~500 bytes | ~500 B |
| **Total** | | | **~22 MB** |

Well within the 512 MB limit. Even at 100K shards completed, Redis uses <25 MB.
