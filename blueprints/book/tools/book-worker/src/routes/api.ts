import { Hono } from 'hono'
import { cors } from 'hono/cors'
import type { HonoEnv } from '../types'
import type { OLSearchResult } from '../types'
import {
  getBook as grGetBook, getAuthor as grGetAuthor, getList as grGetList,
  getPopularLists, parseGoodreadsURL, parseGoodreadsAuthorURL, parseGoodreadsListURL,
  type GoodreadsBook, type GoodreadsAuthor,
} from '../goodreads'
import { searchOL } from '../openlibrary'
import { DEFAULT_LIMIT, MAX_LIMIT, coverURL } from '../config'
import * as db from '../db'
import {
  ENRICH, enrichAuthor, importAuthorWorks, enrichBook, enrichGenre,
  discoverLists, importListBooks, importSeriesBooks, importEditions,
  importSimilarBooks, normalizeGenre,
} from '../enrich'

const app = new Hono<HonoEnv>()
app.use('*', cors())

// ---- Helpers ----

function parsePageLimit(c: { req: { query: (k: string) => string | undefined } }): { page: number; limit: number } {
  const page = Math.max(1, parseInt(c.req.query('page') || '1', 10) || 1)
  let limit = parseInt(c.req.query('limit') || String(DEFAULT_LIMIT), 10) || DEFAULT_LIMIT
  limit = Math.min(Math.max(1, limit), MAX_LIMIT)
  return { page, limit }
}

function extractAuthorID(url: string): string {
  if (!url) return ''
  const m = url.match(/\/author\/show\/(\d+)/)
  return m ? m[1] : ''
}

function grBookToData(gr: GoodreadsBook): Record<string, unknown> {
  return {
    title: gr.title,
    original_title: gr.original_title,
    description: gr.description,
    author_names: gr.author_name || '',
    cover_url: gr.cover_url,
    isbn10: gr.isbn,
    isbn13: gr.isbn13,
    publisher: gr.publisher,
    publish_date: gr.publish_date,
    publish_year: parseInt(gr.publish_date, 10) || 0,
    page_count: gr.page_count,
    language: gr.language,
    edition_language: gr.edition_language,
    format: gr.format,
    subjects: gr.genres,
    characters: gr.characters,
    settings: gr.settings,
    literary_awards: gr.literary_awards,
    series: gr.series,
    editions_count: gr.edition_count,
    average_rating: gr.average_rating,
    ratings_count: gr.ratings_count,
    reviews_count: gr.reviews_count,
    currently_reading: gr.currently_reading,
    want_to_read: gr.want_to_read,
    rating_dist: gr.rating_dist,
    source_id: gr.goodreads_id,
    source_url: gr.url,
    asin: gr.asin,
    first_published: gr.first_published,
  }
}

function grAuthorToData(gr: GoodreadsAuthor): Record<string, unknown> {
  return {
    name: gr.name,
    bio: gr.bio,
    photo_url: gr.photo_url,
    birth_date: gr.born_date,
    death_date: gr.died_date,
    works_count: gr.works_count,
    followers: gr.followers,
    genres: gr.genres,
    influences: gr.influences,
    website: gr.website,
    source_id: gr.goodreads_id,
  }
}

function olToBook(r: OLSearchResult): Record<string, unknown> {
  return {
    id: 0,
    ol_key: r.key,
    title: r.title,
    author_names: r.author_name?.join(', ') || '',
    authors: (r.author_name || []).map(n => ({ id: 0, ol_key: '', name: n })),
    cover_url: r.cover_i ? coverURL(r.cover_i, 'M') : '',
    publish_year: r.first_publish_year || 0,
    subjects: r.subject?.slice(0, 10) || [],
    publisher: r.publisher?.[0] || '',
    language: r.language?.[0] || '',
    average_rating: r.ratings_average || 0,
    ratings_count: r.ratings_count || 0,
    editions_count: r.edition_count || 0,
    isbn13: r.isbn?.[0] || '',
    page_count: 0,
    reviews_count: 0,
    currently_reading: 0,
    want_to_read: 0,
    rating_dist: [],
  }
}

