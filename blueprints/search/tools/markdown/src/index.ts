import { Hono } from 'hono';
import { convert } from './convert';
import { renderDocs } from './docs';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type Env = { AI: any; BROWSER: Fetcher; ASSETS: Fetcher };

const app = new Hono<{ Bindings: Env }>();

// llms.txt — machine-readable API summary for LLM agents
app.get('/llms.txt', (c) =>
  c.text(`# URL to Markdown API
# https://markdown.go-mizu.workers.dev

> Convert any URL to clean Markdown. Free, no auth required.

## Base URL
https://markdown.go-mizu.workers.dev

## Endpoints

### GET /{url}
Convert a URL to Markdown. Append any absolute URL (http:// or https://) to the base.
Returns: text/markdown
Headers:
  X-Conversion-Method: primary | ai | browser
  X-Duration-Ms: <milliseconds>
  X-Title: <percent-encoded page title>
  X-Markdown-Tokens: <approximate token count>
  Cache-Control: public, max-age=300, s-maxage=3600, stale-while-revalidate=86400

Example:
  curl https://markdown.go-mizu.workers.dev/https://example.com

### POST /convert
Convert a URL and receive structured JSON.
Body: {"url": "https://example.com"}
Returns: application/json
  {
    "markdown": "<markdown content>",
    "method": "primary" | "ai" | "browser",
    "durationMs": <number>,
    "title": "<page title>",
    "tokens": <number | undefined>
  }

Example:
  curl -X POST https://markdown.go-mizu.workers.dev/convert \\
    -H 'Content-Type: application/json' \\
    -d '{"url":"https://example.com"}'

## Conversion Pipeline
1. primary — Sites supporting Accept: text/markdown return clean Markdown directly
2. ai      — HTML converted via Cloudflare Workers AI toMarkdown()
3. browser — JS-heavy pages rendered in headless browser before AI conversion

## Usage Notes
- Max response body: 5 MB
- CORS: Access-Control-Allow-Origin: *
- Edge-cached: s-maxage=3600 with stale-while-revalidate=86400
`, { headers: { 'Content-Type': 'text/plain; charset=utf-8', 'Cache-Control': 'public, max-age=86400' } })
);

// JSON API: POST /convert
app.post('/convert', async (c) => {
  let body: { url?: string };
  try {
    body = await c.req.json<{ url?: string }>();
  } catch {
    return c.json({ error: 'Invalid JSON body' }, 400);
  }
  if (!body.url || typeof body.url !== 'string') {
    return c.json({ error: 'url is required' }, 400);
  }
  try {
    const result = await convert(body.url, c.env);
    return c.json(result);
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'Conversion failed';
    return c.json({ error: msg }, 422);
  }
});

// CORS preflight for cross-origin API usage
app.options('/*', (c) => {
  return new Response(null, {
    status: 204,
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type',
      'Access-Control-Max-Age': '86400',
    },
  });
});

// Docs page
app.get('/docs', (c) => c.html(renderDocs()));

// Text API: GET /:url+ (mirrors markdown.new/https://example.com pattern)
// Matches any path starting with http:// or https://
// For other paths, falls through to the static assets handler below.
app.get('/*', async (c, next) => {
  const path = c.req.path.slice(1); // strip leading /
  if (!path.startsWith('http://') && !path.startsWith('https://')) {
    return next(); // fall through to static assets
  }
  // Reconstruct full URL including query string
  const search = new URL(c.req.url).search;
  const url = path + search;
  try {
    const result = await convert(url, c.env);
    return new Response(result.markdown, {
      headers: {
        'Content-Type': 'text/markdown; charset=utf-8',
        'X-Conversion-Method': result.method,
        'X-Duration-Ms': String(result.durationMs),
        'X-Title': encodeURIComponent(result.title.slice(0, 200)),
        ...(result.tokens ? { 'X-Markdown-Tokens': String(result.tokens) } : {}),
        'Access-Control-Allow-Origin': '*',
        'Cache-Control': 'public, max-age=300, s-maxage=3600, stale-while-revalidate=86400',
      },
    });
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'Conversion failed';
    return c.text(`Error: ${msg}`, 422);
  }
});

// Fall through to static assets (public/ directory)
app.get('*', (c) => c.env.ASSETS.fetch(c.req.raw));

export default app;
