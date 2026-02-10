import { Hono } from 'hono'
import { trimTrailingSlash } from 'hono/trailing-slash'
import type { HonoEnv } from './types'
import { cssURL, cssText } from './asset'
import { renderLayout, renderHomePage, renderError } from './html'

import apiRoutes from './routes/api'
import searchRoutes from './routes/search'
import hashtagRoutes from './routes/hashtag'
import locationRoutes from './routes/location'
import storiesRoutes from './routes/stories'
import reelsRoutes from './routes/reels'
import postRoutes from './routes/post'
import followRoutes from './routes/follow'
import profileRoutes from './routes/profile'

const app = new Hono<HonoEnv>()
app.use(trimTrailingSlash())

// Cacheable CSS with content hash
app.get(cssURL, (c) => {
  c.header('Content-Type', 'text/css; charset=utf-8')
  c.header('Cache-Control', 'public, max-age=31536000, immutable')
  return c.body(cssText)
})

// Image proxy — fetch Instagram CDN images server-side to bypass hotlinking protection
app.get('/img/*', async (c) => {
  // The path after /img/ is the encoded Instagram CDN URL
  const encoded = c.req.path.slice(5) // strip "/img/"
  const query = c.req.url.split('?').slice(1).join('?')
  const imgUrl = decodeURIComponent(encoded) + (query ? '?' + query : '')

  // Only allow Instagram CDN domains
  if (!imgUrl.startsWith('https://') || (!imgUrl.includes('.cdninstagram.com/') && !imgUrl.includes('.fbcdn.net/'))) {
    return c.text('Forbidden', 403)
  }

  const resp = await fetch(imgUrl, {
    headers: {
      'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36',
      'Accept': 'image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8',
      'Referer': 'https://www.instagram.com/',
    },
  })

  if (!resp.ok) return c.text('Image not found', 404)

  const ct = resp.headers.get('content-type') || 'image/jpeg'
  c.header('Content-Type', ct)
  c.header('Cache-Control', 'public, max-age=86400') // cache 24h
  return c.body(resp.body as ReadableStream)
})

// Health check
app.get('/api/health', (c) => c.json({ status: 'ok', timestamp: new Date().toISOString() }))

// Session status check
app.get('/api/status', async (c) => {
  const { SessionManager } = await import('./session')
  const status = await new SessionManager(c.env).getStatus()
  return c.json(status)
})

// Force session refresh (manual trigger)
app.post('/api/session/refresh', async (c) => {
  const { SessionManager } = await import('./session')
  try {
    const client = await new SessionManager(c.env).refreshSession()
    const health = await client.checkSession()
    return c.json({ ok: true, health, timestamp: new Date().toISOString() })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.json({ ok: false, error: msg, timestamp: new Date().toISOString() }, 500)
  }
})

// Home page
app.get('/', (c) => {
  return c.html(renderLayout('Insta Viewer', renderHomePage(), { isHome: true }))
})

// JSON API
app.route('/api', apiRoutes)

// Routes (order matters — more specific first)
app.route('/search', searchRoutes)
app.route('/explore/tags', hashtagRoutes)
app.route('/explore/locations', locationRoutes)
app.route('/stories', storiesRoutes)
app.route('/reels', reelsRoutes)
app.route('/p', postRoutes)

// Follow: /:username/followers, /:username/following (before profile catch-all)
app.route('/', followRoutes)

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
