# Arctic Publish — Architecture

## Overview

The `pkg/arctic` package implements a multi-stage pipeline that downloads Reddit
archive data from Academic Torrents (zstd-compressed JSONL), converts it into
Parquet shards via DuckDB, and commits them to a HuggingFace dataset repo —
month by month, resumable after interruption.

```
┌──────────────────────────────────────────────────────────────────┐
│                     CLI (cli/arctic_publish.go)                  │
│  Parses flags, creates HF client, bridges HFOp → hfOperation,  │
│  starts pprof, wires everything together.                       │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                     PipelineTask.Run()
                             │
        ┌────────────────────┼────────────────────┐
        ▼                    ▼                    ▼
   Download(N)         Process(M)           Upload(1)
   ──────────          ──────────           ─────────
   Torrent .zst   →   Scan JSONL     →   HF commit
   from Academic       + DuckDB           (parquet shards
   Torrents            → Parquet           + stats.csv
                                           + README.md)
```

## Execution Modes

| Mode       | When                    | Behavior                                       |
|------------|-------------------------|-------------------------------------------------|
| Pipeline   | RAM ≥ 2 GB, disk ≥ 30 GB | Stages overlap: download N+1 while processing N |
| Sequential | RAM < 2 GB or disk < 30 GB | One month at a time via `PublishTask.Run()`     |

Set `MIZU_ARCTIC_PIPELINE=0` to force sequential mode.

## File Inventory

| File                  | Purpose                                                        |
|-----------------------|----------------------------------------------------------------|
| `config.go`           | `Config` struct, path builders, env var defaults, `logf()` helper, disk checks |
| `budget.go`           | `ComputeBudget()` — derives concurrency limits from hardware   |
| `hwdetect.go`         | `DetectHardware()` — probes CPU, RAM, disk via syscall         |
| `pipeline.go`         | `PipelineTask` — concurrent download/process/upload pipeline   |
| `task_publish.go`     | `PublishTask` — sequential fallback, retry/error classification|
| `torrent.go`          | `DownloadZst()` — torrent download with stall detection        |
| `process.go`          | `ProcessZst()` — zstd decode + DuckDB JSONL→Parquet conversion |
| `stats.go`            | `StatsRow`, CSV read/write, `CommittedSet()` for resume        |
| `hf.go`               | `HFOp`, `CommitFn` types (HF client impl lives in CLI layer)  |
| `live_state.go`       | `LiveState`, `StateSnapshot` — real-time progress tracking     |
| `readme.go`           | README.md generation with live progress sections               |
| `malloc_trim_*.go`    | Platform-specific glibc `malloc_trim(0)` after DuckDB close    |

## Logging

All internal logging uses `logf()` (defined in `config.go`) which prefixes every
line with an ISO timestamp for debuggability:

```
2026-03-16 14:05:02 arctic: pipeline: [2011-01] comments uploading 15 ops (12 shards) to HF…
2026-03-16 14:13:14 arctic: pipeline: [2011-01] comments committed in 492.3s (12 shards, 7,234,561 rows)
2026-03-16 14:13:14 arctic: [2011-01] submissions waiting for zstd decoder semaphore…
```

The HF upload helper (`cli/embed/hf_commit.py`) runs with httpx/urllib3 logs
suppressed at WARNING level. Only the summary lines (`add:`, `committing`,
`committed in Xs`) appear on stderr.

## Pipeline Data Flow

### 1. Download Stage (`downloadJob`)

```
downloadJob(job)
  │
  ├─ Check: does .zst already exist and pass QuickValidateZst?
  │   YES → skip download, reuse file (avoids re-downloading after crash)
  │   NO  → DownloadZst() via torrent
  │
  ├─ DownloadZst()
  │   ├─ Picks info hash: monthlyInfoHashes[ym] for 2024+, bundleInfoHash for older
  │   ├─ Creates torrent client (pkg/torrent), selective file download
  │   ├─ Pre-checks disk space (need 2x file size for .zst + parquet)
  │   ├─ Stall detection: 3-min timeout with no byte progress → ErrTransient
  │   └─ Close() before rename (flush mmap pages)
  │
  └─ QuickValidateZst(): magic bytes + tail check + sampling at 25/50/75%
```

### 2. Process Stage (`processJob` → `ProcessZst`)

```
ProcessZst(zstPath)
  │
  ├─ ACQUIRE zstdDecoderSem (only 1 decoder process-wide)
  │
  ├─ Open .zst + create zstd.NewReader(WithDecoderMaxWindow(1<<31))
  │   └─ The 2 GB window is required — Reddit archives use 2 GB zstd windows
  │
  ├─ SCAN phase: stream lines → write to chunk_N.jsonl (ChunkLines per chunk)
  │   ├─ Each completed chunk dispatched to convert worker pool via chunkCh
  │   ├─ Bounded channel (cap = MaxConvertWorkers) → backpressure if workers busy
  │   └─ No in-memory line accumulation — stream to disk
  │
  ├─ Close decoder + file → runtime.GC() + debug.FreeOSMemory()
  ├─ RELEASE zstdDecoderSem (another worker can now decode)
  │
  ├─ Convert workers continue processing remaining chunks:
  │   └─ convertChunkToShard()
  │       ├─ In-memory DuckDB (no file mmap overhead)
  │       ├─ SET memory_limit (default 512MB)
  │       ├─ read_json_auto → COPY TO parquet (ZSTD compression)
  │       ├─ Delete chunk file immediately after DuckDB import
  │       ├─ ValidateParquet() — PAR1 magic check
  │       └─ defer mallocTrim() — reclaim glibc C heap after DuckDB close
  │
  └─ Collect results, sort by shard index
```

