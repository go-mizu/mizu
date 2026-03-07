# pkg/index/web Task Refactor

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor task system into self-contained, public task types with detailed state emission, non-blocking progress, and consistent Go naming.

**Architecture:** Each task is a pure `core.Task[*State, *Metric]` with zero dependency on Manager. Manager.RunJob is a thin adapter that resolves inputs, creates tasks, and bridges state→progress via buffered channel. Per-task State/Metric types replace shared Workflow* types.

**Tech Stack:** Go 1.22+, parquet-go/parquet-go, pkg/warc, pkg/warc_md, pkg/cc

---

### Task 1: task_download.go — Self-contained download task

**Files:**
- Rewrite: `pkg/index/web/task_download.go`

Public types: `DownloadTask`, `DownloadState`, `DownloadMetric`, `NewDownloadTask()`
- DownloadState: FileIndex, FileTotal, FileName, WARCIndex, BytesReceived, BytesTotal, Progress, BytesPerSec
- DownloadMetric: Files, Bytes, Elapsed
- Helper functions: emitDownloadProgress, downloadFraction, fileProgress
- No JobManager/Manager dependency

### Task 2: task_markdown.go — Self-contained markdown task

**Files:**
- Rewrite: `pkg/index/web/task_markdown.go`

Public types: `MarkdownTask`, `MarkdownState`, `MarkdownMetric`, `NewMarkdownTask()`
- MarkdownState: FileIndex, FileTotal, WARCIndex, Phase, DocsProcessed, DocsTotal, DocsErrors, ReadBytes, WriteBytes, ReadRate, WriteRate, Progress
- MarkdownMetric: Files, Docs, Elapsed
- Helper functions: markdownConfig, emitMarkdownProgress, extractProgress, convertProgress

### Task 3: task_pack.go — Self-contained pack task (no pkg/index/pack)

**Files:**
- Rewrite: `pkg/index/web/task_pack.go`

Public types: `PackTask`, `PackState`, `PackMetric`, `NewPackTask()`
- Two formats only: "warc_md" (.md files → .md.warc.gz), "parquet" (.md files → .parquet)
- Inline parquet writing (parquet-go, schema: doc_id+text, ZSTD, 50K row groups)
- Inline warc_md packing (iterate .md files → WARC conversion records → gzip concat)
- Helper functions: packToWARCMd, packToParquet, walkMarkdownFiles

### Task 4: task_index.go — Self-contained index task

**Files:**
- Rewrite: `pkg/index/web/task_index.go`

Public types: `IndexTask`, `IndexState`, `IndexMetric`, `NewIndexTask()`
- IndexState: FileIndex, FileTotal, WARCIndex, Engine, Source, DocsIndexed, DocsTotal, Progress, DocsPerSec
- Helper functions: openEngine, indexFromFiles, indexFromSource

### Task 5: executors.go — RunJob adapter + helpers

**Files:**
- Rewrite: `pkg/index/web/executors.go`

- Manager.RunJob: resolve manifest+selector → create task → run with non-blocking emit via buffered channel
- Per-task adapter functions: runDownloadJob, runMarkdownJob, runPackJob, runIndexJob
- Manifest cache (moved from Manager struct fields to package-level or dedicated type)
- Pure helpers: warcFileIndex, packPath, parseFileSelector, phaseProgress, phaseRate, mbPerSec, fileExists
- Single execTask compat wrapper

### Task 6: jobs.go — JobManager → Manager rename

**Files:**
- Modify: `pkg/index/web/jobs.go`

- Rename JobManager → Manager everywhere
- Remove manifest cache fields (moved to executors.go)
- Remove manifestFetch, manifestMu, manifestCache from struct

### Task 7: Update references across package

**Files:**
- Modify: `pkg/index/web/server.go` (JobManager→Manager, WorkflowState→per-task)
- Modify: `pkg/index/web/warc_api.go` (JobManager→Manager)
- Modify: `pkg/index/web/meta_manager.go` (if any refs)
- Modify: `pkg/index/web/jobs_test.go` (update test helpers)
- Modify: `pkg/index/web/server_test.go` (update type refs)

### Task 8: Delete task_models.go

**Files:**
- Delete: `pkg/index/web/task_models.go`

### Task 9: ARCHITECTURE.md

**Files:**
- Create: `pkg/index/web/ARCHITECTURE.md`

### Task 10: Build + test verification

Run: `cd blueprints/search && go build ./pkg/index/web/...`
Run: `cd blueprints/search && go test ./pkg/index/web/...`
