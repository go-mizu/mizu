# 0732 — Arctic Publish: Auto Smart Plan (Pipelined Execution)

The current arctic publish pipeline is **fully sequential**: for each (month, type) pair,
it downloads → processes → uploads before starting the next pair. On a server with
256 GB RAM, 20-core CPU, 2 TB NVMe, and gigabit networking, this leaves most hardware
idle most of the time:

- During download: CPU and HF upload bandwidth sit idle
- During processing: network sits idle (both torrent and HF)
- During upload: CPU and torrent bandwidth sit idle

This spec introduces **pipelined execution** that overlaps download, process, and upload
stages across different (month, type) pairs, plus **dynamic hardware detection** to
auto-tune concurrency based on what's actually available.

---

## Design Principles

1. **No OOM** — detect available RAM, budget DuckDB instances accordingly
2. **No HF rate limits** — track commit rate, throttle upload stage if approaching 128/hr
3. **Dynamic** — auto-detect hardware at startup, adapt plan to the machine
4. **Observable** — enhanced states.json shows per-stage pipeline status
5. **Stable** — backpressure between stages prevents resource exhaustion
6. **Resume-safe** — same stats.csv skip logic, just faster throughput

---

## 1. Hardware Detection (`pkg/arctic/hwdetect.go`)

Detect hardware at startup and compute a resource budget. All values dynamic —
works on any Linux/macOS server without configuration.

```go
type HardwareProfile struct {
    Hostname     string  `json:"hostname"`
    OS           string  `json:"os"`           // "linux", "darwin"
    CPUCores     int     `json:"cpu_cores"`     // runtime.NumCPU()
    RAMTotalGB   float64 `json:"ram_total_gb"`  // from /proc/meminfo or sysctl
    RAMAvailGB   float64 `json:"ram_avail_gb"`  // available (not just free)
    DiskTotalGB  float64 `json:"disk_total_gb"` // from syscall.Statfs
    DiskFreeGB   float64 `json:"disk_free_gb"`  // available to non-root
    NetworkMbps  float64 `json:"network_mbps"`  // estimated from first download, 0 initially
}
```

### Resource Budget Computation

```go
type ResourceBudget struct {
    MaxConcurrentDownloads int  // limited by disk space and network
    MaxConcurrentProcess   int  // limited by RAM (each DuckDB = 512MB + overhead)
    MaxConcurrentUploads   int  // always 1 (HF serialized via commitMu)
    MaxPendingDownloaded   int  // downloaded-but-not-yet-processed queue depth
    MaxPendingProcessed    int  // processed-but-not-yet-uploaded queue depth
    DuckDBMemoryMB         int  // per-instance DuckDB memory limit
    ChunkLines             int  // lines per JSONL chunk
}
```

**Budget rules:**

| Resource | Formula | Rationale |
|----------|---------|-----------|
| Downloads | `min(2, diskFreeGB / 60)` | Each .zst up to 50GB; need headroom |
| Processing | `min(cpuCores/4, (ramAvailGB - 4) / 1.5)` capped at 3 | 512MB DuckDB + ~1GB overhead per instance |
| Uploads | Always 1 | HF API doesn't support concurrent commits |
| Download queue | `maxProcess + 1` | Keep process stage fed without hoarding disk |
| Process queue | 2 | Buffer for upload to pick up without waiting |
| DuckDB memory | `min(512, (ramAvailGB * 1024) / (maxProcess * 3))` | Share available RAM across concurrent processes |
| ChunkLines | Config value (default 2M) | Unchanged from current |

**Safety floor:** If RAM < 4GB or disk < 60GB, fall back to sequential mode (budget = 1/1/1).

### Implementation: Linux vs macOS

```go
// Linux: /proc/meminfo
func detectRAMLinux() (total, avail float64)

// macOS: sysctl hw.memsize + vm_stat
func detectRAMDarwin() (total, avail float64)

// Both: syscall.Statfs (already exists in config.go)
func detectDisk(path string) (total, free float64)

// CPU: runtime.NumCPU()

// Network: estimated after first download completes (bytes / duration)
```

