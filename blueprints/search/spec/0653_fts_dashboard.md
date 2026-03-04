# 0653 — FTS Dashboard

> Full admin dashboard for CC FTS pipeline: crawl management, WARC download,
> markdown extraction, packing, indexing, search, data browsing, and stats.

## Overview

Replace `search cc fts web` (search-only SPA) with `search cc fts dashboard` — a
unified admin panel that mirrors every CLI pipeline step in a web GUI with real-time
progress via WebSocket.

**Architecture**: Single embedded HTML (go:embed) + htmx for tab navigation and form
submission + vanilla JS for WebSocket progress streams. No build step, no node_modules.

**CLI**: `search cc fts dashboard [--port 3456] [--crawl CC-MAIN-2026-08] [--open]`

## Data Directory

```
~/data/common-crawl/
├── cache.json                      # 24h cache (crawl list, manifests)
├── {crawlID}/
│   ├── warc.paths.gz               # WARC manifest
│   ├── warc/                       # Downloaded WARC files (~1GB each)
│   ├── markdown/{warcIdx}/**/*.md  # Extracted markdown
│   ├── pack/{format}/{warcIdx}.*   # Packed bundles (parquet/bin/duckdb/gz)
│   ├── fts/{engine}/{warcIdx}/     # Per-engine, per-WARC index shards
│   └── index.duckdb                # Columnar CC index
```

## HTTP API

### Existing (kept from fts web)

| Method | Path | Purpose |
|--------|------|---------|
| `GET`  | `/api/search?q=&limit=&offset=` | Fan-out shard search |
| `GET`  | `/api/stats` | Aggregated index stats |
| `GET`  | `/api/doc/{shard}/{docid...}` | Document fetch + MD→HTML |
| `GET`  | `/api/browse?shard=` | Shard/file listing |

### New

| Method | Path | Purpose |
|--------|------|---------|
| `GET`  | `/api/overview` | Dashboard summary (crawl, sizes, counts) |
| `GET`  | `/api/crawls` | Available CC crawls (cached) |
| `GET`  | `/api/crawl/{id}/warcs` | WARC manifest + local download status |
| `GET`  | `/api/crawl/{id}/data` | Data breakdown (warc/md/pack/fts sizes) |
| `GET`  | `/api/engines` | Registered FTS engine names |
| `POST` | `/api/jobs` | Start a pipeline job |
| `GET`  | `/api/jobs` | List all jobs (running + history) |
| `GET`  | `/api/jobs/{id}` | Single job detail |
| `DELETE`| `/api/jobs/{id}` | Cancel running job |
| `WS`   | `/ws` | WebSocket for real-time progress |

### Job Types

```json
// POST /api/jobs
{"type": "download", "crawl": "CC-MAIN-2026-08", "files": "0-4"}
{"type": "markdown", "crawl": "CC-MAIN-2026-08", "files": "0", "fast": true}
{"type": "pack",     "crawl": "CC-MAIN-2026-08", "files": "0", "format": "parquet"}
{"type": "index",    "crawl": "CC-MAIN-2026-08", "files": "0", "engine": "duckdb", "source": "files"}
```

## WebSocket Protocol

Server → Client:
```json
{"type":"progress","job_id":"j1","pct":0.45,"msg":"indexing 2250/5000 docs","rate":1200,"elapsed":"3.8s"}
{"type":"complete","job_id":"j1","stats":{"docs":5000,"elapsed":"4.2s","disk":"12.3 MB"}}
{"type":"failed","job_id":"j1","error":"connection refused"}
{"type":"cancelled","job_id":"j1"}
{"type":"log","job_id":"j1","line":"creating FTS index (BM25)..."}
```

Client → Server:
```json
{"type":"subscribe","job_ids":["j1","j2"]}
{"type":"unsubscribe","job_ids":["j1"]}
```

## UI Tabs

### 1. Overview (default)

Stat cards row:
- Crawl ID + detected date
- Total docs indexed (sum across engines/shards)
- Total disk usage (warc + markdown + pack + fts)
- WARC files downloaded / total available
- Engines with indexes

Data breakdown table:
| Category | Files | Size | Path |
|----------|-------|------|------|
| WARC | 3 / 90,000 | 2.8 GB | ~/data/.../warc/ |
| Markdown | 3 shards | 450 MB | ~/data/.../markdown/ |
| Pack (parquet) | 3 files | 180 MB | ~/data/.../pack/parquet/ |
| FTS (duckdb) | 3 shards | 95 MB | ~/data/.../fts/duckdb/ |

Quick-action buttons: Download next WARC, Build index, Open search.

### 2. Pipeline

Vertical step-by-step view. Each step shows per-WARC-file status:

```
Step 1: Download WARC
  [00000] ████████████████████ 100%  1.02 GB  done
  [00001] ████████░░░░░░░░░░░░  42%  430 MB   downloading...
  [00002] ░░░░░░░░░░░░░░░░░░░░   —   —        queued
  [Start] [Cancel]

Step 2: Extract Markdown
  [00000] ████████████████████ 100%  21,340 docs  done
  [00001] ░░░░░░░░░░░░░░░░░░░░   —   —           waiting for WARC
  [Start] [Cancel]

Step 3: Pack (format selector: parquet | bin | duckdb | markdown)
  ...

Step 4: Build Index (engine selector: duckdb | sqlite | rose | ...)
  [00000] ████████████████████ 100%  21,340 docs  1,200 docs/s  4.2s
  ...
```