// Import a Goodreads book into D1
async function importGoodreadsBook(d1: D1Database, kv: KVNamespace, grId: string): Promise<Record<string, unknown> | null> {
  // Check if already imported
  const existing = await db.getBookBySourceId(d1, grId)
  if (existing) return db.getBook(d1, existing.id as number)

  const gr = await grGetBook(kv, grId)
  if (!gr) return null

  const book = await db.createBook(d1, grBookToData(gr))
  const bookId = book.id as number

  // Create/link author
  if (gr.author_name) {
    const authorSourceId = extractAuthorID(gr.author_url)
    const author = await db.getOrCreateAuthor(d1, gr.author_name, authorSourceId)
    await db.linkBookAuthor(d1, bookId, author.id as number)

    // If we have author source ID, try to enrich
    if (authorSourceId) {
      const grAuthor = await grGetAuthor(kv, authorSourceId).catch(() => null)
      if (grAuthor) {
        await db.updateAuthor(d1, author.id as number, grAuthorToData(grAuthor))
      }
    }
  }

  // Import reviews
  for (const r of gr.reviews.slice(0, 30)) {
    await db.createReview(d1, bookId, {
      rating: r.rating,
      text: r.text,
      reviewer_name: r.reviewer_name,
      is_spoiler: r.is_spoiler,
      likes_count: r.likes_count,
      source: 'imported',
    })
  }

  // Import quotes
  for (const q of gr.quotes.slice(0, 30)) {
    await db.createQuote(d1, { book_id: bookId, author_name: q.author_name, text: q.text, likes_count: q.likes_count })
  }

  // Add feed item
  await db.addFeedItem(d1, 'book_added', bookId, gr.title)

  return db.getBook(d1, bookId)
}

// ---- Books ----

app.get('/books/search', async (c) => {
  const q = c.req.query('q') || ''
  const { page, limit } = parsePageLimit(c)
  if (!q) return c.json({ books: [], total_count: 0, page, page_size: limit })

  // Search local DB first
  const local = await db.searchBooks(c.env.DB, q, page, limit)
  if (local.total_count > 0) return c.json(local)

  // Check KV flag: have we already imported this search query?
  const searchKey = `enriched:search:${q.toLowerCase().trim()}`
  const alreadyImported = await c.env.KV.get(searchKey)
  if (alreadyImported) {
    return c.json({ books: [], total_count: 0, page, page_size: limit })
  }

  // Fallback to OpenLibrary — auto-import results so they have real IDs
  const ol = await searchOL(c.env.KV, q, limit, (page - 1) * limit).catch(() => ({ docs: [], numFound: 0 }))
  const books: Record<string, unknown>[] = []
  for (const doc of ol.docs) {
    const olKey = doc.key || ''
    if (!olKey) { books.push(olToBook(doc)); continue }
    // Check if already imported
    const existing = await db.getBookByOLKey(c.env.DB, olKey)
    if (existing) { books.push(existing); continue }
    // Auto-import into D1
    const data = olToBook(doc)
    const authorNames = (doc.author_name || []) as string[]
    delete data.id
    delete data.authors
    const created = await db.createBook(c.env.DB, data)
    const bookId = created.id as number
    // Create and link authors
    for (const name of authorNames) {
      const author = await db.getOrCreateAuthor(c.env.DB, name)
      await db.linkBookAuthor(c.env.DB, bookId, author.id as number)
    }
    // Re-fetch to include linked authors
    const full = await db.getBook(c.env.DB, bookId)
    books.push(full || created)
  }
  // Mark this search as imported
  await c.env.KV.put(searchKey, String(books.length))
  return c.json({ books, total_count: ol.numFound, page, page_size: limit })
})

app.get('/books/trending', async (c) => {
  const limit = parseInt(c.req.query('limit') || '20', 10) || 20
  return c.json(await db.getTrendingBooks(c.env.DB, limit))
})

