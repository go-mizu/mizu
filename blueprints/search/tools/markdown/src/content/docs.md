# URL to Markdown

Turn any URL into clean Markdown with a single request. No API key, no account, no setup.

## Overview

Pass any public URL and get back structured Markdown your agent can actually use.

- Works with any HTTP or HTTPS URL
- Three-tier pipeline: native negotiation, AI extraction, browser rendering
- Results cached for 1 hour
- CORS open on all endpoints, call from anywhere

## Quick start

Get Markdown from a URL:

```bash
curl https://markdown.go-mizu.workers.dev/https://example.com
```

Get JSON with metadata:

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

Append any URL to the base endpoint. Returns `text/markdown`. Query strings are preserved.

```bash
curl https://markdown.go-mizu.workers.dev/https://example.com?q=hello
```

## POST /convert

Returns a JSON object with the Markdown plus metadata.

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

- **Tier 1 — Native:** Requests `Accept: text/markdown`. Sites that support this return Markdown directly.
- **Tier 2 — AI:** Fetches HTML and converts via an AI model. Fast, structure-aware extraction.
- **Tier 3 — Browser:** For JS-heavy SPAs. Renders in a headless browser, then converts.

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

All endpoints return `Access-Control-Allow-Origin: *`. Call directly from browser JavaScript with no proxy.

```javascript
const md = await fetch(
  'https://markdown.go-mizu.workers.dev/' + url
).then(r => r.text());
```

## Limits

- Max response body: **5 MB** per URL
- Fetch timeout: **10 seconds** (20 seconds for browser rendering)
- Protocols: **http://** and **https://** only
- No hard rate limit for reasonable use
