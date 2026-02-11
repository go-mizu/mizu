import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { CCClient } from '../cc'
import { Cache } from '../cache'
import { renderLayout, renderCrawlDetailPage, renderCrawlFilesPage, renderCrawlCDXPage, renderError } from '../html'

const app = new Hono<HonoEnv>()

// Crawl detail page
app.get('/:id', async (c) => {
  const id = c.req.param('id')
  try {
    const cache = new Cache(c.env.KV)
    const cc = new CCClient(cache)
    const crawl = await cc.getCrawlDetail(id)
    if (!crawl) return c.html(renderError('Crawl Not Found', `No crawl with ID "${id}"`), 404)
    const stats = await cc.fetchCrawlStats(id)
    return c.html(renderLayout(`${id} - CC Viewer`, renderCrawlDetailPage(crawl, stats)))
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    return c.html(renderError('Failed to Load Crawl', message), 500)
  }
})

// WARC files listing
app.get('/:id/files', async (c) => {
  const id = c.req.param('id')
  const page = parseInt(c.req.query('page') || '0')
  const perPage = 100

  try {
    const cache = new Cache(c.env.KV)
    const cc = new CCClient(cache)
    const listing = await cc.fetchWARCFileList(id)
    const totalPages = Math.ceil(listing.totalFiles / perPage)
    const start = page * perPage
    const pageFiles = listing.files.slice(start, start + perPage)
    return c.html(renderLayout(`Files - ${id}`, renderCrawlFilesPage(id, pageFiles, page, totalPages, listing.totalFiles)))
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    return c.html(renderError('Failed to Load Files', message), 500)
  }
})

// CDX prefix browse
app.get('/:id/cdx', async (c) => {
  const id = c.req.param('id')
  const prefix = c.req.query('prefix') || ''
  const page = parseInt(c.req.query('page') || '0')

  try {
    const cache = new Cache(c.env.KV)
    const cc = new CCClient(cache)

    if (!prefix) {
      // Show cluster.idx entries as CDX page directory
      const cluster = await cc.getClusterEntries(id, page)
      return c.html(renderLayout(`CDX Browse - ${id}`, renderCrawlCDXPage(id, [], 0, 0, '', cluster.entries, page, cluster.totalPages)))
    }

    const result = await cc.browseCDX(id, prefix, page)
    return c.html(renderLayout(`CDX: ${prefix} - ${id}`, renderCrawlCDXPage(id, result.entries, page, result.totalPages, prefix)))
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    return c.html(renderError('CDX Browse Failed', message), 500)
  }
})

export default app