app.get('/books/:id', async (c) => {
  const id = parseInt(c.req.param('id'), 10)
  if (!id) return c.json({ error: 'Invalid ID' }, 400)
  let book = await db.getBook(c.env.DB, id)
  if (!book) return c.json({ error: 'Book not found' }, 404)
  // Auto-link authors if missing (backfill for pre-fix imports)
  const authors = book.authors as unknown[]
  if ((!authors || authors.length === 0) && book.author_names) {
    const names = (book.author_names as string).split(',').map(n => n.trim()).filter(Boolean)
    for (const name of names) {
      const author = await db.getOrCreateAuthor(c.env.DB, name)
      await db.linkBookAuthor(c.env.DB, id, author.id as number)
    }
    book = await db.getBook(c.env.DB, id)
    if (!book) return c.json({ error: 'Book not found' }, 404)
  }
  // Auto-enrich: core data, editions, series (flag-based, idempotent)
  if (!db.hasEnriched(book.enriched as number, ENRICH.BOOK_CORE)) {
    const enriched = await enrichBook(c.env.DB, c.env.KV, id).catch(() => null)
    if (enriched) book = enriched
  }
  if (!db.hasEnriched(book.enriched as number, ENRICH.BOOK_EDITIONS)) {
    await importEditions(c.env.DB, c.env.KV, id).catch(() => 0)
    book = (await db.getBook(c.env.DB, id)) || book
  }
  if (!db.hasEnriched(book.enriched as number, ENRICH.BOOK_SERIES)) {
    await importSeriesBooks(c.env.DB, c.env.KV, id).catch(() => 0)
  }
  return c.json(book)
})

app.post('/books', async (c) => {
  const data = await c.req.json<Record<string, unknown>>()
  if (!data.title) return c.json({ error: 'Title is required' }, 400)
  const book = await db.createBook(c.env.DB, data)
  // Link authors if provided
  if (data.author_names) {
    const names = (data.author_names as string).split(',').map(n => n.trim()).filter(Boolean)
    for (const name of names) {
      const author = await db.getOrCreateAuthor(c.env.DB, name)
      await db.linkBookAuthor(c.env.DB, book.id as number, author.id as number)
    }
  }
  await db.addFeedItem(c.env.DB, 'book_added', book.id as number, data.title as string)
  return c.json(book, 201)
})

app.get('/books/:id/similar', async (c) => {
  const id = parseInt(c.req.param('id'), 10)
  const limit = parseInt(c.req.query('limit') || '10', 10) || 10
  // Auto-import similar books if not already done
  const book = await db.getBook(c.env.DB, id)
  if (book && !db.hasEnriched(book.enriched as number, ENRICH.BOOK_SIMILAR)) {
    await importSimilarBooks(c.env.DB, c.env.KV, id).catch(() => 0)
  }
  return c.json(await db.getSimilarBooks(c.env.DB, id, limit))
})

// ---- Book Reviews ----

app.get('/books/:id/reviews', async (c) => {
  const bookId = parseInt(c.req.param('id'), 10)
  const { page, limit } = parsePageLimit(c)
  const filters: Record<string, unknown> = { page, limit }
  const sort = c.req.query('sort')
  const rating = c.req.query('rating')
  const source = c.req.query('source')
  const has_text = c.req.query('has_text')
  const q = c.req.query('q')
  const include_spoilers = c.req.query('include_spoilers')
  if (sort) filters.sort = sort
  if (rating) filters.rating = parseInt(rating, 10)
  if (source) filters.source = source
  if (has_text) filters.has_text = has_text
  if (q) filters.q = q
  if (include_spoilers) filters.include_spoilers = include_spoilers
  return c.json(await db.getBookReviews(c.env.DB, bookId, filters))
})

app.post('/books/:id/reviews', async (c) => {
  const bookId = parseInt(c.req.param('id'), 10)
  const data = await c.req.json<Record<string, unknown>>()
  const review = await db.createReview(c.env.DB, bookId, data)
  // Add to shelf if specified
  if (data.shelf_id) {
    await db.addBookToShelf(c.env.DB, data.shelf_id as number, bookId)
  }
  const book = await db.getBook(c.env.DB, bookId)
  await db.addFeedItem(c.env.DB, 'review_added', bookId, (book?.title as string) || '', JSON.stringify({ rating: data.rating }))
  return c.json(review, 201)
})

// ---- Book Quotes ----

app.get('/books/:id/quotes', async (c) => {
  const bookId = parseInt(c.req.param('id'), 10)
  return c.json(await db.getBookQuotes(c.env.DB, bookId))
})

// ---- Book Progress ----

app.get('/books/:id/progress', async (c) => {
  const bookId = parseInt(c.req.param('id'), 10)
  return c.json(await db.getProgress(c.env.DB, bookId))
})