---

## 2. Pipeline Architecture

Three stages connected by bounded channels:

```
                    ┌──────────┐     ┌──────────┐     ┌──────────┐
  month queue  ──→  │ DOWNLOAD │ ──→ │ PROCESS  │ ──→ │  UPLOAD  │ ──→ done
                    │ (N workers)│    │ (M workers)│    │ (1 worker)│
                    └──────────┘     └──────────┘     └──────────┘
                         │                │                │
                    bounded chan      bounded chan     commitMu
                    (depth: M+1)      (depth: 2)     (serialized)
```

### Pipeline Job

```go
type PipelineJob struct {
    YM       ymKey
    Type     string  // "comments" | "submissions"
    ZstPath  string  // populated after download
    WorkDir  string  // per-job isolated work directory
    Shards   []ShardResult  // populated after processing
    ProcResult ProcessResult
    DurDown  time.Duration
    DurProc  time.Duration
}
```

### Per-Job Work Directory

**Critical fix for concurrency:** Each job gets its own work directory to avoid
chunk/duckdb filename collisions:

```
WorkDir/
  pipeline_2024-01_comments/    ← job-specific
    chunk_0000.jsonl
    duckdb_0000.db
    comments/2024/01/000.parquet
  pipeline_2024-01_submissions/
    chunk_0000.jsonl
    ...
```

Config gets a `JobWorkDir(ym, typ)` method:

```go
func (c Config) JobWorkDir(ym, typ string) string {
    return filepath.Join(c.WorkDir, fmt.Sprintf("pipeline_%s_%s", ym, typ))
}
```

And ProcessZst uses a per-job Config clone with `WorkDir` set to the job directory.

### Stage Workers

**Download stage** (`downloadWorker`):
```
for job := range downloadCh:
    check disk space (if < minFreeGB, block until upload frees space)
    download .zst via DownloadZst()
    validate (QuickValidateZst)
    send to processCh
```

**Process stage** (`processWorker`):
```
for job := range processCh:
    create per-job work dir
    jobCfg := cfg with WorkDir = job.WorkDir
    ProcessZst(ctx, jobCfg, ...)
    delete .zst (stream exhausted)
    send to uploadCh
```

**Upload stage** (`uploadWorker`):
```
for job := range uploadCh:
    acquire commitMu
    build ops (shards + stats.csv + README.md + states.json)
    batch commit to HF (≤50 ops per call)
    update stats.csv
    delete local shards
    release commitMu
```

### Backpressure

Channels are bounded:
- `downloadCh` → `processCh`: capacity = `MaxConcurrentProcess + 1`
- `processCh` → `uploadCh`: capacity = 2

When `processCh` is full, download workers block naturally (Go channel semantics).
When `uploadCh` is full, process workers block. This prevents unbounded disk usage.

### Disk Space Guard

Before each download, check `FreeDiskGB()`. If below threshold:
1. Log warning
2. Block (wait on a `diskFreeCond sync.Cond`)
3. Upload worker signals `diskFreeCond.Broadcast()` after deleting shards

This prevents the pipeline from filling the disk when downloads are faster than processing.

---

## 3. Enhanced states.json

Extend `StateSnapshot` to show pipeline-level status:

