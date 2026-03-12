# spec/0702 — Browser Worker: More REST API Endpoints

**Date:** 2026-03-11
**Status:** Design
**Scope:** `tools/browser/`

---

## Overview

Extend the `browser` Cloudflare Worker with seven new endpoints that mirror the Cloudflare Browser Rendering REST API exactly:

| Endpoint | Returns | Binary? | D1 cached? |
|---|---|---|---|
| `POST /api/content` | Rendered HTML string | no | yes |
| `POST /api/screenshot` | PNG/JPEG image | **yes** | no |
| `POST /api/pdf` | PDF document | **yes** | no |
| `POST /api/markdown` | Markdown string | no | yes |
| `POST /api/snapshot` | `{content, screenshot}` JSON | no | yes (html + result) |
| `POST /api/scrape` | CSS-selector extraction | no | yes |
| `POST /api/json` | AI-extracted JSON | no | yes |
| `POST /api/links` | Array of URLs | no | yes |

Each text-based endpoint (all except screenshot and pdf) runs through a **4-layer stack**:

```
1. In-memory cache  (Map, no TTL, isolate-scoped)
   ↓ miss
2. D1 cache  (persistent, no forced TTL — cache forever by default)
   ↓ miss
3. CF Browser Rendering proxy  (REST API: api.cloudflare.com)
   ↓ 429 rate-limited
4. Own fallback  (fetch() + HTMLRewriter + our htmlToMarkdown)
```

Binary endpoints (screenshot, pdf) skip layers 1–2 and go directly to CF (layer 3), returning **503** if CF is rate-limited (layer 4 cannot emulate headless rendering).

---

## Request/Response Compatibility

All request bodies and response shapes **exactly match** the Cloudflare Browser Rendering REST API as documented at `https://developers.cloudflare.com/browser-rendering/rest-api/`. The worker acts as a transparent proxy with local caching and a graceful fallback.

### Shared request fields (all endpoints accept these)

```typescript
interface SharedRequest {
  url?: string;                       // target URL (one of url/html required)
  html?: string;                      // raw HTML to render instead of fetching url
  gotoOptions?: {
    waitUntil?: "domcontentloaded" | "networkidle0" | "networkidle2";
    timeout?: number;                 // ms, max 60 000, default ~4500
  };
  cookies?: Array<{
    name: string;
    value: string;
    domain?: string;
    path?: string;                    // default "/"
    secure?: boolean;
    httpOnly?: boolean;
  }>;
  authenticate?: { username: string; password: string };
  setExtraHTTPHeaders?: Record<string, string>;
  userAgent?: string;
  viewport?: {
    width?: number;                   // default 1920
    height?: number;                  // default 1080
    deviceScaleFactor?: number;       // default 1
  };
  waitForSelector?: string | {
    selector: string;
    timeout?: number;
    visible?: boolean;
  };
  addScriptTag?: Array<{ content: string }>;
  addStyleTag?: Array<{ content?: string; url?: string }>;
  setJavaScriptEnabled?: boolean;
  rejectResourceTypes?: string[];     // "image"|"media"|"font"|"stylesheet"|…
  rejectRequestPattern?: string[];
  allowResourceTypes?: string[];
  allowRequestPattern?: string[];
}
```

### Shared response envelope

All text-based endpoints return JSON with this wrapper (matching CF):

```json
{ "success": true,  "result": <payload> }
{ "success": false, "errors": [{ "code": 1001, "message": "..." }], "result": null }
```

Binary endpoints (screenshot, pdf) return raw bytes with the appropriate `Content-Type`.

---

## Endpoint Specifications

### POST /api/content

Mirrors `POST /browser-rendering/content`.

**Additional request fields:** (none beyond shared)

**Response:** `application/json`
```json
{ "success": true, "result": "<html>...</html>" }
```
`result` is the full rendered HTML string.

**Cache key:** `(url, "content", "")` — URL alone identifies the result.

**Fallback:** `fetch(url)` → return `response.text()` (no JS rendering).

---

### POST /api/screenshot

Mirrors `POST /browser-rendering/screenshot`.

**Additional request fields:**
```typescript
interface ScreenshotRequest extends SharedRequest {
  screenshotOptions?: {
    type?: "png" | "jpeg";           // default "png"
    quality?: number;                // 0–100, JPEG only
    fullPage?: boolean;              // default false
    omitBackground?: boolean;        // default false, PNG only
    clip?: { x: number; y: number; width: number; height: number };
    captureBeyondViewport?: boolean;
  };
  selector?: string;                 // capture only this CSS element
}
```

**Response:** `image/png` or `image/jpeg` binary bytes.

