import { Hono } from 'hono'
import type { HonoEnv, CrawlStats } from '../types'
import { CCClient } from '../cc'
import { Cache } from '../cache'
import { renderLayout, renderCrawlsPage, renderError } from '../html'

const app = new Hono<HonoEnv>()

app.get('/', async (c) => {
  try {
    const cache = new Cache(c.env.KV)
    const cc = new CCClient(cache)
    const crawls = await cc.listCrawls()

    // Fetch stats for first 12 crawls in parallel (rest load without stats)
    const withStats = crawls.map((crawl, i) => ({ ...crawl, stats: undefined as CrawlStats | undefined }))
    const statsPromises = crawls.slice(0, 12).map(async (crawl, i) => {
      try {
        withStats[i].stats = await cc.fetchCrawlStats(crawl.id)
      } catch { /* stats are best-effort */ }
    })
    await Promise.all(statsPromises)

    return c.html(renderLayout('Crawls - CC Viewer', renderCrawlsPage(withStats)))
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    return c.html(renderError('Failed to Load Crawls', message), 500)
  }
})

export default app
