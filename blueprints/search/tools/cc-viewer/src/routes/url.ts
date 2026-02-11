import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { CCClient } from '../cc'
import { Cache } from '../cache'
import { renderLayout, renderURLPage, renderError } from '../html'

const app = new Hono<HonoEnv>()

// /url/https://example.com/path
app.get('/*', async (c) => {
  // Extract the URL from the path (everything after /url/)
  const rawURL = c.req.path.substring(5) // remove "/url/"
  if (!rawURL) {
    return c.html(renderError('URL Required', 'Please provide a URL to look up.'), 400)
  }

  // Ensure it's a proper URL
  let url = rawURL
  if (!url.startsWith('http://') && !url.startsWith('https://')) {
    url = 'https://' + url
  }

  const crawl = c.req.query('crawl')

  try {
    const cache = new Cache(c.env.KV)
    const cc = new CCClient(cache)
    const result = await cc.lookupURL(url, crawl || undefined)
    return c.html(renderLayout(`${url} - CC Viewer`, renderURLPage(url, result.entries, result.crawl), { query: url }))
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    return c.html(renderError('Lookup Failed', message), 500)
  }
})

export default app
