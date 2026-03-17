# 0737: Refactor Arctic Pipeline

## Problem

`pipeline.go` is 1167 lines handling download, process, upload, heartbeat, throughput tracking, disk management, and pipeline orchestration all in one file. `task_publish.go` (654 lines) duplicates much of this logic in a sequential path. This makes the code hard to maintain and debug.

**Critical bug**: When a single month's download fails after 5 retries, `setPipeErr` + `pipeCancel()` kills the **entire pipeline** — including in-progress uploads for other months. We observed 2011-05 comments (733 MB upload, 74% done) get aborted because 2011-06 download failed. The 2011-05 data had to be re-downloaded and re-processed from scratch.

## Goals

1. Split pipeline.go into focused files by pipeline stage
2. Make download failures non-fatal (skip month, continue pipeline)
3. Remove the old sequential `PublishTask` — one code path, not two
4. Prefer private functions over methods (Go core library style)
5. Keep pipelining, OOM safety, disk management, and stall detection

## New File Layout

```
pkg/arctic/
  task_download.go   — downloadJob, retryDownload, validateZst, reuse logic
  task_process.go    — processJob, error classification, retry with corruption handling
  task_commit.go     — commitJob, HF batch upload, stats.csv revert on failure, heartbeat
  task_publish.go    — PipelineTask, Run(), pipeline orchestration, channel wiring
  config.go          — unchanged
  stats.go           — unchanged
  live_state.go      — unchanged (+ pipeline slot helpers moved here or kept in task_publish)
  torrent.go         — unchanged
  process.go         — unchanged
  budget.go          — unchanged
```

## Design: Private Functions Over Methods

Inspired by Go core library (e.g. `net/http`), use private functions that take explicit dependencies instead of methods on a god-object. This makes data flow visible and testing easier.

```go
// Instead of:  t.downloadJob(ctx, job, emit)
// Prefer:      downloadJob(ctx, cfg, zstSizes, job, emit)

// Instead of:  t.uploadJob(ctx, job, emit)
// Prefer:      commitJob(ctx, cfg, opts, job, ls, commitMu, lastHFCommit)
```

The `PipelineTask` struct remains as the top-level orchestrator but becomes thin — it wires channels and calls private functions. Each private function receives only what it needs.

## Key Change: Non-Fatal Download Failures

Current (broken):
```
download fails → setPipeErr → pipeCancel() → all stages abort
```

New (resilient):
```
download fails → log warning → increment skipped counter → continue with next job
```

The pipeline continues processing other months. The failed month will be retried on the next run (it won't be in stats.csv as committed). Upload workers are never interrupted by download failures.

## File Details

### task_download.go (~200 lines)

Private functions:
- `downloadJob(ctx, cfg, zstSizes, job, emit) error` — download one .zst via torrent
- `retryDownload(ctx, cfg, zstSizes, job, emit, maxRetries) error` — retry with backoff
- `reuseExistingZst(cfg, zstSizes, job) bool` — check if valid .zst already exists
- `validateZst(path string, expectedBytes int64) error` — size + quick zstd check

Types:
- `ErrCorruption`, `ErrTransient` — error classification (already exist)

### task_process.go (~200 lines)

Private functions:
- `processJob(ctx, cfg, budget, job, emit) error` — run ProcessZst with isolated workdir
- `retryProcess(ctx, cfg, budget, job, emit, maxRetries) error` — retry with corruption detection
- `classifyProcessError(err error) string` — "corruption" | "transient"

### task_commit.go (~250 lines)

Private functions:
- `commitJob(ctx, cfg, opts CommitFn, job, ls, commitMu, lastCommit) error` — full HF upload flow
  - Read existing stats, append new row, write local files
  - Build HF ops, batch upload with retries
  - Revert stats.csv on failure
  - Update DurCommitS inside lock
- `commitHeartbeat(ctx, cfg, opts CommitFn, ls, commitMu, lastCommit, force) error`
- `writeHeartbeatFiles(cfg, ls, zstSizes)` — write states.json + README.md

### task_publish.go (~400 lines, replaces both pipeline.go and old task_publish.go)

The orchestrator. Keeps:
- `PipelineTask` struct (thinner — no throughput/disk fields, those become local vars)
- `Run()` → `runPipeline()`
- Channel wiring: downloadCh → processCh → uploadCh
- Job feeding with disk gate
- Heartbeat goroutine with stall detection
- Throughput tracking (private functions: `recordSpeed`, `appendWindow`, `avg`)
- `cleanupWork`, `monthRange`, `waitForDisk`

Key change in download worker:
```go
for job := range downloadCh {
    if pipeCtx.Err() != nil {
        continue
    }
    err := retryDownload(pipeCtx, cfg, zstSizes, job, emit, maxRetries)
    if err != nil {
        if pipeCtx.Err() != nil {
            continue
        }
        // NON-FATAL: skip this month, log it, continue pipeline
        logf("pipeline: SKIP [%s] %s — download failed: %v", job.YM, job.Type, err)
        ls.Update(func(s *StateSnapshot) { s.Stats.Skipped++ })
        continue  // don't send to processCh
    }
    processCh <- job
}
```

Similarly for process failures — skip month instead of killing pipeline.

Upload failures remain fatal (they indicate HF API problems that affect all months).

## OOM Safety (Preserved)

- GOMEMLIMIT set externally (8 GiB)
- DuckDB memory limited per budget.DuckDBMemoryMB
- .zst deleted immediately after processing (before upload)
- Disk gate prevents downloading when free space < MinFreeGB
- malloc_trim called after each major allocation cycle

## Stall Detection (Preserved)

- Per-download: 3-min BytesCompleted stall → retry (in torrent.go, unchanged)
- Per-pipeline: MaxCommitStall → ErrCommitStall → exit 75 → restart loop
- Heartbeat checks every 2 minutes

## Migration

1. Delete `pipeline.go`
2. Delete old sequential code from `task_publish.go` (`PublishTask.Run`, `processOne`, `processOneWithRetry`, `cleanupWork`, `monthRange`)
3. Keep shared types in `task_publish.go`: `PublishOptions`, `PublishState`, `PublishMetric`, `PipelineJob`, `ymKey`
4. Create `task_download.go`, `task_process.go`, `task_commit.go`
5. Rewrite `task_publish.go` orchestration

The `NewPublishTask` function and sequential fallback (`budget.Sequential`) are removed. All runs use the pipelined path.

## Testing

- Existing `process_go_test.go` continues to work (tests ProcessZst directly)
- `make test` verifies compilation
- Manual verification: deploy to server2, confirm pipeline processes months end-to-end
