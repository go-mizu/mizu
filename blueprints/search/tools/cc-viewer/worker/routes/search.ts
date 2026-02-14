import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { CCClient, groupByURL } from '../cc'
import { Cache } from '../cache'
import { Analytics } from '../analytics'

const app = new Hono<HonoEnv>()

// GET /search - universal search with auto-detect
app.get('/', async (c) => {
  const cc = new CCClient(new Cache(c.env.KV))
  const analytics = new Analytics(c.env.ANALYTICS)
  const q = c.req.query('q')?.trim()
  if (!q) return c.json({ error: 'q parameter required' }, 400)

  const type = c.req.query('type') || detectType(q)
  const crawl = c.req.query('crawl') || await cc.getLatestCrawl()

  analytics.track('search', { query: q, type, resultsCount: 0 })

  if (type === 'url') {
    const url = normalizeURL(q)
    const result = await cc.lookupURL(url, crawl)
    const groups = groupByURL(result.entries)
    return c.json({ type: 'url', query: q, url, crawl, entries: result.entries, groups })
  }

  if (type === 'domain') {
    const domain = normalizeDomain(q)
    const { entries, totalPages } = await cc.lookupDomain(domain, crawl)
    const stats = cc.computeDomainStats(entries)
    return c.json({ type: 'domain', query: q, domain, crawl, entries, stats, totalPages })
  }

  // Prefix search
  const { entries, totalPages } = await cc.browseCDX(crawl, q)
  return c.json({ type: 'prefix', query: q, crawl, entries, totalPages })
})

function detectType(q: string): string {
  if (/^https?:\/\//i.test(q)) return 'url'
  if (q.includes('/') && q.includes('.')) return 'url'
  if (q.includes('.') && !q.includes(' ')) return 'domain'
  return 'prefix'
}

function normalizeURL(q: string): string {
  if (!/^https?:\/\//i.test(q)) return `https://${q}`
  return q
}

function normalizeDomain(q: string): string {
  return q.replace(/^(https?:\/\/)?/, '').replace(/\/.*$/, '').toLowerCase()
}

export default app
