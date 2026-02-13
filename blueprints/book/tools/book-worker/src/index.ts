import { Hono } from 'hono'
import type { Env, HonoEnv } from './types'
import type { QueueJob } from './queue'
import { handleQueue } from './queue'
import apiRoutes from './routes/api'

const app = new Hono<HonoEnv>()

// Health check
app.get('/api/health', (c) => c.json({ status: 'ok', timestamp: new Date().toISOString() }))

// JSON API
app.route('/api', apiRoutes)

// Serve static assets and SPA fallback for non-API GET requests
app.get('*', async (c) => {
  // Unmatched /api/* routes should return JSON 404
  if (c.req.path.startsWith('/api/')) {
    return c.json({ error: 'Not found' }, 404)
  }
  const res = await c.env.ASSETS.fetch(c.req.raw)
  if (res.status !== 404) return res
  // SPA fallback: serve index.html for client-side routes
  return c.env.ASSETS.fetch(new URL('/index.html', c.req.url).toString())
})

// 404 for non-GET or unmatched routes
app.notFound((c) => c.json({ error: 'Not found' }, 404))

// Error handler
app.onError((err, c) => {
  console.error('[Error]', err.message)
  return c.json({ error: err.message }, 500)
})

// Export both fetch (HTTP) and queue (consumer) handlers
export default {
  fetch: app.fetch,
  async queue(batch: MessageBatch<QueueJob>, env: Env): Promise<void> {
    await handleQueue(batch, env)
  },
}
