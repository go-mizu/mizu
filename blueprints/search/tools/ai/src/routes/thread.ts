import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { search } from '../perplexity'
import { ThreadManager } from '../threads'
import { getSessionStore, getThreadStore } from '../storage'
import { renderLayout, renderThreadPage, renderError } from '../html'

const app = new Hono<HonoEnv>()

// GET /thread/:id — view thread
app.get('/:id', async (c) => {
  const id = c.req.param('id')
  const threadStore = getThreadStore(c.env)
  const tm = new ThreadManager(threadStore)
  const thread = await tm.getThread(id)

  if (!thread) {
    return c.html(renderLayout('Not Found', renderError('Thread Not Found', 'This thread does not exist or has expired.'), {}), 404)
  }

  const threads = await tm.listThreads()

  return c.html(renderLayout(thread.title + ' - AI Search', renderThreadPage(thread), {
    query: thread.title,
    threads,
    currentThreadId: thread.id,
  }))
})

// GET /thread/:id/follow-up?q=query&mode=auto — add follow-up (server handles everything)
app.get('/:id/follow-up', async (c) => {
  const id = c.req.param('id')
  const query = c.req.query('q')?.trim()
  if (!query) return c.redirect(`/thread/${id}`)

  const sessionStore = getSessionStore(c.env)
  const threadStore = getThreadStore(c.env)
  const tm = new ThreadManager(threadStore)
  const thread = await tm.getThread(id)
  if (!thread) {
    return c.html(renderLayout('Not Found', renderError('Thread Not Found', 'This thread does not exist or has expired.'), {}), 404)
  }

  const mode = c.req.query('mode') || thread.mode || 'auto'

  try {
    const followUpUUID = tm.getLastBackendUUID(thread)
    const result = await search(sessionStore, query, mode, '', followUpUUID)
    await tm.addFollowUp(id, query, result)

    return c.redirect(`/thread/${id}`)
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    console.error('[Follow-up Error]', msg)
    const threads = await tm.listThreads()
    return c.html(renderLayout('Error', renderError('Follow-up Failed', msg), { query, threads }), 500)
  }
})

export default app
