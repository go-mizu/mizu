import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ensureSchema } from '../db'
import * as db from '../db'
import { renderLayout, renderHomePage } from '../html'

const app = new Hono<HonoEnv>()

app.get('/', async (c) => {
  await ensureSchema(c.env.DB)

  const currentYear = new Date().getFullYear()

  const [trending, shelves, feed, challenge] = await Promise.all([
    db.getTrendingBooks(c.env.DB, 12),
    db.getShelfBySlug(c.env.DB, 'currently-reading').then(async (shelf) => {
      if (!shelf) return []
      const result = await db.getShelfBooks(c.env.DB, shelf.id, 'date_added', 1, 20)
      return result.books
    }),
    db.getFeed(c.env.DB, 10),
    db.getChallenge(c.env.DB, currentYear),
  ])

  return c.html(renderLayout('Books', renderHomePage(trending, shelves, feed, challenge)))
})

export default app
