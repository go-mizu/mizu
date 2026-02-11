import { Hono } from 'hono'
import type { HonoEnv } from './types'
import { cssURL, cssText } from './asset'
import { renderLayout, renderHomePage, renderError } from './html'

import apiRoutes from './routes/api'
import searchRoutes from './routes/search'
import urlRoutes from './routes/url'
import domainRoutes from './routes/domain'
import domainsRoutes from './routes/domains'
import viewRoutes from './routes/view'
import crawlsRoutes from './routes/crawls'
import crawlRoutes from './routes/crawl'

const app = new Hono<HonoEnv>()

// Cacheable CSS with content hash
app.get(cssURL, (c) => {
  c.header('Content-Type', 'text/css; charset=utf-8')
  c.header('Cache-Control', 'public, max-age=31536000, immutable')
  return c.body(cssText)
})

// Health check
app.get('/api/health', (c) => c.json({ status: 'ok', timestamp: new Date().toISOString() }))

// Home page
app.get('/', (c) => {
  return c.html(renderLayout('Common Crawl Viewer', renderHomePage(), { isHome: true }))
})

// JSON API
app.route('/api', apiRoutes)

// Pages (order matters — specific before generic)
app.route('/search', searchRoutes)
app.route('/crawls', crawlsRoutes)
app.route('/crawl', crawlRoutes)
app.route('/domains', domainsRoutes)
app.route('/view', viewRoutes)
app.route('/url', urlRoutes)
app.route('/domain', domainRoutes)

// 404
app.notFound((c) => {
  return c.html(renderError('Page not found', 'The page you\'re looking for doesn\'t exist.'), 404)
})

// Error handler
app.onError((err, c) => {
  console.error('[Error]', err.message)
  return c.html(renderError('Something went wrong', err.message), 500)
})

export default app