app.post('/books/:id/progress', async (c) => {
  const bookId = parseInt(c.req.param('id'), 10)
  const data = await c.req.json<Record<string, unknown>>()
  const progress = await db.createProgress(c.env.DB, bookId, data)
  const book = await db.getBook(c.env.DB, bookId)
  await db.addFeedItem(c.env.DB, 'progress_update', bookId, (book?.title as string) || '', JSON.stringify({ page: data.page }))
  return c.json(progress, 201)
})

// ---- Book Notes ----

app.get('/books/:id/notes', async (c) => {
  const bookId = parseInt(c.req.param('id'), 10)
  const note = await db.getNote(c.env.DB, bookId)
  if (!note) return c.json({ error: 'Note not found' }, 404)
  return c.json(note)
})

app.post('/books/:id/notes', async (c) => {
  const bookId = parseInt(c.req.param('id'), 10)
  const data = await c.req.json<{ text: string }>()
  return c.json(await db.createOrUpdateNote(c.env.DB, bookId, data.text))
})

app.delete('/books/:id/notes', async (c) => {
  const bookId = parseInt(c.req.param('id'), 10)
  await db.deleteNote(c.env.DB, bookId)
  return c.json({ ok: true })
})

// ---- Book Enrich ----

app.post('/books/:id/enrich', async (c) => {
  const id = parseInt(c.req.param('id'), 10)
  const book = await db.getBook(c.env.DB, id)
  if (!book) return c.json({ error: 'Book not found' }, 404)

  const body = await c.req.json<{ force?: boolean }>().catch(() => ({ force: true }))
  const enriched = await enrichBook(c.env.DB, c.env.KV, id, body.force !== false)
  if (!enriched) return c.json({ error: 'Could not enrich book' }, 502)
  return c.json(enriched)
})

// ---- Authors ----

app.get('/authors/search', async (c) => {
  const q = c.req.query('q') || ''
  if (!q) return c.json([])
  const local = await db.searchAuthors(c.env.DB, q)
  if (local.length > 0) return c.json(local)
  // Fallback to OL
  const ol = await searchOL(c.env.KV, q, 10).catch(() => ({ docs: [], numFound: 0 }))
  return c.json(ol.docs.map(d => ({
    id: 0,
    name: d.author_name?.join(', ') || d.title,
    ol_key: d.key,
    works_count: 0,
    followers: 0,
  })))
})

app.get('/authors/:id', async (c) => {
  const id = parseInt(c.req.param('id'), 10)
  let author = await db.getAuthor(c.env.DB, id)
  if (!author) return c.json({ error: 'Author not found' }, 404)
  // Auto-enrich author (flag-based, idempotent)
  if (!db.hasEnriched(author.enriched as number, ENRICH.AUTHOR_CORE)) {
    const enriched = await enrichAuthor(c.env.DB, c.env.KV, id).catch(() => null)
    if (enriched) author = enriched
  }
  return c.json(author)
})

app.get('/authors/:id/books', async (c) => {
  const id = parseInt(c.req.param('id'), 10)
  // Ensure author is enriched first (for ol_key)
  const author = await db.getAuthor(c.env.DB, id)
  if (author && !db.hasEnriched(author.enriched as number, ENRICH.AUTHOR_CORE)) {
    await enrichAuthor(c.env.DB, c.env.KV, id).catch(() => null)
  }
  // Auto-import works (flag-based, idempotent)
  const authorRefresh = await db.getAuthor(c.env.DB, id)
  if (authorRefresh && !db.hasEnriched(authorRefresh.enriched as number, ENRICH.AUTHOR_WORKS)) {
    await importAuthorWorks(c.env.DB, c.env.KV, id).catch(() => 0)
  }
  const books = await db.getBooksByAuthor(c.env.DB, id)
  return c.json(books)
})

// ---- Reviews ----

app.put('/reviews/:id', async (c) => {
  const id = parseInt(c.req.param('id'), 10)
  const data = await c.req.json<Record<string, unknown>>()
  const review = await db.updateReview(c.env.DB, id, data)
  if (!review) return c.json({ error: 'Review not found' }, 404)
  return c.json(review)
})

app.delete('/reviews/:id', async (c) => {
  const id = parseInt(c.req.param('id'), 10)
  await db.deleteReview(c.env.DB, id)
  return c.json({ ok: true })
})

