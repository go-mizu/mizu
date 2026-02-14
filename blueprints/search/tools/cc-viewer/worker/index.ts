import { Hono } from 'hono'
import { cors } from 'hono/cors'
import type { Env, HonoEnv, QueueMessage } from './types'
import apiRoutes from './routes/api'
import { processQueue, handleScheduled } from './queue'

const app = new Hono<HonoEnv>()

// CORS for all API routes
app.use('/api/*', cors({
  origin: '*',
  allowMethods: ['GET', 'OPTIONS'],
  allowHeaders: ['Content-Type'],
}))

// Mount all API routes
app.route('/api', apiRoutes)

// Export for Cloudflare Worker
export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url)

    // API routes — handled by Hono
    if (url.pathname.startsWith('/api/')) {
      return app.fetch(request, env, ctx)
    }

    // SPA fallback via static assets
    if (env.ASSETS) {
      return env.ASSETS.fetch(request)
    }

    // No ASSETS binding (local dev without build) — return a helpful message
    return new Response(JSON.stringify({
      error: 'No static assets available',
      detail: 'Run "npm run build" to generate the SPA, or use /api/ endpoints directly.',
      api: {
        health: '/api/health',
        crawls: '/api/crawls',
        search: '/api/search?q=example.com',
        url: '/api/url?url=https://example.com',
        domains: '/api/domains/lookup?domain=example.com',
        stats: '/api/stats',
        news: '/api/news/dates',
      },
    }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  },

  // Queue consumer
  async queue(batch: MessageBatch<QueueMessage>, env: Env): Promise<void> {
    await processQueue(batch, env)
  },

  // Cron trigger
  async scheduled(event: ScheduledEvent, env: Env, ctx: ExecutionContext): Promise<void> {
    await handleScheduled(event, env, ctx)
  },
}
