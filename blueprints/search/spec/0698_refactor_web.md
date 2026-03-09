# 0698 — Refactor pkg/index/web: Remove Old Duplicates, Use pipeline/ Everywhere

## Problem

`pkg/index/web/` has a partially-completed migration from a monolithic server to a
layered architecture with `pipeline/`, `api/`, and `metastore/` subdirectories. Both
old and new code exist simultaneously, causing ~2,500 lines of pure duplication:

| Old (web/) | New (pipeline/) | Status |
|---|---|---|
| `jobs.go` (Job, JobConfig, Manager, WSHub events) | `pipeline/job.go` + `pipeline/manager.go` | 100% duplicate |
| `executors.go` (RunJob, per-task adapters, helpers) | `pipeline/executor.go` | 100% duplicate |
| `ws.go` (WSHub, WSClient) | `pipeline/ws.go` (Hub, client) | 100% duplicate |
| `task_download.go` | `pipeline/cc/task_download.go` | 100% duplicate |
| `task_markdown.go` | `pipeline/cc/task_markdown.go` | 100% duplicate |
| `task_pack.go` | `pipeline/cc/task_pack.go` | 100% duplicate |
| `task_index.go` | `pipeline/cc/task_index.go` | 100% duplicate |
| `task_scrape.go` | `pipeline/scrape/task_scrape.go` | 100% duplicate |
| `task_scrape_md.go` | `pipeline/scrape/task_scrape_md.go` | 100% duplicate |
| `scrape_store.go` (ScrapeStore) | `pipeline/scrape/store.go` (Store) | 100% duplicate |
| `scrape_handlers.go` (handleScrape*) | `api/scrape.go` (startScrape etc.) | 100% duplicate |
| Job handlers in `server.go` L1075-1127 | `api/jobs.go` | 100% duplicate |

The new `api/` and `pipeline/` packages are **not wired into server.go** — the old code
is still active. The new code is dead/unused at runtime.

## Solution

1. **Delete old duplicate files** from `web/` package
2. **Update `server.go`** to use `pipeline.Manager`, `pipeline.Hub`, `scrape.Store`
3. **Wire `api/` routes** into server.go's Handler() method
4. **Keep non-duplicated files** (doc_store.go, domain_store.go, handler_parquet.go, etc.)

## External API surface (must not break)

- `cli/cc_fts_web.go` → `web.New()`, `web.NewDashboardWithOptions()`, `web.DashboardOptions`
- `cli/cc_warc_pack.go` → `web.NewDocStore()`
- `cli/cc_fts.go` → `pipcc.IndexFromWARCMd()` (already uses pipeline/cc directly)

## Files to DELETE (old duplicates)

```
pkg/index/web/jobs.go              → replaced by pipeline/job.go + pipeline/manager.go
pkg/index/web/executors.go         → replaced by pipeline/executor.go
pkg/index/web/ws.go                → replaced by pipeline/ws.go
pkg/index/web/task_download.go     → replaced by pipeline/cc/task_download.go
pkg/index/web/task_markdown.go     → replaced by pipeline/cc/task_markdown.go
pkg/index/web/task_pack.go         → replaced by pipeline/cc/task_pack.go
pkg/index/web/task_index.go        → replaced by pipeline/cc/task_index.go
pkg/index/web/task_scrape.go       → replaced by pipeline/scrape/task_scrape.go
pkg/index/web/task_scrape_md.go    → replaced by pipeline/scrape/task_scrape_md.go
pkg/index/web/scrape_store.go      → replaced by pipeline/scrape/store.go
pkg/index/web/scrape_handlers.go   → replaced by api/scrape.go
```

## Changes to server.go

1. Replace `Hub *WSHub` → `Hub *pipeline.Hub`
2. Replace `Jobs *Manager` → `Jobs *pipeline.Manager`
3. Replace `Scrape *ScrapeStore` → `Scrape *scrape.Store`
4. `NewDashboardWithOptions`: create `pipeline.Hub`, `pipeline.Manager`, `scrape.Store`
5. Handler(): use `api.RegisterJobRoutes()` and `api.RegisterScrapeRoutes()` instead of
   inline `handleListJobs`, `handleCreateJob`, etc.
6. Remove `handleListJobs`, `handleGetJob`, `handleCreateJob`, `handleCancelJob`,
   `handleClearJobs`, and all `handleScrape*` methods from server.go
7. Update `ListenAndServe` shutdown to call `s.Hub.Close()`
8. Update `JobsListResponse` type alias to use `pipeline.Job`
9. Update `/ws` handler to use `s.Hub.HandleWS`
10. Update `SetCompleteHook` callback signature to `pipeline.CompleteHook`

## Test files

- `jobs_test.go` — depends on old Manager; must be updated to import pipeline.Manager
- `ws_test.go` — depends on old WSHub; must be updated to import pipeline.Hub

## Verification

```bash
go build ./pkg/index/web/...
go build ./cli/...
go test ./pkg/index/web/...
```
