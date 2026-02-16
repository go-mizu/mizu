import { Hono } from 'hono'
import { cors } from 'hono/cors'
import type { HonoEnv } from '../types'
import { initSession, search, streamSearch, getPooledSession, createFreshSession } from '../perplexity'
import { streamSearchAPI } from '../api-stream'
import { ThreadManager } from '../threads'
import { Cache } from '../cache'
import { DEFAULT_MODE } from '../config'

const app = new Hono<HonoEnv>()
app.use('*', cors())

// GET /api/warm — pre-warm Perplexity session (call on page load to reduce TTFB)
app.get('/warm', async (c) => {
  if (c.env.PERPLEXITY_API_KEY) {
    return c.json({ ok: true, backend: 'api', ms: 0 })
  }
  const t0 = Date.now()
  const session = await initSession(c.env.KV)
  return c.json({ ok: true, backend: 'scraper', cached: !!session.csrfToken, ms: Date.now() - t0 })
})

// GET /api/stream?q=query&mode=auto&threadId=xxx — SSE streaming search
app.get('/stream', async (c) => {
  const query = c.req.query('q')?.trim()
  if (!query) return c.json({ error: 'q is required' }, 400)

  const mode = c.req.query('mode') || DEFAULT_MODE
  const threadId = c.req.query('threadId') || ''
  const apiKey = c.env.PERPLEXITY_API_KEY

  let stream: ReadableStream<Uint8Array>

  if (apiKey) {
    // Official API — real token-by-token streaming, ~1.3s TTFB
    // Build conversation history from thread for follow-ups
    let history: Array<{ role: string; content: string }> | undefined
    if (threadId) {
      const tm = new ThreadManager(c.env.KV)
      const thread = await tm.getThread(threadId)
      if (thread) {
        history = thread.messages.map(m => ({ role: m.role, content: m.content }))
      }
    }
    stream = streamSearchAPI(apiKey, query, mode, history)
  } else {
    // Scraping fallback — simulated streaming, ~5-10s TTFB
    let followUpUUID: string | null = null
    const tm = new ThreadManager(c.env.KV)
    const [session, thread] = await Promise.all([
      initSession(c.env.KV),
      threadId ? tm.getThread(threadId) : Promise.resolve(null),
    ])
    if (thread) {
      followUpUUID = tm.getLastBackendUUID(thread)
    }
    stream = streamSearch(c.env.KV, query, mode, '', followUpUUID, session)
  }

  return new Response(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
      'Access-Control-Allow-Origin': '*',
    },
  })
})

// POST /api/search — execute search, return JSON + save thread
app.post('/search', async (c) => {
  const body = await c.req.json<{ query: string; mode?: string; threadId?: string }>()
  if (!body.query?.trim()) return c.json({ error: 'query is required' }, 400)

  const mode = body.mode || DEFAULT_MODE
  const tm = new ThreadManager(c.env.KV)

  try {
    if (body.threadId) {
      const thread = await tm.getThread(body.threadId)
      if (!thread) return c.json({ error: 'thread not found' }, 404)
      const followUpUUID = tm.getLastBackendUUID(thread)
      const result = await search(c.env.KV, body.query, mode, '', followUpUUID)
      const updated = await tm.addFollowUp(body.threadId, body.query, result)
      return c.json({ result, thread: updated })
    }

    const result = await search(c.env.KV, body.query, mode)
    const thread = await tm.createThread(body.query, mode, result.model, result)
    return c.json({ result, thread })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.json({ error: msg }, 500)
  }
})

// POST /api/thread/save — save streaming result as thread
app.post('/thread/save', async (c) => {
  const body = await c.req.json<{
    query: string
    mode: string
    threadId?: string
    result: {
      answer: string
      citations: unknown[]
      webResults: unknown[]
      relatedQueries: string[]
      images: unknown[]
      videos: unknown[]
      thinkingSteps: unknown[]
      backendUUID: string
      model: string
      durationMs: number
    }
  }>()

  const tm = new ThreadManager(c.env.KV)

  try {
    const searchResult = {
      query: body.query,
      answer: body.result.answer,
      citations: body.result.citations as any[],
      webResults: body.result.webResults as any[],
      relatedQueries: body.result.relatedQueries || [],
      images: body.result.images as any[] || [],
      videos: body.result.videos as any[] || [],
      thinkingSteps: body.result.thinkingSteps as any[] || [],
      backendUUID: body.result.backendUUID || '',
      mode: body.mode,
      model: body.result.model || '',
      durationMs: body.result.durationMs || 0,
      createdAt: new Date().toISOString(),
    }

    if (body.threadId) {
      const updated = await tm.addFollowUp(body.threadId, body.query, searchResult)
      return c.json({ thread: updated })
    }

    const thread = await tm.createThread(body.query, body.mode, searchResult.model, searchResult)
    return c.json({ thread })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.json({ error: msg }, 500)
  }
})

