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

  try {
    // Session init + SSE search — fully server-side, invisible to client
    const result = await search(c.env.KV, query, mode)
    const tm = new ThreadManager(c.env.KV)
    const thread = await tm.createThread(query, mode, result.model, result)

    return c.html(renderLayout(query + ' - AI Search', renderSearchResults(result, thread.id), { query }))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    console.error('[Search Error]', msg)
    return c.html(renderLayout('Error', renderError('Search Failed', msg), { query }), 500)
  }
})

export default app
