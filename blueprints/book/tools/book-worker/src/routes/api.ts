import { Hono } from 'hono'
import { cors } from 'hono/cors'
import type { HonoEnv } from '../types'
import {
  ensureSchema, getBook, searchBooks, createBook, getTrendingBooks,
  getSimilarBooks, getBookByOLKey, getAuthor, searchAuthors, getAuthorBooks,
  createAuthor, getShelves, getShelfBySlug, getShelfBooks, addBookToShelf,
  removeBookFromShelf, getUserShelf, createShelf, deleteShelf,
  getBookReviews, createReview, getUserRating, deleteReview, likeReview,
  getReviewComments, createComment, getProgress, addProgress,
  getChallenge, setChallenge, getGenres, getBooksByGenre, getNewReleases,
  getPopularBooks, getLists, getList, createList, addBookToList,
  voteOnListItem, getQuotes, getBookQuotes, createQuote, getStats,
  getFeed, addFeedItem, getNote, upsertNote, deleteNote,
} from '../db'
import { searchOL, olResultToBook, enrichBookFromOL } from '../openlibrary'
import { DEFAULT_LIMIT, MAX_LIMIT } from '../config'

const app = new Hono<HonoEnv>()

app.use('*', cors())

// --- Helpers ---

function parsePageLimit(c: { req: { query: (k: string) => string | undefined } }): { page: number, limit: number } {
  const page = Math.max(1, parseInt(c.req.query('page') || '1', 10) || 1)
  let limit = parseInt(c.req.query('limit') || String(DEFAULT_LIMIT), 10) || DEFAULT_LIMIT
  limit = Math.min(Math.max(1, limit), MAX_LIMIT)
  return { page, limit }
}

// --- Books ---

app.get('/books/search', async (c) => {
  await ensureSchema(c.env.DB)
  const q = c.req.query('q') || ''
  const { page, limit } = parsePageLimit(c)

  const { books, total } = await searchBooks(c.env.DB, q, page, limit)

  let ol_results: any[] | undefined
  if (books.length < limit && q) {
    const ol = await searchOL(c.env.KV, q, limit)
    ol_results = ol.docs.map(olResultToBook)
  }

  return c.json({ books, total_count: total, page, page_size: limit, ol_results })
})

app.get('/books/trending', async (c) => {
  await ensureSchema(c.env.DB)
  const { limit } = parsePageLimit(c)
  const books = await getTrendingBooks(c.env.DB, limit)
  return c.json({ books })
})

app.get('/books/:id', async (c) => {
  await ensureSchema(c.env.DB)
  const id = parseInt(c.req.param('id'), 10)
  const book = await getBook(c.env.DB, id)
  if (!book) return c.json({ error: 'Book not found' }, 404)
  book.user_rating = await getUserRating(c.env.DB, id)
  book.user_shelf = await getUserShelf(c.env.DB, id)
  return c.json({ book })
})

app.post('/books', async (c) => {
  await ensureSchema(c.env.DB)
  const body = await c.req.json()

  if (body.ol_key) {
    const existing = await getBookByOLKey(c.env.DB, body.ol_key)
    if (existing) return c.json({ id: existing.id, book: existing })
  }

  const id = await createBook(c.env.DB, body)
  const book = await getBook(c.env.DB, id)
  return c.json({ id, book })
})

app.get('/books/:id/similar', async (c) => {
  await ensureSchema(c.env.DB)
  const id = parseInt(c.req.param('id'), 10)
  let limit = parseInt(c.req.query('limit') || '10', 10) || 10
  limit = Math.min(Math.max(1, limit), MAX_LIMIT)
  const books = await getSimilarBooks(c.env.DB, id, limit)
  return c.json({ books })
})

// --- Authors ---

app.get('/authors/search', async (c) => {
  await ensureSchema(c.env.DB)
  const q = c.req.query('q') || ''
  const authors = await searchAuthors(c.env.DB, q)
  return c.json({ authors })
})

app.get('/authors/:id', async (c) => {
  await ensureSchema(c.env.DB)
  const id = parseInt(c.req.param('id'), 10)
  const author = await getAuthor(c.env.DB, id)
  if (!author) return c.json({ error: 'Author not found' }, 404)
  return c.json({ author })
})