Each step has a form (htmx POST to `/api/jobs`) with relevant options.
Progress bars update via WebSocket.

### 3. Search

Migrated from existing `fts web` SPA. Same UI, same API.

### 4. Browse

Migrated from existing `fts web` browse view. Shard sidebar + file list + doc viewer.

### 5. Crawls

Table of available CC crawls (from `/api/crawls`):
- Crawl ID, date range, status (local data? indexed?)
- Click to set as active crawl
- Button to download manifest

## Go Architecture

### Package Changes

**`pkg/index/web/`** — enhanced:
- `server.go` → add new API handlers, WebSocket upgrade, job manager
- `jobs.go` (new) → `JobManager` + `Job` types, goroutine lifecycle
- `ws.go` (new) → WebSocket hub (broadcast to subscribers)
- `static/index.html` → new dashboard SPA (replaces search-only SPA)

**`cli/cc_fts_web.go`** → rename command to `dashboard`, add `--crawl` default detection

### Job Manager

```go
type JobManager struct {
    mu      sync.RWMutex
    jobs    map[string]*Job
    hub     *WSHub
    baseDir string // ~/data/common-crawl/{crawlID}
    crawlID string
}

type Job struct {
    ID        string    `json:"id"`
    Type      string    `json:"type"`      // download, markdown, pack, index
    Status    string    `json:"status"`    // queued, running, completed, failed, cancelled
    Config    JobConfig `json:"config"`
    Progress  float64   `json:"progress"`  // 0.0–1.0
    Message   string    `json:"message"`
    Rate      float64   `json:"rate"`      // items/sec
    StartedAt time.Time `json:"started_at"`
    EndedAt   time.Time `json:"ended_at,omitempty"`
    Error     string    `json:"error,omitempty"`
    Stats     any       `json:"stats,omitempty"`
    cancel    context.CancelFunc
}

type JobConfig struct {
    CrawlID string `json:"crawl"`
    Files   string `json:"files"`   // "0", "0-4", "all"
    Engine  string `json:"engine"`  // for index jobs
    Source  string `json:"source"`  // for index jobs
    Format  string `json:"format"`  // for pack jobs
    Fast    bool   `json:"fast"`    // for markdown jobs (go-readability)
}
```

### WebSocket Hub

```go
type WSHub struct {
    mu      sync.RWMutex
    clients map[*WSClient]struct{}
}

type WSClient struct {
    conn   *websocket.Conn
    subs   map[string]bool // subscribed job IDs ("*" = all)
    sendCh chan []byte
}
```

Broadcast model: JobManager calls `hub.Broadcast(jobID, msg)`. Hub iterates
clients, sends to those subscribed to that job ID or "*".

### Data Scanner

`/api/overview` and `/api/crawl/{id}/data` use a lightweight scanner that
walks the data directory to compute sizes and counts without opening DuckDB:

```go
type DataSummary struct {
    CrawlID       string            `json:"crawl_id"`
    WARCCount     int               `json:"warc_count"`
    WARCTotalSize int64             `json:"warc_total_size"`
    WARCAvailable int               `json:"warc_available"`
    MDShards      int               `json:"md_shards"`
    MDTotalSize   int64             `json:"md_total_size"`
    MDDocEstimate int               `json:"md_doc_estimate"`
    PackFormats   map[string]int64  `json:"pack_formats"`   // format → total size
    FTSEngines    map[string]int64  `json:"fts_engines"`    // engine → total size
    FTSShardCount map[string]int    `json:"fts_shard_count"`
}
```

## Style

Continues spec/0652 brutalist design:
- Dark zinc-950 default, Geist font, no rounded corners
- Stat cards: border-only boxes with mono labels
- Progress bars: `bg-zinc-800` track, `bg-zinc-200` fill (no gradients, no colors)
- Tables: minimal lines, mono font for numbers
- htmx for tab switching (`hx-get`, `hx-target="#main"`, `hx-push-url`)
- Tabs in header: Overview | Pipeline | Search | Browse | Crawls

## Implementation Plan

### Phase 1: Backend Foundation (jobs + WebSocket)
1. Create `pkg/index/web/ws.go` — WSHub + WSClient + upgrade handler
2. Create `pkg/index/web/jobs.go` — JobManager + Job lifecycle
3. Add new API routes to `server.go` (overview, crawls, data, engines, jobs, ws)
4. Wire JobManager to existing CLI functions (download, markdown, pack, index)
5. Add data scanner for directory stats

### Phase 2: Dashboard Frontend
6. Replace `static/index.html` with new dashboard SPA
7. Implement tab navigation with htmx
8. Overview tab: stat cards + data table + quick actions
9. Pipeline tab: step forms + progress bars + WebSocket client
10. Migrate search + browse tabs from existing HTML

### Phase 3: Crawl Management
11. `/api/crawls` endpoint (wraps `cc.Client.ListCrawls`)
12. `/api/crawl/{id}/warcs` endpoint (manifest + local file check)
13. Crawls tab UI
14. Set active crawl from dashboard

### Phase 4: Polish
15. Job history persistence (in-memory is fine for v1, survives page reload via API)
16. Error handling + edge cases (no data dir, cancelled downloads)
17. Keyboard shortcuts (Cmd+K search, tab navigation)
18. Mobile responsive layout
