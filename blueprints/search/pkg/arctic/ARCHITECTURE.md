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
│  starts pprof + memory logger, wires everything together.       │
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
| `config.go`           | `Config` struct, path builders, env var defaults, disk checks  |
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
  ├─ Scan lines → write to disk as chunk_N.jsonl files (ChunkLines per chunk)
  │   └─ No in-memory line accumulation — stream to disk
  │
  ├─ For each completed chunk:
  │   └─ convertChunkToShard()
  │       ├─ In-memory DuckDB (no file mmap overhead)
  │       ├─ SET memory_limit (default 512MB)
  │       ├─ read_json_auto → COPY TO parquet (ZSTD compression)
  │       ├─ ValidateParquet() — PAR1 magic check
  │       └─ defer mallocTrim() — reclaim glibc C heap after DuckDB close
  │
  ├─ Close decoder + file BEFORE remaining shard conversion
  ├─ runtime.GC() + debug.FreeOSMemory() — reclaim 2 GB window
  ├─ RELEASE zstdDecoderSem (another worker can now decode)
  │
  └─ Convert remaining chunk → shard (overlaps with next worker's decode)
```

### 3. Upload Stage (`uploadJob`)

```
uploadJob(job)
  │
  ├─ Read stats.csv, append new StatsRow
  ├─ Generate README.md with live state
  ├─ Build HFOp list: parquet shards + stats.csv + README.md + states.json
  │
  ├─ Batch commit (≤50 ops per API call)
  │   └─ Each batch retried 3× internally with 5s/10s backoff
  │
  └─ Cleanup: delete local shard files + job work directory
```

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
```

### Resume After Crash

1. `CommittedSet(ReadStatsCSV())` — months already in stats.csv are skipped
2. `cleanupWork()` — removes stale work dirs and `.part` files, **keeps valid `.zst` files**
3. `downloadJob()` — checks for existing `.zst` via `QuickValidateZst()`, skips download if valid
4. Processing and upload start fresh (parquet shards are regenerated)

## Heartbeat System

The pipeline writes progress to HuggingFace every 5 minutes (or on each data
commit, whichever comes first):

- `states.json` — machine-readable `StateSnapshot` (phase, workers, throughput)
- `README.md` — human-readable with live progress section
- Heartbeat commits are serialized with data commits via `commitMu`

## Configuration

### Environment Variables

| Variable                      | Default                    | Purpose                       |
|-------------------------------|----------------------------|-------------------------------|
| `HF_TOKEN`                    | (required)                 | HuggingFace API token         |
| `MIZU_ARCTIC_REPO_ROOT`      | `$HOME/data/arctic/repo`   | Local repo root               |
| `MIZU_ARCTIC_RAW_DIR`        | `{root}/raw`               | Torrent download directory    |
| `MIZU_ARCTIC_WORK_DIR`       | `{root}/work`              | Temp chunks and DuckDB files  |
| `MIZU_ARCTIC_CHUNK_LINES`    | 500000                     | Lines per JSONL chunk         |
| `MIZU_ARCTIC_MIN_FREE_GB`    | 30                         | Disk space gate               |
| `MIZU_ARCTIC_MAX_DOWNLOADS`  | (auto)                     | Override download concurrency |
| `MIZU_ARCTIC_MAX_PROCESS`    | (auto)                     | Override process concurrency  |
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
