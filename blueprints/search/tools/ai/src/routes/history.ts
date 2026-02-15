import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ThreadManager } from '../threads'
import { renderLayout, renderHistoryPage } from '../html'

const app = new Hono<HonoEnv>()

// GET /history — list all threads
app.get('/', async (c) => {
  const tm = new ThreadManager(c.env.KV)
  const threads = await tm.listThreads()
  return c.html(renderLayout('History - AI Search', renderHistoryPage(threads)))
})

export default app
