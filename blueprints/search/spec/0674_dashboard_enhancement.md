# 0674 — Dashboard Overview Enhancement

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rewrite the dashboard overview page with pipeline-centric layout, per-stage detail panels, and system monitoring widgets. Remove stale pages (Pipeline, Crawls). Clean up navbar.

**Architecture:** Replace the current `/api/overview` with a new `OverviewResponse` Go struct containing 5 pipeline stages (manifest, downloaded, markdown, pack, indexed), each with phase-specific stats. Add system info (memory, disk, runtime). Frontend renders each stage as a standalone panel with a full-width progress summary bar above them.

**Tech Stack:** Go (net/http, runtime), vanilla JS SPA (existing index.html), Tailwind CSS

---

## Current State

The overview page has 6 loosely organized widgets: metric cards, pipeline progress bar, donut chart, engine/pack bars, active jobs, and process stats. The data comes from `DataSummaryWithMeta` which is a flat struct with maps (`PackFormats map[string]int64`, `FTSEngines map[string]int64`).

**Problems:**
- No pipeline narrative — widgets are disconnected
- No Common Crawl manifest total (100K WARCs) — can't show progress against full crawl
- No per-stage detail (docs/WARC, compression ratio, avg sizes)
- Pack shows 4 formats (only parquet + .md.warc.gz matter)
- FTS shows bleve/duckdb (only dahlia + tantivy matter)
- System stats show only heap/stack/goroutines — no disk, no runtime info
- Navbar has stale pages (Pipeline, Crawls) and border-bottom

## Design

### Navbar

Remove `Pipeline` and `Crawls` tabs. Reorder to: `Overview | Search | Browse | WARC | Jobs`.
Make full-width (remove `max-w-6xl mx-auto` from header). Remove border-bottom. Keep search input, engine selector, theme toggle, meta refresh indicator.

### `/api/overview` Response — `OverviewResponse` struct

```go
type OverviewResponse struct {
    CrawlID   string    `json:"crawl_id"`
    CrawlName string    `json:"crawl_name"`
    CrawlFrom string    `json:"crawl_from"`
    CrawlTo   string    `json:"crawl_to"`

    Manifest   ManifestStage   `json:"manifest"`
    Downloaded DownloadedStage `json:"downloaded"`
    Markdown   MarkdownStage   `json:"markdown"`
    Pack       PackStage       `json:"pack"`
    Indexed    IndexedStage    `json:"indexed"`

    Storage StorageInfo `json:"storage"`
    System  SystemInfo  `json:"system"`

    Meta OverviewMeta `json:"meta"`
}

type ManifestStage struct {
    TotalWARCs       int   `json:"total_warcs"`
    EstTotalSizeBytes int64 `json:"est_total_size_bytes"` // total_warcs * avg_warc_size
    EstTotalURLs     int64 `json:"est_total_urls"`       // total_warcs * ~30K URLs/WARC
}

type DownloadedStage struct {
    Count     int   `json:"count"`
    TotalWARCs int  `json:"total_warcs"` // same as manifest for easy %
    SizeBytes int64 `json:"size_bytes"`
    AvgWARCBytes int64 `json:"avg_warc_bytes"`
}

type MarkdownStage struct {
    Count       int   `json:"count"`       // WARCs with .md.warc.gz
    TotalWARCs  int   `json:"total_warcs"` // = downloaded count
    SizeBytes   int64 `json:"size_bytes"`  // total .md.warc.gz size
    TotalDocs   int64 `json:"total_docs"`  // from DocStore or estimate
    AvgDocsPerWARC int64 `json:"avg_docs_per_warc"`
    AvgDocBytes int64 `json:"avg_doc_bytes"`
}

type PackStage struct {
    Count      int   `json:"count"`       // WARCs with parquet
    TotalWARCs int   `json:"total_warcs"` // = downloaded count
    ParquetBytes int64 `json:"parquet_bytes"`
    WARCMdBytes  int64 `json:"warc_md_bytes"` // .md.warc.gz (same as markdown size)
}

type IndexedStage struct {
    Count      int   `json:"count"`       // WARCs with dahlia/tantivy
    TotalWARCs int   `json:"total_warcs"` // = downloaded count
    DahliaBytes  int64 `json:"dahlia_bytes"`
    DahliaShards int   `json:"dahlia_shards"`
    TantivyBytes  int64 `json:"tantivy_bytes"`
    TantivyShards int   `json:"tantivy_shards"`
}

type StorageInfo struct {
    DiskTotal int64 `json:"disk_total"`
    DiskUsed  int64 `json:"disk_used"`
    DiskFree  int64 `json:"disk_free"`
    CrawlBytes int64 `json:"crawl_bytes"` // total local data for this crawl
    ProjectedFullBytes int64 `json:"projected_full_bytes"` // if all 100K WARCs downloaded
}

type SystemInfo struct {
    HeapAlloc   int64  `json:"heap_alloc"`
    HeapSys     int64  `json:"heap_sys"`
    StackInuse  int64  `json:"stack_inuse"`
    NumGC       int64  `json:"num_gc"`
    Goroutines  int    `json:"goroutines"`
    GoVersion   string `json:"go_version"`
    Uptime      int64  `json:"uptime_seconds"`
    PID         int    `json:"pid"`
    GOMEMLIMIT  int64  `json:"gomemlimit"`
}

type OverviewMeta struct {
    Backend     string `json:"backend"`
    GeneratedAt string `json:"generated_at"`
    Stale       bool   `json:"stale"`
    Refreshing  bool   `json:"refreshing"`
}
```

