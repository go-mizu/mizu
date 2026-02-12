import type { Env, Book, Shelf, ShelfBook, Review, Quote, FeedItem, ReadingProgress, BookList, BookListItem, Author, BookNote, ReadingChallenge, ReviewComment } from './types'
import schema from './schema.sql'

let initialized = false

export async function ensureSchema(db: D1Database): Promise<void> {
  if (initialized) return
  // Split by semicolons and execute each statement
  const statements = schema.split(';').map((s: string) => s.trim()).filter((s: string) => s.length > 0)
  for (const stmt of statements) {
    try {
      await db.prepare(stmt).run()
    } catch {
      // Ignore errors from already-existing tables/indexes
    }
  }
  initialized = true
}

// Helper to parse JSON fields on a book row
export function hydrateBook(row: any): Book {
  if (!row) return row
  const b = row as Book
  try { b.subjects = JSON.parse(b.subjects_json || '[]') } catch { b.subjects = [] }
  try { b.characters = JSON.parse(b.characters_json || '[]') } catch { b.characters = [] }
  try { b.settings = JSON.parse(b.settings_json || '[]') } catch { b.settings = [] }
  try { b.literary_awards = JSON.parse(b.literary_awards_json || '[]') } catch { b.literary_awards = [] }
  try { b.rating_distribution = JSON.parse(b.rating_dist || '[]') } catch { b.rating_distribution = [] }
  return b
}

// ---- Books ----

export async function getBook(db: D1Database, id: number): Promise<Book | null> {
  const row = await db.prepare('SELECT * FROM books WHERE id = ?').bind(id).first()
  return row ? hydrateBook(row) : null
}

export async function searchBooks(db: D1Database, q: string, page: number, limit: number): Promise<{ books: Book[], total: number }> {
  const offset = (page - 1) * limit
  const pattern = `%${q}%`
  const countResult = await db.prepare(
    'SELECT COUNT(*) as cnt FROM books WHERE title LIKE ? OR author_names LIKE ? OR isbn13 LIKE ?'
  ).bind(pattern, pattern, pattern).first<{ cnt: number }>()
  const total = countResult?.cnt || 0

  const { results } = await db.prepare(
    'SELECT * FROM books WHERE title LIKE ? OR author_names LIKE ? OR isbn13 LIKE ? ORDER BY ratings_count DESC LIMIT ? OFFSET ?'
  ).bind(pattern, pattern, pattern, limit, offset).all()

  return { books: (results || []).map(hydrateBook), total }
}

