import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { CCClient, groupByURL } from '../cc'
import { Cache } from '../cache'
import { Analytics } from '../analytics'

const app = new Hono<HonoEnv>()

// GET /url - lookup a URL across crawls
app.get('/', async (c) => {
  const cc = new CCClient(new Cache(c.env.KV))
  const analytics = new Analytics(c.env.ANALYTICS)
  const url = c.req.query('url')
  if (!url) return c.json({ error: 'url parameter required' }, 400)

  const crawl = c.req.query('crawl')

  analytics.track('api_call', { endpoint: 'url_lookup', url })

  const result = await cc.lookupURL(url, crawl || undefined)
  const groups = groupByURL(result.entries)

  return c.json({ ...result, groups })
})

export default app