### Frontend Layout

```
┌─────────────────────────────────────────────────────┐
│  CC-MAIN-2026-08 · February 2026 · Feb 6–19        │  Crawl Banner
│  100,000 WARCs · ~3B URLs · ~100 TB estimated       │
├─────────────────────────────────────────────────────┤
│ ████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ │  Pipeline Summary Bar
│ Manifest 100K → Downloaded 7 → MD 1 → Pack 4 → FTS 4│  (full-width, compact)
├─────────────────────────────────────────────────────┤
│ Stage 1: Manifest (Source)                          │
│ Total WARCs: 100,000  Est. URLs: ~3B  Est: ~100 TB │  Each stage is
│ CDX API: index.commoncrawl.org/CC-MAIN-2026-08      │  a standalone panel
├─────────────────────────────────────────────────────┤
│ Stage 2: Downloaded  ███░░░░░░  7 / 100,000 (0.007%)│
│ Total: 5.7 GB · Avg: 815 MB/WARC                   │
├─────────────────────────────────────────────────────┤
│ Stage 3: Markdown    █░░░░░░░░  1 / 7               │
│ 21,184 docs · 37 MB · ~21K docs/WARC · 1.7 KB avg  │
├─────────────────────────────────────────────────────┤
│ Stage 4: Pack        ████░░░░░  4 / 7                │
│ Parquet: 70 MB · .md.warc.gz: 37 MB                │
├─────────────────────────────────────────────────────┤
│ Stage 5: FTS Index   ████░░░░░  4 / 7                │
│ Dahlia: 331 MB (3 shards) · Tantivy: — (0 shards)  │
├──────────────────────┬──────────────────────────────┤
│ Storage              │ System                       │
│ ██████░░ 6.8 GB used │ Heap: 67 MB / 108 MB sys    │
│ Projected full: ~8 TB│ Stack: 832 KB · GC: 45      │
│ Disk: 459/494 GB     │ Goroutines: 13              │
│ Phase bars...        │ Go 1.24 · PID 16941         │
│                      │ Uptime: 2h 15m              │
└──────────────────────┴──────────────────────────────┘
```

### What to Remove
- `renderPipeline()` function and all pipeline page code
- `renderCrawls()` / `renderCrawlsContent()` functions
- Pipeline and Crawls nav tabs
- Old `renderOverview()` function (replace entirely)
- Old storage donut chart, engine/pack bars, process stats widgets
- `handleCrawls` route registration (keep handler for potential future use)
- `handleCrawlWarcs` route registration

---

## Tasks

### Task 1: Backend — OverviewResponse struct + handler

**Files:**
- Create: `pkg/index/web/overview.go`
- Modify: `pkg/index/web/server.go` (handleOverview, route registration)
- Test: `pkg/index/web/overview_test.go`

