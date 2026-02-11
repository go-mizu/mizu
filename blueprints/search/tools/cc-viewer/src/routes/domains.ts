import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { CCClient } from '../cc'
import { Cache } from '../cache'
import { renderLayout, renderDomainsPage, renderError } from '../html'

const app = new Hono<HonoEnv>()

// Domains listing — reads pre-processed cluster.idx data from KV
app.get('/', async (c) => {
  const crawl = c.req.query('crawl')
  const page = parseInt(c.req.query('page') || '0')

  try {
    const cache = new Cache(c.env.KV)
    const cc = new CCClient(cache)
    const crawlID = crawl || await cc.getLatestCrawl()
    const result = await cc.getClusterDomains(crawlID, page)
    const meta = await cc.getClusterMeta(crawlID)

    return c.html(renderLayout(`Domains - CC Viewer`, renderDomainsPage(crawlID, result.domains, page, result.totalPages, meta?.totalDomains || 0, meta?.totalEntries || 0)))
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    return c.html(renderError('Failed to Load Domains', message), 500)
  }
})

export default app
