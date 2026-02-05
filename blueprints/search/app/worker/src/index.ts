import { Hono } from 'hono'
import { cors } from 'hono/cors'
import { timing } from 'hono/timing'

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

type Env = {
  Bindings: {
    SEARCH_KV: KVNamespace
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
// Cloudflare Workers serves static assets from wrangler.toml [site] config
// The index.html is served for all non-API routes (SPA fallback)
app.get('*', async (c) => {
  // In production, static files are served by Cloudflare automatically
  // This is just a fallback that returns the index page content-type hint
  return c.html('<!DOCTYPE html><html><head><meta http-equiv="refresh" content="0;url=/"></head><body></body></html>')
})

export default app
