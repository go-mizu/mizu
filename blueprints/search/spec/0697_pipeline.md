# 0697 — Pipeline Package Refactor

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extract the monolithic `pkg/index/web/` pipeline code (~4,700 LOC) into well-organized subpackages under `pipeline/`, `pipeline/cc/`, `pipeline/scrape/`, and `api/`.

**Architecture:** The `web` package keeps HTTP server lifecycle, search/browse/domain handlers, and static assets. Pipeline concerns (task lifecycle, execution, progress broadcasting) move to `pipeline/`. Task implementations are grouped by use case: CC pipeline tasks in `pipeline/cc/`, scrape tasks in `pipeline/scrape/`. HTTP handlers for jobs and scrape CRUD move to `api/`. All tasks explicitly implement `core.Task[State, Metric]`.

**Tech Stack:** Go 1.26, Mizu framework, gorilla/websocket, DuckDB

---

## Current State

`pkg/index/web/` has 36 Go files. Pipeline-related files (jobs, executors, tasks, scrape, ws, overview) total ~4,700 lines mixed with unrelated HTTP handler code.

### Files to move

| Current file | Destination | Notes |
|---|---|---|
| `jobs.go` (453L) | `pipeline/manager.go` + `pipeline/job.go` | Split types from lifecycle |
| `executors.go` (491L) | `pipeline/executor.go` | RunJob dispatch + helpers |
| `ws.go` (210L) | `pipeline/ws.go` | WSHub + WSClient |
| `ws_test.go` (223L) | `pipeline/ws_test.go` | |
| `jobs_test.go` (373L) | `pipeline/manager_test.go` | |
| `overview.go` (396L) | `pipeline/overview.go` | Stage scanners |
| `overview_test.go` (113L) | `pipeline/overview_test.go` | |
| `task_download.go` (136L) | `pipeline/cc/task_download.go` | |
| `task_markdown.go` (158L) | `pipeline/cc/task_markdown.go` | |
| `task_pack.go` (350L) | `pipeline/cc/task_pack.go` | |
| `task_index.go` (284L) | `pipeline/cc/task_index.go` | |
| `task_scrape.go` (115L) | `pipeline/scrape/task_scrape.go` | |
| `task_scrape_md.go` (192L) | `pipeline/scrape/task_scrape_md.go` | |
| `scrape_store.go` (306L) | `pipeline/scrape/store.go` | |
| `scrape_handlers.go` (238L) | `api/scrape.go` | |
| Job handler code in `server.go` | `api/jobs.go` | ~60 lines |

### External consumers (must update imports)

| File | Uses | After refactor |
|---|---|---|
| `cli/cc_fts.go` | `web.IndexFromWARCMd` | `cc.IndexFromWARCMd` |
| `cli/cc_warc_pack.go` | `web.NewDocStore` | unchanged (stays in `web`) |
| `cli/cc_fts_web.go` | `web.New`, `web.NewDashboardWithOptions`, `web.DashboardOptions` | unchanged (stays in `web`) |

## Target Structure

```
pkg/index/web/
├── server.go              # Server, New, NewDashboard, Handler(), ListenAndServe
├── server_test.go         # (updated imports)
├── doc_store.go           # unchanged
├── domain_store.go        # unchanged
├── domain_cc_store.go     # unchanged
├── meta_manager.go        # unchanged
├── scanner.go             # unchanged
├── logging.go             # unchanged
├── duckdb_ops.go          # unchanged
├── disk_unix.go           # unchanged
├── sysinfo_*.go           # unchanged
├── handler_browse_export.go
├── handler_parquet.go
├── warc_api.go
├── warc_meta_scan.go
├── static/                # unchanged
├── metastore/             # unchanged
│
├── api/                   # HTTP handlers (Mizu)
│   ├── handler.go         # Deps struct, RegisterRoutes
│   ├── jobs.go            # list/get/create/cancel/clear
│   └── scrape.go          # start/resume/stop/list/status/pages/pipeline
│
└── pipeline/              # Task lifecycle framework
    ├── job.go             # Job, JobConfig, Broadcaster, wsEvent types
    ├── manager.go         # Manager (lifecycle, persistence, broadcast)
    ├── manager_test.go
    ├── executor.go        # RunJob, NonBlockingEmit, manifest resolution
    ├── ws.go              # Hub, Client
    ├── ws_test.go
    ├── overview.go        # Response, stage scanners, system/storage info
    ├── overview_test.go
    ├── helpers.go         # ParseFileSelector, WARCFileIndex, etc.
    │
    ├── cc/                # Common Crawl pipeline tasks
    │   ├── task_download.go
    │   ├── task_markdown.go
    │   ├── task_pack.go
    │   └── task_index.go  # includes IndexFromWARCMd (exported, used by CLI)
    │
    └── scrape/            # Scrape pipeline tasks + store
        ├── task_scrape.go
        ├── task_scrape_md.go
        ├── store.go       # Store (DuckDB reads)
        └── types.go       # Domain, Page, response types
```

