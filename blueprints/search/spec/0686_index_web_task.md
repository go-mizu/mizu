# 0686: `pkg/index/web` backend workflow pipeline task refactor

## Goal
Refactor dashboard backend pipeline execution in `pkg/index/web` to use `pkg/core.Task` with explicit task model, runtime state, and final metrics, while preserving current `search cc fts dashboard` behavior.

## Scope
- `pkg/index/web/executors.go`
- `pkg/index/web/task_models.go`
- `pkg/index/web/task_download.go`
- `pkg/index/web/task_markdown.go`
- `pkg/index/web/task_pack.go`
- `pkg/index/web/task_index.go`

## Existing runtime contract (unchanged)
1. API creates a job via `POST /api/jobs`.
2. `JobManager.Create` stores queued job in memory (+ optional metastore).
3. `JobManager.RunJob` executes in goroutine and pushes websocket updates:
   - status transitions: `queued -> running -> completed|failed|cancelled`
   - progress stream: `job_progress` with `progress`, `message`, `rate`
4. UI (`static/js/jobs.js`, `static/js/warc.js`, `static/js/browse.js`) reacts only to this contract.

This contract is preserved exactly.

## New task architecture

### 1. Typed task descriptor/state/metric
`task_models.go` introduces:
- `WorkflowTask` (JSON-serializable execution descriptor)
- `WorkflowState` (JSON-serializable incremental state)
- `WorkflowMetric` (JSON-serializable final summary)

All fields include explicit `json` tags.

### 2. `pkg/core.Task` integration
`TaskRunner` now embeds:
- `core.Task[WorkflowState, WorkflowMetric]`
- plus `Descriptor() *WorkflowTask`

Each pipeline stage is now an isolated `core.Task` implementation:
- `downloadWorkflowTask`
- `markdownWorkflowTask`
- `packWorkflowTask`
- `indexWorkflowTask`

### 3. Stage selection
`JobManager.newWorkflowTask(job)` maps `job.Config.Type` to a stage task and injects resolved crawl context (`crawlID`, `crawlDir`).

### 4. Unified runtime execution
`RunJob` now:
1. marks job running (`SetRunning`)
2. builds concrete task
3. runs task with `emit` callback
4. maps each `WorkflowState` to `UpdateProgress`
5. maps task result/error to `Complete`/`Fail`

This keeps websocket and persistence behavior stable while moving stage logic out of monolithic executor methods.

## Stage behavior details

### Download task
- Fetches manifest via `getManifestPaths` (cache preserved).
- Resolves `files` selector.
- Downloads WARCs into `crawlDir/warc`.
- Emits per-file and per-transfer state updates.
- Returns metrics with processed file count.

### Markdown task
- Fetches manifest and selected indices.
- Validates local WARC existence (actionable error unchanged).
- Runs `warc_md.RunFilePipeline` per shard.
- Emits two-phase progress (extract + convert) and throughput.
- Returns file/doc metrics.

### Pack task
- Supports same output formats: `parquet|bin|duckdb|markdown`.
- Validates source markdown shard directory.
- Runs corresponding packer and emits cumulative progress.
- Returns file/doc metrics.

### Index task
- Defaults preserved: `engine=rose`, `source=files`.
- Opens per-shard engine output in `fts/{engine}/{shard}`.
- Supports source modes: `files|parquet|bin|duckdb|markdown`.
- Executes engine finalizer when available.
- Emits per-shard indexing progress + rate.
- Returns file/doc metrics.

## Compatibility and risk analysis

### Preserved
- Job API payloads (`JobConfig`) and server endpoints.
- Job lifecycle statuses and websocket event schema.
- File selector parsing and manifest cache semantics.
- Crawl override behavior (`cfg.crawl`).
- Error strings for main operator-facing failures.

### Behavioral note
- Internals now have explicit typed state/metric objects, but outbound JSON for existing job endpoints/events remains the same because `WorkflowState` is mapped to existing `UpdateProgress` and websocket message types.

## Validation checklist
- `go test ./pkg/index/web/...`
- `go test ./cli -run TestCCFTSWeb -count=1` (if present)
- `go build ./cmd/search`
- Manual smoke:
  - run `search cc fts dashboard`
  - queue `download -> markdown -> index`
  - verify live progress updates and completion states in Jobs/WARC pages

## Why this refactor improves maintainability
- Removes monolithic executor with mixed concerns.
- Establishes clear boundary:
  - `WorkflowTask`: immutable execution input
  - `WorkflowState`: live observable state
  - `WorkflowMetric`: final outcome
- Enables easier future additions (new stage or source/format) without touching all executors.
