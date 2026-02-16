import { Hono } from 'hono'
import { trimTrailingSlash } from 'hono/trailing-slash'
import type { HonoEnv } from './types'
import { cssURL, cssText } from './asset'
import { renderLayout, renderHomePage, renderError } from './html'
import { ThreadManager } from './threads'
import { getThreadStore } from './storage'

import searchRoutes from './routes/search'
import threadRoutes from './routes/thread'
import historyRoutes from './routes/history'
import apiRoutes from './routes/api'

const app = new Hono<HonoEnv>()
app.use(trimTrailingSlash())

// Cacheable CSS with content hash
app.get(cssURL, (c) => {
  c.header('Content-Type', 'text/css; charset=utf-8')
  c.header('Cache-Control', 'public, max-age=31536000, immutable')
  return c.body(cssText)
})

// Health check
app.get('/api/health', (c) => c.json({ status: 'ok', timestamp: new Date().toISOString() }))

// Home page
app.get('/', async (c) => {
  const threadStore = getThreadStore(c.env)
  const tm = new ThreadManager(threadStore)
  const threads = await tm.listThreads()
  return c.html(renderLayout('AI Search', renderHomePage(threads), { isHome: true, threads }))
})

// JSON API
app.route('/api', apiRoutes)

// Routes
app.route('/search', searchRoutes)
app.route('/thread', threadRoutes)
app.route('/history', historyRoutes)

// 404
app.notFound((c) => {
  return c.html(renderLayout('Not Found', renderError('Page not found', 'The page you\'re looking for doesn\'t exist.')), 404)
})

// Error handler
app.onError((err, c) => {
  console.error('[Error]', err.message)
  return c.html(renderLayout('Error', renderError('Something went wrong', err.message)), 500)
})

export default app
