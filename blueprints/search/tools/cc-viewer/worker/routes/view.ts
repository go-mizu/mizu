import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { CCClient } from '../cc'
import { Cache } from '../cache'
import { Analytics } from '../analytics'

const app = new Hono<HonoEnv>()

// GET /view - fetch and parse a WARC record
app.get('/', async (c) => {
  const cc = new CCClient(new Cache(c.env.KV))
  const analytics = new Analytics(c.env.ANALYTICS)

  const file = c.req.query('file')
  const offset = c.req.query('offset')
  const length = c.req.query('length')

  if (!file || !offset || !length) {
    return c.json({ error: 'file, offset, and length parameters required' }, 400)
  }

  analytics.track('warc_fetch', { file, type: 'warc' })

  const record = await cc.fetchWARC(file, parseInt(offset), parseInt(length))
  return c.json(record)
})

// GET /view/wat - fetch WAT metadata record
app.get('/wat', async (c) => {
  const cc = new CCClient(new Cache(c.env.KV))
  const analytics = new Analytics(c.env.ANALYTICS)

  const file = c.req.query('file')
  const offset = c.req.query('offset')
  const length = c.req.query('length')

  if (!file || !offset || !length) {
    return c.json({ error: 'file, offset, and length parameters required' }, 400)
  }

  // Replace /warc/ with /wat/ in the filename
  const watFile = file.replace('/warc/', '/wat/')
  analytics.track('warc_fetch', { file: watFile, type: 'wat' })

  const record = await cc.fetchWARC(watFile, parseInt(offset), parseInt(length))
  return c.json(record)
})

// GET /view/wet - fetch WET plaintext record
app.get('/wet', async (c) => {
  const cc = new CCClient(new Cache(c.env.KV))
  const analytics = new Analytics(c.env.ANALYTICS)

  const file = c.req.query('file')
  const offset = c.req.query('offset')
  const length = c.req.query('length')

  if (!file || !offset || !length) {
    return c.json({ error: 'file, offset, and length parameters required' }, 400)
  }

  // Replace /warc/ with /wet/ in the filename
  const wetFile = file.replace('/warc/', '/wet/')
  analytics.track('warc_fetch', { file: wetFile, type: 'wet' })

  const record = await cc.fetchWARC(wetFile, parseInt(offset), parseInt(length))
  return c.json(record)
})

// GET /view/robots - fetch robots.txt for a domain
app.get('/robots', async (c) => {
  const cc = new CCClient(new Cache(c.env.KV))
  const domain = c.req.query('domain')
  if (!domain) return c.json({ error: 'domain parameter required' }, 400)

  const crawl = c.req.query('crawl') || await cc.getLatestCrawl()

  // Look up robots.txt via CDX
  const robotsURL = `https://${domain}/robots.txt`
  const result = await cc.lookupURL(robotsURL, crawl)

  if (result.entries.length === 0) {
    return c.json({ found: false, domain, crawl })
  }

  // Fetch the most recent robots.txt
  const latest = result.entries[result.entries.length - 1]
  const record = await cc.fetchWARC(latest.filename, parseInt(latest.offset), parseInt(latest.length))

  return c.json({ found: true, domain, crawl, record, captures: result.entries.length })
})

export default app