app.get('/authors/:id/books', async (c) => {
  await ensureSchema(c.env.DB)
  const id = parseInt(c.req.param('id'), 10)
  const { page, limit } = parsePageLimit(c)
  const { books, total } = await getAuthorBooks(c.env.DB, id, page, limit)
  return c.json({ books, total })
})

// --- Shelves ---

app.get('/shelves', async (c) => {
  await ensureSchema(c.env.DB)
  const shelves = await getShelves(c.env.DB)
  return c.json({ shelves })
})

app.post('/shelves', async (c) => {
  await ensureSchema(c.env.DB)
  const { name, slug, is_exclusive } = await c.req.json()
  const id = await createShelf(c.env.DB, name, slug, !!is_exclusive)
  return c.json({ id })
})

app.delete('/shelves/:id', async (c) => {
  await ensureSchema(c.env.DB)
  const id = parseInt(c.req.param('id'), 10)
  const ok = await deleteShelf(c.env.DB, id)
  if (!ok) return c.json({ error: 'Cannot delete default shelf' }, 400)
  return c.json({ ok: true })
})

app.get('/shelves/:id/books', async (c) => {
  await ensureSchema(c.env.DB)
  const id = parseInt(c.req.param('id'), 10)
  const sort = c.req.query('sort') || 'date_added'
  const { page, limit } = parsePageLimit(c)
  const { books, total } = await getShelfBooks(c.env.DB, id, sort, page, limit)
  return c.json({ books, total })
})

app.post('/shelves/:id/books', async (c) => {
  await ensureSchema(c.env.DB)
  const shelfId = parseInt(c.req.param('id'), 10)
  const { book_id } = await c.req.json()
  await addBookToShelf(c.env.DB, shelfId, book_id)
  const book = await getBook(c.env.DB, book_id)
  await addFeedItem(c.env.DB, 'shelved', book_id, book?.title || '', JSON.stringify({ shelf_id: shelfId }))
  return c.json({ ok: true })
})

app.delete('/shelves/:id/books/:bookId', async (c) => {
  await ensureSchema(c.env.DB)
  const shelfId = parseInt(c.req.param('id'), 10)
  const bookId = parseInt(c.req.param('bookId'), 10)
  await removeBookFromShelf(c.env.DB, shelfId, bookId)
  return c.json({ ok: true })
})

// --- Reviews ---

app.get('/books/:id/reviews', async (c) => {
  await ensureSchema(c.env.DB)
  const bookId = parseInt(c.req.param('id'), 10)
  const sort = c.req.query('sort') || 'default'
  const { page, limit } = parsePageLimit(c)
  const { reviews, total } = await getBookReviews(c.env.DB, bookId, sort, page, limit)
  return c.json({ reviews, total })
})

app.post('/books/:id/reviews', async (c) => {
  await ensureSchema(c.env.DB)
  const bookId = parseInt(c.req.param('id'), 10)
  const body = await c.req.json()
  const id = await createReview(c.env.DB, {
    book_id: bookId,
    rating: body.rating,
    text: body.text,
    is_spoiler: body.is_spoiler ? 1 : 0,
    reviewer_name: body.reviewer_name,
    source: body.source || 'user',
  })
  const book = await getBook(c.env.DB, bookId)
  await addFeedItem(c.env.DB, 'review', bookId, book?.title || '', JSON.stringify({ rating: body.rating, review_id: id }))
  return c.json({ id })
})

app.put('/reviews/:id', async (c) => {
  await ensureSchema(c.env.DB)
  const id = parseInt(c.req.param('id'), 10)
  const body = await c.req.json()
  const sets: string[] = []
  const vals: any[] = []
  if (body.rating !== undefined) { sets.push('rating = ?'); vals.push(body.rating) }
  if (body.text !== undefined) { sets.push('text = ?'); vals.push(body.text) }
  if (body.is_spoiler !== undefined) { sets.push('is_spoiler = ?'); vals.push(body.is_spoiler ? 1 : 0) }
  if (sets.length > 0) {
    sets.push("updated_at = datetime('now')")
    vals.push(id)
    await c.env.DB.prepare(`UPDATE reviews SET ${sets.join(', ')} WHERE id = ?`).bind(...vals).run()
  }
  return c.json({ ok: true })
})

