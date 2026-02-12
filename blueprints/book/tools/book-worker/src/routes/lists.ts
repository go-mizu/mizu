import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ensureSchema } from '../db'
import * as db from '../db'
import { renderLayout, renderListsPage, renderListDetailPage } from '../html'
import { DEFAULT_LIMIT } from '../config'

const app = new Hono<HonoEnv>()

// All lists
app.get('/', async (c) => {
  await ensureSchema(c.env.DB)

  const page = parseInt(c.req.query('page') || '1', 10) || 1
  const listsResult = await db.getLists(c.env.DB, page, DEFAULT_LIMIT)

  return c.html(renderLayout('Lists - Books', renderListsPage(listsResult.lists, page, listsResult.total)))
})

// List detail
app.get('/:id', async (c) => {
  await ensureSchema(c.env.DB)

  const id = parseInt(c.req.param('id'), 10)
  if (!id) return c.redirect('/lists')

  const list = await db.getList(c.env.DB, id)
  if (!list) {
    return c.html(renderLayout('Not Found - Books', '<div class="empty-state"><h2>List not found</h2><p><a href="/lists">Back to lists</a></p></div>'), 404)
  }

  return c.html(renderLayout(`${list.title} - Lists - Books`, renderListDetailPage(list)))
})

// Create list
app.post('/', async (c) => {
  await ensureSchema(c.env.DB)

  const body = await c.req.parseBody()
  const title = (body.title as string || '').trim()
  const description = (body.description as string || '').trim()

  if (!title) return c.redirect('/lists', 302)

  await db.createList(c.env.DB, title, description)

  return c.redirect('/lists', 302)
})

// Vote on list item
app.post('/:id/vote/:bookId', async (c) => {
  await ensureSchema(c.env.DB)

  const id = parseInt(c.req.param('id'), 10)
  const bookId = parseInt(c.req.param('bookId'), 10)
  if (!id || !bookId) return c.redirect('/lists')

  await db.voteOnListItem(c.env.DB, id, bookId)

  return c.redirect(`/lists/${id}`, 302)
})

export default app