export async function createBook(db: D1Database, book: Partial<Book>): Promise<number> {
  const r = await db.prepare(`INSERT INTO books (title, ol_key, google_id, original_title, subtitle, description, author_names, cover_url, cover_id, isbn10, isbn13, publisher, publish_date, publish_year, page_count, language, edition_language, format, subjects_json, characters_json, settings_json, literary_awards_json, editions_count, average_rating, ratings_count, goodreads_id, goodreads_url, asin, series, reviews_count, currently_reading, want_to_read, rating_dist, first_published)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
    .bind(
      book.title || '', book.ol_key || '', book.google_id || '', book.original_title || '',
      book.subtitle || '', book.description || '', book.author_names || '',
      book.cover_url || '', book.cover_id || 0, book.isbn10 || '', book.isbn13 || '',
      book.publisher || '', book.publish_date || '', book.publish_year || 0,
      book.page_count || 0, book.language || 'en', book.edition_language || '',
      book.format || '', book.subjects_json || '[]', book.characters_json || '[]',
      book.settings_json || '[]', book.literary_awards_json || '[]',
      book.editions_count || 0, book.average_rating || 0, book.ratings_count || 0,
      book.goodreads_id || '', book.goodreads_url || '', book.asin || '',
      book.series || '', book.reviews_count || 0, book.currently_reading || 0,
      book.want_to_read || 0, book.rating_dist || '[]', book.first_published || ''
    ).run()
  return r.meta.last_row_id as number
}

export async function getTrendingBooks(db: D1Database, limit: number): Promise<Book[]> {
  const { results } = await db.prepare(
    'SELECT * FROM books ORDER BY ratings_count DESC, average_rating DESC LIMIT ?'
  ).bind(limit).all()
  return (results || []).map(hydrateBook)
}

export async function getSimilarBooks(db: D1Database, bookId: number, limit: number): Promise<Book[]> {
  const book = await getBook(db, bookId)
  if (!book || !book.subjects || book.subjects.length === 0) return []
  const subject = book.subjects[0]
  const { results } = await db.prepare(
    "SELECT * FROM books WHERE id != ? AND subjects_json LIKE ? ORDER BY ratings_count DESC LIMIT ?"
  ).bind(bookId, `%${subject}%`, limit).all()
  return (results || []).map(hydrateBook)
}

export async function getBookByOLKey(db: D1Database, olKey: string): Promise<Book | null> {
  const row = await db.prepare('SELECT * FROM books WHERE ol_key = ?').bind(olKey).first()
  return row ? hydrateBook(row) : null
}

// ---- Authors ----

export async function getAuthor(db: D1Database, id: number): Promise<Author | null> {
  return await db.prepare('SELECT * FROM authors WHERE id = ?').bind(id).first() as Author | null
}

export async function searchAuthors(db: D1Database, q: string): Promise<Author[]> {
  const { results } = await db.prepare(
    'SELECT * FROM authors WHERE name LIKE ? ORDER BY works_count DESC LIMIT 20'
  ).bind(`%${q}%`).all()
  return (results || []) as unknown as Author[]
}

export async function getAuthorBooks(db: D1Database, authorId: number, page: number, limit: number): Promise<{ books: Book[], total: number }> {
  const author = await getAuthor(db, authorId)
  if (!author) return { books: [], total: 0 }
  const pattern = `%${author.name}%`
  const offset = (page - 1) * limit
  const countResult = await db.prepare('SELECT COUNT(*) as cnt FROM books WHERE author_names LIKE ?').bind(pattern).first<{ cnt: number }>()
  const { results } = await db.prepare(
    'SELECT * FROM books WHERE author_names LIKE ? ORDER BY publish_year DESC LIMIT ? OFFSET ?'
  ).bind(pattern, limit, offset).all()
  return { books: (results || []).map(hydrateBook), total: countResult?.cnt || 0 }
}

export async function createAuthor(db: D1Database, author: Partial<Author>): Promise<number> {
  const r = await db.prepare(
    `INSERT INTO authors (ol_key, name, bio, photo_url, birth_date, death_date, works_count, goodreads_id, followers, genres, influences, website)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
  ).bind(
    author.ol_key || '', author.name || '', author.bio || '', author.photo_url || '',
    author.birth_date || '', author.death_date || '', author.works_count || 0,
    author.goodreads_id || '', author.followers || 0, author.genres || '',
    author.influences || '', author.website || ''
  ).run()
  return r.meta.last_row_id as number
}

// ---- Shelves ----

export async function getShelves(db: D1Database): Promise<Shelf[]> {
  const { results } = await db.prepare(
    `SELECT s.*, (SELECT COUNT(*) FROM shelf_books WHERE shelf_id = s.id) as book_count
     FROM shelves s ORDER BY s.sort_order`
  ).all()
  return (results || []) as unknown as Shelf[]
}

export async function getShelfBySlug(db: D1Database, slug: string): Promise<Shelf | null> {
  return await db.prepare(
    `SELECT s.*, (SELECT COUNT(*) FROM shelf_books WHERE shelf_id = s.id) as book_count
     FROM shelves s WHERE s.slug = ?`
  ).bind(slug).first() as Shelf | null
}

export async function getShelfBooks(db: D1Database, shelfId: number, sort: string, page: number, limit: number): Promise<{ books: ShelfBook[], total: number }> {
  const offset = (page - 1) * limit
  const countResult = await db.prepare('SELECT COUNT(*) as cnt FROM shelf_books WHERE shelf_id = ?').bind(shelfId).first<{ cnt: number }>()
  let orderBy = 'sb.date_added DESC'
  if (sort === 'title') orderBy = 'b.title ASC'
  else if (sort === 'rating') orderBy = 'b.average_rating DESC'
  else if (sort === 'date_read') orderBy = 'sb.date_read DESC'

  const { results } = await db.prepare(
    `SELECT sb.*, b.id as bid, b.title, b.author_names, b.cover_url, b.cover_id, b.average_rating, b.ratings_count, b.page_count, b.publish_year, b.subjects_json
     FROM shelf_books sb JOIN books b ON sb.book_id = b.id
     WHERE sb.shelf_id = ? ORDER BY ${orderBy} LIMIT ? OFFSET ?`
  ).bind(shelfId, limit, offset).all()

  const books: ShelfBook[] = (results || []).map((r: any) => ({
    id: r.id,
    shelf_id: r.shelf_id,
    book_id: r.book_id,
    date_added: r.date_added,
    position: r.position,
    date_started: r.date_started,
    date_read: r.date_read,
    read_count: r.read_count,
    book: hydrateBook({
      id: r.bid, title: r.title, author_names: r.author_names,
      cover_url: r.cover_url, cover_id: r.cover_id,
      average_rating: r.average_rating, ratings_count: r.ratings_count,
      page_count: r.page_count, publish_year: r.publish_year,
      subjects_json: r.subjects_json,
    })
  }))

  return { books, total: countResult?.cnt || 0 }
}