app.delete('/reviews/:id', async (c) => {
  await ensureSchema(c.env.DB)
  const id = parseInt(c.req.param('id'), 10)
  await deleteReview(c.env.DB, id)
  return c.json({ ok: true })
})

app.post('/reviews/:id/like', async (c) => {
  await ensureSchema(c.env.DB)
  const id = parseInt(c.req.param('id'), 10)
  await likeReview(c.env.DB, id)
  return c.json({ ok: true })
})

app.get('/reviews/:id/comments', async (c) => {
  await ensureSchema(c.env.DB)
  const id = parseInt(c.req.param('id'), 10)
  const comments = await getReviewComments(c.env.DB, id)
  return c.json({ comments })
})

app.post('/reviews/:id/comments', async (c) => {
  await ensureSchema(c.env.DB)
  const reviewId = parseInt(c.req.param('id'), 10)
  const { author_name, text } = await c.req.json()
  const id = await createComment(c.env.DB, reviewId, author_name, text)
  return c.json({ id })
})

// --- Reading Progress ---

app.get('/books/:id/progress', async (c) => {
  await ensureSchema(c.env.DB)
  const bookId = parseInt(c.req.param('id'), 10)
  const progress = await getProgress(c.env.DB, bookId)
  return c.json({ progress })
})

app.post('/books/:id/progress', async (c) => {
  await ensureSchema(c.env.DB)
  const bookId = parseInt(c.req.param('id'), 10)
  const { page, percent, note } = await c.req.json()
  const id = await addProgress(c.env.DB, bookId, page || 0, percent || 0, note || '')
  const book = await getBook(c.env.DB, bookId)
  await addFeedItem(c.env.DB, 'progress', bookId, book?.title || '', JSON.stringify({ page, percent }))
  return c.json({ id })
})

// --- Challenge ---

app.get('/challenge/:year', async (c) => {
  await ensureSchema(c.env.DB)
  const year = parseInt(c.req.param('year'), 10)
  const challenge = await getChallenge(c.env.DB, year)
  if (!challenge) return c.json({ error: 'Challenge not found' }, 404)
  return c.json({ challenge })
})

app.post('/challenge', async (c) => {
  await ensureSchema(c.env.DB)
  const { year, goal } = await c.req.json()
  await setChallenge(c.env.DB, year, goal)
  return c.json({ ok: true })
})

// --- Browse ---

app.get('/genres', async (c) => {
  await ensureSchema(c.env.DB)
  const genres = await getGenres(c.env.DB)
  return c.json({ genres })
})

app.get('/genres/:genre/books', async (c) => {
  await ensureSchema(c.env.DB)
  const genre = decodeURIComponent(c.req.param('genre'))
  const { page, limit } = parsePageLimit(c)
  const { books, total } = await getBooksByGenre(c.env.DB, genre, page, limit)
  return c.json({ books, total })
})

app.get('/browse/new-releases', async (c) => {
  await ensureSchema(c.env.DB)
  const { limit } = parsePageLimit(c)
  const books = await getNewReleases(c.env.DB, limit)
  return c.json({ books })
})

app.get('/browse/popular', async (c) => {
  await ensureSchema(c.env.DB)
  const { limit } = parsePageLimit(c)
  const books = await getPopularBooks(c.env.DB, limit)
  return c.json({ books })
})

// --- Lists ---

app.get('/lists', async (c) => {
  await ensureSchema(c.env.DB)
  const { page, limit } = parsePageLimit(c)
  const { lists, total } = await getLists(c.env.DB, page, limit)
  return c.json({ lists, total })
})

app.post('/lists', async (c) => {
  await ensureSchema(c.env.DB)
  const { title, description } = await c.req.json()
  const id = await createList(c.env.DB, title, description || '')
  return c.json({ id })
})

app.get('/lists/:id', async (c) => {
  await ensureSchema(c.env.DB)
  const id = parseInt(c.req.param('id'), 10)
  const list = await getList(c.env.DB, id)
  if (!list) return c.json({ error: 'List not found' }, 404)
  return c.json({ list })
})

app.post('/lists/:id/books', async (c) => {
  await ensureSchema(c.env.DB)
  const listId = parseInt(c.req.param('id'), 10)
  const { book_id } = await c.req.json()
  await addBookToList(c.env.DB, listId, book_id)
  return c.json({ ok: true })
})

