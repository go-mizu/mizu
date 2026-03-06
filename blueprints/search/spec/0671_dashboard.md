# 0691 — FTS Dashboard Consistency & DX Review

**Date:** 2026-03-06
**Scope:** `search cc fts dashboard` — full-stack review of `pkg/index/web/` + `cli/cc_fts_web.go`
**Status:** All issues fixed

## Architecture Summary

The FTS dashboard is a dual-mode SPA (search-only `web` vs full-admin `dashboard`) served from a single `index.html` template with mode injection (`__SERVER_MODE__`). Go backend provides REST API + WebSocket for real-time job progress. Metadata cache (SQLite/DuckDB) backs the WARC console and overview stats.

**Files reviewed:**
- `pkg/index/web/server.go` — routes, handlers, core API
- `pkg/index/web/jobs.go` — job manager, state machine
- `pkg/index/web/executors.go` — download/markdown/pack/index executors
- `pkg/index/web/warc_api.go` — WARC list/detail/action handlers
- `pkg/index/web/warc_meta_scan.go` — filesystem scan + WARC record builder
- `pkg/index/web/scanner.go` — DataSummary scan
- `pkg/index/web/meta_manager.go` — metadata cache manager
- `pkg/index/web/ws.go` — WebSocket hub + client
- `pkg/index/web/logging.go` — request logger
- `pkg/index/web/static/index.html` — 2167-line SPA (HTML/CSS/JS)
- `cli/cc_fts_web.go` — CLI registration for `web` and `dashboard`
- `cli/cc_fts.go` — parent command, `index`, `search`, `pack`

---

## Issues Found & Fixed

### 1. Tab comment numbering error (HTML)

**File:** `index.html` — JS section comments

The tab numbering in HTML comments had a duplicate `Tab 4`:
- Tab 4 was used for both "Search" (line 1418) and "WARC" (line 1583)
- Tab 5 said "Browse" but should be 6
- Tab 6 said "Crawls" but should be 7

**Fix:** Corrected to: Tab 1=Overview, 2=Pipeline, 3=Jobs, 4=Search, 5=WARC, 6=Browse, 7=Crawls.

### 2. Default engine inconsistency between `web` and `dashboard`

**File:** `cli/cc_fts_web.go`

- `web` defaulted to `--engine tantivy`
- `dashboard` defaulted to `--engine duckdb`
- All other commands (`index`, `search`, `pack`) default to `duckdb`

Users who build an index with `search cc fts index` (duckdb) then launch `search cc fts web` (tantivy) would get "no FTS index" because the engine doesn't match.

**Fix:** Unified all default engines to `rose` across all FTS commands (`web`, `dashboard`, `index`, `search`) and the executor fallback. The JS frontend now reads `DEFAULT_ENGINE` injected by the server instead of hardcoding.

### 3. `currentSearchEngine()` JS fallback always returned 'duckdb'

**File:** `index.html` line 614

The JS fallback `return 'duckdb'` was hardcoded regardless of what the server was configured with. In web mode, the server might be configured with tantivy or sqlite.

**Fix:** Server now injects the configured engine name alongside `__SERVER_MODE__`. The JS reads it as `DEFAULT_ENGINE` and uses it as the fallback instead of hardcoding 'duckdb'.

### 4. Browse page "No index" message gave wrong guidance

**File:** `index.html` line 1887

The message said: "No index yet. Run `search cc fts index` to get started."
But the Browse tab browses **markdown** documents, not the FTS index. If there are no shards, it's because markdown hasn't been extracted.

**Fix:** Changed to: "No markdown documents found. Run the download and markdown pipeline steps first."

### 5. WARC list table missing pack action button

**File:** `index.html` — `renderWARCContent`

The WARC list showed status chips `dl`, `md`, `pk`, `ix` but action buttons only for `dl`, `md`, `ix` — missing `pk` (pack). Pack is a required intermediate step in the pipeline (markdown -> pack -> index from pack), so omitting it creates a workflow gap.

**Fix:** Added `pk` action button between `md` and `ix` in the WARC list table.

### 6. WARC detail action feedback lost on page re-render

**File:** `index.html` — `warcAction` function

When an action was triggered with `refreshDetail=true`, the function set `warc-action-msg` text, then immediately called `renderWARCDetail(index)` which replaced the entire DOM — the user never saw the message.

**Fix:** Added a small delay (400ms) before re-rendering the detail page so the user sees the feedback. Also persist action message in state so it survives the re-render.

### 7. Overview "Engines" card label was ambiguous

**File:** `index.html` — `renderOverviewContent`

The card labeled "Engines" counted FTS engine types with data on disk. This could be confused with "available engine drivers" (there are 10+).

**Fix:** Renamed to "FTS Engines" for clarity.

### 8. Pipeline page duplicated full Job History table

**File:** `index.html` — `renderPipelineContent`

The Pipeline page rendered the complete job history table (same as Jobs tab), creating redundant content across two tabs. The Pipeline page's purpose is workflow management, not job history.

**Fix:** Replaced the full job history table on Pipeline with a compact "Recent Jobs" section showing only the last 5 jobs in a condensed format, with a link to the Jobs tab for full history.

### 9. Jobs page polling redundant with WebSocket

**File:** `index.html` — `renderJobs`

The Jobs page set up both WebSocket subscription AND a 5-second polling interval. With WebSocket working, the polling was redundant and added unnecessary API load (GET /api/jobs every 5s).

**Fix:** Changed polling to only activate when WebSocket is disconnected. The `wsClient` now exposes a `connected` check, and the polling interval only fires when `!wsClient.connected`.

### 10. `refreshDashboardContext` double-fetched overview data

**File:** `index.html` — `renderOverview`

The overview page called `refreshDashboardContext()` (which fetches `/api/overview`) and then also called `apiOverview()` separately — double-fetching the same data.

**Fix:** Removed the redundant `apiOverview()` call; `renderOverview` now uses `state.overview` populated by `refreshDashboardContext()`.

### 11. Browse shard file_count showed empty string for 0

**File:** `index.html` — `renderShardList`

`(s.file_count || '').toLocaleString()` evaluated to empty string when `file_count` was 0 due to JS falsy behavior.

**Fix:** Changed to `(s.file_count ?? 0).toLocaleString()` to properly display "0".

### 12. `fmtBytes` didn't handle 0/negative gracefully

**File:** `index.html` — `fmtBytes`

`Math.log(0)` returns `-Infinity`, `Math.log(-1)` returns `NaN`. While unlikely to occur, defensive handling prevents display glitches.

**Fix:** Added guard: return '0 B' for values <= 0.

---

## Design Observations (not changed)

### Duplicated Go functions (acceptable)
`warcIndexFromPath` and `packFilePath` are duplicated in `cli/cc_fts.go` and `pkg/index/web/executors.go`. The comment in executors.go explains: "Duplicated from cli/ to avoid circular import (cli imports web)." This is acceptable — extracting to a shared package would be over-engineering for two small pure functions.

### Metadata refresh button redundancy
"Refresh Metadata" appears on Overview, Pipeline, WARC, Browse, Crawls, and header bar (6 places). This is intentional — each page context benefits from local refresh access since the dashboard may be left open on any tab.

### Crawls page is read-only
The Crawls page lists CC crawls but doesn't allow switching the active crawl. This is by design — the active crawl is set at server startup via `--crawl` flag. Dynamic crawl switching would require server-side state changes and data directory re-scanning, which is out of scope for the current dashboard.

### WebSocket origin check
`CheckOrigin: func(r *http.Request) bool { return true }` — permissive for dev convenience. Acceptable for a local-only dashboard; would need CORS restriction for production deployment.
