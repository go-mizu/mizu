import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { CCClient, groupByURL } from '../cc'
import { Cache } from '../cache'
import { renderLayout, renderDomainPage, renderError } from '../html'

const app = new Hono<HonoEnv>()

app.get('/:domain', async (c) => {
  const domain = c.req.param('domain')
  const crawl = c.req.query('crawl')
  const page = parseInt(c.req.query('page') || '0')

  try {
    const cache = new Cache(c.env.KV)
    const cc = new CCClient(cache)
    const result = await cc.lookupDomain(domain, crawl || undefined, page)
    const stats = cc.computeDomainStats(result.entries)
    const groups = groupByURL(result.entries)
    return c.html(renderLayout(`${domain} - CC Viewer`, renderDomainPage(domain, groups, result.crawl, page, result.totalPages, stats), { query: domain }))
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    return c.html(renderError('Domain Lookup Failed', message), 500)
  }
})

export default app
