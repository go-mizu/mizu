# spec/0709 — Cloudflare Browser Rendering `/crawl` Integration

## Overview

`search scrape <domain> --cloudflare` delegates crawling to the Cloudflare
Browser Rendering REST API `/crawl` endpoint instead of running a local
crawler. CF handles JS rendering, robots.txt, and distributed fetching.

**Key design**: CF crawl is async. We submit a job, then **poll every 3s for
partial results as they arrive** and store them immediately — not only when
the job completes.

---

## Credentials

`$HOME/data/cloudflare/cloudflare.json`:
```json
{
  "account_id": "cb5dd73d443a2ec983331204b459380a",
  "api_token":  "<token-with-browser-rendering-edit-permission>"
}
```

Source: `$HOME/data/.local.env` env vars `CLOUDFLARE_BROWSER_ACCOUNT_ID` and
`CLOUDFLARE_BROWSER_TOKEN`.

**Token permissions**: Account > Browser Rendering > Edit.
**Note**: `GET /user/tokens/verify` returns 401 for tokens with only Browser
Rendering scope — that's expected. The token works against the crawl endpoint.

---

## CF API Reference

### Base URL
```
https://api.cloudflare.com/client/v4/accounts/{account_id}/browser-rendering/crawl
```

### Authentication
```
Authorization: Bearer <apiToken>
Content-Type: application/json
```

---

### POST `/crawl` — Submit job

#### Request fields

| Field | Type | Default | Description |
|---|---|---|---|
| `url` | string | **required** | Seed URL to crawl |
| `limit` | number | 10 | Max pages (max 100,000) |
| `depth` | number | 100,000 | Max link depth |
| `source` | string | `"all"` | Discovery: `"all"`, `"sitemaps"`, `"links"` |
| `formats` | array | `["html"]` | Output: `"html"`, `"markdown"`, `"json"` |
| `render` | boolean | `true` | JS rendering; `false` = fast static HTML |
| `maxAge` | number | 86400 | Cache TTL in seconds (max 604,800) |
| `modifiedSince` | number | — | Unix timestamp; only crawl pages modified after this |
| `userAgent` | string | — | Custom User-Agent |
| `rejectResourceTypes` | array | — | Block: `"image"`, `"media"`, `"font"`, `"stylesheet"` |
| `setExtraHTTPHeaders` | object | — | Extra request headers `{"key": "value"}` |
| `authenticate` | object | — | Basic auth `{"username":"…","password":"…"}` |
| `options` | object | — | See below |
| `gotoOptions` | object | — | See below |
| `waitForSelector` | object | — | See below |
| `jsonOptions` | object | — | AI extraction (only with `"json"` format) |

#### `options` object
```json
{
  "includeSubdomains": false,
  "includeExternalLinks": false,
  "includePatterns": ["https://example.com/docs/**"],
  "excludePatterns": ["https://example.com/archive/**"]
}
```
Pattern rules: `*` = any char except `/`; `**` = any char including `/`.
Exclude patterns take precedence over include.

#### `gotoOptions` object
```json
{
  "waitUntil": "networkidle2",
  "timeout": 30000
}
```
`waitUntil` values: `"load"`, `"domcontentloaded"`, `"networkidle0"`, `"networkidle2"`

#### `waitForSelector` object
```json
{
  "selector": "#main-content",
  "timeout": 5000,
  "visible": true
}
```

#### `jsonOptions` object (AI extraction)
```json
{
  "prompt": "Extract product name, price, description",
  "response_format": {
    "type": "json_schema",
    "json_schema": {
      "name": "product",
      "properties": {
        "name": "string",
        "price": "number",
        "description": "string"
      }
    }
  }
}
```

#### Response
```json
{
  "success": true,
  "result": "400745dc-0a47-4d27-9d66-0f2fc7178e7b"
}
```
`result` is the job ID for subsequent GET/DELETE calls.

#### Error response
```json
{
  "success": false,
  "errors": [{"code": 2001, "message": "Rate limit exceeded"}],
  "result": null
}
```

---

### GET `/crawl/{jobId}` — Poll status + records

#### Query parameters

| Param | Type | Description |
|---|---|---|
| `cursor` | number | Start offset for pagination (default 0) |
| `limit` | number | Max records per response (default all) |
| `status` | string | Filter by record status |

#### Response

```json
{
  "success": true,
  "result": {
    "id": "400745dc-0a47-4d27-9d66-0f2fc7178e7b",
    "status": "running",
    "browserSecondsUsed": 12.4,
    "total": 50,
    "finished": 18,
    "cursor": 18,
    "records": [
      {
        "url": "https://sqlite.org/",
        "status": "completed",
        "markdown": "# SQLite Home Page\n\nSQLite is a C-language library...",
        "html": "<html>...</html>",
        "metadata": {
          "status": 200,
          "title": "SQLite Home Page",
          "url": "https://sqlite.org/"
        }
      },
      {
        "url": "https://www.sqlite.org/src/timeline",
        "status": "skipped",
        "markdown": null,
        "html": null,
        "metadata": {
          "status": 0,
          "title": "",
          "url": ""
        }
      }
    ]
  }
}
```