```json
{
  "session_id": "2026-03-15T01:00:00Z",
  "started_at": "2026-03-15T01:00:00Z",
  "updated_at": "2026-03-15T03:45:12Z",
  "phase": "running",
  "hardware": {
    "hostname": "server2",
    "os": "linux",
    "cpu_cores": 20,
    "ram_total_gb": 256.0,
    "ram_avail_gb": 240.5,
    "disk_total_gb": 1863.0,
    "disk_free_gb": 1245.3,
    "network_mbps": 850.2
  },
  "budget": {
    "max_downloads": 2,
    "max_process": 3,
    "max_uploads": 1,
    "duckdb_memory_mb": 512
  },
  "pipeline": {
    "downloading": [
      {"ym": "2009-08", "type": "comments", "bytes_done": 1234567890, "bytes_total": 5432198765, "peers": 12}
    ],
    "processing": [
      {"ym": "2009-07", "type": "submissions", "shard": 3, "rows": 6543210, "rows_per_sec": 45200.0}
    ],
    "uploading": [
      {"ym": "2009-07", "type": "comments", "shards": 5, "rows": 12345678, "phase": "committing"}
    ],
    "queued_for_process": 1,
    "queued_for_upload": 0
  },
  "throughput": {
    "avg_download_mbps": 420.5,
    "avg_process_rows_per_sec": 38500.0,
    "avg_upload_sec_per_commit": 12.3,
    "estimated_completion": "2026-03-18T14:00:00Z"
  },
  "stats": {
    "committed": 47,
    "skipped": 0,
    "retries": 2,
    "total_rows": 1284000000,
    "total_bytes": 87234567890,
    "total_months": 488
  }
}
```

### New types

```go
type PipelineSlot struct {
    YM         string  `json:"ym"`
    Type       string  `json:"type"`
    BytesDone  int64   `json:"bytes_done,omitempty"`
    BytesTotal int64   `json:"bytes_total,omitempty"`
    Peers      int     `json:"peers,omitempty"`
    Shard      int     `json:"shard,omitempty"`
    Rows       int64   `json:"rows,omitempty"`
    RowsPerSec float64 `json:"rows_per_sec,omitempty"`
    Shards     int     `json:"shards,omitempty"`
    Phase      string  `json:"phase,omitempty"`
}

type PipelineState struct {
    Downloading      []PipelineSlot `json:"downloading"`
    Processing       []PipelineSlot `json:"processing"`
    Uploading        []PipelineSlot `json:"uploading"`
    QueuedForProcess int            `json:"queued_for_process"`
    QueuedForUpload  int            `json:"queued_for_upload"`
}

type ThroughputStats struct {
    AvgDownloadMbps      float64   `json:"avg_download_mbps"`
    AvgProcessRowsPerSec float64   `json:"avg_process_rows_per_sec"`
    AvgUploadSecPerCommit float64  `json:"avg_upload_sec_per_commit"`
    EstimatedCompletion  time.Time `json:"estimated_completion,omitempty"`
}
```

`StateSnapshot` gains:

```go
type StateSnapshot struct {
    // ... existing fields ...
    Hardware   *HardwareProfile `json:"hardware,omitempty"`
    Budget     *ResourceBudget  `json:"budget,omitempty"`
    Pipeline   *PipelineState   `json:"pipeline,omitempty"`
    Throughput *ThroughputStats `json:"throughput,omitempty"`
}
```

### Backward Compatibility

- `Current` field kept for single-job view (set to the "most active" pipeline slot)
- `Phase` uses "running" when pipeline is active (replaces "downloading"/"processing"/"committing")
- Old sequential mode still works if budget computes to 1/1/1

---

## 4. Enhanced README

### Pipeline Status Section (replaces simple Live Section)

When pipeline mode is active, the README shows:

```markdown
## Pipeline Status

> Pipelined ingestion running on **server2** (20 cores, 256 GB RAM, 1.2 TB free).
> Auto-updated every ~5 minutes.

**Started:** 2026-03-15 01:00 UTC · **Elapsed:** 2h 45m

### Active Workers

| Stage | Month | Type | Progress |
|-------|-------|------|----------|
| Downloading | 2009-08 | comments | 1.1 GB / 5.1 GB (22%) · 12 peers |
| Processing | 2009-07 | submissions | shard 3 · 6.5M rows · 45.2K rows/s |
| Uploading | 2009-07 | comments | 5 shards · 12.3M rows · committing |

### Throughput

| Metric | Value |
|--------|------:|
| Download | 420 Mbps avg |
| Processing | 38.5K rows/s avg |
| Upload | 12.3s per commit avg |
| ETA | 2026-03-18 14:00 UTC |

### Progress

`████████████░░░░░░░░░░░░░░░░░░` 47 / 488 (9.6%)

| Metric | This Session |
|--------|-------------:|
| Months committed | 47 |
| Total rows | 1.3B |
| Data committed | 87.2 GB |
| Retries | 2 |
```

