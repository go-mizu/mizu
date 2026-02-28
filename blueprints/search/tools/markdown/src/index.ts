import { Hono } from 'hono';
import { convert } from './convert';
import { renderPage } from './page';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type Env = { AI: any; BROWSER: Fetcher };

const app = new Hono<{ Bindings: Env }>();

// Landing page
app.get('/', (c) => c.html(renderPage()));

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

// Text API: GET /:url+ (mirrors markdown.new/https://example.com pattern)
// Matches any path starting with http:// or https://
app.get('/*', async (c) => {
  const path = c.req.path.slice(1); // strip leading /
  if (!path.startsWith('http://') && !path.startsWith('https://')) {
    return c.notFound();
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
        'X-Title': encodeURIComponent(result.title),
        ...(result.tokens ? { 'X-Markdown-Tokens': String(result.tokens) } : {}),
        'Access-Control-Allow-Origin': '*',
        'Cache-Control': 'public, max-age=300',
      },
    });
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'Conversion failed';
    return c.text(`Error: ${msg}`, 422);
  }
});

export default app;
