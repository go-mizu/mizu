import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ensureSchema } from '../db'
import * as db from '../db'
import { renderLayout, renderQuotesPage } from '../html'
import { DEFAULT_LIMIT } from '../config'

const app = new Hono<HonoEnv>()

app.get('/', async (c) => {
  await ensureSchema(c.env.DB)

  const page = parseInt(c.req.query('page') || '1', 10) || 1
  const quotesResult = await db.getQuotes(c.env.DB, page, DEFAULT_LIMIT)

  return c.html(renderLayout('Quotes - Books', renderQuotesPage(quotesResult.quotes, page, quotesResult.total)))
})

export default app
