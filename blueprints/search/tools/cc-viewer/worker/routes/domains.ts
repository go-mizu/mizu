import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { CCClient } from '../cc'
import { Cache } from '../cache'
import { Analytics } from '../analytics'

const app = new Hono<HonoEnv>()

// GET /domains - list top domains from cluster data
app.get('/', async (c) => {
  const cc = new CCClient(new Cache(c.env.KV))
  const analytics = new Analytics(c.env.ANALYTICS)
  const crawl = c.req.query('crawl') || await cc.getLatestCrawl()
  const page = parseInt(c.req.query('page') || '0')

  analytics.track('api_call', { endpoint: 'domains', crawl })

  const result = await cc.getClusterDomains(crawl, page)
  return c.json(result)
})

// GET /domains/lookup - lookup a specific domain
app.get('/lookup', async (c) => {
  const cc = new CCClient(new Cache(c.env.KV))
  const analytics = new Analytics(c.env.ANALYTICS)
  const domain = c.req.query('domain')
  if (!domain) return c.json({ error: 'domain parameter required' }, 400)

  const crawl = c.req.query('crawl') || await cc.getLatestCrawl()
  const page = parseInt(c.req.query('page') || '0')

  analytics.track('api_call', { endpoint: 'domain_lookup', domain, crawl })

  const { entries, totalPages } = await cc.lookupDomain(domain, crawl, page)
  const stats = cc.computeDomainStats(entries)
  const groups = (await import('../cc')).groupByURL(entries)

  return c.json({ entries, stats, groups, crawl, page, totalPages })
})

export default app
