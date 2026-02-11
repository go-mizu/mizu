import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { CCClient } from '../cc'
import { Cache } from '../cache'
import { renderLayout, renderViewPage, renderError } from '../html'

const app = new Hono<HonoEnv>()

app.get('/', async (c) => {
  const file = c.req.query('file')
  const offset = c.req.query('offset')
  const length = c.req.query('length')
  const url = c.req.query('url') || ''

  if (!file || !offset || !length) {
    return c.html(renderError('Missing Parameters', 'file, offset, and length query parameters are required.'), 400)
  }

  try {
    const cache = new Cache(c.env.KV)
    const cc = new CCClient(cache)
    const record = await cc.fetchWARC(file, parseInt(offset), parseInt(length))
    const title = url ? `View: ${url}` : 'WARC Record'
    return c.html(renderLayout(`${title} - CC Viewer`, renderViewPage(record, url || record.targetURI, file), { query: url }))
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    return c.html(renderError('WARC Fetch Failed', message), 500)
  }
})

export default app
