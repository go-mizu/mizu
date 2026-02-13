// Database layer for D1

// ---- Schema (embedded for init endpoint) ----

const SCHEMA = `
CREATE TABLE IF NOT EXISTS books (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ol_key TEXT DEFAULT '',
  google_id TEXT DEFAULT '',
  title TEXT NOT NULL,
  original_title TEXT DEFAULT '',
  subtitle TEXT DEFAULT '',
  description TEXT DEFAULT '',
  author_names TEXT DEFAULT '',
  cover_url TEXT DEFAULT '',
  cover_id INTEGER DEFAULT 0,
  isbn10 TEXT DEFAULT '',
  isbn13 TEXT DEFAULT '',
  publisher TEXT DEFAULT '',
  publish_date TEXT DEFAULT '',
  publish_year INTEGER DEFAULT 0,
  language TEXT DEFAULT '',
  edition_language TEXT DEFAULT '',
  first_published TEXT DEFAULT '',
  page_count INTEGER DEFAULT 0,
  format TEXT DEFAULT '',
  subjects TEXT DEFAULT '[]',
  characters TEXT DEFAULT '[]',
  settings TEXT DEFAULT '[]',
  literary_awards TEXT DEFAULT '[]',
  series TEXT DEFAULT '',
  editions_count INTEGER DEFAULT 0,
  average_rating REAL DEFAULT 0,
  ratings_count INTEGER DEFAULT 0,
  reviews_count INTEGER DEFAULT 0,
  currently_reading INTEGER DEFAULT 0,
  want_to_read INTEGER DEFAULT 0,
  rating_dist TEXT DEFAULT '[]',
  source_id TEXT DEFAULT '',
  source_url TEXT DEFAULT '',
  asin TEXT DEFAULT '',
  created_at TEXT DEFAULT (datetime('now')),
  updated_at TEXT DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS authors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ol_key TEXT DEFAULT '',
  name TEXT NOT NULL,
  bio TEXT DEFAULT '',
  photo_url TEXT DEFAULT '',
  birth_date TEXT DEFAULT '',
  death_date TEXT DEFAULT '',
  works_count INTEGER DEFAULT 0,
  followers INTEGER DEFAULT 0,
  genres TEXT DEFAULT '',
  influences TEXT DEFAULT '',
  website TEXT DEFAULT '',
  source_id TEXT DEFAULT '',
  created_at TEXT DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS book_authors (
  book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  author_id INTEGER NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
  PRIMARY KEY (book_id, author_id)
);
CREATE TABLE IF NOT EXISTS shelves (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  is_exclusive INTEGER DEFAULT 0,
  is_default INTEGER DEFAULT 0,
  sort_order INTEGER DEFAULT 0,
  created_at TEXT DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS shelf_books (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  shelf_id INTEGER NOT NULL REFERENCES shelves(id) ON DELETE CASCADE,
  book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  date_added TEXT DEFAULT (datetime('now')),
  position INTEGER DEFAULT 0,
  date_started TEXT,
  date_read TEXT,
  read_count INTEGER DEFAULT 0,
  UNIQUE(shelf_id, book_id)
);
CREATE TABLE IF NOT EXISTS reviews (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  rating INTEGER DEFAULT 0,
  text TEXT DEFAULT '',
  is_spoiler INTEGER DEFAULT 0,
  likes_count INTEGER DEFAULT 0,
  comments_count INTEGER DEFAULT 0,
  reviewer_name TEXT DEFAULT '',
  source TEXT DEFAULT 'user',
  started_at TEXT,
  finished_at TEXT,
  created_at TEXT DEFAULT (datetime('now')),
  updated_at TEXT DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS review_comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  review_id INTEGER NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
  author_name TEXT DEFAULT '',
  text TEXT NOT NULL,
  created_at TEXT DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS reading_progress (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  page INTEGER DEFAULT 0,
  percent REAL DEFAULT 0,
  note TEXT DEFAULT '',
  created_at TEXT DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS reading_challenges (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  year INTEGER UNIQUE NOT NULL,
  goal INTEGER DEFAULT 0,
  progress INTEGER DEFAULT 0,
  created_at TEXT DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS book_lists (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  item_count INTEGER DEFAULT 0,
  source_url TEXT DEFAULT '',
  voter_count INTEGER DEFAULT 0,
  created_at TEXT DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS book_list_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  list_id INTEGER NOT NULL REFERENCES book_lists(id) ON DELETE CASCADE,
  book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  position INTEGER DEFAULT 0,
  votes INTEGER DEFAULT 0,
  UNIQUE(list_id, book_id)
);
CREATE TABLE IF NOT EXISTS quotes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  author_name TEXT DEFAULT '',
  text TEXT NOT NULL,
  likes_count INTEGER DEFAULT 0,
  created_at TEXT DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS book_notes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  book_id INTEGER UNIQUE NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  text TEXT NOT NULL,
  created_at TEXT DEFAULT (datetime('now')),
  updated_at TEXT DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS feed_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL,
  book_id INTEGER DEFAULT 0,
  book_title TEXT DEFAULT '',
  data TEXT DEFAULT '',
  created_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_books_source_id ON books(source_id);
CREATE INDEX IF NOT EXISTS idx_books_title ON books(title);
CREATE INDEX IF NOT EXISTS idx_books_ol_key ON books(ol_key);
CREATE INDEX IF NOT EXISTS idx_authors_ol_key ON authors(ol_key);
CREATE INDEX IF NOT EXISTS idx_shelf_books_book ON shelf_books(book_id);
CREATE INDEX IF NOT EXISTS idx_shelf_books_shelf ON shelf_books(shelf_id);
CREATE INDEX IF NOT EXISTS idx_reviews_book ON reviews(book_id);
CREATE INDEX IF NOT EXISTS idx_quotes_book ON quotes(book_id);
CREATE INDEX IF NOT EXISTS idx_feed_created ON feed_items(created_at);
INSERT OR IGNORE INTO shelves (name, slug, is_exclusive, is_default, sort_order) VALUES
  ('Want to Read', 'want-to-read', 1, 1, 1),
  ('Currently Reading', 'currently-reading', 1, 1, 2),
  ('Read', 'read', 1, 1, 3);
`

