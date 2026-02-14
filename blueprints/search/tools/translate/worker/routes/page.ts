import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { makePageRewriter } from '../page-rewriter'

const route = new Hono<HonoEnv>()

// GET /page/:tl/*  — fetch, translate, and proxy an HTML page
route.get('/page/:tl{[a-zA-Z]{2,3}(-[a-zA-Z]{2})?}/*', async (c) => {
  const tl = c.req.param('tl')

  // Extract the target URL from the path after /page/<tl>/
  // Browsers may encode ":" as "%3A" and collapse "//" to "/" in the path,
  // so we decode percent-encoding and fix the scheme separator.
  const reqUrl = new URL(c.req.url)
  const prefix = `/page/${tl}/`
  const prefixIdx = reqUrl.pathname.indexOf(prefix)
  if (prefixIdx === -1) {
    return c.json({ error: 'Invalid URL. Use /page/<lang>/https://example.com' }, 400)
  }

  let targetUrl = decodeURIComponent(reqUrl.pathname.substring(prefixIdx + prefix.length))
  // Fix single-slash after scheme: "https:/example.com" → "https://example.com"
  targetUrl = targetUrl.replace(/^(https?):\/(?!\/)/, '$1://')
  // Append query string
  targetUrl += reqUrl.search

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
