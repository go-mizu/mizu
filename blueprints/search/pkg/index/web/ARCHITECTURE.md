# pkg/index/web — Architecture

## Overview

`pkg/index/web` is the dashboard backend for the Common Crawl FTS pipeline. It exposes an HTTP server with a WebSocket hub, a job manager, and handlers for browsing WARC metadata, searching the index, and triggering pipeline jobs (download → markdown → pack → index).

---

## Package Layout

```
pkg/index/web/
├── server.go          — Server struct, HTTP routes, handler functions
├── warc_api.go        — WARC list/detail/action REST API
├── jobs.go            — Manager: job lifecycle (create/run/cancel/complete/fail)
├── executors.go       — RunJob adapter + per-task bridges + pure helpers
├── task_download.go   — Self-contained download task
├── task_markdown.go   — Self-contained markdown conversion task
├── task_pack.go       — Self-contained pack task (warc_md, parquet)
├── task_index.go      — Self-contained index task (FTS engine indexing)
├── ws.go              — WSHub: WebSocket broadcast hub
├── meta_manager.go    — MetaManager: WARC metadata cache
├── doc_store.go       — DocStore: per-document DuckDB browse metadata
├── scanner.go         — Data directory scanner (WARC, markdown, pack, FTS)
├── overview.go        — Pipeline overview stats
├── duckdb_ops.go      — DuckDB pipeline runner (build-tagged)
├── duckdb_ops_chdb.go — chdb shim (build-tagged)
└── metastore/         — Job persistence (SQLite/DuckDB drivers)
```

---

## Core Abstractions

### Server

`Server` is the top-level struct. It owns the HTTP mux, FTS engine connection, WebSocket hub, job manager, and metadata caches. Two constructors:

- `New(...)` — lightweight, for search-only mode (no dashboard).
- `NewDashboard(...)` — full dashboard with job management, metadata, and WebSocket.

### Manager (jobs.go)

`Manager` tracks pipeline jobs in memory with optional metastore persistence. It is safe for concurrent use. Public surface:

```
Create(cfg) *Job
Get(id) *Job
List() []*Job
Cancel(id) bool
SetRunning(id, cancel)
UpdateProgress(id, pct, msg, rate)
Complete(id, msg)
Fail(id, err)
Clear() int
SetStore(metastore.Store)
LoadHistory(ctx)
SetCompleteHook(JobCompleteHook)
RunJob(*Job)
```

`Job` has status: `queued → running → completed | failed | cancelled`.

### Self-Contained Tasks (task_*.go)

Each pipeline stage is a self-contained task type that implements the `core.Task` pattern:

```go
func (t *XxxTask) Run(ctx context.Context, emit func(*XxxState)) (XxxMetric, error)
```

All inputs are injected as value fields on construction. Tasks have **no dependency on Manager** — they are pure data-in / data-out.

| Task | Input | State | Metric |
|---|---|---|---|
| `DownloadTask` | `CrawlDir, Paths, Selected` | `DownloadState` | `DownloadMetric` |
| `MarkdownTask` | `CrawlID, CrawlDir, Paths, Selected` | `MarkdownState` | `MarkdownMetric` |
| `PackTask` | `CrawlDir, Paths, Selected, Format` | `PackState` | `PackMetric` |
| `IndexTask` | `CrawlDir, Paths, Selected, Engine, Source` | `IndexState` | `IndexMetric` |

State types are detailed: per-file progress, bytes transferred, docs/sec, phase, WARC index, etc. — never a generic `Message string`.

### RunJob Adapter (executors.go)

`Manager.RunJob` bridges the Manager and self-contained tasks:

1. Resolves manifest (with TTL cache) + file selector → `paths []string`, `selected []int`.
2. Constructs the appropriate `*XxxTask` with all data injected.
3. Runs the task with a **non-blocking emit** wrapper — a 64-entry buffered channel drained by a goroutine that calls `Manager.UpdateProgress`. The task goroutine never blocks on slow WS broadcasts.
4. On success: `Manager.Complete`. On error: `Manager.Fail`.

```
RunJob → goroutine:
  resolveFiles → paths, selected
  NewXxxTask(...)
  nonBlockingEmit(fn) → ch(64) → goroutine → UpdateProgress → WSHub.Broadcast
  task.Run(ctx, emit)
  Complete | Fail
```

### WSHub (ws.go)

Broadcast hub for WebSocket connections. Clients subscribe to a job ID. `Manager` calls `hub.Broadcast(jobID, event)` on state transitions.

---

## Pipeline Stages

```
WARC manifest (S3)
  └── DownloadTask   → crawlDir/warc/*.warc.gz
        └── MarkdownTask → crawlDir/markdown/{warcIdx}/*.md
              └── PackTask     → crawlDir/pack/{format}/{warcIdx}.{ext}
                                 crawlDir/warc_md/{warcIdx}.md.warc.gz
                    └── IndexTask  → crawlDir/fts/{engine}/{warcIdx}/
```

**Pack formats:**
- `warc_md` — `.md files → .md.warc.gz` (concatenated-gzip WARC, per-record gzip members)
- `parquet` — `.md files → .parquet` (ZSTD, 50K row groups, `doc_id + text` schema)

**Index sources:** `files` (markdown dir), `parquet`, `bin`, `duckdb`, `markdown` (bin.gz).

---

## On-Disk Layout

```
~/data/common-crawl/{crawlID}/
├── warc/               — downloaded .warc.gz files
├── markdown/{warcIdx}/ — extracted .md files (one per HTML page)
├── pack/
│   ├── parquet/        — {warcIdx}.parquet
│   ├── bin/            — {warcIdx}.bin
│   ├── duckdb/         — {warcIdx}.duckdb
│   └── markdown/       — {warcIdx}.bin.gz
├── warc_md/            — {warcIdx}.md.warc.gz
└── fts/{engine}/{warcIdx}/ — FTS index shards
```

WARC index is the zero-padded 5-digit shard number extracted from the filename (e.g. `CC-MAIN-...-00042.warc.gz` → `"00042"`).

---

## Concurrency Model

- **Manager** is protected by `sync.RWMutex` for job state; manifest cache has its own `sync.Mutex`.
- **RunJob** spawns one goroutine per job; cancellation via `context.WithCancel`.
- **nonBlockingEmit** decouples task progress from WS broadcast latency using a 64-entry buffered channel. Intermediate states are dropped when the channel is full — the task never blocks.
- **WSHub** uses a `sync.Mutex` per broadcast; slow clients are disconnected.
- **DocStore** / **MetaManager** maintain their own internal locks.

---

## Key Pure Helpers (executors.go)

| Function | Purpose |
|---|---|
| `warcFileIndex(path, fallback)` | Extract 5-digit WARC shard index from filename |
| `packPath(packDir, format, warcIdx)` | Compute pack file path for a given format |
| `parseFileSelector(s, total)` | Parse `"0"`, `"0-4"`, `"all"` → `[]int` |
| `phaseProgress(done, total)` | Compute 0–1 fraction (caps at 0.95 for unknown total) |
| `phaseRate(done, elapsed)` | Items per second |
| `mbPerSec(bytes, elapsed)` | Megabytes per second |
| `fileExists(path)` | Stat-based existence check |