export async function addBookToShelf(db: D1Database, shelfId: number, bookId: number): Promise<void> {
  // If shelf is exclusive, remove from other exclusive shelves first
  const shelf = await db.prepare('SELECT * FROM shelves WHERE id = ?').bind(shelfId).first<Shelf>()
  if (shelf?.is_exclusive) {
    const exclusiveShelves = await db.prepare('SELECT id FROM shelves WHERE is_exclusive = 1').all()
    for (const s of exclusiveShelves.results || []) {
      await db.prepare('DELETE FROM shelf_books WHERE shelf_id = ? AND book_id = ?').bind((s as any).id, bookId).run()
    }
  }
  await db.prepare(
    'INSERT OR REPLACE INTO shelf_books (shelf_id, book_id) VALUES (?, ?)'
  ).bind(shelfId, bookId).run()
}

export async function removeBookFromShelf(db: D1Database, shelfId: number, bookId: number): Promise<void> {
  await db.prepare('DELETE FROM shelf_books WHERE shelf_id = ? AND book_id = ?').bind(shelfId, bookId).run()
}

export async function getUserShelf(db: D1Database, bookId: number): Promise<string> {
  const row = await db.prepare(
    `SELECT s.slug FROM shelf_books sb JOIN shelves s ON sb.shelf_id = s.id WHERE sb.book_id = ? AND s.is_exclusive = 1 LIMIT 1`
  ).bind(bookId).first<{ slug: string }>()
  return row?.slug || ''
}

export async function createShelf(db: D1Database, name: string, slug: string, isExclusive: boolean): Promise<number> {
  const maxOrder = await db.prepare('SELECT MAX(sort_order) as m FROM shelves').first<{ m: number }>()
  const r = await db.prepare(
    'INSERT INTO shelves (name, slug, is_exclusive, is_default, sort_order) VALUES (?, ?, ?, 0, ?)'
  ).bind(name, slug, isExclusive ? 1 : 0, (maxOrder?.m || 0) + 1).run()
  return r.meta.last_row_id as number
}

export async function deleteShelf(db: D1Database, id: number): Promise<boolean> {
  const shelf = await db.prepare('SELECT * FROM shelves WHERE id = ?').bind(id).first<Shelf>()
  if (!shelf || shelf.is_default) return false
  await db.prepare('DELETE FROM shelf_books WHERE shelf_id = ?').bind(id).run()
  await db.prepare('DELETE FROM shelves WHERE id = ?').bind(id).run()
  return true
}

// ---- Reviews ----

export async function getBookReviews(db: D1Database, bookId: number, sort: string, page: number, limit: number): Promise<{ reviews: Review[], total: number }> {
  const offset = (page - 1) * limit
  const countResult = await db.prepare('SELECT COUNT(*) as cnt FROM reviews WHERE book_id = ?').bind(bookId).first<{ cnt: number }>()
  let orderBy = 'likes_count DESC, created_at DESC'
  if (sort === 'newest') orderBy = 'created_at DESC'
  else if (sort === 'oldest') orderBy = 'created_at ASC'
  else if (sort === 'rating_desc') orderBy = 'rating DESC'
  else if (sort === 'rating_asc') orderBy = 'rating ASC'

  const { results } = await db.prepare(
    `SELECT * FROM reviews WHERE book_id = ? ORDER BY ${orderBy} LIMIT ? OFFSET ?`
  ).bind(bookId, limit, offset).all()
  return { reviews: (results || []) as unknown as Review[], total: countResult?.cnt || 0 }
}