**Step 1: Write overview_test.go**

Test that `buildOverviewResponse` returns correct struct with all pipeline stages populated from a temp directory layout.

```go
package web

import (
    "os"
    "path/filepath"
    "testing"
)

func TestBuildOverviewResponse(t *testing.T) {
    root := t.TempDir()

    // Create data layout: 2 WARCs, 1 warc_md, 1 pack/parquet, 1 fts/dahlia
    mustMkdir(t, filepath.Join(root, "warc"))
    writeFile(t, filepath.Join(root, "warc", "CC-MAIN-x-00000.warc.gz"), 1024*1024)
    writeFile(t, filepath.Join(root, "warc", "CC-MAIN-x-00001.warc.gz"), 2048*1024)

    mustMkdir(t, filepath.Join(root, "warc_md"))
    writeFile(t, filepath.Join(root, "warc_md", "00000.md.warc.gz"), 512*1024)

    mustMkdir(t, filepath.Join(root, "pack", "parquet"))
    writeFile(t, filepath.Join(root, "pack", "parquet", "00000.parquet"), 256*1024)

    mustMkdir(t, filepath.Join(root, "fts", "dahlia", "00000"))
    writeFile(t, filepath.Join(root, "fts", "dahlia", "00000", "seg.bin"), 128*1024)

    resp := buildOverviewResponse("CC-TEST-2026", root, 1000, nil)

    // Manifest
    if resp.Manifest.TotalWARCs != 1000 {
        t.Fatalf("manifest total_warcs: got %d, want 1000", resp.Manifest.TotalWARCs)
    }

    // Downloaded
    if resp.Downloaded.Count != 2 {
        t.Fatalf("downloaded count: got %d, want 2", resp.Downloaded.Count)
    }
    if resp.Downloaded.SizeBytes != 3*1024*1024 {
        t.Fatalf("downloaded size: got %d", resp.Downloaded.SizeBytes)
    }
    if resp.Downloaded.AvgWARCBytes != (3*1024*1024)/2 {
        t.Fatalf("downloaded avg: got %d", resp.Downloaded.AvgWARCBytes)
    }

    // Markdown
    if resp.Markdown.Count != 1 {
        t.Fatalf("markdown count: got %d, want 1", resp.Markdown.Count)
    }
    if resp.Markdown.SizeBytes != 512*1024 {
        t.Fatalf("markdown size: got %d", resp.Markdown.SizeBytes)
    }

    // Pack
    if resp.Pack.Count != 1 {
        t.Fatalf("pack count: got %d, want 1", resp.Pack.Count)
    }
    if resp.Pack.ParquetBytes != 256*1024 {
        t.Fatalf("parquet bytes: got %d", resp.Pack.ParquetBytes)
    }

    // Indexed
    if resp.Indexed.Count != 1 {
        t.Fatalf("indexed count: got %d, want 1", resp.Indexed.Count)
    }
    if resp.Indexed.DahliaShards != 1 {
        t.Fatalf("dahlia shards: got %d", resp.Indexed.DahliaShards)
    }

    // Storage
    if resp.Storage.CrawlBytes <= 0 {
        t.Fatal("expected positive crawl bytes")
    }

    // System
    if resp.System.Goroutines <= 0 {
        t.Fatal("expected positive goroutines")
    }
    if resp.System.GoVersion == "" {
        t.Fatal("expected go version")
    }
    if resp.System.PID <= 0 {
        t.Fatal("expected positive PID")
    }
}

func TestBuildOverviewResponse_EmptyDir(t *testing.T) {
    root := t.TempDir()
    resp := buildOverviewResponse("CC-TEST-2026", root, 0, nil)

    if resp.CrawlID != "CC-TEST-2026" {
        t.Fatalf("crawl_id: got %q", resp.CrawlID)
    }
    if resp.Downloaded.Count != 0 {
        t.Fatalf("downloaded count: got %d, want 0", resp.Downloaded.Count)
    }
    if resp.Manifest.TotalWARCs != 0 {
        t.Fatalf("manifest total: got %d, want 0", resp.Manifest.TotalWARCs)
    }
}

func TestBuildOverviewResponse_ProjectedSize(t *testing.T) {
    root := t.TempDir()
    mustMkdir(t, filepath.Join(root, "warc"))
    // 1 WARC of 1 GB, manifest says 100K WARCs
    writeFile(t, filepath.Join(root, "warc", "CC-MAIN-x-00000.warc.gz"), 1024*1024*1024)

    resp := buildOverviewResponse("CC-TEST-2026", root, 100000, nil)

    // projected = avg_warc * total = 1GB * 100K = 100 TB
    if resp.Storage.ProjectedFullBytes != 1024*1024*1024*100000 {
        t.Fatalf("projected: got %d", resp.Storage.ProjectedFullBytes)
    }
}
```

