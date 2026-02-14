import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { extractTexts, batchTranslate, makePageRewriter } from '../page-rewriter'
import { needsBrowserRender, renderWithBrowser } from '../renderer'

const route = new Hono<HonoEnv>()

// Fixed nonce for CSP — blocks all original page scripts while allowing our injected ones.
// Not random because cached HTML must work with the same CSP header.
const CSP_NONCE = 'tl'
const CSP_HEADER = `script-src 'nonce-${CSP_NONCE}'`

function cacheKey(tl: string, url: string): string {
  // v2: invalidates old cache entries poisoned by Next.js hydration
  // (old translate-cs cached DOM after async scripts reverted translations)
  return `page:v2:${tl}:${url}`
}

// GET /page/:tl?url=...  — fetch, translate, and proxy an HTML page
route.get('/page/:tl{[a-zA-Z]{2,3}(-[a-zA-Z]{2})?}', async (c) => {
  const tl = c.req.param('tl')
  const targetUrl = c.req.query('url')
  const forceRender = c.req.query('render') === '1'
  const noCache = c.req.query('nocache') === '1'

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
  const t0 = Date.now()
  console.log(`[page] START tl=${tl} url=${targetUrl} render=${forceRender} nocache=${noCache}`)

  // Purge old v1 cache key on every request (idempotent, ensures migration)
  const v1Key = `page:${tl}:${targetUrl}`
  c.executionCtx.waitUntil(kv.delete(v1Key))

  // 1. Check KV cache
  const ck = cacheKey(tl, targetUrl)
  if (!noCache) {
    const cached = await kv.get(ck, 'text')
    if (cached) {
      console.log(`[page] CACHE HIT key=${ck} size=${cached.length} ms=${Date.now() - t0}`)
      return new Response(cached, {
        headers: {
          'Content-Type': 'text/html; charset=utf-8',
          'Content-Security-Policy': CSP_HEADER,
          'Cache-Control': 'public, max-age=3600',
          'X-Translate-Cache': 'HIT',
          'X-Robots-Tag': 'noindex',
        },
      })
    }
  }
  console.log(`[page] CACHE MISS key=${ck} nocache=${noCache} ms=${Date.now() - t0}`)

  // 2. Try normal fetch first (fast path)
  let html: string | null = null
  let usedBrowser = false

  if (!forceRender) {
    try {
      const response = await fetch(originUrl.toString(), {
        headers: {
          'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
          'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
          'Accept-Language': 'en-US,en;q=0.9',
        },
        redirect: 'follow',
      })

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

      const body = await response.text()
      console.log(`[page] FETCH status=${response.status} size=${body.length} ct=${contentType} ms=${Date.now() - t0}`)

      // 3. Check if we need browser rendering
      if (needsBrowserRender(response.status, body)) {
        console.log(`[page] NEEDS_BROWSER_RENDER status=${response.status} size=${body.length}`)
        html = null
      } else {
        html = body
      }
    } catch (e) {
      console.log(`[page] FETCH_ERROR err=${e instanceof Error ? e.message : e} ms=${Date.now() - t0}`)
      html = null
    }
  }

  // 4. Browser rendering fallback
  if (html === null) {
    console.log(`[page] BROWSER_RENDER_START url=${targetUrl}`)
    try {
      html = await renderWithBrowser(c.env, originUrl.toString())
      usedBrowser = true
      console.log(`[page] BROWSER_RENDER_OK size=${html.length} ms=${Date.now() - t0}`)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      console.log(`[page] BROWSER_RENDER_FAIL err=${message} ms=${Date.now() - t0}`)
      const is429 = message.includes('429')
      const errorHtml = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Translation Unavailable</title>
<style>*{margin:0;padding:0;box-sizing:border-box}body{font-family:-apple-system,system-ui,sans-serif;background:#f8f9fa;color:#202124;display:flex;align-items:center;justify-content:center;min-height:100vh;padding:24px}
.card{background:#fff;border-radius:12px;box-shadow:0 2px 12px rgba(0,0,0,.08);max-width:520px;width:100%;padding:32px;text-align:center}
h1{font-size:20px;font-weight:600;margin-bottom:8px}p{font-size:15px;color:#5f6368;line-height:1.6;margin-bottom:16px}
a.btn{display:inline-block;padding:10px 24px;background:#1a73e8;color:#fff;text-decoration:none;border-radius:8px;font-size:14px;font-weight:500}
a.btn:hover{background:#1557b0}.hint{font-size:13px;color:#80868b;margin-top:12px}</style></head>
<body><div class="card">
<h1>${is429 ? 'Rate Limited' : 'Rendering Failed'}</h1>
<p>${is429 ? 'Browser rendering is temporarily rate limited. Please try again in a minute.' : 'This page requires JavaScript rendering which is currently unavailable.'}</p>
<a class="btn" href="${originUrl.toString()}">View original page</a>
<p class="hint">${message}</p>
</div></body></html>`
      return new Response(errorHtml, {
        status: 502,
        headers: {
          'Content-Type': 'text/html; charset=utf-8',
          'Cache-Control': 'no-cache',
          'X-Robots-Tag': 'noindex',
        },
      })
    }
  }

  // 5. Two-pass translation: extract → batch translate → apply
  console.log(`[page] EXTRACT_START htmlSize=${html!.length} usedBrowser=${usedBrowser} ms=${Date.now() - t0}`)
  const proxyBase = new URL(c.req.url).origin

  // Pass 1: Extract all translatable text (no API calls, no subrequests)
  const texts = await extractTexts(html!)
  console.log(`[page] EXTRACT_DONE texts=${texts.length} ms=${Date.now() - t0}`)

  // Batch translate all texts in 2-5 API calls instead of N
  const { translations, detectedSl } = await batchTranslate(texts, 'auto', tl)
  console.log(`[page] BATCH_DONE translated=${translations.size}/${texts.length} sl=${detectedSl} ms=${Date.now() - t0}`)

  // Pass 2: Apply translations via HTMLRewriter with Map lookup (no API calls)
  const rewriter = makePageRewriter(originUrl, proxyBase, tl, 'auto', CSP_NONCE, translations, detectedSl)
  const translated = rewriter.transform(new Response(html, {
    headers: { 'Content-Type': 'text/html; charset=utf-8' },
  }))

  // Always buffer the result for caching (batch approach is already non-streaming)
  const translatedHtml = await translated.text()
  console.log(`[page] TRANSLATED size=${translatedHtml.length} render=${usedBrowser ? 'browser' : 'fetch'} ms=${Date.now() - t0}`)

  // Cache the translated HTML
  c.executionCtx.waitUntil(
    kv.put(cacheKey(tl, targetUrl), translatedHtml, { expirationTtl: 86400 })
  )

  return new Response(translatedHtml, {
    headers: {
      'Content-Type': 'text/html; charset=utf-8',
      'Content-Security-Policy': CSP_HEADER,
      'Cache-Control': 'public, max-age=3600',
      'X-Translate-Cache': 'MISS',
      'X-Translate-Render': usedBrowser ? 'browser' : 'fetch',
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

// DELETE /page/cache?url=...&tl=...  — purge cached translation
route.delete('/page/cache', async (c) => {
  const url = c.req.query('url')
  const tl = c.req.query('tl')
  if (!url || !tl) return c.json({ error: 'Missing url or tl query param' }, 400)

  const kv = c.env.TRANSLATE_CACHE
  // Delete both v1 and v2 cache keys
  await Promise.all([
    kv.delete(`page:${tl}:${url}`),
    kv.delete(cacheKey(tl, url)),
  ])
  return c.json({ ok: true, deleted: [`page:${tl}:${url}`, cacheKey(tl, url)] })
})

// GET /page/inspect/:tl?url=...  — fetch, translate, return JSON metadata for debugging
route.get('/page/inspect/:tl{[a-zA-Z]{2,3}(-[a-zA-Z]{2})?}', async (c) => {
  const tl = c.req.param('tl')
  const targetUrl = c.req.query('url')
  const forceRender = c.req.query('render') === '1'

  if (!targetUrl || !targetUrl.startsWith('http')) {
    return c.json({ error: 'Invalid URL. Use /page/inspect/<lang>?url=https://example.com' }, 400)
  }

  let originUrl: URL
  try {
    originUrl = new URL(targetUrl)
  } catch {
    return c.json({ error: 'Invalid URL format' }, 400)
  }

  const kv = c.env.TRANSLATE_CACHE
  const t0 = Date.now()

  // Check both v1 and v2 cache
  const v1Key = `page:${tl}:${targetUrl}`
  const v2Key = cacheKey(tl, targetUrl)
  const [v1Cached, v2Cached] = await Promise.all([
    kv.get(v1Key, 'text'),
    kv.get(v2Key, 'text'),
  ])

  const cache = {
    v1: v1Cached ? { key: v1Key, size: v1Cached.length } : null,
    v2: v2Cached ? { key: v2Key, size: v2Cached.length } : null,
  }

  // Fetch the page
  let fetchStatus = 0
  let fetchSize = 0
  let fetchNeedsBrowser = false
  let html: string | null = null
  let usedBrowser = false
  let renderError: string | null = null

  if (!forceRender) {
    try {
      const response = await fetch(originUrl.toString(), {
        headers: {
          'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
          'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
        },
        redirect: 'follow',
      })
      const body = await response.text()
      fetchStatus = response.status
      fetchSize = body.length
      fetchNeedsBrowser = needsBrowserRender(response.status, body)
      if (!fetchNeedsBrowser) html = body
    } catch (e) {
      renderError = `fetch failed: ${e instanceof Error ? e.message : e}`
    }
  }

  if (html === null) {
    try {
      html = await renderWithBrowser(c.env, originUrl.toString())
      usedBrowser = true
    } catch (e) {
      renderError = `browser failed: ${e instanceof Error ? e.message : e}`
    }
  }

  if (!html) {
    return c.json({
      error: renderError || 'No HTML',
      cache,
      fetch: { status: fetchStatus, size: fetchSize, needsBrowser: fetchNeedsBrowser },
      ms: Date.now() - t0,
    })
  }

  // Two-pass: extract → batch translate → apply
  const proxyBase = new URL(c.req.url).origin
  const texts = await extractTexts(html)
  const { translations, detectedSl } = await batchTranslate(texts, 'auto', tl)
  const rewriter = makePageRewriter(originUrl, proxyBase, tl, 'auto', CSP_NONCE, translations, detectedSl)
  const translated = rewriter.transform(new Response(html, {
    headers: { 'Content-Type': 'text/html; charset=utf-8' },
  }))
  const translatedHtml = await translated.text()

  // Analyze the output
  const scriptMatches = translatedHtml.match(/<script[\s>]/gi)
  const tlSegMatches = translatedHtml.match(/class="tl-seg"/g)
  const tlBlockMatches = translatedHtml.match(/class="[^"]*tl-block/g)
  const hasBase = translatedHtml.includes('<base href=')
  const hasBanner = translatedHtml.includes('Translated from')
  const hasForceVisible = translatedHtml.includes('tl-force-visible')
  const hasLearnerScript = translatedHtml.includes('id="tl-learner"')
  const langMatch = translatedHtml.match(/<html[^>]*lang="([^"]*)"/)

  return c.json({
    cache,
    fetch: { status: fetchStatus, size: fetchSize, needsBrowser: fetchNeedsBrowser },
    render: { usedBrowser, error: renderError },
    input: { size: html.length },
    output: {
      size: translatedHtml.length,
      lang: langMatch ? langMatch[1] : 'not found',
      hasBase,
      hasBanner,
      hasForceVisible,
      hasLearnerScript,
      scripts: scriptMatches ? scriptMatches.length : 0,
      tlSegments: tlSegMatches ? tlSegMatches.length : 0,
      tlBlocks: tlBlockMatches ? tlBlockMatches.length : 0,
      first500: translatedHtml.slice(0, 500),
    },
    ms: Date.now() - t0,
  })
})

export default route