app.post('/lists/:id/vote/:bookId', async (c) => {
  await ensureSchema(c.env.DB)
  const listId = parseInt(c.req.param('id'), 10)
  const bookId = parseInt(c.req.param('bookId'), 10)
  await voteOnListItem(c.env.DB, listId, bookId)
  return c.json({ ok: true })
})

// --- Quotes ---

app.get('/quotes', async (c) => {
  await ensureSchema(c.env.DB)
  const { page, limit } = parsePageLimit(c)
  const { quotes, total } = await getQuotes(c.env.DB, page, limit)
  return c.json({ quotes, total })
})

app.post('/quotes', async (c) => {
  await ensureSchema(c.env.DB)
  const { book_id, author_name, text } = await c.req.json()
  const id = await createQuote(c.env.DB, book_id, author_name, text)
  return c.json({ id })
})

app.get('/books/:id/quotes', async (c) => {
  await ensureSchema(c.env.DB)
  const bookId = parseInt(c.req.param('id'), 10)
  const quotes = await getBookQuotes(c.env.DB, bookId)
  return c.json({ quotes })
})

// --- Stats ---

app.get('/stats', async (c) => {
  await ensureSchema(c.env.DB)
  const stats = await getStats(c.env.DB)
  return c.json(stats)
})

app.get('/stats/:year', async (c) => {
  await ensureSchema(c.env.DB)
  const year = parseInt(c.req.param('year'), 10)
  const stats = await getStats(c.env.DB, year)
  return c.json(stats)
})

// --- Feed ---

app.get('/feed', async (c) => {
  await ensureSchema(c.env.DB)
  let limit = parseInt(c.req.query('limit') || String(DEFAULT_LIMIT), 10) || DEFAULT_LIMIT
  limit = Math.min(Math.max(1, limit), MAX_LIMIT)
  const feed = await getFeed(c.env.DB, limit)
  return c.json({ feed })
})

// --- Notes ---

app.get('/books/:id/notes', async (c) => {
  await ensureSchema(c.env.DB)
  const bookId = parseInt(c.req.param('id'), 10)
  const note = await getNote(c.env.DB, bookId)
  if (!note) return c.json({ error: 'Note not found' }, 404)
  return c.json({ note })
})

app.post('/books/:id/notes', async (c) => {
  await ensureSchema(c.env.DB)
  const bookId = parseInt(c.req.param('id'), 10)
  const { text } = await c.req.json()
  await upsertNote(c.env.DB, bookId, text)
  return c.json({ ok: true })
})

app.delete('/books/:id/notes', async (c) => {
  await ensureSchema(c.env.DB)
  const bookId = parseInt(c.req.param('id'), 10)
  await deleteNote(c.env.DB, bookId)
  return c.json({ ok: true })
})

// --- Import/Export ---

app.get('/export/csv', async (c) => {
  await ensureSchema(c.env.DB)
  const readShelf = await getShelfBySlug(c.env.DB, 'read')
  if (!readShelf) return c.text('Title,Author,ISBN13,Rating,Date Read,Shelves\n')

  const { books } = await getShelfBooks(c.env.DB, readShelf.id, 'date_read', 1, MAX_LIMIT)
  const lines = ['Title,Author,ISBN13,Rating,Date Read,Shelves']
  for (const sb of books) {
    const b = sb.book
    if (!b) continue
    const rating = await getUserRating(c.env.DB, b.id)
    const title = `"${(b.title || '').replace(/"/g, '""')}"`
    const author = `"${(b.author_names || '').replace(/"/g, '""')}"`
    lines.push(`${title},${author},${b.isbn13 || ''},${rating || ''},${sb.date_read || ''},read`)
  }
  c.header('Content-Type', 'text/csv')
  c.header('Content-Disposition', 'attachment; filename="books.csv"')
  return c.text(lines.join('\n'))
})