---

## 5. Estimated Completion Time

Track running averages of each stage and compute ETA:

```go
type throughputTracker struct {
    mu            sync.Mutex
    downloadSecs  []float64  // last 10 download durations
    downloadBytes []int64    // corresponding byte counts
    processSecs   []float64  // last 10 process durations
    processRows   []int64    // corresponding row counts
    commitSecs    []float64  // last 10 commit durations
    networkMbps   float64    // computed after first download
}

func (t *throughputTracker) EstimateCompletion(remaining int, avgBytesPerMonth int64) time.Time
```

Uses a sliding window of the last 10 completed jobs to estimate:
- Average wall-time per (month, type) pair (download + process + commit, accounting for overlap)
- Remaining pairs × average time ÷ pipeline parallelism factor

---

## 6. Retry Integration with Pipeline

When a job fails at any stage:
1. Error is classified (corruption vs transient) — existing logic
2. Job is re-queued at the appropriate stage:
   - Corruption during process → re-queue at download stage (with .zst renamed to .part)
   - Transient during download → re-queue at download stage (keep .part)
   - HF commit failure → re-queue at upload stage (keep shards)
3. Exponential backoff applied per-job (not per-stage)
4. Failed job doesn't block other pipeline slots

---

## 7. Graceful Shutdown

On context cancellation (ctrl-C / SIGTERM):
1. Stop feeding new jobs to download stage
2. Let in-flight downloads finish current piece (torrent client handles this)
3. Let in-flight processing finish current shard
4. Attempt to commit any fully-processed jobs (best effort)
5. Write final states.json with phase="interrupted"
6. All temp files left in place for resume

---

## 8. Implementation Order

### Phase 1: Hardware Detection (new file)
- `pkg/arctic/hwdetect.go` — detect RAM, CPU, disk
- `pkg/arctic/hwdetect_linux.go` — Linux-specific (build tag)
- `pkg/arctic/hwdetect_darwin.go` — macOS-specific (build tag)
- `pkg/arctic/budget.go` — compute resource budget from hardware profile

### Phase 2: Per-Job Work Directories (modify existing)
- `pkg/arctic/config.go` — add `JobWorkDir()`, `JobChunkPath()`, `JobShardLocalPath()`
- `pkg/arctic/process.go` — accept per-job config (already works, just pass different WorkDir)

### Phase 3: Pipeline Orchestrator (new file)
- `pkg/arctic/pipeline.go` — `PipelineTask` with download/process/upload workers
- Channels, backpressure, disk space guard
- Integrates with existing `DownloadZst`, `ProcessZst`, HF commit functions

### Phase 4: Enhanced State (modify existing)
- `pkg/arctic/live_state.go` — add `PipelineState`, `ThroughputStats`, `HardwareProfile`
- `pkg/arctic/readme.go` — pipeline status section in README template
- Throughput tracker with sliding window

### Phase 5: CLI Integration (modify existing)
- `cli/arctic_publish.go` — detect hardware, compute budget, launch pipeline mode
- Print hardware profile and budget at startup
- Enhanced progress display for concurrent stages

### Phase 6: Fallback & Safety
- If budget = 1/1/1, use existing sequential `PublishTask.Run()` (zero risk)
- Environment variable `MIZU_ARCTIC_PIPELINE=0` to force sequential
- `MIZU_ARCTIC_MAX_DOWNLOADS`, `MIZU_ARCTIC_MAX_PROCESS` to override auto-detection