**Caching:** None (binary, too large for D1 1MB row limit).

**Fallback:** Returns `503 Service Unavailable` JSON:
```json
{ "success": false, "errors": [{ "code": 429, "message": "CF rate limited; screenshot requires a real browser" }], "result": null }
```

---

### POST /api/pdf

Mirrors `POST /browser-rendering/pdf`.

**Additional request fields:**
```typescript
interface PdfRequest extends SharedRequest {
  pdfOptions?: {
    format?: string;                 // "a4"|"letter"|"a5"|… (Puppeteer paper formats)
    landscape?: boolean;             // default false
    printBackground?: boolean;       // default false
    preferCSSPageSize?: boolean;     // default false
    scale?: number;                  // default 1.0
    displayHeaderFooter?: boolean;   // default false
    headerTemplate?: string;         // HTML; placeholders: .pageNumber .totalPages .date .title
    footerTemplate?: string;
    margin?: { top?: string; bottom?: string; left?: string; right?: string };
    timeout?: number;
  };
}
```

**Response:** `application/pdf` binary bytes.

**Caching:** None.

**Fallback:** Returns `503 Service Unavailable` (same shape as screenshot fallback).

---

### POST /api/markdown

Mirrors `POST /browser-rendering/markdown`.

**Additional request fields:** (none beyond shared)

**Response:** `application/json`
```json
{ "success": true, "result": "# Page Title\n\nMarkdown content…" }
```

**Cache key:** `(url, "markdown", "")`.

**Fallback:** `fetch(url)` → `htmlToMarkdown(html)` using our existing converter in `markdown.ts`.

---

### POST /api/snapshot

Mirrors `POST /browser-rendering/snapshot`.

**Additional request fields:**
```typescript
interface SnapshotRequest extends SharedRequest {
  screenshotOptions?: {
    fullPage?: boolean;              // default false
  };
}
```

**Response:** `application/json`
```json
{
  "success": true,
  "result": {
    "content": "<html>…</html>",
    "screenshot": "<base64-encoded PNG string>"
  }
}
```

**Cache key:** `(url, "snapshot", "")`.
- D1 `html` column → `content` field.
- D1 `result` column → JSON `{ "screenshot": "<base64>" }` (base64 string is large but text; snapshots are typically small pages).

**Fallback:** CF-returned 429 → fetch HTML ourselves, set `screenshot: null` in result:
```json
{ "success": true, "result": { "content": "<html>…</html>", "screenshot": null } }
```
This is a **graceful degradation** — callers must handle `screenshot: null`.

---

### POST /api/scrape

Mirrors `POST /browser-rendering/scrape`.

**Additional request fields:**
```typescript
interface ScrapeRequest extends SharedRequest {
  elements: Array<{ selector: string }>;  // required
}
```

**Response:** `application/json`
```json
{
  "success": true,
  "result": [
    {
      "selector": "h1",
      "results": [
        {
          "text": "Page Title",
          "html": "<h1>Page Title</h1>",
          "attributes": [{ "name": "class", "value": "title" }],
          "height": 42,
          "width": 800,
          "top": 120,
          "left": 0
        }
      ]
    }
  ]
}
```

**Cache key:** `(url, "scrape", hash(selectors))`.
`hash(selectors)` = sorted selector strings joined with `\0`, SHA-256 hex truncated to 16 chars.

**Fallback:** `fetch(url)` → use `HTMLRewriter` to extract matching elements.
- `text`, `html`, `attributes` available via HTMLRewriter.
- `height`, `width`, `top`, `left` = **0** (no layout engine in fallback; document clearly in response as approximation).

---

### POST /api/json

Mirrors `POST /browser-rendering/json`.

**Additional request fields:**
```typescript
interface JsonRequest extends SharedRequest {
  prompt?: string;
  response_format?: {
    type: "json_schema";
    schema: Record<string, unknown>;  // JSON Schema object
  };
  custom_ai?: Array<{
    model: string;                    // "<provider>/<model>" format
    authorization: string;
  }>;
}
```

**Response:** `application/json`
```json
{ "success": true, "result": { /* extracted structured data */ } }
```

**Cache key:** `(url, "json", hash({prompt, schema}))`.
`hash` = SHA-256 hex of `JSON.stringify({prompt, schema})` truncated to 16 chars.

**Fallback:** `fetch(url)` → `htmlToMarkdown(html)` → call Workers AI (`@cf/meta/llama-3.1-8b-instruct-fast` or similar) if `AI` binding is available; else return:
```json
{ "success": false, "errors": [{ "code": 503, "message": "AI extraction unavailable; CF rate limited" }], "result": null }
```

