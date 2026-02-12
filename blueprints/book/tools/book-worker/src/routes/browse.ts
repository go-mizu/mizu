import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ensureSchema } from '../db'
import * as db from '../db'
import { renderLayout, renderBrowsePage, renderGenrePage } from '../html'
import { DEFAULT_LIMIT } from '../config'

const app = new Hono<HonoEnv>()

// Browse page
app.get('/', async (c) => {
  await ensureSchema(c.env.DB)

  const [genres, popular, newReleases] = await Promise.all([
    db.getGenres(c.env.DB),
    db.getPopularBooks(c.env.DB, 12),
    db.getNewReleases(c.env.DB, 12),
  ])

  return c.html(renderLayout('Browse - Books', renderBrowsePage(genres, popular, newReleases)))
})

// Genre page
app.get('/genre/:slug', async (c) => {
  await ensureSchema(c.env.DB)

  const slug = decodeURIComponent(c.req.param('slug'))
  const page = parseInt(c.req.query('page') || '1', 10) || 1

  const booksResult = await db.getBooksByGenre(c.env.DB, slug, page, DEFAULT_LIMIT)

  return c.html(renderLayout(
    `${slug} - Browse - Books`,
    renderGenrePage(slug, booksResult.books, page, booksResult.total)
  ))
})

export default app
