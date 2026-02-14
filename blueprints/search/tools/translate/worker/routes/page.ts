import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { makePageRewriter } from '../page-rewriter'

const route = new Hono<HonoEnv>()

function cacheKey(tl: string, url: string): string {
  return `page:${tl}:${url}`
}

// GET /page/:tl?url=...  — fetch, translate, and proxy an HTML page
route.get('/page/:tl{[a-zA-Z]{2,3}(-[a-zA-Z]{2})?}', async (c) => {
  const tl = c.req.param('tl')
  const targetUrl = c.req.query('url')

  if (!targetUrl || !targetUrl.startsWith('http')) {
    return c.json({ error: 'Invalid URL. Use /page/<lang>?url=https://example.com' }, 400)
  }

  let originUrl: URL
  try {
    originUrl = new URL(targetUrl)
  } catch {
    return c.json({ error: 'Invalid URL format' }, 400)
  }

  const kv = c.env.TRANSLATE_CACHE

  // 1. Check KV cache
  const cached = await kv.get(cacheKey(tl, targetUrl), 'text')
  if (cached) {
    return new Response(cached, {
      headers: {
        'Content-Type': 'text/html; charset=utf-8',
        'Cache-Control': 'public, max-age=3600',
        'X-Translate-Cache': 'HIT',
        'X-Robots-Tag': 'noindex',
      },
    })
  }

  // 2. Fetch the original page
  let response: Response
  try {
    response = await fetch(originUrl.toString(), {
      headers: {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
        'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
        'Accept-Language': 'en-US,en;q=0.9',
      },
      redirect: 'follow',
    })
  } catch {
    return c.json({ error: 'Failed to fetch page' }, 502)
  }

  const contentType = response.headers.get('content-type') || ''

  // Non-HTML content: proxy through directly
  if (!contentType.includes('text/html')) {
    return new Response(response.body, {
      status: response.status,
      headers: {
        'Content-Type': contentType,
        'Cache-Control': 'public, max-age=3600',
      },
    })
  }

  // 3. Stream translated response — no buffering, no timeout risk
  const proxyBase = new URL(c.req.url).origin
  const rewriter = makePageRewriter(originUrl, proxyBase, tl, 'auto')

  return new Response(rewriter.transform(response).body, {
    headers: {
      'Content-Type': 'text/html; charset=utf-8',
      'Cache-Control': 'no-cache',
      'X-Translate-Cache': 'MISS',
      'X-Robots-Tag': 'noindex',
    },
  })
})

// POST /page/cache  — client pushes fully translated HTML for KV storage
route.post('/page/cache', async (c) => {
  const body = await c.req.json<{ url: string; tl: string; html: string }>()
  if (!body.url || !body.tl || !body.html) {
    return c.json({ error: 'Missing url, tl, or html' }, 400)
  }

  const kv = c.env.TRANSLATE_CACHE
  await kv.put(cacheKey(body.tl, body.url), body.html, { expirationTtl: 86400 })

  return c.json({ ok: true })
})

export default route
