import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ensureSchema } from '../db'
import * as db from '../db'
import { renderLayout, renderSearchResults } from '../html'
import { searchOL } from '../openlibrary'
import { DEFAULT_LIMIT } from '../config'

const app = new Hono<HonoEnv>()

app.get('/', async (c) => {
  await ensureSchema(c.env.DB)

  const q = (c.req.query('q') || '').trim()
  const page = parseInt(c.req.query('page') || '1', 10) || 1

  if (!q) {
    return c.html(renderLayout('Search - Books', renderSearchResults(q, [], [], page, 0), { query: q }))
  }

  // Search local DB and Open Library in parallel
  const [localResults, olResults] = await Promise.all([
    db.searchBooks(c.env.DB, q, page, DEFAULT_LIMIT),
    searchOL(c.env.KV, q, DEFAULT_LIMIT, (page - 1) * DEFAULT_LIMIT).catch(() => ({ docs: [], numFound: 0 })),
  ])

  const total = Math.max(localResults.total, olResults.numFound)
  return c.html(renderLayout(`Search: ${q} - Books`, renderSearchResults(q, localResults.books, olResults.docs, page, total), { query: q }))
})

export default app