## Design Principles

### Go naming (stdlib style)

- Package names: `pipeline`, `cc`, `scrape`, `api` — short, lowercase, no underscores
- No type stutter: `pipeline.Manager` not `pipeline.PipelineManager`; `cc.DownloadTask` not `cc.CCDownloadTask`; `scrape.Store` not `scrape.ScrapeStore`
- Constructors: `pipeline.NewManager`, `cc.NewDownloadTask`, `scrape.NewStore`

### Private functions over private methods

- Helper functions that don't need receiver state become package-level private functions
- Example: `emitDownloadProgress(emit, ...)` stays a function, not `(t *DownloadTask).emitProgress`
- `snapshotJob(job)` stays a function, not `(m *Manager).snapshotJob`
- `nonBlockingEmit[S]` stays a generic function

### Interfaces at the consumer

- `pipeline.Broadcaster` interface (satisfied by `pipeline.Hub`):
  ```go
  type Broadcaster interface {
      Broadcast(jobID string, msg any)
      BroadcastAll(msg any)
  }
  ```
- Manager accepts `Broadcaster`, not concrete `*Hub`
- `api.Deps` struct references concrete types (it's the composition root)

### core.Task compliance

All tasks implement `core.Task[State, Metric]` with compile-time assertions:
```go
var _ core.Task[DownloadState, DownloadMetric] = (*DownloadTask)(nil)
```

## Dependency Flow

```
server.go ──→ api/       (RegisterRoutes)
    │    ──→ pipeline/   (Manager, Hub, BuildOverview)
    │    ──→ pipeline/cc/
    │    ──→ pipeline/scrape/
    │
api/     ──→ pipeline/   (Manager, Job, JobConfig)
    │    ──→ pipeline/scrape/ (Store)
    │
pipeline/ ──→ pipeline/cc/     (task constructors, for executor dispatch)
    │     ──→ pipeline/scrape/  (task constructors, for executor dispatch)
    │     ──→ metastore         (JobRecord for persistence)
    │
pipeline/cc/     ──→ core       (Task interface)
pipeline/scrape/ ──→ core       (Task interface)
```

No circular dependencies. `pipeline/cc/` and `pipeline/scrape/` are leaf packages.

---

## Implementation Plan

### Task 1: Create `pipeline/` package — types and interfaces

**Files:**
- Create: `pkg/index/web/pipeline/job.go`
- Create: `pkg/index/web/pipeline/helpers.go`

**Step 1: Create `pipeline/job.go`**

Move from `jobs.go`: `JobConfig`, `Job`, `JobCompleteHook`, WS event types.
Add `Broadcaster` interface.

```go
package pipeline

import (
    "context"
    "time"
)

// JobConfig describes the parameters for a pipeline task.
type JobConfig struct {
    Type    string `json:"type"`
    CrawlID string `json:"crawl"`
    Files   string `json:"files"`
    Engine  string `json:"engine"`
    Source  string `json:"source"`
    Format  string `json:"format"`
    Domain  string `json:"domain,omitempty"`
}

// Job represents a single pipeline task tracked by the Manager.
type Job struct {
    ID        string     `json:"id"`
    Type      string     `json:"type"`
    Status    string     `json:"status"`
    Config    JobConfig  `json:"config"`
    Progress  float64    `json:"progress"`
    Message   string     `json:"message"`
    Rate      float64    `json:"rate,omitempty"`
    StartedAt time.Time  `json:"started_at"`
    EndedAt   *time.Time `json:"ended_at,omitempty"`
    Error     string     `json:"error,omitempty"`
    cancel    context.CancelFunc
}

// CompleteHook is called when a job transitions to completed status.
type CompleteHook func(job *Job, crawlID, crawlDir string)

// Broadcaster delivers real-time updates to connected clients.
type Broadcaster interface {
    Broadcast(jobID string, msg any)
    BroadcastAll(msg any)
}

// WS event payloads (exported for api/ package).

type JobUpdate struct {
    Type   string `json:"type"`
    JobID  string `json:"job_id"`
    Status string `json:"status"`
    Error  string `json:"error,omitempty"`
}

type JobProgress struct {
    Type     string  `json:"type"`
    JobID    string  `json:"job_id"`
    Progress float64 `json:"progress"`
    Message  string  `json:"message"`
    Rate     float64 `json:"rate"`
}
```

**Step 2: Create `pipeline/helpers.go`**

Move pure helper functions from `executors.go`:

```go
package pipeline

import (
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
)

// ParseFileSelector parses a file selector string into a list of indices.
// Supports: "0", "0-4", "all", "".
func ParseFileSelector(s string, total int) ([]int, error) { ... }

// WARCFileIndex extracts the zero-padded 5-digit WARC index from a path.
func WARCFileIndex(warcPath string, fallback int) string { ... }

// PackPath returns the expected pack file path for a format and WARC index.
func PackPath(packDir, format, warcIdx string) (string, error) { ... }

func PhaseProgress(done, total int64) float64 { ... }
func PhaseRate(done int64, elapsed time.Duration) float64 { ... }
func MBPerSec(bytes int64, elapsed time.Duration) float64 { ... }
func FileProgress(fileIdx, fileTotal int, fileFraction float64) float64 { ... }
func FileExists(path string) bool { ... }

// NonBlockingEmit wraps an emit callback with a buffered channel.
func NonBlockingEmit[S any](fn func(*S)) func(*S) { ... }
```

**Step 3: Run tests**

```
go build ./pkg/index/web/pipeline/...
```

**Step 4: Commit**

```
git add pkg/index/web/pipeline/
git commit -m "refactor(pipeline): add job types, helpers, broadcaster interface"
```

---

### Task 2: Create `pipeline/ws.go` — Hub and Client

**Files:**
- Create: `pkg/index/web/pipeline/ws.go`
- Create: `pkg/index/web/pipeline/ws_test.go`

Move `WSHub` → `pipeline.Hub`, `WSClient` → `pipeline.Client`. Hub implements `Broadcaster`.
Use private functions for `readPump`, `writePump`.

```go
package pipeline

// Hub satisfies Broadcaster.
var _ Broadcaster = (*Hub)(nil)

type Hub struct { ... }
func NewHub() *Hub { ... }
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) { ... }
func (h *Hub) Broadcast(jobID string, msg any) { ... }
func (h *Hub) BroadcastAll(msg any) { ... }
func (h *Hub) Close() { ... }

type Client struct { ... }
// readPump, writePump, isSubscribed as private functions taking *Client
```

**Step: Run tests**

```
go test ./pkg/index/web/pipeline/ -run TestHub -v
```

**Step: Commit**

```
git commit -m "refactor(pipeline): move WSHub/WSClient to pipeline.Hub/Client"
```

---

### Task 3: Create `pipeline/manager.go` — Manager lifecycle

**Files:**
- Create: `pkg/index/web/pipeline/manager.go`
- Create: `pkg/index/web/pipeline/manager_test.go`

Move Manager from `jobs.go`. Accept `Broadcaster` interface instead of `*WSHub`.
Private functions: `snapshotJob`, `enqueuePersist`, `persistFlusher`.

```go
package pipeline

func NewManager(bc Broadcaster, baseDir, crawlID string) *Manager { ... }
func (m *Manager) Create(cfg JobConfig) *Job { ... }
func (m *Manager) Get(id string) *Job { ... }
func (m *Manager) List() []*Job { ... }
func (m *Manager) Cancel(id string) bool { ... }
func (m *Manager) UpdateProgress(id string, pct float64, msg string, rate float64) { ... }
func (m *Manager) Complete(id string, msg string) { ... }
func (m *Manager) Fail(id string, err error) { ... }
func (m *Manager) SetRunning(id string, cancel context.CancelFunc) { ... }
func (m *Manager) Clear() int { ... }
func (m *Manager) SetCompleteHook(h CompleteHook) { ... }
func (m *Manager) SetStore(s metastore.Store) { ... }
func (m *Manager) LoadHistory(ctx context.Context) { ... }
func (m *Manager) StopPersist() { ... }

// private functions
func snapshotJob(job *Job) metastore.JobRecord { ... }
func enqueuePersist(ch chan metastore.JobRecord, rec metastore.JobRecord) { ... }
```

**Step: Run tests**

```
go test ./pkg/index/web/pipeline/ -v
```

**Step: Commit**

```
git commit -m "refactor(pipeline): move Manager to pipeline package"
```

---

### Task 4: Create `pipeline/overview.go` — stage scanners

**Files:**
- Create: `pkg/index/web/pipeline/overview.go`
- Create: `pkg/index/web/pipeline/overview_test.go`

Move `OverviewResponse`, all stage types, `buildOverviewResponse` → `BuildOverview`,
and all `scan*Stage`, `collect*` functions. These are all standalone functions
(no struct methods needed).

Rename for export:
- `buildOverviewResponse` → `BuildOverview`
- `scanDownloadedStage` → `scanDownloaded` (private, only used within package)
- `collectSystemInfo` → `collectSystem` (private)
- `collectStorageInfo` → `collectStorage` (private)

`DocStore` dependency: accept as an interface or pass doc count directly.
Simplest: pass `docCountFn func(shard string) int64` callback.

**Step: Commit**

```
git commit -m "refactor(pipeline): move overview stage scanners to pipeline"
```

---

### Task 5: Create `pipeline/cc/` — CC pipeline tasks

**Files:**
- Create: `pkg/index/web/pipeline/cc/task_download.go`
- Create: `pkg/index/web/pipeline/cc/task_markdown.go`
- Create: `pkg/index/web/pipeline/cc/task_pack.go`
- Create: `pkg/index/web/pipeline/cc/task_index.go`

Each file moves its task from `web/task_*.go` to `cc/`.
Change `package web` → `package cc`.
Import `pipeline` for helpers (`pipeline.WARCFileIndex`, `pipeline.PhaseProgress`, etc.).
Add `core.Task` compile-time assertion per task.

Parquet download logic (currently inline in `executors.go:runParquetDownloadJob`)
stays in `pipeline/executor.go` since it's tightly coupled to Manager + cc.Client.

**Example for download:**

```go
package cc

import "github.com/go-mizu/mizu/blueprints/search/pkg/core"

var _ core.Task[DownloadState, DownloadMetric] = (*DownloadTask)(nil)

type DownloadTask struct { ... }
type DownloadState struct { ... }
type DownloadMetric struct { ... }

func NewDownloadTask(crawlDir string, paths []string, selected []int) *DownloadTask { ... }
func (t *DownloadTask) Run(ctx context.Context, emit func(*DownloadState)) (DownloadMetric, error) { ... }
```

**Step: Build**

```
go build ./pkg/index/web/pipeline/cc/...
```

**Step: Commit**

```
git commit -m "refactor(pipeline/cc): move CC pipeline tasks (download, markdown, pack, index)"
```

---

### Task 6: Create `pipeline/scrape/` — scrape tasks + store

**Files:**
- Create: `pkg/index/web/pipeline/scrape/task_scrape.go`
- Create: `pkg/index/web/pipeline/scrape/task_scrape_md.go`
- Create: `pkg/index/web/pipeline/scrape/store.go`
- Create: `pkg/index/web/pipeline/scrape/types.go`

Move `ScrapeTask` → `scrape.Task`, `ScrapeStore` → `scrape.Store`.
Move `ScrapeMarkdownTask` → `scrape.MarkdownTask`.
Move all scrape types (ScrapeDomain, ScrapePage, etc.) to `types.go`.
Move `buildDCrawlerConfig` → `scrape.BuildCrawlerConfig` (used by api/).

```go
package scrape

import "github.com/go-mizu/mizu/blueprints/search/pkg/core"

var _ core.Task[TaskState, TaskMetric] = (*Task)(nil)
var _ core.Task[MarkdownState, MarkdownMetric] = (*MarkdownTask)(nil)
```

**Step: Build**

```
go build ./pkg/index/web/pipeline/scrape/...
```

**Step: Commit**

```
git commit -m "refactor(pipeline/scrape): move scrape tasks and store"
```

---

### Task 7: Create `pipeline/executor.go` — RunJob dispatch

**Files:**
- Create: `pkg/index/web/pipeline/executor.go`

Move `RunJob` and all `run*Job` methods from `executors.go`.
Import `cc` and `scrape` packages for task constructors.
Keep `resolveFiles`, `getManifestPaths`, `resolveJobCrawl` as Manager methods
(they need Manager state).
Keep parquet download logic here.

```go
package pipeline

func (m *Manager) RunJob(job *Job) { ... }

// private per-type runners
func (m *Manager) runDownloadJob(ctx context.Context, job *Job) error { ... }
func (m *Manager) runMarkdownJob(ctx context.Context, job *Job) error { ... }
// ... etc
```

**Step: Build**

```
go build ./pkg/index/web/pipeline/...
```

**Step: Commit**

```
git commit -m "refactor(pipeline): move RunJob executor dispatch"
```

---

### Task 8: Create `api/` — HTTP handlers

**Files:**
- Create: `pkg/index/web/api/handler.go`
- Create: `pkg/index/web/api/jobs.go`
- Create: `pkg/index/web/api/scrape.go`

**handler.go**: Deps struct + RegisterRoutes

```go
package api

import (
    mizu "github.com/go-mizu/mizu"
    "github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline"
    "github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/scrape"
)

// Deps holds dependencies for pipeline API handlers.
type Deps struct {
    Manager *pipeline.Manager
    Scrape  *scrape.Store
    BaseDir string
    CrawlID string
}

// RegisterRoutes registers pipeline API routes on the router.
func RegisterRoutes(router *mizu.Router, deps Deps) {
    h := &handler{deps: deps}
    router.Get("/api/jobs", h.listJobs)
    router.Get("/api/jobs/{id}", h.getJob)
    router.Post("/api/jobs", h.createJob)
    router.Delete("/api/jobs/{id}", h.cancelJob)
    router.Delete("/api/jobs", h.clearJobs)

    router.Post("/api/scrape", h.scrapeStart)
    router.Get("/api/scrape/list", h.scrapeList)
    router.Post("/api/scrape/{domain}/resume", h.scrapeResume)
    router.Delete("/api/scrape/{domain}", h.scrapeStop)
    router.Get("/api/scrape/{domain}/status", h.scrapeStatus)
    router.Get("/api/scrape/{domain}/pages", h.scrapePages)
    router.Post("/api/scrape/{domain}/pipeline", h.scrapePipeline)
}

type handler struct {
    deps Deps
}
```

**jobs.go**: Move `handleListJobs`, `handleGetJob`, `handleCreateJob`, `handleCancelJob`, `handleClearJobs` from `server.go`.

**scrape.go**: Move all handlers from `scrape_handlers.go`. Move `findActiveScrapeJob` as private function.

**Step: Build**

```
go build ./pkg/index/web/api/...
```

**Step: Commit**

```
git commit -m "refactor(api): extract job and scrape HTTP handlers"
```

---

### Task 9: Wire everything in `server.go`

**Files:**
- Modify: `pkg/index/web/server.go`

1. Replace `*WSHub` with `*pipeline.Hub`
2. Replace `*Manager` with `*pipeline.Manager`
3. Replace `*ScrapeStore` with `*scrape.Store`
4. Remove `handleListJobs`, `handleGetJob`, `handleCreateJob`, `handleCancelJob`, `handleClearJobs` methods
5. Remove scrape handler methods
6. In `Handler()`, call `api.RegisterRoutes(router, api.Deps{...})` for pipeline routes
7. Update `NewDashboard` to construct `pipeline.Hub`, `pipeline.NewManager`
8. Update overview handler to call `pipeline.BuildOverview(...)`

**Step: Build + test**

```
go build ./pkg/index/web/...
go test ./pkg/index/web/... -count=1
```

**Step: Commit**

```
git commit -m "refactor(web): wire pipeline and api packages into server"
```

---

### Task 10: Delete old files + update external imports

**Files:**
- Delete: `pkg/index/web/jobs.go`, `jobs_test.go`
- Delete: `pkg/index/web/executors.go`
- Delete: `pkg/index/web/ws.go`, `ws_test.go`
- Delete: `pkg/index/web/overview.go`, `overview_test.go`
- Delete: `pkg/index/web/task_download.go`, `task_markdown.go`, `task_pack.go`, `task_index.go`
- Delete: `pkg/index/web/task_scrape.go`, `task_scrape_md.go`
- Delete: `pkg/index/web/scrape_handlers.go`, `scrape_store.go`
- Modify: `cli/cc_fts.go` — `web.IndexFromWARCMd` → `cc.IndexFromWARCMd`

**Step: Build + full test**

```
go build ./...
go test ./pkg/index/web/... -count=1
```

**Step: Commit**

```
git commit -m "refactor: delete old files, update external imports"
```

---

### Task 11: Update `server_test.go`

**Files:**
- Modify: `pkg/index/web/server_test.go`

Update type references:
- `Job` → `pipeline.Job`
- `JobConfig` → `pipeline.JobConfig`
- `JobsListResponse` → keep as `web.JobsListResponse` or inline
- `OverviewResponse` → `pipeline.Response`
- `NewWSHub` → `pipeline.NewHub`

All tests should pass without behavior changes.

**Step: Test**

```
go test ./pkg/index/web/... -count=1 -v
```

**Step: Commit**

```
git commit -m "refactor: update server_test.go for pipeline types"
```

---

## Verification

Final check after all tasks:

```bash
go build ./...
go test ./pkg/index/web/... -count=1
go test ./pkg/index/web/pipeline/... -count=1
go test ./pkg/core/... -count=1
go vet ./pkg/index/web/...
```

No behavior changes — only package reorganization.
