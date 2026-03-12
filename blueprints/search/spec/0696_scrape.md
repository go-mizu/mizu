# 0696 — Domain Scrape Pipeline

## Goal

Integrate `pkg/dcrawler` (HTTP and rod/browser modes) into the dashboard as a
new **Scrape** tab. Users can crawl any domain from the UI, monitor progress in
real-time via WebSocket, browse scraped pages, and feed results into the
existing pipeline (markdown → pack → index) for full-text search.

## Architecture

```
Dashboard UI (Scrape tab)
  │
  ├─ POST /api/scrape           → start scrape job
  ├─ POST /api/scrape/:domain/resume → resume from checkpoint
  ├─ DELETE /api/scrape/:domain → stop running scrape
  ├─ GET  /api/scrape/list      → list all scraped domains
  ├─ GET  /api/scrape/:domain/status → job status + stats
  ├─ GET  /api/scrape/:domain/pages  → paginated scraped pages
  ├─ POST /api/scrape/:domain/pipeline → trigger md→pack→index
  │
  └─ WebSocket: job_progress / job_update events
```

### Storage Layout (per domain)

```
~/data/crawler/{domain}/
  results/              # dcrawler sharded DuckDB (pages + links)
  state.duckdb          # frontier checkpoint for resume
  markdown/             # ScrapeMarkdownTask output
  pack/                 # PackTask output (reused)
  fts/                  # IndexTask output (reused)
```

### Data Flow

```
┌─────────────┐     ┌──────────────────┐     ┌──────────┐     ┌───────────┐
│ ScrapeTask  │────▶│ScrapeMarkdownTask│────▶│ PackTask │────▶│ IndexTask │
│ (dcrawler)  │     │ DuckDB → .md     │     │ (reused) │     │ (reused)  │
└─────────────┘     └──────────────────┘     └──────────┘     └───────────┘
       │                     │                     │                │
  results/*.duckdb      markdown/*.md         pack/{fmt}/       fts/{engine}/
```

## New Components

### 1. ScrapeTask (`task_scrape.go`)

Wraps `dcrawler.Crawler` as `core.Task[ScrapeState, ScrapeMetric]`.

```go
type ScrapeConfig struct {
    Domain    string
    Mode      string // "http" or "browser"
    MaxPages  int
    MaxDepth  int
    Workers   int
    Timeout   time.Duration
    StoreBody bool
    Resume    bool
    DataDir   string // ~/data/crawler
}

type ScrapeState struct {
    Domain      string  // domain being scraped
    Pages       int64   // total pages processed
    Success     int64   // successful fetches
    Failed      int64   // failed fetches
    Frontier    int     // URLs pending in frontier
    InFlight    int64   // workers currently fetching
    BytesRecv   int64   // total bytes downloaded
    LinksFound  int64   // total links extracted
    PagesPerSec float64 // rolling speed
    Elapsed     time.Duration
    Progress    float64 // 0-1 (maxPages based, or 0.5 if unlimited)
}

type ScrapeMetric struct {
    Domain    string
    Pages     int64
    Success   int64
    Failed    int64
    Bytes     int64
    Links     int64
    Elapsed   time.Duration
}
```

**Implementation**:
- Creates `dcrawler.Config` from `ScrapeConfig`
- Calls `dcrawler.New(cfg)` + `crawler.Run(ctx)`
- Polls `crawler.Stats()` every 500ms → emits `ScrapeState`
- On context cancellation: dcrawler saves frontier to state.duckdb
- Resume: sets `cfg.Resume = true`, dcrawler handles bloom rebuild + frontier restore

### 2. ScrapeMarkdownTask (`task_scrape_md.go`)

Converts DuckDB pages to markdown files.

```go
type ScrapeMarkdownConfig struct {
    Domain   string
    DataDir  string // ~/data/crawler
    Converter string // "default", "fast", "light"
}

type ScrapeMarkdownState struct {
    Domain        string
    DocsProcessed int64
    DocsTotal     int64
    DocsPerSec    float64
    Progress      float64
}

type ScrapeMarkdownMetric struct {
    Domain string
    Docs   int64
    Elapsed time.Duration
}
```

**Implementation**:
- Opens all result shard DuckDB files in read-only mode
- Queries: `SELECT url, url_hash, body, title, content_type FROM pages
  WHERE status_code >= 200 AND status_code < 400 AND body IS NOT NULL`
- Decompresses zstd body → calls `markdown.ConvertFast(html, url)`
- Writes to `{domain}/markdown/{url_hash}.md`
- Skips files that already exist (incremental)
- Emits progress every 100 docs