app.post('/reviews/:id/like', async (c) => {
  const id = parseInt(c.req.param('id'), 10)
  const likes = await db.likeReview(c.env.DB, id)
  return c.json({ likes_count: likes })
})

app.get('/reviews/:id/comments', async (c) => {
  const reviewId = parseInt(c.req.param('id'), 10)
  const { page, limit } = parsePageLimit(c)
  return c.json(await db.getComments(c.env.DB, reviewId, page, limit))
})

app.post('/reviews/:id/comments', async (c) => {
  const reviewId = parseInt(c.req.param('id'), 10)
  const data = await c.req.json<Record<string, unknown>>()
  return c.json(await db.createComment(c.env.DB, reviewId, data), 201)
})

app.delete('/reviews/:id/comments/:commentId', async (c) => {
  const reviewId = parseInt(c.req.param('id'), 10)
  const commentId = parseInt(c.req.param('commentId'), 10)
  await db.deleteComment(c.env.DB, reviewId, commentId)
  return c.json({ ok: true })
})

// ---- Shelves ----

app.get('/shelves', async (c) => {
  return c.json(await db.getShelves(c.env.DB))
})

app.post('/shelves', async (c) => {
  const data = await c.req.json<{ name: string }>()
  if (!data.name) return c.json({ error: 'Name is required' }, 400)
  const slug = data.name.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '')
  return c.json(await db.createShelf(c.env.DB, data.name, slug), 201)
})

app.put('/shelves/:id', async (c) => {
  const id = parseInt(c.req.param('id'), 10)
  const data = await c.req.json<{ name: string }>()
  const shelf = await db.updateShelf(c.env.DB, id, data.name)
  if (!shelf) return c.json({ error: 'Cannot update default shelf' }, 400)
  return c.json(shelf)
})

app.delete('/shelves/:id', async (c) => {
  const id = parseInt(c.req.param('id'), 10)
  const ok = await db.deleteShelf(c.env.DB, id)
  if (!ok) return c.json({ error: 'Cannot delete default shelf' }, 400)
  return c.json({ ok: true })
})

app.get('/shelves/:id/books', async (c) => {
  const id = parseInt(c.req.param('id'), 10)
  const { page, limit } = parsePageLimit(c)
  return c.json(await db.getShelfBooks(c.env.DB, id, page, limit))
})

app.post('/shelves/:id/books', async (c) => {
  const shelfId = parseInt(c.req.param('id'), 10)
  const data = await c.req.json<{ book_id: number }>()
  await db.addBookToShelf(c.env.DB, shelfId, data.book_id)
  const book = await db.getBook(c.env.DB, data.book_id)
  const shelf = await c.env.DB.prepare('SELECT name FROM shelves WHERE id = ?').bind(shelfId).first<{ name: string }>()
  await db.addFeedItem(c.env.DB, 'shelved', data.book_id, (book?.title as string) || '', JSON.stringify({ shelf: shelf?.name }))
  return c.json({ ok: true })
})

app.delete('/shelves/:id/books/:bookId', async (c) => {
  const shelfId = parseInt(c.req.param('id'), 10)
  const bookId = parseInt(c.req.param('bookId'), 10)
  await db.removeBookFromShelf(c.env.DB, shelfId, bookId)
  return c.json({ ok: true })
})

app.put('/shelves/:id/books/:bookId', async (c) => {
  const shelfId = parseInt(c.req.param('id'), 10)
  const bookId = parseInt(c.req.param('bookId'), 10)
  const data = await c.req.json<Record<string, unknown>>()
  await db.updateShelfBook(c.env.DB, shelfId, bookId, data)
  return c.json({ ok: true })
})

// ---- Browse ----

app.get('/genres', async (c) => {
  return c.json(await db.getGenres(c.env.DB, normalizeGenre))
})

