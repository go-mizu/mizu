import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ensureSchema } from '../db'
import * as db from '../db'
import { renderLayout, renderBookDetail } from '../html'

const app = new Hono<HonoEnv>()

// Book detail page
app.get('/:id', async (c) => {
  await ensureSchema(c.env.DB)

  const id = parseInt(c.req.param('id'), 10)
  if (!id) return c.redirect('/')

  const [book, reviewsResult, quotes, similar, progress, note, shelves, userShelf, userRating] = await Promise.all([
    db.getBook(c.env.DB, id),
    db.getBookReviews(c.env.DB, id, 'popular', 1, 10),
    db.getBookQuotes(c.env.DB, id),
    db.getSimilarBooks(c.env.DB, id, 6),
    db.getProgress(c.env.DB, id),
    db.getNote(c.env.DB, id),
    db.getShelves(c.env.DB),
    db.getUserShelf(c.env.DB, id),
    db.getUserRating(c.env.DB, id),
  ])

  if (!book) {
    return c.html(renderLayout('Not Found - Books', '<div class="empty-state"><h2>Book not found</h2><p><a href="/">Go home</a></p></div>'), 404)
  }

  book.user_rating = userRating
  book.user_shelf = userShelf

  return c.html(renderLayout(
    `${book.title} - Books`,
    renderBookDetail(book, reviewsResult.reviews, quotes, similar, progress, note, shelves, userShelf, userRating)
  ))
})

// Add to shelf
app.post('/:id/shelf', async (c) => {
  await ensureSchema(c.env.DB)

  const id = parseInt(c.req.param('id'), 10)
  if (!id) return c.redirect('/')

  const body = await c.req.parseBody()
  const shelfId = parseInt(body.shelf_id as string, 10)
  if (!shelfId) return c.redirect(`/book/${id}`)

  const book = await db.getBook(c.env.DB, id)
  await db.addBookToShelf(c.env.DB, shelfId, id)

  if (book) {
    await db.addFeedItem(c.env.DB, 'shelved', id, book.title, JSON.stringify({ shelf_id: shelfId }))
  }

  return c.redirect(`/book/${id}`, 302)
})

// Remove from shelf
app.post('/:id/shelf/remove', async (c) => {
  await ensureSchema(c.env.DB)

  const id = parseInt(c.req.param('id'), 10)
  if (!id) return c.redirect('/')

  // Find which exclusive shelf the book is on and remove it
  const userShelf = await db.getUserShelf(c.env.DB, id)
  if (userShelf) {
    const shelf = await db.getShelfBySlug(c.env.DB, userShelf)
    if (shelf) await db.removeBookFromShelf(c.env.DB, shelf.id, id)
  }

  return c.redirect(`/book/${id}`, 302)
})

// Rate book
app.post('/:id/rate', async (c) => {
  await ensureSchema(c.env.DB)

  const id = parseInt(c.req.param('id'), 10)
  if (!id) return c.redirect('/')

  const body = await c.req.parseBody()
  const rating = parseInt(body.rating as string, 10)
  if (!rating || rating < 1 || rating > 5) return c.redirect(`/book/${id}`)

  await db.createReview(c.env.DB, {
    book_id: id,
    rating,
    source: 'user',
  })

  return c.redirect(`/book/${id}`, 302)
})

// Submit review
app.post('/:id/review', async (c) => {
  await ensureSchema(c.env.DB)

  const id = parseInt(c.req.param('id'), 10)
  if (!id) return c.redirect('/')

  const body = await c.req.parseBody()
  const text = (body.text as string || '').trim()
  const rating = parseInt(body.rating as string, 10) || 0
  const isSpoiler = body.is_spoiler === 'on' || body.is_spoiler === '1' ? 1 : 0

  await db.createReview(c.env.DB, {
    book_id: id,
    rating,
    text,
    is_spoiler: isSpoiler,
    source: 'user',
  })

  const book = await db.getBook(c.env.DB, id)
  if (book) {
    await db.addFeedItem(c.env.DB, 'review', id, book.title, JSON.stringify({ rating, text: text.slice(0, 100) }))
  }

  return c.redirect(`/book/${id}`, 302)
})

// Update progress
app.post('/:id/progress', async (c) => {
  await ensureSchema(c.env.DB)

  const id = parseInt(c.req.param('id'), 10)
  if (!id) return c.redirect('/')

  const body = await c.req.parseBody()
  const page = parseInt(body.page as string, 10) || 0
  const percent = parseFloat(body.percent as string) || 0
  const note = (body.note as string || '').trim()

  await db.addProgress(c.env.DB, id, page, percent, note)

  const book = await db.getBook(c.env.DB, id)
  if (book) {
    await db.addFeedItem(c.env.DB, 'progress', id, book.title, JSON.stringify({ page, percent }))
  }

  return c.redirect(`/book/${id}`, 302)
})

// Save notes
app.post('/:id/notes', async (c) => {
  await ensureSchema(c.env.DB)

  const id = parseInt(c.req.param('id'), 10)
  if (!id) return c.redirect('/')

  const body = await c.req.parseBody()
  const text = (body.text as string || '').trim()

  await db.upsertNote(c.env.DB, id, text)

  return c.redirect(`/book/${id}`, 302)
})

// Delete notes
app.post('/:id/notes/delete', async (c) => {
  await ensureSchema(c.env.DB)

  const id = parseInt(c.req.param('id'), 10)
  if (!id) return c.redirect('/')

  await db.deleteNote(c.env.DB, id)

  return c.redirect(`/book/${id}`, 302)
})

export default app
