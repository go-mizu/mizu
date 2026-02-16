import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ThreadManager } from '../threads'
import { getThreadStore } from '../storage'
import { renderLayout, renderHistoryPage } from '../html'

const app = new Hono<HonoEnv>()

// GET /history — list all threads
app.get('/', async (c) => {
  const threadStore = getThreadStore(c.env)
  const tm = new ThreadManager(threadStore)
  const threads = await tm.listThreads()
  return c.html(renderLayout('History - AI Search', renderHistoryPage(threads), { threads }))
})

export default app
