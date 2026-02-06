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
import { securityHeaders } from './middleware/security'
import { rateLimit } from './middleware/rate-limit'
import { contextMiddleware } from './middleware/context'
import type { HonoEnv } from './types'

const assetManifest = JSON.parse(manifestJSON)

const MIME_TYPES: Record<string, string> = {
  html: 'text/html; charset=utf-8',
  js: 'application/javascript; charset=utf-8',
  css: 'text/css; charset=utf-8',
  json: 'application/json; charset=utf-8',
  svg: 'image/svg+xml',
  png: 'image/png',
  jpg: 'image/jpeg',
  jpeg: 'image/jpeg',
  gif: 'image/gif',
  webp: 'image/webp',
  ico: 'image/x-icon',
  woff: 'font/woff',
  woff2: 'font/woff2',
  ttf: 'font/ttf',
  txt: 'text/plain; charset=utf-8',
  xml: 'application/xml',
  webmanifest: 'application/manifest+json',
}

function getMimeType(path: string): string {
  const ext = path.split('.').pop()?.toLowerCase() ?? ''
  return MIME_TYPES[ext] || 'application/octet-stream'
}

const app = new Hono<HonoEnv>()

// Global middleware (lightweight - no security headers on static assets)
app.use('*', errorHandler)
app.use('*', cors())
app.use('*', timing())
app.use('*', sessionMiddleware())

// API-specific middleware: security headers, rate limiting, service injection
app.use('/api/*', securityHeaders())
app.use('/api/*', rateLimit({
  windowMs: 60_000,    // 1 minute
  maxRequests: 100,    // 100 requests per minute
}))
app.use('/api/*', contextMiddleware)

// Health check (no rate limiting, no service injection needed)
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
    // Explicitly set Content-Type based on requested path
    const contentType = getMimeType(new URL(c.req.url).pathname)
    const headers = new Headers(asset.headers)
    headers.set('Content-Type', contentType)
    return new Response(asset.body, { headers, status: asset.status })
  } catch {
    // SPA fallback: serve index.html for all non-asset routes
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
      const headers = new Headers(asset.headers)
      headers.set('Content-Type', 'text/html; charset=utf-8')
      return new Response(asset.body, { headers, status: 200 })
    } catch {
      return c.text('Not found', 404)
    }
  }
})

export default app