export async function createReview(db: D1Database, review: Partial<Review>): Promise<number> {
  const r = await db.prepare(
    `INSERT INTO reviews (book_id, rating, text, is_spoiler, reviewer_name, source, started_at, finished_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
  ).bind(
    review.book_id, review.rating || 0, review.text || '', review.is_spoiler || 0,
    review.reviewer_name || '', review.source || 'user',
    review.started_at || null, review.finished_at || null
  ).run()
  return r.meta.last_row_id as number
}

export async function getUserRating(db: D1Database, bookId: number): Promise<number> {
  const row = await db.prepare(
    "SELECT rating FROM reviews WHERE book_id = ? AND source = 'user' ORDER BY created_at DESC LIMIT 1"
  ).bind(bookId).first<{ rating: number }>()
  return row?.rating || 0
}

export async function deleteReview(db: D1Database, id: number): Promise<void> {
  await db.prepare('DELETE FROM review_comments WHERE review_id = ?').bind(id).run()
  await db.prepare('DELETE FROM reviews WHERE id = ?').bind(id).run()
}

export async function likeReview(db: D1Database, id: number): Promise<void> {
  await db.prepare('UPDATE reviews SET likes_count = likes_count + 1 WHERE id = ?').bind(id).run()
}

export async function getReviewComments(db: D1Database, reviewId: number): Promise<ReviewComment[]> {
  const { results } = await db.prepare(
    'SELECT * FROM review_comments WHERE review_id = ? ORDER BY created_at ASC'
  ).bind(reviewId).all()
  return (results || []) as unknown as ReviewComment[]
}

export async function createComment(db: D1Database, reviewId: number, authorName: string, text: string): Promise<number> {
  const r = await db.prepare(
    'INSERT INTO review_comments (review_id, author_name, text) VALUES (?, ?, ?)'
  ).bind(reviewId, authorName, text).run()
  await db.prepare('UPDATE reviews SET comments_count = comments_count + 1 WHERE id = ?').bind(reviewId).run()
  return r.meta.last_row_id as number
}

// ---- Reading Progress ----

export async function getProgress(db: D1Database, bookId: number): Promise<ReadingProgress[]> {
  const { results } = await db.prepare(
    'SELECT * FROM reading_progress WHERE book_id = ? ORDER BY created_at DESC'
  ).bind(bookId).all()
  return (results || []) as unknown as ReadingProgress[]
}

export async function addProgress(db: D1Database, bookId: number, page: number, percent: number, note: string): Promise<number> {
  const r = await db.prepare(
    'INSERT INTO reading_progress (book_id, page, percent, note) VALUES (?, ?, ?, ?)'
  ).bind(bookId, page, percent, note).run()
  return r.meta.last_row_id as number
}

// ---- Reading Challenge ----

export async function getChallenge(db: D1Database, year: number): Promise<ReadingChallenge | null> {
  const challenge = await db.prepare('SELECT * FROM reading_challenges WHERE year = ?').bind(year).first() as ReadingChallenge | null
  if (challenge) {
    const countResult = await db.prepare(
      `SELECT COUNT(*) as cnt FROM shelf_books sb JOIN shelves s ON sb.shelf_id = s.id
       WHERE s.slug = 'read' AND sb.date_read LIKE ?`
    ).bind(`${year}%`).first<{ cnt: number }>()
    challenge.progress = countResult?.cnt || 0
  }
  return challenge
}

export async function setChallenge(db: D1Database, year: number, goal: number): Promise<void> {
  await db.prepare(
    'INSERT INTO reading_challenges (year, goal) VALUES (?, ?) ON CONFLICT(year) DO UPDATE SET goal = ?'
  ).bind(year, goal, goal).run()
}

// ---- Browse ----

export async function getGenres(db: D1Database): Promise<{ name: string, book_count: number }[]> {
  // Extract unique subjects from all books
  const { results } = await db.prepare(
    "SELECT subjects_json FROM books WHERE subjects_json != '[]' AND subjects_json != '' LIMIT 1000"
  ).all()
  const counts: Record<string, number> = {}
  for (const row of results || []) {
    try {
      const subjects = JSON.parse((row as any).subjects_json || '[]')
      for (const s of subjects.slice(0, 3)) {
        const name = s.trim()
        if (name) counts[name] = (counts[name] || 0) + 1
      }
    } catch { /* ignore */ }
  }
  return Object.entries(counts)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 50)
    .map(([name, book_count]) => ({ name, book_count }))
}

export async function getBooksByGenre(db: D1Database, genre: string, page: number, limit: number): Promise<{ books: Book[], total: number }> {
  const offset = (page - 1) * limit
  const pattern = `%${genre}%`
  const countResult = await db.prepare('SELECT COUNT(*) as cnt FROM books WHERE subjects_json LIKE ?').bind(pattern).first<{ cnt: number }>()
  const { results } = await db.prepare(
    'SELECT * FROM books WHERE subjects_json LIKE ? ORDER BY ratings_count DESC LIMIT ? OFFSET ?'
  ).bind(pattern, limit, offset).all()
  return { books: (results || []).map(hydrateBook), total: countResult?.cnt || 0 }
}

export async function getNewReleases(db: D1Database, limit: number): Promise<Book[]> {
  const { results } = await db.prepare(
    'SELECT * FROM books WHERE publish_year > 0 ORDER BY publish_year DESC, created_at DESC LIMIT ?'
  ).bind(limit).all()
  return (results || []).map(hydrateBook)
}

export async function getPopularBooks(db: D1Database, limit: number): Promise<Book[]> {
  const { results } = await db.prepare(
    'SELECT * FROM books ORDER BY ratings_count DESC LIMIT ?'
  ).bind(limit).all()
  return (results || []).map(hydrateBook)
}

// ---- Lists ----

export async function getLists(db: D1Database, page: number, limit: number): Promise<{ lists: BookList[], total: number }> {
  const offset = (page - 1) * limit
  const countResult = await db.prepare('SELECT COUNT(*) as cnt FROM book_lists').first<{ cnt: number }>()
  const { results } = await db.prepare(
    `SELECT bl.*, (SELECT COUNT(*) FROM book_list_items WHERE list_id = bl.id) as item_count
     FROM book_lists bl ORDER BY bl.created_at DESC LIMIT ? OFFSET ?`
  ).bind(limit, offset).all()
  return { lists: (results || []) as unknown as BookList[], total: countResult?.cnt || 0 }
}

export async function getList(db: D1Database, id: number): Promise<BookList | null> {
  const list = await db.prepare(
    `SELECT bl.*, (SELECT COUNT(*) FROM book_list_items WHERE list_id = bl.id) as item_count FROM book_lists bl WHERE bl.id = ?`
  ).bind(id).first() as BookList | null
  if (!list) return null
  const { results } = await db.prepare(
    `SELECT bli.*, b.title, b.author_names, b.cover_url, b.cover_id, b.average_rating, b.ratings_count
     FROM book_list_items bli JOIN books b ON bli.book_id = b.id
     WHERE bli.list_id = ? ORDER BY bli.votes DESC, bli.position ASC`
  ).bind(id).all()
  list.items = (results || []).map((r: any) => ({
    id: r.id, list_id: r.list_id, book_id: r.book_id,
    position: r.position, votes: r.votes,
    book: hydrateBook({ id: r.book_id, title: r.title, author_names: r.author_names, cover_url: r.cover_url, cover_id: r.cover_id, average_rating: r.average_rating, ratings_count: r.ratings_count })
  }))
  return list
}

export async function createList(db: D1Database, title: string, description: string): Promise<number> {
  const r = await db.prepare(
    'INSERT INTO book_lists (title, description) VALUES (?, ?)'
  ).bind(title, description).run()
  return r.meta.last_row_id as number
}

export async function addBookToList(db: D1Database, listId: number, bookId: number): Promise<void> {
  const maxPos = await db.prepare('SELECT MAX(position) as m FROM book_list_items WHERE list_id = ?').bind(listId).first<{ m: number }>()
  await db.prepare(
    'INSERT OR IGNORE INTO book_list_items (list_id, book_id, position) VALUES (?, ?, ?)'
  ).bind(listId, bookId, (maxPos?.m || 0) + 1).run()
}

export async function voteOnListItem(db: D1Database, listId: number, bookId: number): Promise<void> {
  await db.prepare(
    'UPDATE book_list_items SET votes = votes + 1 WHERE list_id = ? AND book_id = ?'
  ).bind(listId, bookId).run()
}

// ---- Quotes ----

export async function getQuotes(db: D1Database, page: number, limit: number): Promise<{ quotes: Quote[], total: number }> {
  const offset = (page - 1) * limit
  const countResult = await db.prepare('SELECT COUNT(*) as cnt FROM quotes').first<{ cnt: number }>()
  const { results } = await db.prepare(
    `SELECT q.*, b.title as book_title FROM quotes q LEFT JOIN books b ON q.book_id = b.id
     ORDER BY q.likes_count DESC LIMIT ? OFFSET ?`
  ).bind(limit, offset).all()
  const quotes = (results || []).map((r: any) => ({
    ...r,
    book: r.book_title ? { id: r.book_id, title: r.book_title } : undefined
  })) as Quote[]
  return { quotes, total: countResult?.cnt || 0 }
}

export async function getBookQuotes(db: D1Database, bookId: number): Promise<Quote[]> {
  const { results } = await db.prepare(
    'SELECT * FROM quotes WHERE book_id = ? ORDER BY likes_count DESC'
  ).bind(bookId).all()
  return (results || []) as unknown as Quote[]
}

export async function createQuote(db: D1Database, bookId: number, authorName: string, text: string): Promise<number> {
  const r = await db.prepare(
    'INSERT INTO quotes (book_id, author_name, text) VALUES (?, ?, ?)'
  ).bind(bookId, authorName, text).run()
  return r.meta.last_row_id as number
}

// ---- Stats ----

export async function getStats(db: D1Database, year?: number): Promise<any> {
  const yearFilter = year ? `AND sb.date_read LIKE '${year}%'` : ''
  const booksRead = await db.prepare(
    `SELECT b.* FROM shelf_books sb JOIN shelves s ON sb.shelf_id = s.id JOIN books b ON sb.book_id = b.id
     WHERE s.slug = 'read' ${yearFilter}`
  ).all()

  const books = (booksRead.results || []).map(hydrateBook)
  const totalBooks = books.length
  const totalPages = books.reduce((sum, b) => sum + (b.page_count || 0), 0)

  const ratings = books.map(b => b.average_rating).filter(r => r > 0)
  const avgRating = ratings.length > 0 ? ratings.reduce((s, r) => s + r, 0) / ratings.length : 0

  const booksPerMonth: Record<string, number> = {}
  const pagesPerMonth: Record<string, number> = {}
  const genreBreakdown: Record<string, number> = {}
  const ratingDist: Record<number, number> = { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0 }

  for (const b of books) {
    // Genre
    if (b.subjects && b.subjects.length > 0) {
      const g = b.subjects[0]
      genreBreakdown[g] = (genreBreakdown[g] || 0) + 1
    }
  }

  // Reviews for user ratings
  for (const b of books) {
    const review = await db.prepare(
      "SELECT rating FROM reviews WHERE book_id = ? AND source = 'user' LIMIT 1"
    ).bind(b.id).first<{ rating: number }>()
    if (review?.rating) ratingDist[review.rating] = (ratingDist[review.rating] || 0) + 1
  }

  // Sort books
  const sorted = [...books].filter(b => b.page_count > 0).sort((a, b) => a.page_count - b.page_count)
  const shortestBook = sorted[0] || null
  const longestBook = sorted[sorted.length - 1] || null
  const highestRated = [...books].sort((a, b) => b.average_rating - a.average_rating)[0] || null
  const mostPopular = [...books].sort((a, b) => b.ratings_count - a.ratings_count)[0] || null

  return {
    total_books: totalBooks,
    total_pages: totalPages,
    average_rating: Math.round(avgRating * 100) / 100,
    books_per_month: booksPerMonth,
    pages_per_month: pagesPerMonth,
    genre_breakdown: genreBreakdown,
    rating_distribution: ratingDist,
    shortest_book: shortestBook,
    longest_book: longestBook,
    highest_rated: highestRated,
    most_popular: mostPopular,
  }
}

// ---- Feed ----

export async function getFeed(db: D1Database, limit: number): Promise<FeedItem[]> {
  const { results } = await db.prepare(
    'SELECT * FROM feed ORDER BY created_at DESC LIMIT ?'
  ).bind(limit).all()
  return (results || []) as unknown as FeedItem[]
}

export async function addFeedItem(db: D1Database, type: string, bookId: number, bookTitle: string, data: string): Promise<void> {
  await db.prepare(
    'INSERT INTO feed (type, book_id, book_title, data) VALUES (?, ?, ?, ?)'
  ).bind(type, bookId, bookTitle, data).run()
}

// ---- Notes ----

export async function getNote(db: D1Database, bookId: number): Promise<BookNote | null> {
  return await db.prepare('SELECT * FROM book_notes WHERE book_id = ?').bind(bookId).first() as BookNote | null
}

export async function upsertNote(db: D1Database, bookId: number, text: string): Promise<void> {
  await db.prepare(
    `INSERT INTO book_notes (book_id, text) VALUES (?, ?)
     ON CONFLICT(book_id) DO UPDATE SET text = ?, updated_at = datetime('now')`
  ).bind(bookId, text, text).run()
}

export async function deleteNote(db: D1Database, bookId: number): Promise<void> {
  await db.prepare('DELETE FROM book_notes WHERE book_id = ?').bind(bookId).run()
}