Key insight: the zstd decoder semaphore is released **after scan completes**,
not after all shards are converted. This allows the next file to start decoding
while the current file's DuckDB conversions run in parallel.

### 3. Upload Stage (`uploadJob`)

```
uploadJob(job)
  │
  ├─ Read stats.csv, append new StatsRow
  ├─ Generate README.md with live state
  ├─ Build HFOp list: parquet shards + stats.csv + README.md + states.json
  │
  ├─ Batch commit (≤50 ops per API call)
  │   ├─ Hold commitMu to serialize with heartbeat commits
  │   ├─ Each batch retried 3× internally with 5s/10s backoff
  │   └─ On failure: revert stats.csv to prevent false "committed" state
  │
  ├─ Log: "committed in Xs (N shards, M rows)"
  └─ Cleanup: delete local shard files + job work directory
```

The upload stage is always single-worker (`MaxUploads = 1`) because the HF API
is serialized via `commitMu`. The upload worker runs the embedded `hf_commit.py`
via `uv`, which uses `huggingface_hub` + `hf-xet` for native xet storage.

## Memory Architecture

**The #1 constraint**: each zstd decoder allocates a 2 GB window buffer
(`startStreamDecoder.func2`). On a server with 11 GB RAM, two concurrent
decoders = 4+ GB → OOM.

### Memory Budget Per Processing Slot

| Component           | Memory    |
|---------------------|-----------|
| zstd decoder window | ~2,048 MB |
| DuckDB instance     | ~512 MB   |
| Scanner buffers     | ~16 MB    |
| Overhead            | ~200 MB   |
| **Total**           | **~3 GB** |

### Protection Mechanisms

1. **`zstdDecoderSem`** (process.go) — `sync.Mutex` ensuring at most one 2 GB
   zstd decoder exists process-wide. Acquired before `zstd.NewReader()`,
   released after `dec.Close()` + `runtime.GC()`. Other workers continue with
   DuckDB shard conversion while blocked.

2. **`ComputeBudget()`** (budget.go) — Calculates `MaxProcess` as
   `(RAMTotal - 4 GB) / 3 GB`, capped by CPU cores and hard max of 4.
   On server2 (11 GB): `MaxProcess = 2` (one decoding, one doing DuckDB).

3. **`mallocTrim()`** — After DuckDB `db.Close()`, calls glibc
   `malloc_trim(0)` (Linux only) to return freed C heap pages to OS.

4. **`runtime.GC()` + `debug.FreeOSMemory()`** — Called after closing the zstd
   decoder and before releasing the semaphore, ensuring the 2 GB buffer is
   reclaimed before another worker can allocate a new one.

### Memory Timeline (Single Worker)

```
Time →
│ zstdDecoderSem.Lock()
│ ┌──────────────────────┐
│ │  zstd decode (2 GB)  │
│ │  scanning JSONL      │
│ └──────────────────────┘
│ dec.Close() + GC → 2 GB freed
│ zstdDecoderSem.Unlock()
│ ┌───────────────┐  ┌───────────────┐
│ │ DuckDB shard1 │  │ DuckDB shard2 │  ... (512 MB each, sequential)
│ └───────────────┘  └───────────────┘
```

### Memory Timeline (2 Workers, Overlapped)

```
Worker 1:  [== zstd decode 2GB ==][=== DuckDB shard conversion ===]
Worker 2:       (blocked)         [== zstd decode 2GB ==][=== DuckDB ===]
                                   ^                      ^
                                   sem released by W1     sem released by W2

Peak: 2 GB (decoder) + 512 MB (DuckDB) = ~2.5 GB
```

## Pipeline Overlap

The three-stage pipeline (`download → process → upload`) overlaps across
different months. While the upload worker pushes month N to HuggingFace
(~8 min), the process workers decode + convert month N+1.

```
Timeline with pipeline overlap:

Month N:    [download] [process ────────] [upload ────────────────]
Month N+1:             [download] [process ────────] [upload ────]
Month N+2:                        [download] [process ────]

Upload worker idle between commits only if no processed job is ready.
```

The `uploadCh` channel (capacity = `ProcessQueue`, default 2) buffers completed
jobs. Once `processJob` finishes all shards for a month, it sends the job to
`uploadCh`. The upload worker picks it up immediately.

## Auto-Heal / Retry Logic

### Error Classification

