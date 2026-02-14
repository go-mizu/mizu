import { Hono } from 'hono'
import type { HonoEnv } from '../types'

import crawlsRoutes from './crawls'
import domainsRoutes from './domains'
import urlRoutes from './url'
import viewRoutes from './view'
import statsRoutes from './stats'
import graphRoutes from './graph'
import newsRoutes from './news'
import searchRoutes from './search'
import analyticsRoutes from './analytics'

const app = new Hono<HonoEnv>()

// CORS middleware for all API routes
app.use('*', async (c, next) => {
  await next()
  c.header('Access-Control-Allow-Origin', '*')
  c.header('Access-Control-Allow-Methods', 'GET, OPTIONS')
  c.header('Access-Control-Allow-Headers', 'Content-Type')
})

// Handle preflight
app.options('*', (c) => {
  return new Response(null, { status: 204 })
})

// Health check
app.get('/health', (c) => c.json({
  status: 'ok',
  timestamp: new Date().toISOString(),
  environment: c.env.ENVIRONMENT || 'development',
}))

// Mount sub-routes
app.route('/crawls', crawlsRoutes)
app.route('/domains', domainsRoutes)
app.route('/url', urlRoutes)
app.route('/view', viewRoutes)
app.route('/stats', statsRoutes)
app.route('/graph', graphRoutes)
app.route('/news', newsRoutes)
app.route('/search', searchRoutes)
app.route('/analytics', analyticsRoutes)

export default app
