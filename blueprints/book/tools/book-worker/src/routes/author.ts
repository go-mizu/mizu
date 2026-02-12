import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ensureSchema } from '../db'
import * as db from '../db'
import { renderLayout, renderAuthorPage } from '../html'
import { DEFAULT_LIMIT } from '../config'

const app = new Hono<HonoEnv>()

app.get('/:id', async (c) => {
  await ensureSchema(c.env.DB)

  const id = parseInt(c.req.param('id'), 10)
  if (!id) return c.redirect('/')

  const page = parseInt(c.req.query('page') || '1', 10) || 1

  const author = await db.getAuthor(c.env.DB, id)
  if (!author) {
    return c.html(renderLayout('Not Found - Books', '<div class="empty-state"><h2>Author not found</h2><p><a href="/">Go home</a></p></div>'), 404)
  }

  const booksResult = await db.getAuthorBooks(c.env.DB, id, page, DEFAULT_LIMIT)

  return c.html(renderLayout(
    `${author.name} - Books`,
    renderAuthorPage(author, booksResult.books, page, booksResult.total)
  ))
})

export default app