**Step 2: Run tests — verify they fail**

```bash
go test -run TestBuildOverviewResponse ./pkg/index/web/ -v -count=1
```

Expected: FAIL (buildOverviewResponse not defined)

**Step 3: Implement overview.go**

Create `pkg/index/web/overview.go` with:
- All type definitions (`OverviewResponse`, `ManifestStage`, etc.)
- `buildOverviewResponse(crawlID, crawlDir string, manifestTotal int, docs *DocStore) OverviewResponse`
- `scanDownloaded(crawlDir) DownloadedStage`
- `scanMarkdownStage(crawlDir, docs) MarkdownStage`
- `scanPackStage(crawlDir) PackStage`
- `scanIndexedStage(crawlDir) IndexedStage`
- `collectSystemInfo() SystemInfo`
- `collectStorageInfo(crawlDir, manifestTotal, avgWARC) StorageInfo`

Key implementation notes:
- `manifestTotal` comes from `len(manifestPaths)` (CC API manifest)
- `EstTotalURLs = manifestTotal * 30_000` (empirical: ~30K URLs per WARC)
- `EstTotalSizeBytes = manifestTotal * avgWARCBytes` (from downloaded sample)
- `ProjectedFullBytes = sum of all phase bytes * (manifestTotal / downloadedCount)`
- Markdown count: count `.md.warc.gz` files in `warc_md/`
- Pack count: count `.parquet` files in `pack/parquet/`
- Indexed count: count shard dirs in `fts/dahlia/`; tantivy in `fts/tantivy/`
- DocStore integration: if docs != nil, sum `GetShardMeta` for total_docs
- SystemInfo: `runtime.ReadMemStats`, `runtime.NumGoroutine`, `runtime.Version()`, `os.Getpid()`, `debug.ReadBuildInfo()` for go version, `time.Since(startTime)` for uptime
- Package-level `var startTime = time.Now()` for uptime

**Step 4: Run tests — verify they pass**

```bash
go test -run TestBuildOverviewResponse ./pkg/index/web/ -v -count=1
```

**Step 5: Update handleOverview in server.go**

Replace the current `handleOverview` to use `buildOverviewResponse`. The handler needs manifest paths — cache them on Server struct or fetch lazily.

Add to Server struct:
```go
manifestTotal int // cached count of WARCs in CC manifest
```

In `NewDashboardWithOptions`, after meta init:
```go
go func() {
    client := cc.NewClient("", 4)
    paths, err := client.DownloadManifest(context.Background(), crawlID, "warc.paths.gz")
    if err == nil {
        s.manifestTotal = len(paths)
    }
}()
```

Also fetch crawl metadata (name, dates) and store on Server:
```go
CrawlName string
CrawlFrom time.Time
CrawlTo   time.Time
```

Update `handleOverview`:
```go
func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
    resp := buildOverviewResponse(s.CrawlID, s.CrawlDir, s.manifestTotal, s.Docs)
    resp.CrawlName = s.CrawlName
    if !s.CrawlFrom.IsZero() {
        resp.CrawlFrom = s.CrawlFrom.Format(time.RFC3339)
    }
    if !s.CrawlTo.IsZero() {
        resp.CrawlTo = s.CrawlTo.Format(time.RFC3339)
    }
    if s.Meta != nil {
        // Add meta status
        summary := s.Meta.GetSummary(r.Context(), s.CrawlID, s.CrawlDir)
        resp.Meta.Backend = summary.MetaBackend
        resp.Meta.GeneratedAt = summary.MetaGeneratedAt
        resp.Meta.Stale = summary.MetaStale
        resp.Meta.Refreshing = summary.MetaRefreshing
    }
    writeJSON(w, 200, resp)
}
```