#### Job `status` values

| Value | Meaning |
|---|---|
| `running` | Job in progress |
| `completed` | All pages processed |
| `errored` | Job-level error |
| `cancelled_due_to_timeout` | Exceeded 7-day job limit |
| `cancelled_due_to_limits` | Account limit reached |
| `cancelled_by_user` | Manually deleted |

#### Record `status` values

| Value | Stored? | Counted? | Meaning |
|---|---|---|---|
| `completed` | ✓ | ok or fail (by HTTP code) | Successfully fetched |
| `errored` | ✓ | fail | CF failed to fetch this URL |
| `disallowed` | ✗ | — | Blocked by robots.txt |
| `skipped` | ✗ | — | CF filtered (pattern, domain, etc.) |
| `queued` | ✗ | — | Not yet processed |
| `cancelled` | ✗ | — | Cancelled before processing |

---

### DELETE `/crawl/{jobId}` — Cancel job

```bash
curl -X DELETE \
  -H "Authorization: Bearer <token>" \
  "https://api.cloudflare.com/client/v4/accounts/{account_id}/browser-rendering/crawl/{jobId}"
```

---

## Example curl Requests

### Basic static crawl (our default, `render=false`)
```bash
curl -X POST \
  -H "Authorization: Bearer $CF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://sqlite.org",
    "limit": 10,
    "formats": ["markdown"],
    "render": false
  }' \
  "https://api.cloudflare.com/client/v4/accounts/$CF_ACCOUNT/browser-rendering/crawl"
```

### JS-rendered crawl with subdomain + pattern filter
```bash
curl -X POST \
  -H "Authorization: Bearer $CF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://docs.example.com",
    "limit": 500,
    "depth": 5,
    "formats": ["markdown"],
    "render": true,
    "rejectResourceTypes": ["image", "font", "stylesheet"],
    "options": {
      "includeSubdomains": true,
      "includePatterns": ["https://docs.example.com/docs/**"],
      "excludePatterns": ["https://docs.example.com/archive/**"]
    },
    "gotoOptions": {
      "waitUntil": "networkidle2",
      "timeout": 15000
    }
  }' \
  "https://api.cloudflare.com/client/v4/accounts/$CF_ACCOUNT/browser-rendering/crawl"
```

### Poll for results
```bash
# Initial poll
curl -H "Authorization: Bearer $CF_TOKEN" \
  "https://api.cloudflare.com/client/v4/accounts/$CF_ACCOUNT/browser-rendering/crawl/$JOB_ID?cursor=0&limit=100"

# Next page
curl -H "Authorization: Bearer $CF_TOKEN" \
  "https://api.cloudflare.com/client/v4/accounts/$CF_ACCOUNT/browser-rendering/crawl/$JOB_ID?cursor=100&limit=100"
```

---

## Polling Strategy

```
POST /crawl → jobID
cursor = 0

loop (every 3s):
  GET /crawl/{jobID}?cursor={cursor}&limit=100
  → { records: [...N], cursor: N, status: "running"|"completed"|... }

  for each record: storeCFRecord() → ResultDB
  if len(records) > 0: cursor = result.cursor

  if status == "running":  sleep 3s
  if status != "running" && len(records) < 100:  close channel → done
  if status != "running" && len(records) == 100:  fetch next page immediately (no sleep)
```

The poll loop runs in a goroutine, sends records to a buffered channel (cap=200).
Main goroutine reads channel and writes to ResultDB. Display goroutine ticks every 1s.

---

## Live Progress Display

```
00:07  1 ok  recv:1  CF:1/1  browser:0.0s  [running]
```

| Field | Meaning |
|---|---|
| `00:07` | Elapsed time |
| `1 ok` | Records stored with HTTP 2xx |
| `N err` | Records stored with non-2xx or CF error (only shown if > 0) |
| `recv:N` | Total records received from CF |
| `CF:X/Y` | CF's internal finished/total count |
| `browser:Xs` | CF browser seconds consumed |
| `[status]` | Current CF job status |

Final summary:
```
  Completed  4/4 pages  stored:4  browser:0.0s  elapsed:13s
  Results: /Users/apple/data/crawler/sqlite.org/results/
  Errors (1):
    HTTP 404  https://sqlite.org/missing-page
```

---

## CLI Usage

```bash
# Static HTML fetch (default, fastest)
search scrape sqlite.org --cloudflare --cf-limit 10

# JS rendering (React/Next.js apps, dynamic pages)
search scrape app.example.com --cloudflare --cf-limit 50 --cf-render

# Docs-only deep crawl with resource blocking
search scrape docs.example.com --cloudflare --cf-limit 500 --cf-depth 5 \
  --cf-include '**docs.example.com/docs/**' \
  --cf-reject-resources image,font,stylesheet

# Include subdomains, wait for React root
search scrape example.com --cloudflare --cf-subdomains \
  --cf-wait-selector "#root" --cf-render

# Sitemap-only discovery
search scrape sqlite.org --cloudflare --cf-source sitemaps
```