// GET /api/thread/:id
app.get('/thread/:id', async (c) => {
  const tm = new ThreadManager(c.env.KV)
  const thread = await tm.getThread(c.req.param('id'))
  if (!thread) return c.json({ error: 'not found' }, 404)
  return c.json(thread)
})

// POST /api/thread/:id/follow-up
app.post('/thread/:id/follow-up', async (c) => {
  const id = c.req.param('id')
  const body = await c.req.json<{ query: string; mode?: string }>()
  if (!body.query?.trim()) return c.json({ error: 'query is required' }, 400)

  const tm = new ThreadManager(c.env.KV)
  const thread = await tm.getThread(id)
  if (!thread) return c.json({ error: 'thread not found' }, 404)

  const mode = body.mode || thread.mode || DEFAULT_MODE

  try {
    const followUpUUID = tm.getLastBackendUUID(thread)
    const result = await search(c.env.KV, body.query, mode, '', followUpUUID)
    const updated = await tm.addFollowUp(id, body.query, result)
    return c.json({ result, thread: updated })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.json({ error: msg }, 500)
  }
})

// DELETE /api/thread/:id
app.delete('/thread/:id', async (c) => {
  const tm = new ThreadManager(c.env.KV)
  const ok = await tm.deleteThread(c.req.param('id'))
  if (!ok) return c.json({ error: 'not found' }, 404)
  return c.json({ ok: true })
})

// GET /api/threads
app.get('/threads', async (c) => {
  const tm = new ThreadManager(c.env.KV)
  const threads = await tm.listThreads()
  return c.json({ threads })
})

// GET /api/og?url=... — extract Open Graph metadata from a URL (for link previews)
app.get('/og', async (c) => {
  const url = c.req.query('url')
  if (!url) return c.json({ error: 'url required' }, 400)

  // Check KV cache
  const cache = new Cache(c.env.KV)
  const cacheKey = `og:${url}`
  const cached = await cache.get<{ title: string; description: string; image: string; siteName: string }>(cacheKey)
  if (cached) return c.json(cached)

  try {
    const resp = await fetch(url, {
      headers: {
        'User-Agent': 'Mozilla/5.0 (compatible; AI-Search/1.0; +https://ai-search.go-mizu.workers.dev)',
        'Accept': 'text/html',
      },
      redirect: 'follow',
      signal: AbortSignal.timeout(5000),
    })
    if (!resp.ok) return c.json({ error: `HTTP ${resp.status}` }, 502)

    // Only read first 50KB to extract meta tags
    const reader = resp.body?.getReader()
    if (!reader) return c.json({ error: 'no body' }, 502)
    let html = ''
    const decoder = new TextDecoder()
    while (html.length < 50000) {
      const { done, value } = await reader.read()
      if (done) break
      html += decoder.decode(value, { stream: true })
      // Stop once we've passed </head> — no need to read the body
      if (html.includes('</head>')) break
    }
    reader.cancel()

    const og = extractOGMeta(html)
    // Resolve relative image URLs
    if (og.image && !og.image.startsWith('http')) {
      try {
        og.image = new URL(og.image, url).href
      } catch { /* leave as-is */ }
    }

    await cache.set(cacheKey, og, 86400) // 24h cache
    return c.json(og)
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.json({ error: msg }, 502)
  }
})

function extractOGMeta(html: string): { title: string; description: string; image: string; siteName: string } {
  const getMeta = (property: string): string => {
    // Match property="X" or name="X" with content="Y" in either order
    const re1 = new RegExp(`<meta[^>]*(?:property|name)=["']${property}["'][^>]*content=["']([^"']*)["']`, 'i')
    const re2 = new RegExp(`<meta[^>]*content=["']([^"']*)["'][^>]*(?:property|name)=["']${property}["']`, 'i')
    return (html.match(re1)?.[1] || html.match(re2)?.[1] || '').trim()
  }
  // Extract <title> tag as fallback
  const getTitle = (): string => {
    const m = html.match(/<title[^>]*>([^<]+)<\/title>/i)
    return m?.[1]?.trim() || ''
  }
  // Extract first large image from page as last resort
  const getFirstImage = (): string => {
    const m = html.match(/<img[^>]*src=["']([^"']+(?:\.jpg|\.jpeg|\.png|\.webp)[^"']*)["']/i)
    return m?.[1] || ''
  }
  return {
    title: getMeta('og:title') || getMeta('twitter:title') || getTitle(),
    description: getMeta('og:description') || getMeta('twitter:description') || getMeta('description') || '',
    image: getMeta('og:image') || getMeta('twitter:image') || getMeta('twitter:image:src') || getFirstImage(),
    siteName: getMeta('og:site_name') || '',
  }
}

app.onError((err, c) => {
  console.error('[API Error]', err.message)
  return c.json({ error: err.message }, 500)
})

export default app