### 3. ScrapeStore (`scrape_store.go`)

Reads per-domain crawl metadata for the dashboard.

```go
type ScrapeStore struct {
    dataDir string // ~/data/crawler
}

type ScrapeDomain struct {
    Domain    string    `json:"domain"`
    Pages     int64     `json:"pages"`
    Success   int64     `json:"success"`
    Failed    int64     `json:"failed"`
    Links     int64     `json:"links"`
    BodyBytes int64     `json:"body_bytes"`
    LastCrawl time.Time `json:"last_crawl"`
    HasMD     bool      `json:"has_markdown"`
    HasIndex  bool      `json:"has_index"`
}
```

**Implementation**:
- `ListDomains()`: scans `~/data/crawler/*/results/` directories
- `GetDomainStats(domain)`: queries DuckDB shards for aggregate stats
- `GetPages(domain, page, pageSize, q, sort)`: paginated page listing
- `GetPageDetail(domain, urlHash)`: single page with body

### 4. Server Routes (`server.go` additions)

New scrape routes registered in the dashboard block:

```go
// Scrape pipeline
router.Post("/api/scrape", s.handleScrapeStart)
router.Delete("/api/scrape/{domain}", s.handleScrapeStop)
router.Post("/api/scrape/{domain}/resume", s.handleScrapeResume)
router.Get("/api/scrape/list", s.handleScrapeList)
router.Get("/api/scrape/{domain}/status", s.handleScrapeStatus)
router.Get("/api/scrape/{domain}/pages", s.handleScrapePages)
router.Post("/api/scrape/{domain}/pipeline", s.handleScrapePipeline)
```

### 5. Job Integration (`executors.go` additions)

New job types in RunJob switch:
- `"scrape"` → `runScrapeJob(ctx, job)` — wraps ScrapeTask
- `"scrape_markdown"` → `runScrapeMarkdownJob(ctx, job)` — wraps ScrapeMarkdownTask

JobConfig extended:
```go
type JobConfig struct {
    // ... existing fields ...
    Domain string `json:"domain,omitempty"` // for scrape jobs
}
```

### 6. Frontend (`scrape.js`)

New Scrape tab at `#/scrape`:

**Main view**: List of scraped domains with stats (pages, last crawl, pipeline status)

**Domain detail view** (`#/scrape/{domain}`):
- Live progress panel (when scraping): pages/s, frontier, in-flight, bytes, elapsed
- Controls: Start / Stop / Resume
- Config form: mode (http/browser), max pages, max depth, store body
- Scraped pages table: URL, status, title, content type, fetch time
- Pipeline controls: "Convert to Markdown" → "Pack" → "Build Index"
- Pipeline progress (reuses job progress pattern)

**State additions**:
```js
scrapeList: null,
scrapeDomain: '',
scrapePage: 1,
scrapeSort: 'crawled_at',
scrapeQ: '',
```

## Resumability & Checkpointing

Leverages dcrawler's existing resume infrastructure:

1. **Frontier checkpoint**: On graceful stop (context cancel), dcrawler drains
   frontier to `state.duckdb`. On resume, frontier restored.

2. **Bloom rebuild**: On resume, dcrawler reads all successful URLs from
   result shards, marks in bloom filter → prevents re-crawling.

3. **Failed retry**: Previously failed URLs are re-attempted on resume.

4. **Stale re-crawl**: Optional `StaleHours` config to re-crawl pages older
   than N hours (incremental freshness).

5. **Markdown incremental**: ScrapeMarkdownTask skips pages whose
   `{url_hash}.md` file already exists.

## Files Changed

| File | Change |
|------|--------|
| `pkg/index/web/task_scrape.go` | New: ScrapeTask |
| `pkg/index/web/task_scrape_md.go` | New: ScrapeMarkdownTask |
| `pkg/index/web/scrape_store.go` | New: ScrapeStore |
| `pkg/index/web/executors.go` | Add scrape/scrape_markdown dispatchers |
| `pkg/index/web/jobs.go` | Add Domain to JobConfig |
| `pkg/index/web/server.go` | Add scrape routes + handlers + ScrapeStore field |
| `pkg/index/web/static/index.html` | Add Scrape tab to nav |
| `pkg/index/web/static/js/scrape.js` | New: Scrape tab frontend |
| `pkg/index/web/static/js/api.js` | Add scrape API client functions |
| `pkg/index/web/static/js/state.js` | Add scrape state fields |
| `pkg/index/web/static/js/router.js` | Add scrape routes |