### Flag reference

| Flag | Default | Description |
|---|---|---|
| `--cloudflare` | false | Enable CF mode |
| `--cf-limit N` | 0 (CF default: 10) | Max pages |
| `--cf-depth N` | 0 (CF default: unlimited) | Max link depth |
| `--cf-render` | false | Enable JS rendering (default: static HTML) |
| `--cf-source` | `"all"` | Discovery: `all`, `sitemaps`, `links` |
| `--cf-subdomains` | false | Follow subdomain links |
| `--cf-include` | — | Wildcard URL patterns to include |
| `--cf-exclude` | — | Wildcard URL patterns to exclude |
| `--cf-reject-resources` | — | Block: `image`, `media`, `font`, `stylesheet` |
| `--cf-wait-selector` | — | CSS selector to wait for before extracting |
| `--cf-goto-wait` | — | Nav event: `load`, `domcontentloaded`, `networkidle0`, `networkidle2` |
| `--cf-goto-timeout` | 0 | Per-page navigation timeout in ms |
| `--cf-user-agent` | — | Custom User-Agent string |

---

## Implementation Files

| File | Description |
|---|---|
| `pkg/scrape/cloudflare.go` | `CloudflareCredentials`, `CloudflareClient`, `CFOptions`, `cfPollLoop`, `RunCloudflareCrawl`, `storeCFRecord` |
| `pkg/scrape/config.go` | `UseCloudflare`, `CloudflareLimit`, `CloudflareDepth`, `CFOptions` in `Config` |
| `cli/dcrawl.go` | All CF flags; `runCloudflareScrape()` dispatched before normal crawl |

---

## Key Design Decisions

**`render=false` default**: Static HTML is ~3–5× faster and sufficient for most
documentation/content sites. Use `--cf-render` for dynamic apps.

**Why `formats: ["markdown"]` not `["html"]`**: Markdown is lighter, directly
usable for FTS indexing. If HTML is also needed, add `"html"` to formats.

**Why poll instead of webhook**: CF provides no streaming — polling is the only
option. 3s interval balances freshness vs. rate limits.

**Why skip `skipped`/`cancelled`/`disallowed` records**: These are CF-side
routing decisions (pattern filter, robots.txt, etc.), not crawl failures.
Counting them as errors would be misleading.

**Error display**: Actual failures (CF `errored` or HTTP non-2xx) are collected
and printed after the final summary so the rolling progress line isn't
interrupted.

---

## Limits & Rate Limits

| Limit | Value |
|---|---|
| Max pages per job | 100,000 |
| Max job runtime | 7 days |
| Results retention | 14 days post-completion |
| Response size | 10 MB per GET |
| Free plan browser time | 10 min/day |
| Rate limit error | HTTP 429, code 2001 |

---

## Rate Limit Handling

The free plan allows 10 min/day of browser time. When exceeded, CF returns
HTTP 429 with a `Retry-After` header (seconds until reset).

The CLI detects and displays this clearly:
```
  Start CF crawl: CF Browser Rendering rate limit exceeded
  Retry-After: 4h 48m (resets at 13:41:40 +07)
  Free plan: 10 min/day browser time. Upgrade at dash.cloudflare.com
```

Actual 429 response (from real test 2026-03-11):
```
HTTP/2 429
retry-after: 17280
cf-ray: 9da6eb778c990442-HKG

{"success":false,"errors":[{"code":2001,"message":"Rate limit exceeded"}]}
```

---

## Real Test Results

### Local (macOS, 2026-03-11)

```
search scrape sqlite.org --cloudflare --cf-limit 10

Scraping sqlite.org via Cloudflare Browser Rendering
  Data: /Users/apple/data/crawler/sqlite.org
  Submitting CF crawl job for https://sqlite.org
  Limit: 10 pages  Depth: 0
  Job ID: 8d30fcce-4321-4bcf-ade7-ab50c4ccba8a

  Completed  3/3 pages  stored:4  browser:0.0s  elapsed:13s
  Results: /Users/apple/data/crawler/sqlite.org/results/
```

Result shard contents:
```sql
SELECT url, status_code, length(markdown) as md_len, title FROM pages WHERE status_code > 0;

url                         status_code  md_len   title
https://sqlite.org/         200          3344     SQLite Home Page
https://sqlite.org/forum    200          4738     SQLite User Forum: Forum
https://sqlite.org/fiddle   200          2629     SQLite3 Fiddle
https://sqlite.org/cli.html 200          84053    Command Line Shell For SQLite
```

4 pages with full markdown content. External links (sqlite.org/src/*, zlib.net)
returned with `status="skipped"` — correctly not counted as errors.

### Server 2 (Ubuntu 24.04 Noble) — rate limited
Server 2 test hit the free plan 10 min/day limit (Retry-After: 17280s = 4.8h).
Binary deployed: `v0.5.26-256-gac22bf20-dirty` ✓
