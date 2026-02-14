import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { makePageRewriter } from '../page-rewriter'

const route = new Hono<HonoEnv>()

// GET /page/:tl/*  — fetch, translate, and proxy an HTML page
route.get('/page/:tl{[a-zA-Z]{2,3}(-[a-zA-Z]{2})?}/*', async (c) => {
  const tl = c.req.param('tl')

  // Extract the target URL from the wildcard portion of the path
  const urlPath = c.req.path.replace(`/page/${tl}/`, '')
  const queryString = new URL(c.req.url).search
  const targetUrl = urlPath + queryString

  if (!targetUrl || !targetUrl.startsWith('http')) {
    return c.json({ error: 'Invalid URL. Use /page/<lang>/https://example.com' }, 400)
  }

  let originUrl: URL
  try {
    originUrl = new URL(targetUrl)
  } catch {
    return c.json({ error: 'Invalid URL format' }, 400)
  }

  // Fetch the original page
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

  // Non-HTML content: proxy through directly (images, CSS, JS, etc.)
  if (!contentType.includes('text/html')) {
    return new Response(response.body, {
      status: response.status,
      headers: {
        'Content-Type': contentType,
        'Cache-Control': 'public, max-age=3600',
      },
    })
  }

  const proxyBase = new URL(c.req.url).origin
  const rewriter = makePageRewriter(originUrl, proxyBase, tl, 'auto')

  return rewriter.transform(response)
})

export default route
