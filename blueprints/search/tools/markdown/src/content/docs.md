# URL → Markdown

Free, instant URL-to-Markdown conversion for AI agents and LLM pipelines. No API key, no account.

## Overview

Convert any HTTP/HTTPS URL to clean, structured Markdown with a single request.

- Works with any HTTP/HTTPS URL
- Three-tier pipeline: native negotiation → Workers AI → Browser rendering
- Edge-cached for 1 hour with stale-while-revalidate
- CORS-enabled — fetch from any origin, no proxy needed

## Quick start

Fetch as Markdown:

```bash
curl https://markdown.go-mizu.workers.dev/https://example.com
```

Use the JSON API:

```bash
curl -X POST https://markdown.go-mizu.workers.dev/convert \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}'
```

JavaScript:

```javascript
const md = await fetch(
  'https://markdown.go-mizu.workers.dev/' + url
).then(r => r.text());
```

Python:

```python
import httpx
md = httpx.get('https://markdown.go-mizu.workers.dev/' + url).text
```

## GET /{url}

Convert a URL to Markdown. Append any `http://` or `https://` URL to the worker base URL. Query strings are preserved.

```bash
curl https://markdown.go-mizu.workers.dev/https://example.com?q=hello
```

## POST /convert

Convert a URL and receive a structured JSON response.

Request body:

```json
{"url": "https://example.com"}
```

Response:

```json
{
  "markdown": "# Example Domain\n\n...",
  "method": "primary",
  "durationMs": 342,
  "title": "Example Domain",
  "tokens": 1248
}
```

`method` is one of `primary`, `ai`, or `browser`.

## Conversion pipeline

Every URL goes through up to three tiers, falling back automatically:

- **Tier 1 — Native:** Requests with `Accept: text/markdown`. Sites that support this return structured Markdown directly.
- **Tier 2 — Workers AI:** Fetches HTML and converts via Cloudflare Workers AI `toMarkdown()`.
- **Tier 3 — Browser:** For JS-heavy SPAs. Renders in a headless browser via Puppeteer, then passes to Workers AI.

## Response headers

The `GET /{url}` endpoint returns these headers:

| Header | Description |
|---|---|
| `X-Conversion-Method` | `primary`, `ai`, or `browser` |
| `X-Duration-Ms` | Server-side processing time in milliseconds |
| `X-Title` | Percent-encoded page title (max 200 chars) |
| `X-Markdown-Tokens` | Approximate token count (when available) |
| `Cache-Control` | `public, max-age=300, s-maxage=3600, stale-while-revalidate=86400` |

## Error responses

| Status | When |
|---|---|
| `400` | Missing or invalid `url` field in POST body |
| `422` | Conversion failed (fetch error, unsupported content) |

Error body for `POST /convert`:

```json
{"error": "description of what went wrong"}
```

The `GET /{url}` endpoint returns plain text: `Error: description`

## CORS

All endpoints return `Access-Control-Allow-Origin: *`. You can call the API directly from browser JavaScript with no proxy needed.

```javascript
// Works in browser — no CORS errors
const md = await fetch(
  'https://markdown.go-mizu.workers.dev/' + url
).then(r => r.text());
```

## Limits

- Max response body: **5 MB** per URL
- Fetch timeout: **10 seconds** (30 seconds for browser rendering)
- Protocols: **http://** and **https://** only
- Rate limits: Cloudflare Workers free tier (100,000 requests/day)
