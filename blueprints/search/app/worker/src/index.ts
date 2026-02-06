import { Hono } from 'hono'
import { cors } from 'hono/cors'
import { timing } from 'hono/timing'
import { getAssetFromKV } from '@cloudflare/kv-asset-handler'
// @ts-expect-error - __STATIC_CONTENT_MANIFEST is injected by wrangler
import manifestJSON from '__STATIC_CONTENT_MANIFEST'

import healthRoutes from './routes/health'
import searchRoutes from './routes/search'
import suggestRoutes from './routes/suggest'
import instantRoutes from './routes/instant'
import knowledgeRoutes from './routes/knowledge'
import preferencesRoutes from './routes/preferences'
import lensesRoutes from './routes/lenses'
import historyRoutes from './routes/history'
import settingsRoutes from './routes/settings'
import bangsRoutes from './routes/bangs'
import newsRoutes from './routes/news'
import readRoutes from './routes/read'
import {
  widgetsRoutes,
  cheatsheetRoutes,
  cheatsheetsListRoutes,
  relatedRoutes,
} from './routes/widgets'
import { sessionMiddleware } from './middleware/session'
import { errorHandler } from './middleware/error-handler'
import { rateLimit } from './middleware/rate-limit'
import { contextMiddleware } from './middleware/context'
import type { HonoEnv } from './types'

const assetManifest = JSON.parse(manifestJSON)

const app = new Hono<HonoEnv>()

// Global middleware
app.use('*', errorHandler)
app.use('*', cors())
app.use('*', timing())
app.use('*', sessionMiddleware())

// API middleware: rate limiting + service injection
app.use('/api/*', rateLimit({
  windowMs: 60_000,
  maxRequests: 100,
}))
app.use('/api/*', contextMiddleware)

// Health check
app.route('/health', healthRoutes)

// API route groups
app.route('/api/search', searchRoutes)
app.route('/api/suggest', suggestRoutes)
app.route('/api/instant', instantRoutes)
app.route('/api/knowledge', knowledgeRoutes)
app.route('/api/preferences', preferencesRoutes)
app.route('/api/lenses', lensesRoutes)
app.route('/api/history', historyRoutes)
app.route('/api/settings', settingsRoutes)
app.route('/api/bangs', bangsRoutes)
app.route('/api/widgets', widgetsRoutes)
app.route('/api/cheatsheet', cheatsheetRoutes)
app.route('/api/cheatsheets', cheatsheetsListRoutes)
app.route('/api/related', relatedRoutes)
app.route('/api/news', newsRoutes)
app.route('/api/read', readRoutes)

// Serve static frontend files for all other routes
app.get('*', async (c) => {
  try {
    // Try to serve the exact static file
    const asset = await getAssetFromKV(
      {
        request: c.req.raw,
        waitUntil: (promise) => c.executionCtx.waitUntil(promise),
      },
      {
        ASSET_NAMESPACE: c.env.__STATIC_CONTENT,
        ASSET_MANIFEST: assetManifest,
      }
    )
    return new Response(asset.body, asset)
  } catch {
    // SPA fallback: serve index.html for client-side routes
    try {
      const notFoundRequest = new Request(new URL('/index.html', c.req.url).toString(), {
        method: 'GET',
      })
      const asset = await getAssetFromKV(
        {
          request: notFoundRequest,
          waitUntil: (promise) => c.executionCtx.waitUntil(promise),
        },
        {
          ASSET_NAMESPACE: c.env.__STATIC_CONTENT,
          ASSET_MANIFEST: assetManifest,
        }
      )
      return new Response(asset.body, asset)
    } catch {
      return c.text('Not found', 404)
    }
  }
})

export default app
