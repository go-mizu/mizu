# browser — Cloudflare Worker for headless rendering

A Cloudflare Worker that exposes a set of HTTP endpoints for rendering web pages using Cloudflare's Browser Rendering API, with a D1-backed cache and queue-driven async crawl.

Deployed at: `https://browser.go-mizu.workers.dev`

## Auth

All `/api/*` routes require a Bearer token:

```
Authorization: Bearer <MIZU_TOKEN>
```

## Endpoints

### `POST /api/content`
Fetch raw HTML for a URL. Falls back to plain `fetch` if CF Browser Rendering is unavailable.

```json
{ "url": "https://example.com" }
```

Response: `{ "success": true, "result": "<html>..." }`

---

### `POST /api/markdown`
Fetch a page and convert its HTML to Markdown.

```json
{ "url": "https://example.com" }
```

Response: `{ "success": true, "result": "# Page Title\n..." }`

---

### `POST /api/screenshot`
Capture a full-page PNG screenshot.

```json
{
  "url": "https://example.com",
  "screenshotOptions": { "fullPage": true, "type": "png" }
}
```

Response: base64-encoded image in `result`.

---

### `POST /api/pdf`
Render a page as PDF.

```json
{
  "url": "https://example.com",
  "pdfOptions": { "format": "A4", "printBackground": true }
}
```

Response: base64-encoded PDF in `result`.

---

### `POST /api/scrape`
Extract elements matching CSS selectors (text, HTML, attributes, bounding box).

```json
{
  "url": "https://example.com",
  "elements": [{ "selector": "h1" }, { "selector": ".price" }]
}
```

Response: array of `{ selector, results: [{ text, html, attributes, height, width, top, left }] }`.

---

### `POST /api/snapshot`
Returns both HTML content and a full-page screenshot in one call.

```json
{ "url": "https://example.com" }
```

Response: `{ "content": "<html>...", "screenshot": "<base64>" }`.

---

### `POST /api/json`
Extract structured JSON from a page using CF Workers AI (or a custom AI model).

```json
{
  "url": "https://example.com",
  "prompt": "Extract the product name and price",
  "response_format": {
    "type": "json_schema",
    "schema": { "name": { "type": "string" }, "price": { "type": "number" } }
  }
}
```

---

### `POST /api/links`
Return all hyperlinks found on a page.

```json
{ "url": "https://example.com", "excludeExternalLinks": true }
```

---

### `POST /api/crawl`
Start an async multi-page crawl job (queued via Cloudflare Queues).

```json
{
  "url": "https://example.com",
  "limit": 100,
  "depth": 3,
  "formats": ["markdown"],
  "options": {
    "includeSubdomains": false,
    "includeExternalLinks": false
  }
}
```

Response: `{ "success": true, "result": { "id": "<job-id>", ... } }`

Poll status: `GET /api/crawl/:id`
Cancel: `DELETE /api/crawl/:id`

---

## Shared Request Options

All single-URL endpoints accept these optional fields:

| Field | Description |
|---|---|
| `userAgent` | Override User-Agent header |
| `setExtraHTTPHeaders` | Extra request headers |
| `cookies` | Set cookies before navigation |
| `authenticate` | HTTP Basic auth credentials |
| `viewport` | `{ width, height, deviceScaleFactor }` |
| `gotoOptions` | `{ waitUntil, timeout }` |
| `waitForSelector` | Wait for selector before returning |
| `addScriptTag` | Inject `<script>` tags |
| `addStyleTag` | Inject `<style>` tags |
| `setJavaScriptEnabled` | Disable JS (default: enabled) |
| `rejectResourceTypes` | Block resource types (e.g. `["image"]`) |

---

## Rendering Layers

Requests are served through a layered fallback chain:

1. **L1/L2 cache** — D1 database (`browser-db`). Cache hit returns immediately.
2. **L3 CF Browser Rendering** — Cloudflare's headless Chrome API. Requires `CF_ACCOUNT_ID` + `CF_API_TOKEN` worker secrets with *Browser Rendering > Edit* permission.
3. **L4 own fallback** — plain `fetch()` from the Worker runtime. No JS execution.

If CF Browser Rendering is not configured or is rate-limited, the worker falls back to L4 automatically.

---

## Deployment

```bash
cd tools/browser
npx wrangler deploy

# Set required secrets
npx wrangler secret put AUTH_TOKEN       # set to $MIZU_TOKEN
npx wrangler secret put CF_ACCOUNT_ID   # Cloudflare Account ID (for L3)
npx wrangler secret put CF_API_TOKEN    # CF API token with Browser Rendering perm
```

Required D1 database and Queue bindings are defined in `wrangler.toml`.