**Step 6: Update existing overview tests**

Update `TestHandleOverview` and `TestHandleOverview_EmptyDir` in `server_test.go` to decode into `OverviewResponse` instead of `map[string]any` / `DataSummary`.

**Step 7: Remove stale routes**

In `server.go` Handler():
- Remove: `mux.HandleFunc("GET /api/crawls", s.handleCrawls)`
- Remove: `mux.HandleFunc("GET /api/crawl/{id}/warcs", s.handleCrawlWarcs)`

Keep the handler functions (don't delete code — just unregister routes).

Remove compile-time checks:
```go
// Remove these lines from server_test.go:
var _ http.HandlerFunc = (*Server)(nil).handleCrawlWarcs
var _ http.HandlerFunc = (*Server)(nil).handleCrawls
```

Update `TestIntegrationNewNoDashboardRoutes` to remove `/api/crawls` from the test loop.

**Step 8: Run all tests**

```bash
go test ./pkg/index/web/... -v -count=1
```

**Step 9: Commit**

```bash
git add pkg/index/web/overview.go pkg/index/web/overview_test.go pkg/index/web/server.go pkg/index/web/server_test.go
git commit -m "feat(overview): structured OverviewResponse with pipeline stages and system info"
```

---

### Task 2: Frontend — Navbar cleanup

**Files:**
- Modify: `pkg/index/web/static/index.html`

**Step 1: Update navbar**

In the `<header>` element:
- Remove `max-w-6xl mx-auto` from the inner div (make full-width, keep px-4 md:px-6)
- Remove `border-b border-[var(--border)]` from `app-header` style
- Remove Pipeline tab: `<a href="#/pipeline" ...>Pipeline</a>`
- Remove Crawls tab: `<a href="#/crawls" ...>Crawls</a>`
- Reorder remaining tabs: Overview, Search, Browse, WARC, Jobs

**Step 2: Remove renderPipeline and renderCrawls**

Delete:
- `renderPipeline()` function and all helper functions it calls (`renderPipelineContent`, etc.)
- `renderCrawls()` and `renderCrawlsContent()` functions
- Router cases for `pipeline` and `crawls` in the hash router

**Step 3: Build and verify**

```bash
make build && search cc fts dashboard --port 3460
```

Open http://localhost:3460 — verify navbar has 5 tabs, full-width, no bottom border.

**Step 4: Commit**

```bash
git add pkg/index/web/static/index.html
git commit -m "fix(navbar): remove Pipeline/Crawls tabs, full-width, no border"
```

---

### Task 3: Frontend — Overview page rewrite

**Files:**
- Modify: `pkg/index/web/static/index.html`

**Step 1: Replace renderOverview()**

Delete the entire old `renderOverview()` function and replace with new implementation that:

1. Fetches `/api/overview`
2. Renders **Crawl Banner**: crawl_id, name, date range, manifest total
3. Renders **Pipeline Summary Bar**: full-width horizontal bar with 5 segments
4. Renders **5 Stage Panels** (each a `<section>` with border):
   - Stage 1: Manifest — total WARCs, est URLs, est total size
   - Stage 2: Downloaded — count/total, progress bar, size, avg/WARC
   - Stage 3: Markdown — count/total, docs, size, avg docs/WARC, avg doc size
   - Stage 4: Pack — count/total, parquet bytes, .md.warc.gz bytes
   - Stage 5: FTS Index — count/total, dahlia bytes+shards, tantivy bytes+shards
5. Renders **Storage + System** (2-column grid):
   - Left: disk total/used/free bar, crawl size, projected full crawl
   - Right: memory (heap/sys/stack/GC), runtime (goroutines, Go version, PID, uptime)

Design guidelines:
- Each stage panel: left side = stage label + icon, center = stats grid, right = progress badge
- Progress bars use accent color with `height: 4px`
- Numbers formatted with `fmtBytes()` and `fmtNum()` helpers
- Stage panels have subtle left border color (green for complete, yellow for partial, gray for 0)
- System monitor uses monospace font for values

**Step 2: Build and test**

```bash
make build && search cc fts dashboard --port 3460
```

Open http://localhost:3460 — verify:
- Banner shows CC-MAIN-2026-08, date range
- Pipeline bar shows 5 stages with correct counts
- All 5 stage panels render with real data
- Storage shows disk info
- System shows heap, goroutines, Go version

**Step 3: Commit**

```bash
git add pkg/index/web/static/index.html
git commit -m "feat(overview): pipeline stage panels, system monitor, storage gauge"
```

---

### Task 4: Integration test

**Files:**
- Modify: `pkg/index/web/server_test.go`

**Step 1: Write TestHandleOverview_StructuredResponse**

```go
func TestHandleOverview_StructuredResponse(t *testing.T) {
    root := t.TempDir()

    // Create realistic data layout
    mustMkdir(t, filepath.Join(root, "warc"))
    writeFile(t, filepath.Join(root, "warc", "CC-MAIN-x-00000.warc.gz"), 1024*1024)
    writeFile(t, filepath.Join(root, "warc", "CC-MAIN-x-00001.warc.gz"), 1024*1024)

    mustMkdir(t, filepath.Join(root, "warc_md"))
    writeFile(t, filepath.Join(root, "warc_md", "00000.md.warc.gz"), 512*1024)

    mustMkdir(t, filepath.Join(root, "pack", "parquet"))
    writeFile(t, filepath.Join(root, "pack", "parquet", "00000.parquet"), 256*1024)

    mustMkdir(t, filepath.Join(root, "fts", "dahlia", "00000"))
    writeFile(t, filepath.Join(root, "fts", "dahlia", "00000", "index.bin"), 128*1024)

    srv := NewDashboard("dahlia", "CC-TEST-2026", "", root)

    req := httptest.NewRequest("GET", "/api/overview", nil)
    w := httptest.NewRecorder()
    srv.handleOverview(w, req)

    if w.Code != 200 {
        t.Fatalf("expected 200, got %d", w.Code)
    }

    var resp OverviewResponse
    if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
        t.Fatalf("decode: %v", err)
    }

    // Pipeline stages
    if resp.Downloaded.Count != 2 {
        t.Fatalf("downloaded: got %d, want 2", resp.Downloaded.Count)
    }
    if resp.Markdown.Count != 1 {
        t.Fatalf("markdown: got %d, want 1", resp.Markdown.Count)
    }
    if resp.Pack.Count != 1 {
        t.Fatalf("pack: got %d, want 1", resp.Pack.Count)
    }
    if resp.Indexed.Count != 1 {
        t.Fatalf("indexed: got %d, want 1", resp.Indexed.Count)
    }
    if resp.Indexed.DahliaShards != 1 {
        t.Fatalf("dahlia shards: got %d, want 1", resp.Indexed.DahliaShards)
    }

    // System
    if resp.System.GoVersion == "" {
        t.Fatal("missing go version")
    }
    if resp.System.Goroutines <= 0 {
        t.Fatal("expected positive goroutines")
    }
    if resp.System.PID <= 0 {
        t.Fatal("expected positive PID")
    }

    // Storage
    if resp.Storage.CrawlBytes <= 0 {
        t.Fatal("expected positive crawl bytes")
    }
}
```

**Step 2: Run all tests**

```bash
go test ./pkg/index/web/... -v -count=1 -timeout 60s
```

**Step 3: Commit**

```bash
git add pkg/index/web/server_test.go
git commit -m "test(overview): add structured OverviewResponse integration tests"
```

---

### Task 5: End-to-end verification

**Step 1: Build and start server**

```bash
make build && search cc fts dashboard --port 3460
```

**Step 2: Verify API response**

```bash
curl -s http://localhost:3460/api/overview | python3 -m json.tool
```

Check: all pipeline stages present with correct counts, system info populated.

**Step 3: Verify UI**

Open http://localhost:3460/#/ in browser.
Check: banner, pipeline bar, 5 stage panels, storage, system monitor all render.

**Step 4: Write spec doc and commit**

```bash
git add spec/0674_dashboard_enhancement.md
git commit -m "docs(spec): dashboard overview enhancement spec"
```