app.get('/genres/:genre/books', async (c) => {
  const genre = decodeURIComponent(c.req.param('genre')).replace(/-/g, ' ')
  const { page, limit } = parsePageLimit(c)
  let result = await db.getBooksByGenre(c.env.DB, genre, page, limit)
  // Auto-import from OL if genre not yet enriched (KV flag, idempotent)
  if (page === 1) {
    const genreKey = `enriched:genre:${genre.toLowerCase().replace(/\s+/g, '_')}`
    const alreadyEnriched = await c.env.KV.get(genreKey)
    if (!alreadyEnriched) {
      const imported = await enrichGenre(c.env.DB, c.env.KV, genre).catch(() => 0)
      await c.env.KV.put(genreKey, String(imported))
      if (imported > 0) {
        result = await db.getBooksByGenre(c.env.DB, genre, page, limit)
      }
    }
  }
  return c.json(result)
})

app.get('/browse/new-releases', async (c) => {
  const limit = parseInt(c.req.query('limit') || '20', 10) || 20
  return c.json(await db.getNewReleases(c.env.DB, limit))
})

app.get('/browse/popular', async (c) => {
  const limit = parseInt(c.req.query('limit') || '20', 10) || 20
  return c.json(await db.getPopularBooks(c.env.DB, limit))
})

// ---- Lists ----

app.get('/lists', async (c) => {
  const result = await db.getLists(c.env.DB)
  // Auto-discover popular GR lists (KV flag, idempotent)
  const tag = c.req.query('tag') || ''
  const discoverKey = `enriched:lists:${tag || 'all'}`
  const alreadyDiscovered = await c.env.KV.get(discoverKey)
  if (!alreadyDiscovered) {
    await discoverLists(c.env.DB, c.env.KV, tag).catch(() => [])
    await c.env.KV.put(discoverKey, '1')
    if (result.total === 0) return c.json(await db.getLists(c.env.DB))
  }
  return c.json(result)
})

app.post('/lists', async (c) => {
  const data = await c.req.json<Record<string, unknown>>()
  if (!data.title) return c.json({ error: 'Title is required' }, 400)
  return c.json(await db.createList(c.env.DB, data), 201)
})

app.get('/lists/discover', async (c) => {
  const tag = c.req.query('tag') || ''
  const lists = await discoverLists(c.env.DB, c.env.KV, tag).catch(() => [])
  return c.json({ lists, total: lists.length })
})

app.get('/lists/:id', async (c) => {
  const id = parseInt(c.req.param('id'), 10)
  let list = await db.getList(c.env.DB, id)
  if (!list) return c.json({ error: 'List not found' }, 404)
  // Auto-import books from GR source (flag-based, idempotent)
  if (!db.hasEnriched(list.enriched as number, ENRICH.LIST_BOOKS)) {
    await importListBooks(c.env.DB, c.env.KV, id).catch(() => 0)
    list = (await db.getList(c.env.DB, id)) || list
  }
  return c.json(list)
})

app.post('/lists/:id/books', async (c) => {
  const listId = parseInt(c.req.param('id'), 10)
  const data = await c.req.json<{ book_id: number }>()
  await db.addBookToList(c.env.DB, listId, data.book_id)
  return c.json({ ok: true })
})

app.post('/lists/:id/vote/:bookId', async (c) => {
  const listId = parseInt(c.req.param('id'), 10)
  const bookId = parseInt(c.req.param('bookId'), 10)
  await db.voteOnBook(c.env.DB, listId, bookId)
  return c.json({ ok: true })
})

// ---- Quotes ----

app.get('/quotes', async (c) => {
  const { page, limit } = parsePageLimit(c)
  return c.json(await db.getQuotes(c.env.DB, page, limit))
})

app.post('/quotes', async (c) => {
  const data = await c.req.json<Record<string, unknown>>()
  if (!data.text) return c.json({ error: 'Text is required' }, 400)
  return c.json(await db.createQuote(c.env.DB, data), 201)
})

// ---- Stats ----

app.get('/stats', async (c) => {
  const year = new Date().getFullYear()
  return c.json(await db.getStats(c.env.DB, year))
})

app.get('/stats/:year', async (c) => {
  const year = parseInt(c.req.param('year'), 10) || new Date().getFullYear()
  return c.json(await db.getStats(c.env.DB, year))
})

// ---- Feed ----

app.get('/feed', async (c) => {
  const limit = parseInt(c.req.query('limit') || '20', 10) || 20
  return c.json(await db.getFeed(c.env.DB, limit))
})

// ---- Challenge ----

app.get('/challenge/:year', async (c) => {
  const year = parseInt(c.req.param('year'), 10)
  const challenge = await db.getChallenge(c.env.DB, year)
  if (!challenge) return c.json({ error: 'Challenge not found' }, 404)
  return c.json(challenge)
})

