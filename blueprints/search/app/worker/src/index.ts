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
import {
  widgetsRoutes,
  cheatsheetRoutes,
  cheatsheetsListRoutes,
  relatedRoutes,
} from './routes/widgets'

const assetManifest = JSON.parse(manifestJSON)

type Env = {
  Bindings: {
    SEARCH_KV: KVNamespace
    __STATIC_CONTENT: KVNamespace
    ENVIRONMENT: string
  }
}

const app = new Hono<Env>()

// Global middleware
app.use('*', cors())
app.use('*', timing())

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
    return new Response(asset.body, asset)
  } catch {
    // For SPA routing, return index.html for non-asset requests
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
      return new Response(asset.body, {
        ...asset,
        status: 200,
      })
    } catch {
      return c.text('Not found', 404)
    }
  }
})

export default app
