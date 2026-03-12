# spec/0710 — Browser API: Self-hosted Crawl Endpoint

## Overview

A Cloudflare Worker at `https://browser.go-mizu.workers.dev` that exposes
the same API shape as CF's Browser Rendering `/crawl` endpoint but uses plain
`fetch()` instead of browser rendering. No rate limits, no browser time quota.

Stack: **Hono + D1 + CF Queue**, Bearer token auth.

---

## API Reference

### Authentication

All endpoints require:
```
Authorization: Bearer <token>
```

Token is stored as CF Worker secret `AUTH_TOKEN`.

---

### POST `/api/crawl` — Submit job

#### Request body

| Field | Type | Default | Description |
|---|---|---|---|
| `url` | string | **required** | Seed URL to crawl |
| `limit` | number | 10 | Max pages to crawl |
| `depth` | number | 100 | Max link depth from seed |
| `formats` | string[] | `["markdown"]` | Output formats: `"html"`, `"markdown"` |
| `userAgent` | string | `"mizu-browser/1.0"` | Custom User-Agent |
| `setExtraHTTPHeaders` | object | — | Extra headers `{"key": "value"}` |
| `options` | object | — | Crawl options (see below) |

#### `options` object

| Field | Type | Default | Description |
|---|---|---|---|
| `includeSubdomains` | boolean | false | Follow links to subdomains |
| `includeExternalLinks` | boolean | false | Follow links to other domains |
| `includePatterns` | string[] | — | Glob patterns to include |
| `excludePatterns` | string[] | — | Glob patterns to exclude (take precedence) |

Pattern rules: `*` = any char except `/`; `**` = any char including `/`.

#### Success response

```json
{
  "success": true,
  "result": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### Error response

```json
{
  "success": false,
  "errors": [{"code": 1001, "message": "url is required"}],
  "result": null
}
```

---

### GET `/api/crawl/:id` — Poll job status + records

#### Query parameters

| Param | Type | Default | Description |
|---|---|---|---|
| `cursor` | number | 0 | Start offset for pagination |
| `limit` | number | 100 | Max records per response |

#### Response

```json
{
  "success": true,
  "result": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "running",
    "total": 50,
    "finished": 18,
    "cursor": 18,
    "records": [
      {
        "url": "https://example.com/",
        "status": "completed",
        "markdown": "# Example\n\nContent here...",
        "html": "<html>...</html>",
        "metadata": {
          "status": 200,
          "title": "Example Page",
          "url": "https://example.com/"
        }
      }
    ]
  }
}
```

#### Job status values

| Value | Meaning |
|---|---|
| `running` | Job in progress |
| `completed` | All pages processed |
| `errored` | Job-level error |
| `cancelled_by_user` | Manually deleted |

#### Record status values

| Value | Meaning |
|---|---|
| `completed` | Successfully fetched |
| `errored` | Fetch failed |
| `skipped` | Filtered by pattern/domain rules |
| `queued` | Not yet processed |

---

### DELETE `/api/crawl/:id` — Cancel job

#### Response

```json
{
  "success": true,
  "result": { "id": "...", "status": "cancelled_by_user" }
}
```

---

## Architecture

```
POST /api/crawl
  → validate request
  → create job row in D1
  → insert seed URL as page row (depth=0, status=queued)
  → enqueue seed URL to CF Queue
  → return job ID

Queue consumer (batch processor):
  for each message:
    → check job not cancelled, page limit not reached
    → fetch URL via fetch()
    → parse HTML: extract <title>, links
    → convert HTML → markdown (if requested)
    → update page row: status=completed, html, markdown, title, http_status
    → discover new links:
      - filter by domain/subdomain/pattern rules
      - skip already-seen URLs (check pages table)
      - respect depth limit
      - respect page limit (total < job.limit)
      - insert new page rows + enqueue

GET /api/crawl/:id
  → read job from D1
  → read pages with cursor/limit pagination (ordered by id)
  → return CF-compatible response shape

DELETE /api/crawl/:id
  → set job status = cancelled_by_user
  → queue consumer checks status before processing
