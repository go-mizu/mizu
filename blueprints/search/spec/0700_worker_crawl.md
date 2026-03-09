# 0700: Worker-Proxied Domain Crawling

## Summary

Add `--worker` mode to `search crawl-domain` that proxies page fetches through a
Cloudflare Worker (`tools/crawler/`), getting both raw HTML and markdown in a single
request. The worker uses CF `fetch()` for standard mode and `@cloudflare/puppeteer`
for browser mode.

## Motivation

- CF edge network fetches pages from 300+ global PoPs → lower latency to targets
- Workers AI `toMarkdown()` converts HTML→markdown server-side (free, no local deps)
- Browser Rendering binding handles JS-heavy sites without local Chrome
- In-memory cache per isolate enables fast recrawl without re-fetching

## Architecture

```
┌─────────────────────────────┐
│  Go Crawler (pkg/dcrawler)  │
│  frontier → batcher → POST  │
│  parse response → links     │
│  store html+md → DuckDB     │
└──────────┬──────────────────┘
           │ POST /crawl (batch 50 URLs)
           ▼
┌─────────────────────────────┐
│  CF Worker (tools/crawler)  │
│  fetch HTML (or browser)    │
│  Workers AI toMarkdown()    │
│  in-memory result cache     │
└─────────────────────────────┘
```

## CF Worker Limits (paid plan)

- **CPU time**: 30s per request (wall time can be longer due to I/O waits)
- **Memory**: 128MB per isolate
- **Subrequests**: 1000 per request (fetch calls count)
- **Body size**: ~25MB response (practical; no hard limit documented)
- **Browser sessions**: 2 concurrent per account (Browser Rendering)
- **Batch size**: 10 URLs per request (conservative; avoids 30s CPU timeout)
  - Each URL: fetch (~1-5s I/O) + toMarkdown (~0.5-2s CPU) ≈ 2-7s wall time
  - 10 URLs × 3s avg CPU = ~30s worst case → safe margin
  - Go side sends 20 concurrent batches = 200 URLs in-flight

## Worker API

### POST /crawl

```json
Request:
{
  "urls": ["https://example.com/page1", ...],  // max 10
  "browser": false,    // use CF Browser Rendering
  "timeout": 15000     // per-URL timeout (ms)
}

Response:
[{
  "url": "https://example.com/page1",
  "status": 200,
  "html": "<html>...</html>",
  "markdown": "# Page Title\n\nContent...",
  "title": "Page Title",
  "content_type": "text/html; charset=utf-8",
  "content_length": 12345,
  "redirect_url": null,
  "fetch_time_ms": 450,
  "error": null
}]
```

### GET /

Health check → `{"status": "ok", "version": "1.0.0"}`

## DuckDB Schema Changes

Add two columns to `pages` table:

```sql
ALTER TABLE pages ADD COLUMN html BLOB;       -- zstd-compressed raw HTML
ALTER TABLE pages ADD COLUMN markdown VARCHAR; -- clean markdown text
```

In worker mode, both are populated automatically. The existing `body` column
remains for backward compatibility (non-worker mode with --store-body).

## Go Integration

### Config additions

```go
UseWorker      bool   // --worker flag
WorkerURL      string // --worker-url (default https://crawler.go-mizu.workers.dev)
WorkerToken    string // --worker-token or CRAWLER_WORKER_TOKEN env
WorkerBrowser  bool   // --worker-browser (CF Browser Rendering)
WorkerBatch    int    // batch size (default 10)
WorkerParallel int    // concurrent batches (default 20)
```

### worker.go

- `WorkerClient` struct with HTTP client, URL, token
- `FetchBatch(ctx, urls) → []WorkerResult`
- Concurrent batch dispatch from frontier channel
- Retry on 429/5xx from worker with exponential backoff

### Crawler integration

When `UseWorker` is true:
1. Workers pull from frontier into a batch buffer (channel)
2. Batch dispatcher sends POST /crawl every 10 URLs or 500ms timeout
3. Response parsed: HTML → ExtractLinksAndMeta (Go-side) → frontier
4. Result stored with html (zstd compressed) + markdown columns

## CLI Flags

```
--worker              Use CF Worker proxy for fetching + markdown
--worker-url URL      Worker endpoint (default https://crawler.go-mizu.workers.dev)
--worker-token TOKEN  Auth token (default $CRAWLER_WORKER_TOKEN)
--worker-browser      Enable CF Browser Rendering on worker side
```

## Testing

```bash
# Test with qiita.com (Japanese tech blog, 1000+ pages)
search crawl-domain qiita.com --worker --max-pages 1000

# With browser mode
search crawl-domain qiita.com --worker --worker-browser --max-pages 100
```

## Files Changed

- `tools/crawler/` — new CF Worker (Hono + TypeScript)
- `pkg/dcrawler/worker.go` — Go worker client
- `pkg/dcrawler/config.go` — new config fields
- `pkg/dcrawler/types.go` — Result fields for html/markdown
- `pkg/dcrawler/resultdb.go` — schema + write changes
- `pkg/dcrawler/crawler.go` — worker mode integration
- `cli/dcrawl.go` — new CLI flags