// ---- Helpers ----

function parseJSON(s: string | null | undefined, fallback: unknown = []): unknown {
  if (!s) return fallback
  try { return JSON.parse(s) } catch { return fallback }
}

function toJSON(v: unknown): string {
  return JSON.stringify(v ?? [])
}

function now(): string {
  return new Date().toISOString().replace('T', ' ').replace('Z', '')
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function rowToBook(row: any): Record<string, unknown> {
  if (!row) return row
  return {
    ...row,
    subjects: parseJSON(row.subjects),
    characters: parseJSON(row.characters),
    settings: parseJSON(row.settings),
    literary_awards: parseJSON(row.literary_awards),
    rating_dist: parseJSON(row.rating_dist),
    is_exclusive: undefined,
    is_default: undefined,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function rowToReview(row: any): Record<string, unknown> {
  if (!row) return row
  return { ...row, is_spoiler: Boolean(row.is_spoiler) }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function rowToShelf(row: any): Record<string, unknown> {
  if (!row) return row
  return {
    ...row,
    is_exclusive: Boolean(row.is_exclusive),
    is_default: Boolean(row.is_default),
  }
}

// ---- Init ----

export async function initDB(db: D1Database): Promise<void> {
  const stmts = SCHEMA.split(';').map(s => s.trim()).filter(s => s.length > 0)
  for (const stmt of stmts) {
    await db.exec(stmt + ';')
  }
}

// ---- Books ----

const BOOK_SELECT = `SELECT b.*,
  (SELECT r.rating FROM reviews r WHERE r.book_id = b.id AND r.source = 'user' ORDER BY r.created_at DESC LIMIT 1) as user_rating,
  (SELECT s.slug FROM shelf_books sb JOIN shelves s ON s.id = sb.shelf_id WHERE sb.book_id = b.id AND s.is_exclusive = 1 LIMIT 1) as user_shelf
FROM books b`

export async function getBook(db: D1Database, id: number): Promise<Record<string, unknown> | null> {
  const row = await db.prepare(`${BOOK_SELECT} WHERE b.id = ?`).bind(id).first()
  if (!row) return null
  const book = rowToBook(row)
  // Fetch authors
  const authors = await db.prepare(
    `SELECT a.* FROM authors a JOIN book_authors ba ON ba.author_id = a.id WHERE ba.book_id = ?`
  ).bind(id).all()
  book.authors = authors.results || []
  return book
}

export async function getBookBySourceId(db: D1Database, sourceId: string): Promise<Record<string, unknown> | null> {
  const row = await db.prepare(`${BOOK_SELECT} WHERE b.source_id = ?`).bind(sourceId).first()
  return row ? rowToBook(row) : null
}

export async function getBookByOLKey(db: D1Database, olKey: string): Promise<Record<string, unknown> | null> {
  const row = await db.prepare(`${BOOK_SELECT} WHERE b.ol_key = ?`).bind(olKey).first()
  return row ? rowToBook(row) : null
}

export async function createBook(db: D1Database, data: Record<string, unknown>): Promise<Record<string, unknown>> {
  const ts = now()
  const result = await db.prepare(`INSERT INTO books
    (ol_key, google_id, title, original_title, subtitle, description, author_names,
     cover_url, cover_id, isbn10, isbn13, publisher, publish_date, publish_year,
     language, edition_language, first_published, page_count, format,
     subjects, characters, settings, literary_awards, series, editions_count,
     average_rating, ratings_count, reviews_count, currently_reading, want_to_read,
     rating_dist, source_id, source_url, asin, created_at, updated_at)
    VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
  ).bind(
    data.ol_key || '', data.google_id || '', data.title || '', data.original_title || '',
    data.subtitle || '', data.description || '', data.author_names || '',
    data.cover_url || '', data.cover_id || 0, data.isbn10 || '', data.isbn13 || '',
    data.publisher || '', data.publish_date || '', data.publish_year || 0,
    data.language || '', data.edition_language || '', data.first_published || '',
    data.page_count || 0, data.format || '',
    toJSON(data.subjects), toJSON(data.characters), toJSON(data.settings),
    toJSON(data.literary_awards), data.series || '', data.editions_count || 0,
    data.average_rating || 0, data.ratings_count || 0, data.reviews_count || 0,
    data.currently_reading || 0, data.want_to_read || 0,
    toJSON(data.rating_dist), data.source_id || '', data.source_url || '',
    data.asin || '', ts, ts,
  ).run()

  const id = result.meta?.last_row_id
  return (await getBook(db, id as number))!
}

export async function updateBook(db: D1Database, id: number, data: Record<string, unknown>): Promise<Record<string, unknown> | null> {
  const fields: string[] = []
  const values: unknown[] = []
  const strFields = ['ol_key', 'google_id', 'title', 'original_title', 'subtitle', 'description',
    'author_names', 'cover_url', 'isbn10', 'isbn13', 'publisher', 'publish_date',
    'language', 'edition_language', 'first_published', 'format', 'series', 'source_id', 'source_url', 'asin']
  const numFields = ['cover_id', 'publish_year', 'page_count', 'editions_count',
    'average_rating', 'ratings_count', 'reviews_count', 'currently_reading', 'want_to_read']
  const jsonFields = ['subjects', 'characters', 'settings', 'literary_awards', 'rating_dist']

  for (const f of strFields) {
    if (f in data) { fields.push(`${f} = ?`); values.push(data[f] || '') }
  }
  for (const f of numFields) {
    if (f in data) { fields.push(`${f} = ?`); values.push(data[f] || 0) }
  }
  for (const f of jsonFields) {
    if (f in data) { fields.push(`${f} = ?`); values.push(toJSON(data[f])) }
  }
  if (fields.length === 0) return getBook(db, id)
  fields.push('updated_at = ?')
  values.push(now())
  values.push(id)
  await db.prepare(`UPDATE books SET ${fields.join(', ')} WHERE id = ?`).bind(...values).run()
  return getBook(db, id)
}

export async function deleteBook(db: D1Database, id: number): Promise<void> {
  await db.prepare('DELETE FROM books WHERE id = ?').bind(id).run()
}

export async function searchBooks(db: D1Database, q: string, page: number, limit: number) {
  if (!q) {
    const total = await db.prepare('SELECT COUNT(*) as c FROM books').first<{ c: number }>()
    const rows = await db.prepare(`${BOOK_SELECT} ORDER BY b.created_at DESC LIMIT ? OFFSET ?`)
      .bind(limit, (page - 1) * limit).all()
    return { books: (rows.results || []).map(rowToBook), total_count: total?.c || 0, page, page_size: limit }
  }
  const pattern = `%${q}%`
  const total = await db.prepare('SELECT COUNT(*) as c FROM books WHERE title LIKE ? OR author_names LIKE ?')
    .bind(pattern, pattern).first<{ c: number }>()
  const rows = await db.prepare(`${BOOK_SELECT} WHERE b.title LIKE ? OR b.author_names LIKE ? ORDER BY b.average_rating DESC LIMIT ? OFFSET ?`)
    .bind(pattern, pattern, limit, (page - 1) * limit).all()
  return { books: (rows.results || []).map(rowToBook), total_count: total?.c || 0, page, page_size: limit }
}

export async function getTrendingBooks(db: D1Database, limit: number) {
  const rows = await db.prepare(`${BOOK_SELECT} ORDER BY b.ratings_count DESC LIMIT ?`).bind(limit).all()
  return (rows.results || []).map(rowToBook)
}

export async function getPopularBooks(db: D1Database, limit: number) {
  const rows = await db.prepare(`${BOOK_SELECT} ORDER BY b.average_rating DESC, b.ratings_count DESC LIMIT ?`).bind(limit).all()
  return (rows.results || []).map(rowToBook)
}

export async function getNewReleases(db: D1Database, limit: number) {
  const rows = await db.prepare(`${BOOK_SELECT} ORDER BY b.publish_year DESC, b.created_at DESC LIMIT ?`).bind(limit).all()
  return (rows.results || []).map(rowToBook)
}

export async function getSimilarBooks(db: D1Database, bookId: number, limit: number) {
  // Find books with overlapping subjects
  const book = await db.prepare('SELECT subjects FROM books WHERE id = ?').bind(bookId).first<{ subjects: string }>()
  if (!book) return []
  const subjects = parseJSON(book.subjects) as string[]
  if (subjects.length === 0) return []
  const pattern = `%${subjects[0]}%`
  const rows = await db.prepare(`${BOOK_SELECT} WHERE b.id != ? AND b.subjects LIKE ? LIMIT ?`)
    .bind(bookId, pattern, limit).all()
  return (rows.results || []).map(rowToBook)
}

export async function getBooksByAuthor(db: D1Database, authorId: number) {
  const rows = await db.prepare(
    `${BOOK_SELECT} JOIN book_authors ba ON ba.book_id = b.id WHERE ba.author_id = ? ORDER BY b.publish_year DESC`
  ).bind(authorId).all()
  return (rows.results || []).map(rowToBook)
}

export async function getBooksByGenre(db: D1Database, genre: string, page: number, limit: number) {
  const pattern = `%${genre}%`
  const total = await db.prepare('SELECT COUNT(*) as c FROM books WHERE subjects LIKE ?').bind(pattern).first<{ c: number }>()
  const rows = await db.prepare(`${BOOK_SELECT} WHERE b.subjects LIKE ? ORDER BY b.average_rating DESC LIMIT ? OFFSET ?`)
    .bind(pattern, limit, (page - 1) * limit).all()
  return { books: (rows.results || []).map(rowToBook), total_count: total?.c || 0, page, page_size: limit }
}

// ---- Authors ----

export async function getAuthor(db: D1Database, id: number): Promise<Record<string, unknown> | null> {
  return await db.prepare('SELECT * FROM authors WHERE id = ?').bind(id).first()
}

export async function getAuthorBySourceId(db: D1Database, sourceId: string): Promise<Record<string, unknown> | null> {
  return await db.prepare('SELECT * FROM authors WHERE source_id = ?').bind(sourceId).first()
}

export async function createAuthor(db: D1Database, data: Record<string, unknown>): Promise<Record<string, unknown>> {
  const result = await db.prepare(
    `INSERT INTO authors (ol_key, name, bio, photo_url, birth_date, death_date, works_count, followers, genres, influences, website, source_id, created_at)
     VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
  ).bind(
    data.ol_key || '', data.name || '', data.bio || '', data.photo_url || '',
    data.birth_date || '', data.death_date || '', data.works_count || 0,
    data.followers || 0, data.genres || '', data.influences || '',
    data.website || '', data.source_id || '', now(),
  ).run()
  return (await getAuthor(db, result.meta?.last_row_id as number))!
}

export async function getOrCreateAuthor(db: D1Database, name: string, sourceId?: string): Promise<Record<string, unknown>> {
  if (sourceId) {
    const existing = await getAuthorBySourceId(db, sourceId)
    if (existing) return existing
  }
  const byName = await db.prepare('SELECT * FROM authors WHERE name = ? LIMIT 1').bind(name).first()
  if (byName) return byName
  return createAuthor(db, { name, source_id: sourceId || '' })
}

export async function searchAuthors(db: D1Database, q: string) {
  const pattern = `%${q}%`
  const rows = await db.prepare('SELECT * FROM authors WHERE name LIKE ? LIMIT 20').bind(pattern).all()
  return rows.results || []
}

export async function getAuthorByOLKey(db: D1Database, olKey: string): Promise<Record<string, unknown> | null> {
  return await db.prepare('SELECT * FROM authors WHERE ol_key = ?').bind(olKey).first()
}

export async function updateAuthor(db: D1Database, id: number, data: Record<string, unknown>): Promise<Record<string, unknown> | null> {
  const fields: string[] = []
  const values: unknown[] = []
  const strFields = ['ol_key', 'name', 'bio', 'photo_url', 'birth_date', 'death_date', 'genres', 'influences', 'website', 'source_id']
  const numFields = ['works_count', 'followers']

  for (const f of strFields) {
    if (f in data && data[f]) { fields.push(`${f} = ?`); values.push(data[f]) }
  }
  for (const f of numFields) {
    if (f in data && data[f]) { fields.push(`${f} = ?`); values.push(data[f]) }
  }
  if (fields.length === 0) return getAuthor(db, id)
  values.push(id)
  await db.prepare(`UPDATE authors SET ${fields.join(', ')} WHERE id = ?`).bind(...values).run()
  return getAuthor(db, id)
}

export async function linkBookAuthor(db: D1Database, bookId: number, authorId: number): Promise<void> {
  await db.prepare('INSERT OR IGNORE INTO book_authors (book_id, author_id) VALUES (?, ?)').bind(bookId, authorId).run()
}

// ---- Shelves ----

export async function getShelves(db: D1Database) {
  const rows = await db.prepare(
    `SELECT s.*, (SELECT COUNT(*) FROM shelf_books sb WHERE sb.shelf_id = s.id) as book_count
     FROM shelves s ORDER BY s.sort_order, s.id`
  ).all()
  return (rows.results || []).map(rowToShelf)
}

export async function createShelf(db: D1Database, name: string, slug: string): Promise<Record<string, unknown>> {
  const result = await db.prepare(
    'INSERT INTO shelves (name, slug, is_exclusive, is_default, sort_order, created_at) VALUES (?,?,0,0,99,?)'
  ).bind(name, slug, now()).run()
  const id = result.meta?.last_row_id as number
  const row = await db.prepare('SELECT *, 0 as book_count FROM shelves WHERE id = ?').bind(id).first()
  return rowToShelf(row)!
}

export async function updateShelf(db: D1Database, id: number, name: string): Promise<Record<string, unknown> | null> {
  const shelf = await db.prepare('SELECT * FROM shelves WHERE id = ?').bind(id).first()
  if (!shelf || shelf.is_default) return null
  const slug = name.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '')
  await db.prepare('UPDATE shelves SET name = ?, slug = ? WHERE id = ?').bind(name, slug, id).run()
  const row = await db.prepare(
    `SELECT s.*, (SELECT COUNT(*) FROM shelf_books sb WHERE sb.shelf_id = s.id) as book_count FROM shelves s WHERE s.id = ?`
  ).bind(id).first()
  return rowToShelf(row)
}

export async function deleteShelf(db: D1Database, id: number): Promise<boolean> {
  const shelf = await db.prepare('SELECT * FROM shelves WHERE id = ?').bind(id).first()
  if (!shelf || shelf.is_default) return false
  await db.prepare('DELETE FROM shelves WHERE id = ?').bind(id).run()
  return true
}

export async function getShelfBooks(db: D1Database, shelfId: number, page: number, limit: number) {
  const total = await db.prepare('SELECT COUNT(*) as c FROM shelf_books WHERE shelf_id = ?').bind(shelfId).first<{ c: number }>()
  const rows = await db.prepare(
    `SELECT sb.*, b.title, b.author_names, b.cover_url, b.page_count, b.average_rating,
            b.publish_year, b.source_id, b.isbn13, b.publisher, b.format,
            b.subjects, b.rating_dist, b.ratings_count, b.reviews_count, b.description
     FROM shelf_books sb JOIN books b ON b.id = sb.book_id
     WHERE sb.shelf_id = ? ORDER BY sb.date_added DESC LIMIT ? OFFSET ?`
  ).bind(shelfId, limit, (page - 1) * limit).all()

  const books = (rows.results || []).map((row: Record<string, unknown>) => ({
    id: row.id,
    shelf_id: row.shelf_id,
    book_id: row.book_id,
    date_added: row.date_added,
    position: row.position,
    date_started: row.date_started,
    date_read: row.date_read,
    read_count: row.read_count,
    book: rowToBook({
      id: row.book_id, title: row.title, author_names: row.author_names,
      cover_url: row.cover_url, page_count: row.page_count, average_rating: row.average_rating,
      publish_year: row.publish_year, source_id: row.source_id, isbn13: row.isbn13,
      publisher: row.publisher, format: row.format, subjects: row.subjects,
      rating_dist: row.rating_dist, ratings_count: row.ratings_count,
      reviews_count: row.reviews_count, description: row.description,
    }),
  }))
  return { books, total: total?.c || 0, page }
}

export async function addBookToShelf(db: D1Database, shelfId: number, bookId: number): Promise<void> {
  // If exclusive shelf, remove from other exclusive shelves first
  const shelf = await db.prepare('SELECT * FROM shelves WHERE id = ?').bind(shelfId).first()
  if (shelf?.is_exclusive) {
    await db.prepare(
      `DELETE FROM shelf_books WHERE book_id = ? AND shelf_id IN (SELECT id FROM shelves WHERE is_exclusive = 1)`
    ).bind(bookId).run()
  }
  await db.prepare('INSERT OR REPLACE INTO shelf_books (shelf_id, book_id, date_added) VALUES (?,?,?)')
    .bind(shelfId, bookId, now()).run()
}

export async function removeBookFromShelf(db: D1Database, shelfId: number, bookId: number): Promise<void> {
  await db.prepare('DELETE FROM shelf_books WHERE shelf_id = ? AND book_id = ?').bind(shelfId, bookId).run()
}

export async function updateShelfBook(db: D1Database, shelfId: number, bookId: number, data: Record<string, unknown>): Promise<void> {
  const fields: string[] = []
  const values: unknown[] = []
  if ('date_started' in data) { fields.push('date_started = ?'); values.push(data.date_started || null) }
  if ('date_read' in data) { fields.push('date_read = ?'); values.push(data.date_read || null) }
  if ('read_count' in data) { fields.push('read_count = ?'); values.push(data.read_count || 0) }
  if (fields.length === 0) return
  values.push(shelfId, bookId)
  await db.prepare(`UPDATE shelf_books SET ${fields.join(', ')} WHERE shelf_id = ? AND book_id = ?`).bind(...values).run()
}

// ---- Reviews ----

export async function getBookReviews(db: D1Database, bookId: number, filters: Record<string, unknown>) {
  const page = (filters.page as number) || 1
  const limit = (filters.limit as number) || 20
  const conditions = ['r.book_id = ?']
  const params: unknown[] = [bookId]

  if (filters.rating) { conditions.push('r.rating = ?'); params.push(filters.rating) }
  if (filters.source) { conditions.push('r.source = ?'); params.push(filters.source) }
  if (filters.has_text === 'true') conditions.push("r.text != ''")
  if (filters.has_text === 'false') conditions.push("r.text = ''")
  if (filters.q) { conditions.push('r.text LIKE ?'); params.push(`%${filters.q}%`) }
  if (filters.include_spoilers !== 'true') conditions.push('r.is_spoiler = 0')

  const where = conditions.join(' AND ')
  let orderBy = 'r.created_at DESC'
  if (filters.sort === 'popular') orderBy = 'r.likes_count DESC'
  else if (filters.sort === 'oldest') orderBy = 'r.created_at ASC'
  else if (filters.sort === 'rating_desc') orderBy = 'r.rating DESC'
  else if (filters.sort === 'rating_asc') orderBy = 'r.rating ASC'

  const totalRow = await db.prepare(`SELECT COUNT(*) as c FROM reviews r WHERE ${where}`).bind(...params).first<{ c: number }>()
  const rows = await db.prepare(`SELECT r.* FROM reviews r WHERE ${where} ORDER BY ${orderBy} LIMIT ? OFFSET ?`)
    .bind(...params, limit, (page - 1) * limit).all()
  return { reviews: (rows.results || []).map(rowToReview), total: totalRow?.c || 0 }
}

export async function createReview(db: D1Database, bookId: number, data: Record<string, unknown>): Promise<Record<string, unknown>> {
  const ts = now()
  const result = await db.prepare(
    `INSERT INTO reviews (book_id, rating, text, is_spoiler, reviewer_name, source, started_at, finished_at, created_at, updated_at)
     VALUES (?,?,?,?,?,?,?,?,?,?)`
  ).bind(
    bookId, data.rating || 0, data.text || '', data.is_spoiler ? 1 : 0,
    data.reviewer_name || 'You', data.source || 'user',
    data.started_at || null, data.finished_at || null, ts, ts,
  ).run()
  const row = await db.prepare('SELECT * FROM reviews WHERE id = ?').bind(result.meta?.last_row_id).first()
  return rowToReview(row)!
}

export async function updateReview(db: D1Database, id: number, data: Record<string, unknown>): Promise<Record<string, unknown> | null> {
  const fields: string[] = []
  const values: unknown[] = []
  if ('rating' in data) { fields.push('rating = ?'); values.push(data.rating) }
  if ('text' in data) { fields.push('text = ?'); values.push(data.text) }
  if ('is_spoiler' in data) { fields.push('is_spoiler = ?'); values.push(data.is_spoiler ? 1 : 0) }
  if ('started_at' in data) { fields.push('started_at = ?'); values.push(data.started_at || null) }
  if ('finished_at' in data) { fields.push('finished_at = ?'); values.push(data.finished_at || null) }
  if (fields.length === 0) return null
  fields.push('updated_at = ?')
  values.push(now())
  values.push(id)
  await db.prepare(`UPDATE reviews SET ${fields.join(', ')} WHERE id = ?`).bind(...values).run()
  const row = await db.prepare('SELECT * FROM reviews WHERE id = ?').bind(id).first()
  return rowToReview(row)
}

export async function deleteReview(db: D1Database, id: number): Promise<void> {
  await db.prepare('DELETE FROM reviews WHERE id = ?').bind(id).run()
}

export async function likeReview(db: D1Database, id: number): Promise<number> {
  await db.prepare('UPDATE reviews SET likes_count = likes_count + 1 WHERE id = ?').bind(id).run()
  const row = await db.prepare('SELECT likes_count FROM reviews WHERE id = ?').bind(id).first<{ likes_count: number }>()
  return row?.likes_count || 0
}

// ---- Review Comments ----

export async function getComments(db: D1Database, reviewId: number, page: number, limit: number) {
  const total = await db.prepare('SELECT COUNT(*) as c FROM review_comments WHERE review_id = ?').bind(reviewId).first<{ c: number }>()
  const rows = await db.prepare('SELECT * FROM review_comments WHERE review_id = ? ORDER BY created_at LIMIT ? OFFSET ?')
    .bind(reviewId, limit, (page - 1) * limit).all()
  return { comments: rows.results || [], total: total?.c || 0 }
}

export async function createComment(db: D1Database, reviewId: number, data: Record<string, unknown>): Promise<Record<string, unknown>> {
  const result = await db.prepare(
    'INSERT INTO review_comments (review_id, author_name, text, created_at) VALUES (?,?,?,?)'
  ).bind(reviewId, data.author_name || 'You', data.text || '', now()).run()
  await db.prepare('UPDATE reviews SET comments_count = comments_count + 1 WHERE id = ?').bind(reviewId).run()
  return (await db.prepare('SELECT * FROM review_comments WHERE id = ?').bind(result.meta?.last_row_id).first())!
}

export async function deleteComment(db: D1Database, reviewId: number, commentId: number): Promise<void> {
  await db.prepare('DELETE FROM review_comments WHERE id = ? AND review_id = ?').bind(commentId, reviewId).run()
  await db.prepare('UPDATE reviews SET comments_count = MAX(0, comments_count - 1) WHERE id = ?').bind(reviewId).run()
}

// ---- Reading Progress ----

export async function getProgress(db: D1Database, bookId: number) {
  const rows = await db.prepare('SELECT * FROM reading_progress WHERE book_id = ? ORDER BY created_at DESC').bind(bookId).all()
  return rows.results || []
}

export async function createProgress(db: D1Database, bookId: number, data: Record<string, unknown>): Promise<Record<string, unknown>> {
  const result = await db.prepare(
    'INSERT INTO reading_progress (book_id, page, percent, note, created_at) VALUES (?,?,?,?,?)'
  ).bind(bookId, data.page || 0, data.percent || 0, data.note || '', now()).run()
  return (await db.prepare('SELECT * FROM reading_progress WHERE id = ?').bind(result.meta?.last_row_id).first())!
}

// ---- Reading Challenge ----

export async function getChallenge(db: D1Database, year: number) {
  const row = await db.prepare('SELECT * FROM reading_challenges WHERE year = ?').bind(year).first()
  if (!row) return null
  // Compute progress
  const readShelf = await db.prepare("SELECT id FROM shelves WHERE slug = 'read'").first<{ id: number }>()
  if (readShelf) {
    const count = await db.prepare(
      `SELECT COUNT(*) as c FROM shelf_books WHERE shelf_id = ? AND date_read LIKE ?`
    ).bind(readShelf.id, `${year}%`).first<{ c: number }>()
    return { ...row, progress: count?.c || 0 }
  }
  return row
}

export async function createOrUpdateChallenge(db: D1Database, year: number, goal: number): Promise<Record<string, unknown>> {
  await db.prepare(
    'INSERT INTO reading_challenges (year, goal, created_at) VALUES (?,?,?) ON CONFLICT(year) DO UPDATE SET goal = ?'
  ).bind(year, goal, now(), goal).run()
  return (await getChallenge(db, year))!
}

// ---- Lists ----

export async function getLists(db: D1Database) {
  const rows = await db.prepare('SELECT * FROM book_lists ORDER BY created_at DESC').all()
  return { lists: rows.results || [], total: rows.results?.length || 0 }
}

export async function getList(db: D1Database, id: number) {
  const list = await db.prepare('SELECT * FROM book_lists WHERE id = ?').bind(id).first()
  if (!list) return null
  const items = await db.prepare(
    `SELECT bli.*, b.title, b.author_names, b.cover_url, b.average_rating, b.ratings_count, b.source_id
     FROM book_list_items bli JOIN books b ON b.id = bli.book_id
     WHERE bli.list_id = ? ORDER BY bli.position`
  ).bind(id).all()
  return {
    ...list,
    items: (items.results || []).map((row: Record<string, unknown>) => ({
      id: row.id,
      list_id: row.list_id,
      book_id: row.book_id,
      position: row.position,
      votes: row.votes,
      book: {
        id: row.book_id, title: row.title, author_names: row.author_names,
        cover_url: row.cover_url, average_rating: row.average_rating,
        ratings_count: row.ratings_count, source_id: row.source_id,
      },
    })),
  }
}

export async function createList(db: D1Database, data: Record<string, unknown>): Promise<Record<string, unknown>> {
  const result = await db.prepare(
    'INSERT INTO book_lists (title, description, source_url, voter_count, created_at) VALUES (?,?,?,?,?)'
  ).bind(data.title || '', data.description || '', data.source_url || '', data.voter_count || 0, now()).run()
  return (await db.prepare('SELECT * FROM book_lists WHERE id = ?').bind(result.meta?.last_row_id).first())!
}

export async function addBookToList(db: D1Database, listId: number, bookId: number): Promise<void> {
  const maxPos = await db.prepare('SELECT MAX(position) as p FROM book_list_items WHERE list_id = ?').bind(listId).first<{ p: number }>()
  await db.prepare('INSERT OR IGNORE INTO book_list_items (list_id, book_id, position) VALUES (?,?,?)')
    .bind(listId, bookId, (maxPos?.p || 0) + 1).run()
  await db.prepare('UPDATE book_lists SET item_count = (SELECT COUNT(*) FROM book_list_items WHERE list_id = ?) WHERE id = ?')
    .bind(listId, listId).run()
}

export async function voteOnBook(db: D1Database, listId: number, bookId: number): Promise<void> {
  await db.prepare('UPDATE book_list_items SET votes = votes + 1 WHERE list_id = ? AND book_id = ?')
    .bind(listId, bookId).run()
}

// ---- Quotes ----

export async function getQuotes(db: D1Database, page: number, limit: number) {
  const rows = await db.prepare(
    `SELECT q.*, b.title as book_title, b.cover_url as book_cover_url
     FROM quotes q LEFT JOIN books b ON b.id = q.book_id
     ORDER BY q.likes_count DESC LIMIT ? OFFSET ?`
  ).bind(limit, (page - 1) * limit).all()
  return (rows.results || []).map((row: Record<string, unknown>) => ({
    id: row.id, book_id: row.book_id, author_name: row.author_name,
    text: row.text, likes_count: row.likes_count, created_at: row.created_at,
    book: row.book_title ? { id: row.book_id, title: row.book_title, cover_url: row.book_cover_url } : undefined,
  }))
}

export async function getBookQuotes(db: D1Database, bookId: number) {
  const rows = await db.prepare('SELECT * FROM quotes WHERE book_id = ? ORDER BY likes_count DESC').bind(bookId).all()
  return rows.results || []
}

export async function createQuote(db: D1Database, data: Record<string, unknown>): Promise<Record<string, unknown>> {
  const result = await db.prepare(
    'INSERT INTO quotes (book_id, author_name, text, likes_count, created_at) VALUES (?,?,?,?,?)'
  ).bind(data.book_id || 0, data.author_name || '', data.text || '', data.likes_count || 0, now()).run()
  return (await db.prepare('SELECT * FROM quotes WHERE id = ?').bind(result.meta?.last_row_id).first())!
}

// ---- Book Notes ----

export async function getNote(db: D1Database, bookId: number) {
  return await db.prepare('SELECT * FROM book_notes WHERE book_id = ?').bind(bookId).first()
}

export async function createOrUpdateNote(db: D1Database, bookId: number, text: string): Promise<Record<string, unknown>> {
  const ts = now()
  await db.prepare(
    'INSERT INTO book_notes (book_id, text, created_at, updated_at) VALUES (?,?,?,?) ON CONFLICT(book_id) DO UPDATE SET text = ?, updated_at = ?'
  ).bind(bookId, text, ts, ts, text, ts).run()
  return (await getNote(db, bookId))!
}

export async function deleteNote(db: D1Database, bookId: number): Promise<void> {
  await db.prepare('DELETE FROM book_notes WHERE book_id = ?').bind(bookId).run()
}

// ---- Feed ----

export async function getFeed(db: D1Database, limit: number) {
  const rows = await db.prepare('SELECT * FROM feed_items ORDER BY created_at DESC LIMIT ?').bind(limit).all()
  return rows.results || []
}

export async function addFeedItem(db: D1Database, type: string, bookId: number, bookTitle: string, data: string = ''): Promise<void> {
  await db.prepare('INSERT INTO feed_items (type, book_id, book_title, data, created_at) VALUES (?,?,?,?,?)')
    .bind(type, bookId, bookTitle, data, now()).run()
}

// ---- Stats ----

export async function getStats(db: D1Database, year: number) {
  const readShelf = await db.prepare("SELECT id FROM shelves WHERE slug = 'read'").first<{ id: number }>()
  if (!readShelf) {
    return {
      total_books: 0, total_pages: 0, average_rating: 0,
      books_per_month: {}, pages_per_month: {},
      genre_breakdown: {}, rating_distribution: {},
      shortest_book: null, longest_book: null, highest_rated: null, most_popular: null,
    }
  }
  const sid = readShelf.id
  const yearPattern = `${year}%`

  // Total books read this year
  const totalBooks = await db.prepare(
    'SELECT COUNT(*) as c FROM shelf_books WHERE shelf_id = ? AND date_read LIKE ?'
  ).bind(sid, yearPattern).first<{ c: number }>()

  // Get all read book IDs for this year
  const readBookIds = await db.prepare(
    'SELECT book_id, date_read FROM shelf_books WHERE shelf_id = ? AND date_read LIKE ?'
  ).bind(sid, yearPattern).all()
  const bookIds = (readBookIds.results || []).map((r: Record<string, unknown>) => r.book_id as number)

  let totalPages = 0
  let avgRating = 0
  const booksPerMonth: Record<string, number> = {}
  const pagesPerMonth: Record<string, number> = {}
  const genreBreakdown: Record<string, number> = {}
  const ratingDist: Record<string, number> = {}
  let shortestBook: Record<string, unknown> | null = null
  let longestBook: Record<string, unknown> | null = null
  let highestRated: Record<string, unknown> | null = null
  let mostPopular: Record<string, unknown> | null = null

  if (bookIds.length > 0) {
    const placeholders = bookIds.map(() => '?').join(',')

    // Pages and rating stats
    const stats = await db.prepare(
      `SELECT SUM(page_count) as tp, AVG(average_rating) as ar FROM books WHERE id IN (${placeholders})`
    ).bind(...bookIds).first<{ tp: number; ar: number }>()
    totalPages = stats?.tp || 0
    avgRating = Math.round((stats?.ar || 0) * 100) / 100

    // Books/pages per month
    for (const r of (readBookIds.results || []) as Record<string, unknown>[]) {
      const dateRead = r.date_read as string
      if (!dateRead) continue
      const month = dateRead.slice(0, 7) // YYYY-MM
      booksPerMonth[month] = (booksPerMonth[month] || 0) + 1
    }
    // Get page counts for monthly breakdown
    const booksData = await db.prepare(`SELECT id, page_count FROM books WHERE id IN (${placeholders})`).bind(...bookIds).all()
    const pageMap = new Map<number, number>()
    for (const b of (booksData.results || []) as Record<string, unknown>[]) {
      pageMap.set(b.id as number, (b.page_count as number) || 0)
    }
    for (const r of (readBookIds.results || []) as Record<string, unknown>[]) {
      const dateRead = r.date_read as string
      if (!dateRead) continue
      const month = dateRead.slice(0, 7)
      const pages = pageMap.get(r.book_id as number) || 0
      pagesPerMonth[month] = (pagesPerMonth[month] || 0) + pages
    }

    // Genre breakdown
    const genreRows = await db.prepare(`SELECT subjects FROM books WHERE id IN (${placeholders})`).bind(...bookIds).all()
    for (const row of (genreRows.results || []) as Record<string, unknown>[]) {
      const subjects = parseJSON(row.subjects as string) as string[]
      for (const s of subjects) {
        genreBreakdown[s] = (genreBreakdown[s] || 0) + 1
      }
    }

    // Rating distribution from user reviews
    const ratingRows = await db.prepare(
      `SELECT rating, COUNT(*) as c FROM reviews WHERE book_id IN (${placeholders}) AND source = 'user' AND rating > 0 GROUP BY rating`
    ).bind(...bookIds).all()
    for (const row of (ratingRows.results || []) as Record<string, unknown>[]) {
      ratingDist[String(row.rating)] = row.c as number
    }

    // Notable books
    const shortest = await db.prepare(
      `${BOOK_SELECT} WHERE b.id IN (${placeholders}) AND b.page_count > 0 ORDER BY b.page_count ASC LIMIT 1`
    ).bind(...bookIds).first()
    shortestBook = shortest ? rowToBook(shortest) : null

    const longest = await db.prepare(
      `${BOOK_SELECT} WHERE b.id IN (${placeholders}) ORDER BY b.page_count DESC LIMIT 1`
    ).bind(...bookIds).first()
    longestBook = longest ? rowToBook(longest) : null

    const highest = await db.prepare(
      `${BOOK_SELECT} WHERE b.id IN (${placeholders}) ORDER BY b.average_rating DESC LIMIT 1`
    ).bind(...bookIds).first()
    highestRated = highest ? rowToBook(highest) : null

    const popular = await db.prepare(
      `${BOOK_SELECT} WHERE b.id IN (${placeholders}) ORDER BY b.ratings_count DESC LIMIT 1`
    ).bind(...bookIds).first()
    mostPopular = popular ? rowToBook(popular) : null
  }

  return {
    total_books: totalBooks?.c || 0,
    total_pages: totalPages,
    average_rating: avgRating,
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

// ---- Genres ----

export async function getGenres(db: D1Database) {
  const rows = await db.prepare('SELECT subjects FROM books WHERE subjects != \'[]\' ').all()
  const counts: Record<string, number> = {}
  for (const row of (rows.results || []) as Record<string, unknown>[]) {
    const subjects = parseJSON(row.subjects as string) as string[]
    for (const s of subjects) {
      counts[s] = (counts[s] || 0) + 1
    }
  }
  return Object.entries(counts)
    .sort((a, b) => b[1] - a[1])
    .map(([name, count]) => ({
      name,
      slug: name.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, ''),
      book_count: count,
    }))
}

// ---- CSV Export ----

export async function exportCSV(db: D1Database): Promise<string> {
  const header = ['Title', 'Author', 'ISBN', 'ISBN13', 'My Rating', 'Average Rating',
    'Publisher', 'Number of Pages', 'Year Published', 'Date Read', 'Date Added',
    'Exclusive Shelf', 'My Review']

  const rows = await db.prepare(`
    SELECT b.*, sb.date_read, sb.date_added, s.slug as shelf_slug,
           (SELECT r.rating FROM reviews r WHERE r.book_id = b.id AND r.source = 'user' ORDER BY r.created_at DESC LIMIT 1) as user_rating,
           (SELECT r.text FROM reviews r WHERE r.book_id = b.id AND r.source = 'user' ORDER BY r.created_at DESC LIMIT 1) as review_text
    FROM books b
    LEFT JOIN shelf_books sb ON sb.book_id = b.id
    LEFT JOIN shelves s ON s.id = sb.shelf_id AND s.is_exclusive = 1
    ORDER BY b.title
  `).all()

  const lines = [header.map(escapeCSV).join(',')]
  for (const row of (rows.results || []) as Record<string, unknown>[]) {
    lines.push([
      row.title, row.author_names, row.isbn10, row.isbn13,
      row.user_rating || '', row.average_rating || '',
      row.publisher, row.page_count, row.publish_year,
      row.date_read || '', row.date_added || '',
      row.shelf_slug || '', row.review_text || '',
    ].map(v => escapeCSV(String(v ?? ''))).join(','))
  }
  return lines.join('\n')
}

function escapeCSV(s: string): string {
  if (s.includes(',') || s.includes('"') || s.includes('\n')) {
    return '"' + s.replace(/"/g, '""') + '"'
  }
  return s
}

// ---- CSV Import ----

export async function importCSV(db: D1Database, text: string): Promise<number> {
  const rows = parseCSVText(text)
  if (rows.length < 2) return 0

  const header = rows[0]
  const cols: Record<string, number> = {}
  for (let i = 0; i < header.length; i++) {
    cols[header[i].trim()] = i
  }

  let imported = 0
  for (let i = 1; i < rows.length; i++) {
    const row = rows[i]
    const get = (key: string) => {
      const idx = cols[key]
      return idx !== undefined && idx < row.length ? row[idx].trim() : ''
    }

    const title = get('Title')
    if (!title) continue

    const data: Record<string, unknown> = {
      title,
      author_names: get('Author'),
      isbn10: get('ISBN').replace(/[="]/g, ''),
      isbn13: get('ISBN13').replace(/[="]/g, ''),
      publisher: get('Publisher'),
      page_count: parseInt(get('Number of Pages')) || 0,
      publish_year: parseInt(get('Year Published')) || 0,
      average_rating: parseFloat(get('Average Rating')) || 0,
    }

    const book = await createBook(db, data)
    const bookId = book.id as number

    // Handle author
    const authorName = get('Author')
    if (authorName) {
      const author = await getOrCreateAuthor(db, authorName)
      await linkBookAuthor(db, bookId, author.id as number)
    }

    // Handle shelf
    const shelfName = get('Exclusive Shelf')
    if (shelfName) {
      const shelf = await db.prepare('SELECT id FROM shelves WHERE slug = ?').bind(shelfName).first<{ id: number }>()
      if (shelf) {
        await addBookToShelf(db, shelf.id, bookId)
        // Set dates
        const dateRead = get('Date Read')
        const dateAdded = get('Date Added')
        if (dateRead || dateAdded) {
          await updateShelfBook(db, shelf.id, bookId, {
            date_read: dateRead ? dateRead.replace(/\//g, '-') : null,
          })
        }
      }
    }

    // Handle rating/review
    const rating = parseInt(get('My Rating'))
    if (rating > 0) {
      await createReview(db, bookId, {
        rating,
        text: get('My Review'),
        finished_at: get('Date Read')?.replace(/\//g, '-') || null,
      })
    }

    imported++
  }
  return imported
}

function parseCSVText(text: string): string[][] {
  const rows: string[][] = []
  const lines = text.split('\n')
  for (const line of lines) {
    if (!line.trim()) continue
    const row: string[] = []
    let field = ''
    let inQuotes = false
    for (let i = 0; i < line.length; i++) {
      const ch = line[i]
      if (ch === '"') {
        if (inQuotes && line[i + 1] === '"') { field += '"'; i++ }
        else inQuotes = !inQuotes
      } else if (ch === ',' && !inQuotes) {
        row.push(field)
        field = ''
      } else {
        field += ch
      }
    }
    row.push(field)
    rows.push(row)
  }
  return rows
}
