import { Hono } from 'hono'
import type { HonoEnv } from './types'
import { cssURL, cssText } from './asset'
import { renderLayout, renderHomePage, renderError } from './html'
import { ensureSchema } from './db'
import * as db from './db'
import { searchOL } from './openlibrary'

import apiRoutes from './routes/api'
import homeRoutes from './routes/home'
import searchRoutes from './routes/search'
import bookRoutes from './routes/book'
import authorRoutes from './routes/author'
import shelfRoutes from './routes/shelf'
import browseRoutes from './routes/browse'
import listsRoutes from './routes/lists'
import quotesRoutes from './routes/quotes'
import statsRoutes from './routes/stats'
import challengeRoutes from './routes/challenge'
import importRoutes from './routes/import'

const app = new Hono<HonoEnv>()

// Cacheable CSS with content hash
app.get(cssURL, (c) => {
  c.header('Content-Type', 'text/css; charset=utf-8')
  c.header('Cache-Control', 'public, max-age=31536000, immutable')
  return c.body(cssText)
})

// Health check
app.get('/api/health', (c) => c.json({ status: 'ok', timestamp: new Date().toISOString() }))

// JSON API
app.route('/api', apiRoutes)

// SSR pages (order matters — more specific first)
app.route('/search', searchRoutes)
app.route('/book', bookRoutes)
app.route('/author', authorRoutes)
app.route('/shelf', shelfRoutes)
app.route('/browse', browseRoutes)
app.route('/lists', listsRoutes)
app.route('/list', listsRoutes)
app.route('/quotes', quotesRoutes)
app.route('/stats', statsRoutes)
app.route('/challenge', challengeRoutes)
app.route('/import', importRoutes)

// Home page (catch-all, must be last)
app.route('/', homeRoutes)

// 404
app.notFound((c) => {
  return c.html(renderLayout('Not Found', renderError('Page not found', 'The page you\'re looking for doesn\'t exist.'), { currentPath: '' }), 404)
})

// Error handler
app.onError((err, c) => {
  console.error('[Error]', err.message)
  return c.html(renderLayout('Error', renderError('Something went wrong', err.message), { currentPath: '' }), 500)
})

export default app
