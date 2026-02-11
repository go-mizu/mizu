import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { CCClient } from '../cc'
import { Cache } from '../cache'

const app = new Hono<HonoEnv>()

// CORS middleware
app.use('*', async (c, next) => {
  await next()
  c.header('Access-Control-Allow-Origin', '*')
  c.header('Access-Control-Allow-Methods', 'GET')
})

app.get('/health', (c) => c.json({ status: 'ok', timestamp: new Date().toISOString() }))

app.get('/crawls', async (c) => {
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const crawls = await cc.listCrawls()
  return c.json(crawls)
})

app.get('/url', async (c) => {
  const url = c.req.query('url')
  if (!url) return c.json({ error: 'url parameter required' }, 400)
  const crawl = c.req.query('crawl')
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const result = await cc.lookupURL(url, crawl || undefined)
  return c.json(result)
})

app.get('/domain', async (c) => {
  const domain = c.req.query('domain')
  if (!domain) return c.json({ error: 'domain parameter required' }, 400)
  const crawl = c.req.query('crawl')
  const page = parseInt(c.req.query('page') || '0')
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const result = await cc.lookupDomain(domain, crawl || undefined, page)
  const stats = cc.computeDomainStats(result.entries)
  return c.json({ ...result, stats })
})

app.get('/view', async (c) => {
  const file = c.req.query('file')
  const offset = c.req.query('offset')
  const length = c.req.query('length')
  if (!file || !offset || !length) return c.json({ error: 'file, offset, length parameters required' }, 400)
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const record = await cc.fetchWARC(file, parseInt(offset), parseInt(length))
  return c.json(record)
})

export default app