```

---

## D1 Schema

```sql
CREATE TABLE jobs (
  id TEXT PRIMARY KEY,
  url TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'running',
  config TEXT NOT NULL,
  total INTEGER NOT NULL DEFAULT 0,
  finished INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE pages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id TEXT NOT NULL,
  url TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'queued',
  http_status INTEGER NOT NULL DEFAULT 0,
  title TEXT NOT NULL DEFAULT '',
  html TEXT,
  markdown TEXT,
  depth INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (job_id) REFERENCES jobs(id)
);

CREATE INDEX idx_pages_job_id ON pages(job_id, id);
CREATE UNIQUE INDEX idx_pages_job_url ON pages(job_id, url);
```

---

## Link Discovery

Uses CF `HTMLRewriter` for streaming link extraction (zero-copy, native API):

```typescript
class LinkExtractor {
  links: string[] = [];
  element(el: Element) {
    const href = el.getAttribute("href");
    if (href) this.links.push(href);
  }
}
```

Link filtering pipeline:
1. Resolve relative URLs against page base URL
2. Normalize: strip fragment, trailing slash
3. Skip non-http(s) schemes
4. Apply domain/subdomain filter
5. Apply include/exclude glob patterns
6. Check not already in pages table for this job
7. Check depth < job depth limit
8. Check total < job page limit

---

## HTML → Markdown

Lightweight regex-based conversion (no external deps):
- `<h1>`...`<h6>` → `#` headers
- `<p>` → double newline
- `<a href>` → `[text](url)`
- `<ul>/<ol>/<li>` → bullet/numbered lists
- `<code>/<pre>` → backtick/fenced blocks
- `<strong>/<em>` → `**bold**`/`*italic*`
- `<br>` → newline
- Strip all other tags

---

## Auth

- Worker secret: `AUTH_TOKEN`
- Middleware checks `Authorization: Bearer <token>` on all `/api/*` routes
- Generate token: `openssl rand -hex 32` → store in `$HOME/data/.local.env` as
  `BROWSER_API_TOKEN=<value>`, then `wrangler secret put AUTH_TOKEN`
- Returns 401 `{ success: false, errors: [{ code: 1000, message: "Unauthorized" }] }`

---

## Not Implemented (v1)

These CF crawl endpoint features are **intentionally omitted** from v1:

| Feature | CF Field | Reason |
|---|---|---|
| JS rendering | `render` | We use plain `fetch()` — no browser needed |
| Navigation options | `gotoOptions` | Browser-only (waitUntil, timeout) |
| Wait for selector | `waitForSelector` | Browser-only (CSS selector wait) |
| Resource blocking | `rejectResourceTypes` | Browser-only (image/font/css blocking) |
| Basic auth | `authenticate` | Can be added later via `setExtraHTTPHeaders` |
| AI extraction | `jsonOptions` | Requires Workers AI binding |
| Browser seconds | `browserSecondsUsed` | No browser = always 0 |
| Source discovery | `source` | No sitemap/robots.txt parsing yet |
| Cache TTL | `maxAge` | No result caching yet |
| Modified since | `modifiedSince` | No conditional crawling yet |

These can be added incrementally in future versions.

---

## Deployment

```bash
cd tools/browser
npm install
npx wrangler d1 create browser-db
npx wrangler queues create browser-crawl
# Update wrangler.toml with D1 database_id and queue name
npx wrangler secret put AUTH_TOKEN
npx wrangler deploy
```

Worker URL: `https://browser.go-mizu.workers.dev`

---

## File Structure

```
tools/browser/
├── wrangler.toml
├── package.json
├── tsconfig.json
├── schema.sql
└── src/
    ├── index.ts          # Hono app, routes, queue consumer
    ├── types.ts          # Request/response types, Env bindings
    ├── auth.ts           # Bearer token middleware
    ├── crawl.ts          # POST/GET/DELETE handlers
    ├── queue.ts          # Queue consumer: fetch + parse + enqueue
    ├── links.ts          # HTMLRewriter link extraction + filtering
    ├── markdown.ts       # HTML → Markdown converter
    └── patterns.ts       # Glob pattern matching for include/exclude
```
