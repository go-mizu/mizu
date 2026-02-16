import { Hono } from 'hono'
import { cors } from 'hono/cors'
import type { HonoEnv } from '../types'
import { search, streamSearch } from '../perplexity'
import { ThreadManager } from '../threads'
import { DEFAULT_MODE } from '../config'

const app = new Hono<HonoEnv>()
app.use('*', cors())

// GET /api/stream?q=query&mode=auto&threadId=xxx — SSE streaming search
app.get('/stream', async (c) => {
  const query = c.req.query('q')?.trim()
  if (!query) return c.json({ error: 'q is required' }, 400)

  const mode = c.req.query('mode') || DEFAULT_MODE
  const threadId = c.req.query('threadId') || ''

  let followUpUUID: string | null = null
  if (threadId) {
    const tm = new ThreadManager(c.env.KV)
    const thread = await tm.getThread(threadId)
    if (thread) {
      followUpUUID = tm.getLastBackendUUID(thread)
    }
  }

  const stream = streamSearch(c.env.KV, query, mode, '', followUpUUID)

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

app.onError((err, c) => {
  console.error('[API Error]', err.message)
  return c.json({ error: err.message }, 500)
})

export default app
