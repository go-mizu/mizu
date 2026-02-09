import { Hono } from 'hono'
import type { HonoEnv } from './types'
import { cssURL, cssText } from './asset'
import { renderLayout, renderHomePage, renderError } from './html'

import searchRoutes from './routes/search'
import hashtagRoutes from './routes/hashtag'
import listRoutes from './routes/list'
import tweetRoutes from './routes/tweet'
import profileRoutes from './routes/profile'

const app = new Hono<HonoEnv>()

// Cacheable CSS with content hash
app.get(cssURL, (c) => {
  c.header('Content-Type', 'text/css; charset=utf-8')
  c.header('Cache-Control', 'public, max-age=31536000, immutable')
  return c.body(cssText)
})

// Health check
app.get('/api/health', (c) => c.json({ status: 'ok', timestamp: new Date().toISOString() }))

// Home page — Google-inspired single search box
app.get('/', (c) => {
  return c.html(renderLayout('Z - the X/Twitter Viewer', renderHomePage(), { isHome: true }))
})

// Routes (order matters — more specific first)
app.route('/search', searchRoutes)
app.route('/hashtag', hashtagRoutes)
app.route('/i/lists', listRoutes)

// Tweet detail: /:username/status/:id  (must be before profile catch-all)
app.route('/', tweetRoutes)

// Profile: /:username (catch-all, must be last)
app.route('/', profileRoutes)

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
