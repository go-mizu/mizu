import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ensureSchema } from '../db'
import * as db from '../db'
import { renderLayout, renderShelfPage } from '../html'
import { DEFAULT_LIMIT } from '../config'

const app = new Hono<HonoEnv>()

// All books view
app.get('/', async (c) => {
  await ensureSchema(c.env.DB)

  const sort = c.req.query('sort') || 'date_added'
  const page = parseInt(c.req.query('page') || '1', 10) || 1

  const shelves = await db.getShelves(c.env.DB)

  // Get books from all shelves
  let allBooks: any[] = []
  let total = 0
  for (const s of shelves) {
    const result = await db.getShelfBooks(c.env.DB, s.id, sort, 1, 1000)
    allBooks = allBooks.concat(result.books)
  }
  // Deduplicate by book_id
  const seen = new Set<number>()
  allBooks = allBooks.filter(sb => {
    if (seen.has(sb.book_id)) return false
    seen.add(sb.book_id)
    return true
  })
  total = allBooks.length
  const offset = (page - 1) * DEFAULT_LIMIT
  const pagedBooks = allBooks.slice(offset, offset + DEFAULT_LIMIT)

  return c.html(renderLayout(
    'My Books',
    renderShelfPage(shelves, null, pagedBooks, page, total)
  ))
})

// Create shelf
app.post('/create', async (c) => {
  await ensureSchema(c.env.DB)

  const body = await c.req.parseBody()
  const name = (body.name as string || '').trim()
  if (!name) return c.redirect('/shelf', 302)

  const slug = name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
  await db.createShelf(c.env.DB, name, slug, false)

  return c.redirect(`/shelf/${slug}`, 302)
})

// Specific shelf
app.get('/:slug', async (c) => {
  await ensureSchema(c.env.DB)

  const slug = c.req.param('slug')
  const sort = c.req.query('sort') || 'date_added'
  const page = parseInt(c.req.query('page') || '1', 10) || 1

  const [shelves, shelf] = await Promise.all([
    db.getShelves(c.env.DB),
    db.getShelfBySlug(c.env.DB, slug),
  ])

  if (!shelf) {
    return c.html(renderLayout('Not Found - Books', '<div class="empty-state"><h2>Shelf not found</h2><p><a href="/shelf">Go back</a></p></div>'), 404)
  }

  const booksResult = await db.getShelfBooks(c.env.DB, shelf.id, sort, page, DEFAULT_LIMIT)

  return c.html(renderLayout(
    `${shelf.name} - Books`,
    renderShelfPage(shelves, shelf, booksResult.books, page, booksResult.total)
  ))
})

export default app