app.post('/challenge', async (c) => {
  const data = await c.req.json<{ year: number; goal: number }>()
  return c.json(await db.createOrUpdateChallenge(c.env.DB, data.year, data.goal))
})

// ---- Source (Goodreads proxy + import) ----

app.get('/source/:id', async (c) => {
  const id = parseGoodreadsURL(c.req.param('id'))
  const gr = await grGetBook(c.env.KV, id).catch(() => null)
  if (!gr) return c.json({ error: 'Book not found' }, 404)
  return c.json({
    ...grBookToData(gr),
    id: parseInt(gr.goodreads_id, 10) || 0,
    authors: gr.author_name ? [{ id: 0, name: gr.author_name, source_id: extractAuthorID(gr.author_url) }] : [],
  })
})

app.get('/source/author/:id', async (c) => {
  const id = parseGoodreadsAuthorURL(c.req.param('id'))
  const gr = await grGetAuthor(c.env.KV, id).catch(() => null)
  if (!gr) return c.json({ error: 'Author not found' }, 404)
  return c.json({ ...grAuthorToData(gr), id: parseInt(gr.goodreads_id, 10) || 0 })
})

app.get('/source/lists', async (c) => {
  const tag = c.req.query('tag') || ''
  const lists = await getPopularLists(c.env.KV, tag).catch(() => [])
  return c.json(lists.map(l => ({
    source_id: l.goodreads_id,
    title: l.title,
    url: l.url,
    book_count: l.book_count,
    voter_count: l.voter_count,
    tag: l.tag,
  })))
})

// ---- Import from external source ----

app.post('/import-source', async (c) => {
  const data = await c.req.json<{ url: string }>()
  if (!data.url) return c.json({ error: 'URL is required' }, 400)
  const grId = parseGoodreadsURL(data.url)
  const book = await importGoodreadsBook(c.env.DB, c.env.KV, grId)
  if (!book) return c.json({ error: 'Could not import book' }, 502)
  return c.json(book, 201)
})

app.post('/import-source-list', async (c) => {
  const data = await c.req.json<{ url: string }>()
  if (!data.url) return c.json({ error: 'URL is required' }, 400)
  const listId = parseGoodreadsListURL(data.url)
  const grList = await grGetList(c.env.KV, listId).catch(() => null)
  if (!grList) return c.json({ error: 'Could not fetch list' }, 502)

  // Create list in D1
  const list = await db.createList(c.env.DB, {
    title: grList.title,
    description: grList.description,
    source_url: data.url,
    voter_count: grList.voter_count,
  })
  const dbListId = list.id as number

  // Import each book
  for (const item of grList.books.slice(0, 100)) {
    if (!item.goodreads_id) continue
    const book = await importGoodreadsBook(c.env.DB, c.env.KV, item.goodreads_id).catch(() => null)
    if (book) {
      await db.addBookToList(c.env.DB, dbListId, book.id as number)
    }
  }

  return c.json(await db.getList(c.env.DB, dbListId), 201)
})

// ---- Import/Export CSV ----

app.post('/import/csv', async (c) => {
  const body = await c.req.parseBody()
  const file = body['file']
  if (!file || typeof file === 'string') return c.json({ error: 'File is required' }, 400)
  const text = await (file as File).text()
  const imported = await db.importCSV(c.env.DB, text)
  return c.json({ imported })
})

app.get('/export/csv', async (c) => {
  const csv = await db.exportCSV(c.env.DB)
  return new Response(csv, {
    headers: {
      'Content-Type': 'text/csv; charset=utf-8',
      'Content-Disposition': 'attachment; filename="book_export.csv"',
    },
  })
})

// ---- OL search ----

app.get('/ol/search', async (c) => {
  const q = c.req.query('q') || ''
  const { docs, numFound } = await searchOL(c.env.KV, q).catch(() => ({ docs: [] as OLSearchResult[], numFound: 0 }))
  return c.json({ results: docs.map(olToBook), total: numFound })
})

// ---- Init DB ----

app.post('/init', async (c) => {
  await db.initDB(c.env.DB)
  return c.json({ ok: true, message: 'Database initialized' })
})

export default app
