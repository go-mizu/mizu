import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { search } from '../perplexity'
import { ThreadManager } from '../threads'
import { DEFAULT_MODE } from '../config'
import { renderLayout, renderSearchResults, renderError } from '../html'

const app = new Hono<HonoEnv>()

// GET /search?q=query&mode=auto
app.get('/', async (c) => {
  const query = c.req.query('q')?.trim()
  if (!query) return c.redirect('/')

  const mode = c.req.query('mode') || DEFAULT_MODE
  const tm = new ThreadManager(c.env.KV)

  try {
    const result = await search(c.env.KV, query, mode)
    const thread = await tm.createThread(query, mode, result.model, result)
    const threads = await tm.listThreads()

    return c.html(renderLayout(query + ' - AI Search', renderSearchResults(result, thread.id), {
      query,
      threads,
      currentThreadId: thread.id,
    }))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    console.error('[Search Error]', msg)
    const threads = await tm.listThreads()
    return c.html(renderLayout('Error', renderError('Search Failed', msg), { query, threads }), 500)
  }
})

export default app