app.post('/import/csv', async (c) => {
  await ensureSchema(c.env.DB)
  const text = await c.req.text()
  const lines = text.split('\n').map(l => l.trim()).filter(l => l.length > 0)
  if (lines.length < 2) return c.json({ imported: 0, errors: [] })

  // Parse header to find column indices
  const header = parseCSVRow(lines[0])
  const colIndex = (name: string) => header.findIndex(h => h.toLowerCase().trim() === name.toLowerCase())
  const iTitle = colIndex('title')
  const iAuthor = colIndex('author')
  const iISBN = colIndex('isbn13')
  const iRating = colIndex('rating')
  const iDateRead = colIndex('date read')
  const iShelves = colIndex('shelves')

  let imported = 0
  const errors: string[] = []

  // Get default shelves for mapping
  const readShelf = await getShelfBySlug(c.env.DB, 'read')
  const wantShelf = await getShelfBySlug(c.env.DB, 'want-to-read')
  const currentShelf = await getShelfBySlug(c.env.DB, 'currently-reading')

  for (let i = 1; i < lines.length; i++) {
    try {
      const cols = parseCSVRow(lines[i])
      const title = iTitle >= 0 ? cols[iTitle] : ''
      const author = iAuthor >= 0 ? cols[iAuthor] : ''
      const isbn13 = iISBN >= 0 ? cols[iISBN] : ''
      const ratingStr = iRating >= 0 ? cols[iRating] : ''
      const dateRead = iDateRead >= 0 ? cols[iDateRead] : ''
      const shelvesStr = iShelves >= 0 ? cols[iShelves] : ''

      if (!title) { errors.push(`Row ${i + 1}: missing title`); continue }

      // Find or create book
      let bookId: number | undefined
      if (isbn13) {
        const { books } = await searchBooks(c.env.DB, isbn13, 1, 1)
        if (books.length > 0) bookId = books[0].id
      }
      if (!bookId) {
        const { books } = await searchBooks(c.env.DB, title, 1, 1)
        if (books.length > 0) bookId = books[0].id
      }
      if (!bookId) {
        bookId = await createBook(c.env.DB, { title, author_names: author, isbn13 })
      }

      // Add to shelf
      const shelfName = (shelvesStr || 'read').trim().toLowerCase()
      let targetShelf = readShelf
      if (shelfName.includes('want') || shelfName.includes('to-read')) targetShelf = wantShelf
      else if (shelfName.includes('current')) targetShelf = currentShelf
      if (targetShelf) await addBookToShelf(c.env.DB, targetShelf.id, bookId)

      // Create review if rating
      const rating = parseInt(ratingStr, 10)
      if (rating > 0 && rating <= 5) {
        await createReview(c.env.DB, { book_id: bookId, rating, source: 'import' })
      }

      imported++
    } catch (err: any) {
      errors.push(`Row ${i + 1}: ${err.message || 'unknown error'}`)
    }
  }

  return c.json({ imported, errors })
})

// Simple CSV row parser handling quoted fields
function parseCSVRow(row: string): string[] {
  const result: string[] = []
  let current = ''
  let inQuotes = false
  for (let i = 0; i < row.length; i++) {
    const ch = row[i]
    if (inQuotes) {
      if (ch === '"') {
        if (i + 1 < row.length && row[i + 1] === '"') {
          current += '"'
          i++
        } else {
          inQuotes = false
        }
      } else {
        current += ch
      }
    } else {
      if (ch === '"') {
        inQuotes = true
      } else if (ch === ',') {
        result.push(current.trim())
        current = ''
      } else {
        current += ch
      }
    }
  }
  result.push(current.trim())
  return result
}

// --- Open Library ---

app.get('/ol/search', async (c) => {
  await ensureSchema(c.env.DB)
  const q = c.req.query('q') || ''
  const { docs, numFound } = await searchOL(c.env.KV, q)
  const results = docs.map(olResultToBook)
  return c.json({ results, total: numFound })
})

app.post('/ol/import', async (c) => {
  await ensureSchema(c.env.DB)
  const { ol_key } = await c.req.json()

  // Check if already imported
  const existing = await getBookByOLKey(c.env.DB, ol_key)
  if (existing) return c.json({ id: existing.id, book: existing })

  // Search OL for the work
  const { docs } = await searchOL(c.env.KV, ol_key, 1)
  let bookData = docs.length > 0 ? olResultToBook(docs[0]) : { ol_key }

  // Enrich with work details
  const enriched = await enrichBookFromOL(c.env.KV, ol_key)
  bookData = { ...bookData, ...enriched, ol_key }

  const id = await createBook(c.env.DB, bookData)
  const book = await getBook(c.env.DB, id)
  return c.json({ id, book })
})

export default app