| Error Type   | Signature                       | Action                            |
|--------------|---------------------------------|-----------------------------------|
| Corruption   | zstd decode error, scan error   | Delete .zst, re-download, retry   |
| Transient    | Timeout, network, context       | Keep .zst, backoff, retry         |
| Rate Limit   | HF 429                          | Sleep Retry-After + 30s, retry    |
| Conn Reset   | "connection reset by peer"      | Verify commit landed, then decide |

### Pipeline Retry Flow

```
downloadJob fails?
  └─ retryDownload(): up to 5 attempts, exponential backoff (10s, 20s, 40s...)
       ├─ Corruption → rename .zst to .part, re-download
       └─ Transient → keep .part for torrent resume

processJob fails?
  └─ Process worker retry loop: up to 5 attempts
       ├─ Corruption → delete .zst, re-download via downloadJob(), retry process
       └─ Transient → keep .zst, backoff (10s, 20s...), retry process

uploadJob fails?
  └─ Upload worker retry loop: up to 5 attempts, backoff (30s, 60s, 120s...)
       └─ Internal: each HF commit batch already retried 3× (5s, 10s)
       └─ On failure: revert stats.csv to prevent false committed state
```

### Resume After Crash

1. `CommittedSet(ReadStatsCSV())` — months already in stats.csv are skipped
2. `cleanupWork()` — removes stale work dirs and `.part` files, **keeps valid `.zst` files**
3. `downloadJob()` — checks for existing `.zst` via `QuickValidateZst()`, skips download if valid
4. Processing and upload start fresh (parquet shards are regenerated)

## Heartbeat System

The pipeline writes progress to HuggingFace every 10 minutes (or on each data
commit, whichever comes first):

- `states.json` — machine-readable `StateSnapshot` (phase, workers, throughput)
- `README.md` — human-readable with live progress section
- Heartbeat commits are serialized with data commits via `commitMu`

## HF Upload Architecture

The Go layer (`cli/arctic_publish.go`) bridges `arctic.HFOp` to the Python
upload helper via stdin JSON:

```
Go (hfCommitFn)
  │
  ├─ Convert arctic.HFOp[] → JSON payload
  ├─ Handle rate limits (429) with Retry-After + 30s
  ├─ Handle "connection reset by peer" with verify-then-retry
  │
  └─ cli/embed/hf_commit.py (via uv)
       ├─ huggingface_hub.create_commit() with hf-xet
       ├─ Xet tuning: HF_XET_FIXED_UPLOAD_CONCURRENCY=8
       ├─ 30-min hard timeout per upload
       └─ Returns JSON: {commit_url: "..."} or {error: "...", retry_after: N}
```

## Configuration

### Environment Variables

| Variable                      | Default                    | Purpose                       |
|-------------------------------|----------------------------|-------------------------------|
| `HF_TOKEN`                    | (required)                 | HuggingFace API token         |
| `MIZU_ARCTIC_REPO_ROOT`      | `$HOME/data/arctic/repo`   | Local repo root               |
| `MIZU_ARCTIC_RAW_DIR`        | `$HOME/data/arctic/raw`    | Torrent download directory    |
| `MIZU_ARCTIC_WORK_DIR`       | `$HOME/data/arctic/work`   | Temp chunks and DuckDB files  |
| `MIZU_ARCTIC_CHUNK_LINES`    | 500000                     | Lines per JSONL chunk         |
| `MIZU_ARCTIC_MIN_FREE_GB`    | 30                         | Disk space gate               |
| `MIZU_ARCTIC_MAX_DOWNLOADS`  | (auto)                     | Override download concurrency |
| `MIZU_ARCTIC_MAX_PROCESS`    | (auto)                     | Override process concurrency  |
| `MIZU_ARCTIC_MAX_CONVERT`    | (auto)                     | Override convert workers      |
| `MIZU_ARCTIC_DUCKDB_MB`      | 512                        | Override DuckDB memory limit  |
| `MIZU_ARCTIC_PIPELINE`       | 1                          | Set to 0 for sequential mode  |
| `TORRENT_STORAGE_DEFAULT_FILE_IO` | (unset)               | Set to "classic" to avoid mmap|

## Torrent Strategy

- **Months ≤ 2023-12**: Bundle torrent (`bundleInfoHash`) — one torrent for all files
- **Months ≥ 2024-01**: Individual monthly torrents (`monthlyInfoHashes` map)
- Selective file download: only the needed `{comments,submissions}/R{C,S}_YYYY-MM.zst`
- Boundary-file workaround: enables adjacent file at low priority to complete shared last piece
- Classic file I/O (via env var) to avoid mmap zero-fill corruption

## DuckDB Processing

- In-memory DuckDB (`sql.Open("duckdb", "")`) — no file-backed mmap
- `read_json_auto` with `ignore_errors=true` and `union_by_name=true`
- ZSTD-compressed Parquet output with 128K row groups
- Comments: 12 columns (id, author, subreddit, body, score, created_utc, ...)
- Submissions: 14 columns (id, author, subreddit, title, selftext, score, ...)
- `mallocTrim()` after each DuckDB close to return glibc heap to OS
