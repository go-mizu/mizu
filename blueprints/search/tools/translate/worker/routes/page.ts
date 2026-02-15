/**
 * Page translation route — Queue-based architecture.
 *
 * Flow:
 *   1. Page-level KV cache → HIT → return instantly
 *   2. MISS → Fetch HTML (normal or browser rendering)
 *   3. Extract translatable text segments
 *   4. Batch KV lookup for text-level translations (t:{tl}:{hash})
 *   5. Cached texts → server-translated, uncached → Queue + client-side fallback
 *   6. Queue consumer translates and writes to KV asynchronously
 *   7. Next visit: all texts cached → instant fully-translated page
 *
 * Zero inline translation API calls. Page handler only does KV reads + Queue sends.
 */

import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { extractTexts, makePageRewriter, fixTitle, debugTitle } from '../page-rewriter'
import { lookupTranslations } from '../translate'
import { needsBrowserRender, renderWithBrowser } from '../renderer'
import { track } from '../analytics'

const route = new Hono<HonoEnv>()

// Fixed nonce for CSP — blocks all original page scripts while allowing our injected ones.
// Not random because cached HTML must work with the same CSP header.
const CSP_NONCE = 'tl'
const CSP_HEADER = `script-src 'nonce-${CSP_NONCE}'`

function cacheKey(tl: string, url: string): string {
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

  // 1. Check page-level KV cache
  const ck = cacheKey(tl, targetUrl)
  if (!noCache) {
    const cached = await kv.get(ck, 'text')
    if (cached) {
      console.log(`[page] CACHE HIT key=${ck} size=${cached.length} ms=${Date.now() - t0}`)
      track(c.env, {
        event: 'page',
        sl: targetUrl,
        tl,
        cache: 'HIT',
        latencyMs: Date.now() - t0,
        chars: cached.length,
        success: true,
      })
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

  // 2. Fetch HTML (normal fetch or browser rendering)
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

  // 3. Browser rendering fallback
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
      track(c.env, {
        event: 'page',
        sl: targetUrl,
        tl,
        extra: message,
        cache: 'MISS',
        latencyMs: Date.now() - t0,
        success: false,
      })
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

  // 4. Extract translatable text segments
  console.log(`[page] EXTRACT_START htmlSize=${html!.length} usedBrowser=${usedBrowser} ms=${Date.now() - t0}`)
  const proxyBase = new URL(c.req.url).origin
  const texts = await extractTexts(html!)
  console.log(`[page] EXTRACT_DONE texts=${texts.length} ms=${Date.now() - t0}`)

  // 5. Batch KV lookup for text-level translations
  const { cached, uncached, detectedSl } = await lookupTranslations(kv, texts, tl)
  console.log(`[page] KV_LOOKUP cached=${cached.size} uncached=${uncached.length} sl=${detectedSl} ms=${Date.now() - t0}`)

  // 6. Queue uncached texts for background translation
  if (uncached.length > 0) {
    console.log(`[page] QUEUE_SEND texts=${uncached.length} tl=${tl}`)
    c.executionCtx.waitUntil(
      c.env.TRANSLATE_QUEUE.send({ texts: uncached, tl })
    )
  }

  // 7. Build page — cached texts get real translations, uncached get data-tp="1"
  const rewriter = makePageRewriter(originUrl, proxyBase, tl, 'auto', CSP_NONCE, cached, detectedSl)
  const translated = rewriter.transform(new Response(html, {
    headers: { 'Content-Type': 'text/html; charset=utf-8' },
  }))
  let translatedHtml = await translated.text()
  const beforeLen = translatedHtml.length
  const titleDbg = debugTitle(translatedHtml)
  translatedHtml = fixTitle(translatedHtml, cached)
  console.log(`[page] TRANSLATED size=${translatedHtml.length} beforeFix=${beforeLen} titleFixed=${beforeLen !== translatedHtml.length} titleDbg=${titleDbg} render=${usedBrowser ? 'browser' : 'fetch'} ms=${Date.now() - t0}`)

  // 8. Cache the page HTML (translate-cs will re-cache with full translations if needed)
  c.executionCtx.waitUntil(
    kv.put(ck, translatedHtml, { expirationTtl: 86400 })
  )

  track(c.env, {
    event: 'page',
    sl: targetUrl,
    tl,
    extra: usedBrowser ? 'browser' : 'fetch',
    cache: 'MISS',
    latencyMs: Date.now() - t0,
    chars: translatedHtml.length,
    success: true,
    cacheHits: cached.size,
    total: texts.length,
  })

  return new Response(translatedHtml, {
    headers: {
      'Content-Type': 'text/html; charset=utf-8',
      'Content-Security-Policy': CSP_HEADER,
      'Cache-Control': 'public, max-age=3600',
      'X-Translate-Cache': 'MISS',
      'X-Translate-Render': usedBrowser ? 'browser' : 'fetch',
      'X-Translate-Texts': `${cached.size}/${texts.length}`,
      'X-Translate-Title-Fixed': `${beforeLen !== translatedHtml.length}`,
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
  await Promise.all([
    kv.delete(`page:${tl}:${url}`),
    kv.delete(cacheKey(tl, url)),
  ])
  return c.json({ ok: true, deleted: [`page:${tl}:${url}`, cacheKey(tl, url)] })
})

// GET /page/inspect/:tl?url=...  — fetch, extract, show KV cache status (no translation)
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

  // Check page cache
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

  // Extract texts and check KV cache (no translation, just analysis)
  const texts = await extractTexts(html)
  const { cached: kvCached, uncached, detectedSl } = await lookupTranslations(kv, texts, tl)

  // Build page to analyze output
  const proxyBase = new URL(c.req.url).origin
  const rewriter = makePageRewriter(originUrl, proxyBase, tl, 'auto', CSP_NONCE, kvCached, detectedSl)
  const translated = rewriter.transform(new Response(html, {
    headers: { 'Content-Type': 'text/html; charset=utf-8' },
  }))
  const translatedHtml = fixTitle(await translated.text(), kvCached)

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
    input: { size: html.length, texts: texts.length },
    translation: {
      kvCached: kvCached.size,
      uncached: uncached.length,
      detectedSl,
      sampleUncached: uncached.slice(0, 5),
    },
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
