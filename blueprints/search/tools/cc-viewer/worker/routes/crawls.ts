import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { CCClient } from '../cc'
import { Cache } from '../cache'
import { Analytics } from '../analytics'

const app = new Hono<HonoEnv>()

// GET /crawls — list all crawls with optional stats
app.get('/', async (c) => {
  const start = Date.now()
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const analytics = new Analytics(c.env.ANALYTICS)
  const withStats = c.req.query('stats') === 'true'

  try {
    const crawls = await cc.listCrawls()

    if (!withStats) {
      analytics.apiCall('/api/crawls', Date.now() - start, false)
      return c.json({ crawls, total: crawls.length })
    }

    // Fetch stats for first 12 crawls in parallel
    const enriched = await Promise.all(
      crawls.slice(0, 12).map(async (crawl) => {
        try {
          const stats = await cc.fetchCrawlStats(crawl.id)
          return { ...crawl, stats }
        } catch {
          return { ...crawl, stats: null }
        }
      })
    )

    // Append remaining crawls without stats
    const remaining = crawls.slice(12).map(crawl => ({ ...crawl, stats: null }))
    const all = [...enriched, ...remaining]

    analytics.apiCall('/api/crawls', Date.now() - start, false)
    return c.json({ crawls: all, total: all.length })
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    analytics.error('/api/crawls', 500, message)
    return c.json({ error: message }, 500)
  }
})

// GET /crawls/:id — crawl detail + stats
app.get('/:id', async (c) => {
  const start = Date.now()
  const id = c.req.param('id')
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const analytics = new Analytics(c.env.ANALYTICS)

  try {
    const crawl = await cc.getCrawlDetail(id)
    if (!crawl) {
      return c.json({ error: `Crawl "${id}" not found` }, 404)
    }

    const stats = await cc.fetchCrawlStats(id)
    analytics.apiCall(`/api/crawls/${id}`, Date.now() - start, false)
    return c.json({ crawl, stats })
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    analytics.error(`/api/crawls/${id}`, 500, message)
    return c.json({ error: message }, 500)
  }
})

// GET /crawls/:id/files — WARC file listing (paginated, 100/page)
app.get('/:id/files', async (c) => {
  const start = Date.now()
  const id = c.req.param('id')
  const page = parseInt(c.req.query('page') || '0')
  const perPage = parseInt(c.req.query('limit') || '100')
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const analytics = new Analytics(c.env.ANALYTICS)

  try {
    const listing = await cc.fetchWARCFileList(id)
    const totalPages = Math.ceil(listing.totalFiles / perPage)
    const startIdx = page * perPage
    const files = listing.files.slice(startIdx, startIdx + perPage)

    analytics.apiCall(`/api/crawls/${id}/files`, Date.now() - start, false)
    return c.json({
      crawl: id,
      files,
      page,
      perPage,
      totalPages,
      totalFiles: listing.totalFiles,
    })
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    analytics.error(`/api/crawls/${id}/files`, 500, message)
    return c.json({ error: message }, 500)
  }
})

// GET /crawls/:id/cdx — CDX browse (prefix search)
app.get('/:id/cdx', async (c) => {
  const start = Date.now()
  const id = c.req.param('id')
  const prefix = c.req.query('prefix') || ''
  const page = parseInt(c.req.query('page') || '0')
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const analytics = new Analytics(c.env.ANALYTICS)

  try {
    if (!prefix) {
      // Return cluster.idx entries as a directory listing
      const cluster = await cc.getClusterEntries(id, page)
      analytics.apiCall(`/api/crawls/${id}/cdx`, Date.now() - start, false)
      return c.json({
        crawl: id,
        entries: [],
        clusterEntries: cluster.entries,
        page,
        totalPages: cluster.totalPages,
        prefix: '',
      })
    }

    const result = await cc.browseCDX(id, prefix, page)
    analytics.apiCall(`/api/crawls/${id}/cdx`, Date.now() - start, false)
    return c.json({
      crawl: id,
      entries: result.entries,
      page,
      totalPages: result.totalPages,
      prefix,
    })
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    analytics.error(`/api/crawls/${id}/cdx`, 500, message)
    return c.json({ error: message }, 500)
  }
})

export default app