> The `AI` binding is optional. If not configured in wrangler.toml the fallback skips the AI step.

---

### POST /api/links

Mirrors `POST /browser-rendering/links`.

**Additional request fields:**
```typescript
interface LinksRequest extends SharedRequest {
  visibleLinksOnly?: boolean;        // default false
  excludeExternalLinks?: boolean;    // default false
}
```

**Response:** `application/json`
```json
{ "success": true, "result": ["https://example.com/a", "https://example.com/b"] }
```

**Cache key:** `(url, "links", "${visibleLinksOnly}:${excludeExternalLinks}")`.
The filtering flags are deterministic and short enough to use as a literal params_hash.

**Fallback:** `fetch(url)` → `extractLinks(html, url)` (reuse existing `links.ts`).
- `visibleLinksOnly`: not enforceable without a real browser; return all links when falling back, add header `X-Fallback: true`.
- `excludeExternalLinks`: enforced in our code via URL hostname comparison.

---

## D1 Cache Schema

Add a new `page_cache` table to `schema.sql`:

```sql
-- Cache for single-URL rendering endpoints
-- PK = (url, endpoint, params_hash) covers all parameterized variants
CREATE TABLE IF NOT EXISTS page_cache (
  url          TEXT    NOT NULL,
  endpoint     TEXT    NOT NULL,            -- 'content'|'markdown'|'links'|'snapshot'|'scrape'|'json'
  params_hash  TEXT    NOT NULL DEFAULT '', -- '' for simple endpoints; hash for parameterized ones
  html         TEXT,                        -- rendered HTML (content, snapshot)
  markdown     TEXT,                        -- markdown text (markdown)
  result       TEXT,                        -- JSON-encoded payload (links array, scrape array, json obj, snapshot screenshot)
  title        TEXT,
  created_at   INTEGER NOT NULL,

  PRIMARY KEY (url, endpoint, params_hash)
);

-- Lookup by URL across all endpoints (e.g. to invalidate all cached data for a URL)
CREATE INDEX IF NOT EXISTS idx_page_cache_url ON page_cache(url);

-- TTL sweeping: find oldest entries
CREATE INDEX IF NOT EXISTS idx_page_cache_created ON page_cache(created_at);
```

**Column usage by endpoint:**

| Endpoint | `html` | `markdown` | `result` | `title` |
|---|---|---|---|---|
| `/content` | full HTML | — | — | extracted |
| `/markdown` | — | markdown string | — | extracted |
| `/snapshot` | full HTML | — | `{"screenshot":"<b64>"}` | extracted |
| `/scrape` | — | — | JSON array (CF shape) | — |
| `/json` | — | — | JSON object | — |
| `/links` | — | — | JSON array of URLs | — |

**No TTL by default.** Cache entries are permanent unless explicitly invalidated. A future `DELETE /api/cache?url=…` endpoint can clear entries. TTL enforcement is out of scope for this spec.

---

## In-Memory Cache

A module-level `Map` in each new handler file acts as the L1 cache:

```typescript
// Shared across all endpoint handlers in the same isolate
const memCache = new Map<string, MemCacheEntry>();

interface MemCacheEntry {
  html?: string | null;
  markdown?: string | null;
  result?: string | null;  // JSON string
  title?: string | null;
  ts: number;              // Date.now() at insertion
}
```

**Cache key format:** `` `${url}\0${endpoint}\0${paramsHash}` `` — null byte separator prevents collisions.

**No TTL.** The isolate is typically recycled after ~30s idle; the map lives for the isolate's lifetime. For high-traffic deployments this provides meaningful L1 hit rates on repeated calls within the same request burst.

---

## CF Browser Rendering Credentials

The worker needs the CF account credentials to proxy requests. These are stored as Worker secrets (not in wrangler.toml):

```
CF_ACCOUNT_ID   — Cloudflare account ID
CF_API_TOKEN    — Browser Rendering Edit token
```

When both secrets are present, layer 3 (CF proxy) is active. When absent, the stack goes directly to layer 4 (own fallback).

The proxy function:

```typescript
async function proxyCF(
  endpoint: string,
  body: unknown,
  env: Env
): Promise<{ ok: boolean; rateLimited: boolean; headers: Headers; blob: Blob | null; json: unknown }> {
  const url = `https://api.cloudflare.com/client/v4/accounts/${env.CF_ACCOUNT_ID}/browser-rendering/${endpoint}`;
  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${env.CF_API_TOKEN}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });
  return {
    ok: res.ok,
    rateLimited: res.status === 429,
    headers: res.headers,
    blob: res.ok ? await res.blob() : null,
    json: !res.ok ? await res.json() : null,
  };
}
```

The `X-Browser-Ms-Used` response header from CF is forwarded to the caller unchanged.

---

## Error Handling

| Situation | HTTP status | Response |
|---|---|---|
| Neither `url` nor `html` provided | 400 | `errors: [{code:1001, message:"url or html is required"}]` |
| `url` not a valid URL | 400 | `errors: [{code:1001, message:"url is not a valid URL"}]` |
| `/scrape` missing `elements` | 400 | `errors: [{code:1001, message:"elements is required"}]` |
| CF returns 429 (text endpoints) | — | fall through to layer 4 |
| CF returns 429 (binary endpoints) | 503 | `errors: [{code:429, message:"CF rate limited; …"}]` |
| CF returns 422 | 422 | forward CF error body |
| Fallback fetch fails | 502 | `errors: [{code:502, message:"Failed to fetch URL"}]` |

---

## File Layout

```
tools/browser/src/
  index.ts          ← add 8 new routes
  types.ts          ← add new request/response interfaces
  cache.ts          ← NEW: in-memory Map + D1 read/write helpers + paramsHash()
  cf.ts             ← NEW: proxyCF() function
  content.ts        ← NEW: handleContent()
  screenshot.ts     ← NEW: handleScreenshot()
  pdf.ts            ← NEW: handlePdf()
  markdown-ep.ts    ← NEW: handleMarkdown() (avoid name clash with markdown.ts)
  snapshot.ts       ← NEW: handleSnapshot()
  scrape.ts         ← NEW: handleScrape()
  json-ep.ts        ← NEW: handleJson() (avoid name clash)
  links-ep.ts       ← NEW: handleLinks()
  -- existing --
  auth.ts
  crawl.ts
  links.ts          ← reused by links-ep.ts and queue.ts
  markdown.ts       ← reused by markdown-ep.ts, json-ep.ts, queue.ts
  patterns.ts
  queue.ts
schema.sql          ← add page_cache table + indexes
wrangler.toml       ← add CF_ACCOUNT_ID, CF_API_TOKEN secret declarations
```

---

## wrangler.toml Changes

```toml
# Secrets — set via: wrangler secret put CF_ACCOUNT_ID / CF_API_TOKEN
# (no values in toml; listed here for documentation)

# Optional AI binding for /json fallback
# [[ai]]
# binding = "AI"
```

---

## Implementation Notes

### params_hash computation

Use `crypto.subtle.digest` (available in CF Workers):

```typescript
async function hashParams(obj: unknown): Promise<string> {
  if (obj === null || obj === undefined) return "";
  const text = JSON.stringify(obj, Object.keys(obj as object).sort());
  const buf = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(text));
  return Array.from(new Uint8Array(buf)).map(b => b.toString(16).padStart(2, "0")).join("").slice(0, 16);
}
```

### Cache write-through

After a successful CF proxy or fallback response, write to D1 then update the in-memory map. D1 write uses `INSERT OR REPLACE` (upsert):

```sql
INSERT OR REPLACE INTO page_cache (url, endpoint, params_hash, html, markdown, result, title, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
```

### Fallback HTMLRewriter for /scrape

```typescript
// For each selector, run a separate HTMLRewriter pass
const results = await Promise.all(
  elements.map(async ({ selector }) => {
    const matches: ScrapeResult[] = [];
    const rw = new HTMLRewriter().on(selector, {
      element(el) {
        const attrs = [];
        for (const [name, value] of el.attributes) attrs.push({ name, value });
        // Note: text/html require async handlers (element.onEndTag or text handler)
        matches.push({ text: "", html: "", attributes: attrs, height: 0, width: 0, top: 0, left: 0 });
      },
      text(chunk) { /* accumulate */ },
    }).transform(new Response(html, { headers: { "Content-Type": "text/html" } }));
    await rw.text();
    return { selector, results: matches };
  })
);
```

### Route registration (index.ts additions)

```typescript
app.post("/api/content",    handleContent);
app.post("/api/screenshot", handleScreenshot);
app.post("/api/pdf",        handlePdf);
app.post("/api/markdown",   handleMarkdown);
app.post("/api/snapshot",   handleSnapshot);
app.post("/api/scrape",     handleScrape);
app.post("/api/json",       handleJson);
app.post("/api/links",      handleLinks);
```

---

## Out of Scope

- Cache invalidation endpoint (`DELETE /api/cache`) — future work
- TTL-based eviction — future work
- `/api/crawl` changes — existing, unchanged
- R2 storage for screenshot/pdf caching — future work (see spec/0702 alternative B)
- Workers AI binding for `/json` fallback — optional; listed in wrangler.toml but not required
