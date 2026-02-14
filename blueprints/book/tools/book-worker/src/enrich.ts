// Auto-enrichment module for authors and books
// On-demand: fetch from OL + GR, update D1, return enriched data
// Uses bit flags to track what enrichment has been done (idempotent)

import { searchOLAuthors, fetchOLAuthor, fetchOLAuthorWorks, fetchOLWork, fetchOLSubjects, fetchOLEditions } from './openlibrary'
import { authorPhotoURL, coverURL } from './config'
import {
  getAuthor as grGetAuthor, searchBook as grSearchBook, getBook as grGetBook,
  getPopularLists, getList as grGetList,
  type GoodreadsAuthor, type GoodreadsBook,
} from './goodreads'
import * as db from './db'

// ---- Enrichment Bit Flags ----

export const ENRICH = {
  // Book flags
  BOOK_CORE: 1,       // GR data, description, reviews, quotes
  BOOK_GENRES: 2,     // Genre merge from OL + GR
  BOOK_EDITIONS: 4,   // Edition data from OL
  BOOK_SIMILAR: 8,    // Similar books imported
  BOOK_SERIES: 16,    // Series books imported
  // Author flags
  AUTHOR_CORE: 1,     // Bio, photo, dates, GR enrichment
  AUTHOR_WORKS: 2,    // OL works imported
  // List flags
  LIST_BOOKS: 1,      // Books imported from GR source
} as const

// ---- Genre Normalization ----

// Map of variant → canonical name. Covers common duplicates across OL + GR.
const GENRE_ALIASES: Record<string, string> = {
  'sci-fi': 'Science Fiction',
  'science fiction': 'Science Fiction',
  'scifi': 'Science Fiction',
  'sf': 'Science Fiction',
  'mysteries': 'Mystery',
  'mystery': 'Mystery',
  'mystery thriller': 'Mystery',
  'fantasies': 'Fantasy',
  'fantasy': 'Fantasy',
  'romances': 'Romance',
  'romance': 'Romance',
  'historical fiction': 'Historical Fiction',
  'historical': 'Historical Fiction',
  'nonfiction': 'Nonfiction',
  'non-fiction': 'Nonfiction',
  'non fiction': 'Nonfiction',
  'biography': 'Biography',
  'biographies': 'Biography',
  'autobiography': 'Memoir',
  'autobiographies': 'Memoir',
  'memoir': 'Memoir',
  'memoirs': 'Memoir',
  'thriller': 'Thriller',
  'thrillers': 'Thriller',
  'horror': 'Horror',
  'classics': 'Classics',
  'classic': 'Classics',
  'classic literature': 'Classics',
  'poetry': 'Poetry',
  'poems': 'Poetry',
  'young adult': 'Young Adult',
  'ya': 'Young Adult',
  'children': "Children's",
  "children's": "Children's",
  'childrens': "Children's",
  "children's literature": "Children's",
  'graphic novels': 'Graphic Novels',
  'graphic novel': 'Graphic Novels',
  'comics': 'Comics',
  'comic': 'Comics',
  'dystopia': 'Dystopian',
  'dystopian': 'Dystopian',
  'self-help': 'Self-Help',
  'self help': 'Self-Help',
  'psychology': 'Psychology',
  'philosophy': 'Philosophy',
  'religion': 'Religion',
  'spirituality': 'Spirituality',
  'travel': 'Travel',
  'adventure': 'Adventure',
  'adventures': 'Adventure',
  'crime': 'Crime',
  'true crime': 'True Crime',
  'war': 'War',
  'military': 'Military',
  'humor': 'Humor',
  'humour': 'Humor',
  'comedy': 'Humor',
  'funny': 'Humor',
  'cooking': 'Cooking',
  'cookbooks': 'Cooking',
  'food': 'Food',
  'art': 'Art',
  'music': 'Music',
  'science': 'Science',
  'nature': 'Nature',
  'environment': 'Environment',
  'politics': 'Politics',
  'political': 'Politics',
  'economics': 'Economics',
  'business': 'Business',
  'technology': 'Technology',
  'computers': 'Technology',
  'programming': 'Programming',
  'math': 'Mathematics',
  'mathematics': 'Mathematics',
  'history': 'History',
  'fiction': 'Fiction',
  'literary fiction': 'Literary Fiction',
  'literature': 'Literature',
  'suspense': 'Suspense',
  'paranormal': 'Paranormal',
  'supernatural': 'Supernatural',
  'urban fantasy': 'Urban Fantasy',
  'epic fantasy': 'Epic Fantasy',
  'high fantasy': 'High Fantasy',
  'dark fantasy': 'Dark Fantasy',
  'space opera': 'Space Opera',
  'steampunk': 'Steampunk',
  'cyberpunk': 'Cyberpunk',
  'magical realism': 'Magical Realism',
  'magic realism': 'Magical Realism',
  'contemporary': 'Contemporary',
  'modern': 'Contemporary',
  'medieval': 'Medieval',
  'short stories': 'Short Stories',
  'short story': 'Short Stories',
  'essays': 'Essays',
  'essay': 'Essays',
  'drama': 'Drama',
  'plays': 'Drama',
  'feminism': 'Feminism',
  'feminist': 'Feminism',
  'lgbtq': 'LGBTQ+',
  'lgbt': 'LGBTQ+',
  'queer': 'LGBTQ+',
}

/** Normalize a single genre string to its canonical form */
export function normalizeGenre(genre: string): string {
  const lower = genre.trim().toLowerCase()
  return GENRE_ALIASES[lower] || genre.trim()
}

/** Merge + normalize + deduplicate genre arrays from multiple sources */
export function mergeGenres(...sources: (string[] | undefined)[]): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const source of sources) {
    if (!source) continue
    for (const g of source) {
      const normalized = normalizeGenre(g)
      const key = normalized.toLowerCase()
      if (!key || seen.has(key)) continue
      seen.add(key)
      result.push(normalized)
    }
  }
  return result
}

// ---- Helpers ----

function extractAuthorID(url: string): string {
  if (!url) return ''
  const m = url.match(/\/author\/show\/(\d+)/)
  return m ? m[1] : ''
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

// ---- Author Enrichment ----

export async function enrichAuthor(d1: D1Database, kv: KVNamespace, authorId: number): Promise<Record<string, unknown> | null> {
  const author = await db.getAuthor(d1, authorId)
  if (!author) return null

  // Already enriched? Check flag
  if (db.hasEnriched(author.enriched as number, ENRICH.AUTHOR_CORE)) return author

  const name = author.name as string
  if (!name) return author

  const updates: Record<string, unknown> = {}

  // Step 1: Search OL by name to get ol_key
  let olKey = author.ol_key as string
  if (!olKey) {
    const olMatch = await searchOLAuthors(kv, name).catch(() => null)
    if (olMatch?.key) {
      olKey = olMatch.key
      updates.ol_key = olKey
      if (olMatch.birth_date) updates.birth_date = olMatch.birth_date
      if (olMatch.death_date) updates.death_date = olMatch.death_date
      if (olMatch.work_count) updates.works_count = olMatch.work_count
    }
  }

  // Step 2: Fetch OL author detail (bio, dates, photos, remote_ids.goodreads)
  if (olKey) {
    const olDetail = await fetchOLAuthor(kv, olKey).catch(() => null)
    if (olDetail) {
      if (olDetail.bio && !author.bio) updates.bio = olDetail.bio
      if (olDetail.birth_date && !author.birth_date) updates.birth_date = olDetail.birth_date
      if (olDetail.death_date && !author.death_date) updates.death_date = olDetail.death_date
      if (olDetail.photos?.length > 0 && !author.photo_url) {
        updates.photo_url = authorPhotoURL(olKey)
      }

      // Extract GR ID from OL remote_ids
      if (!author.source_id && olDetail.remote_ids?.goodreads) {
        updates.source_id = String(olDetail.remote_ids.goodreads)
      }
    }
  }

  // Step 3: If GR ID found, fetch richer data from Goodreads
  const grId = (updates.source_id || author.source_id) as string
  if (grId) {
    const grAuthor = await grGetAuthor(kv, grId).catch(() => null)
    if (grAuthor) {
      const grData = grAuthorToData(grAuthor)
      // GR overrides OL when both present (richer data)
      if (grData.bio) updates.bio = grData.bio
      if (grData.photo_url) updates.photo_url = grData.photo_url
      if (grData.birth_date) updates.birth_date = grData.birth_date
      if (grData.death_date) updates.death_date = grData.death_date
      if (grData.works_count) updates.works_count = grData.works_count
      if (grData.followers) updates.followers = grData.followers
      if (grData.genres) updates.genres = grData.genres
      if (grData.influences) updates.influences = grData.influences
      if (grData.website) updates.website = grData.website
      if (grData.source_id) updates.source_id = grData.source_id
    }
  }

  // Step 4: Persist + set flag
  if (Object.keys(updates).length > 0) {
    await db.updateAuthor(d1, authorId, updates)
  }
  await db.setEnriched(d1, 'authors', authorId, ENRICH.AUTHOR_CORE)
  return db.getAuthor(d1, authorId)
}

// ---- Author Works Import ----

export async function importAuthorWorks(d1: D1Database, kv: KVNamespace, authorId: number, limit: number = 50): Promise<number> {
  const author = await db.getAuthor(d1, authorId)
  if (!author) return 0

  // Already imported? Check flag
  if (db.hasEnriched(author.enriched as number, ENRICH.AUTHOR_WORKS)) return 0

  const olKey = author.ol_key as string
  if (!olKey) {
    // Mark as done even if no ol_key — avoid retrying
    await db.setEnriched(d1, 'authors', authorId, ENRICH.AUTHOR_WORKS)
    return 0
  }

  const deadline = Date.now() + 15_000 // 15s time budget

  const works = await fetchOLAuthorWorks(kv, olKey, limit).catch(() => [])
  let imported = 0

  for (const work of works) {
    if (Date.now() > deadline) break
    imported += await importOLWork(d1, kv, work, author.name as string, authorId)
  }

  await db.setEnriched(d1, 'authors', authorId, ENRICH.AUTHOR_WORKS)
  return imported
}

/** Import a single OL work entry into D1. Returns 1 if imported, 0 if skipped. */
async function importOLWork(d1: D1Database, kv: KVNamespace, work: { key: string; title: string }, authorName: string, authorId: number): Promise<number> {
  const workOLKey = work.key || ''
  if (!workOLKey || !work.title) return 0

  // Dedup: check if already in D1
  const existing = await db.getBookByOLKey(d1, workOLKey)
  if (existing) return 0

  // Fetch OL work detail for description/covers
  const olWork = await fetchOLWork(kv, workOLKey).catch(() => null)

  let description = ''
  if (olWork?.description) {
    description = typeof olWork.description === 'string' ? olWork.description : olWork.description?.value || ''
  }

  let coverUrl = ''
  if (olWork?.covers?.length > 0) {
    coverUrl = coverURL(olWork.covers[0])
  }

  const book = await db.createBook(d1, {
    ol_key: workOLKey,
    title: work.title,
    description,
    cover_url: coverUrl,
    author_names: authorName,
    subjects: olWork?.subjects?.slice(0, 10) || [],
    first_published: olWork?.first_publish_date || '',
  })

  await db.linkBookAuthor(d1, book.id as number, authorId)
  return 1
}

/** Background-friendly paginated import of author works. Imports one batch per call.
 *  Uses KV to track offset progress. Safe to call multiple times (idempotent per batch). */
export async function importAuthorWorksBackground(d1: D1Database, kv: KVNamespace, authorId: number, batchSize: number = 50): Promise<{ imported: number; done: boolean; total: number }> {
  const author = await db.getAuthor(d1, authorId)
  if (!author) return { imported: 0, done: true, total: 0 }

  const olKey = author.ol_key as string
  if (!olKey) return { imported: 0, done: true, total: 0 }

  // Get current offset from KV
  const progressKey = `author-works-progress:${authorId}`
  const progress = await kv.get(progressKey).then(v => v ? JSON.parse(v) : null) as { offset: number; total: number } | null
  const offset = progress?.offset || 0

  // If first batch is not done yet, skip background (let initial import finish)
  if (!db.hasEnriched(author.enriched as number, ENRICH.AUTHOR_WORKS)) {
    return { imported: 0, done: false, total: (author.works_count as number) || 0 }
  }

  const works = await fetchOLAuthorWorks(kv, olKey, batchSize, offset).catch(() => [])
  const totalWorks = works.length > 0 ? (works[0].size || 0) : (progress?.total || 0)

  if (works.length === 0) {
    // No more works — mark as fully done
    await kv.put(progressKey, JSON.stringify({ offset, total: totalWorks, done: true }))
    return { imported: 0, done: true, total: totalWorks }
  }

  const deadline = Date.now() + 25_000 // 25s budget for background
  let imported = 0
  for (const work of works) {
    if (Date.now() > deadline) break
    imported += await importOLWork(d1, kv, work, author.name as string, authorId)
  }

  const newOffset = offset + works.length
  const done = works.length < batchSize || newOffset >= totalWorks
  await kv.put(progressKey, JSON.stringify({ offset: newOffset, total: totalWorks, done }))

  return { imported, done, total: totalWorks }
}

// ---- Book Enrichment ----

export async function enrichBook(d1: D1Database, kv: KVNamespace, bookId: number, force: boolean = false): Promise<Record<string, unknown> | null> {
  let book = await db.getBook(d1, bookId)
  if (!book) return null

  const enriched = (book.enriched as number) || 0
  const needsCore = force || !db.hasEnriched(enriched, ENRICH.BOOK_CORE)
  const needsGenres = force || !db.hasEnriched(enriched, ENRICH.BOOK_GENRES)

  if (!needsCore && !needsGenres) return book

  const currentSubjects = (book.subjects as string[]) || []

  // Collect OL subjects if book has ol_key
  let olSubjects: string[] = []
  const olKey = book.ol_key as string
  if (olKey && needsGenres) {
    const olWork = await fetchOLWork(kv, olKey).catch(() => null)
    if (olWork?.subjects) {
      olSubjects = (olWork.subjects as string[]).slice(0, 20)
    }
  }

  // Try GR enrichment for core data + genres
  let gr: GoodreadsBook | null = null
  if (needsCore) {
    let sourceId = book.source_id as string
    if (!sourceId) {
      sourceId = await grSearchBook(kv, book.title as string).catch(() => '')
    }
    if (sourceId) {
      gr = await grGetBook(kv, sourceId).catch(() => null)
    }
  }

  if (gr) {
    // Full GR enrichment
    const enrichData = grBookToData(gr)
    // Merge genres: existing + OL subjects + GR genres (normalized, deduped)
    enrichData.subjects = mergeGenres(currentSubjects, olSubjects, gr.genres)
    book = await db.updateBook(d1, bookId, enrichData)

    // Create and link authors from GR data
    if (gr.author_name) {
      const names = gr.author_name.split(',').map((n: string) => n.trim()).filter(Boolean)
      const authorSourceId = extractAuthorID(gr.author_url)
      for (const name of names) {
        const author = await db.getOrCreateAuthor(d1, name, authorSourceId)
        await db.linkBookAuthor(d1, bookId, author.id as number)
      }
    }

    // Import reviews if none exist
    const reviews = await db.getBookReviews(d1, bookId, { page: 1, limit: 1 })
    if (reviews.total === 0) {
      for (const r of gr.reviews.slice(0, 20)) {
        await db.createReview(d1, bookId, {
          rating: r.rating, text: r.text, reviewer_name: r.reviewer_name,
          is_spoiler: r.is_spoiler, likes_count: r.likes_count, source: 'imported',
        })
      }
    }

    // Import quotes if none exist
    const quotes = await db.getBookQuotes(d1, bookId)
    if (quotes.length === 0) {
      for (const q of gr.quotes.slice(0, 20)) {
        await db.createQuote(d1, { book_id: bookId, author_name: q.author_name, text: q.text, likes_count: q.likes_count })
      }
    }

    // Set both core and genres flags
    await db.setEnriched(d1, 'books', bookId, ENRICH.BOOK_CORE | ENRICH.BOOK_GENRES)
  } else if (olSubjects.length > 0) {
    // No GR data but have OL subjects — merge with existing
    const merged = mergeGenres(currentSubjects, olSubjects)
    if (merged.length > currentSubjects.length) {
      await db.updateBook(d1, bookId, { subjects: merged })
    }
    await db.setEnriched(d1, 'books', bookId, ENRICH.BOOK_GENRES)
    // Mark core as attempted even with no GR data
    await db.setEnriched(d1, 'books', bookId, ENRICH.BOOK_CORE)
  } else {
    // No data found, still mark as attempted
    await db.setEnriched(d1, 'books', bookId, ENRICH.BOOK_CORE | ENRICH.BOOK_GENRES)
  }

  return db.getBook(d1, bookId)
}

// ---- Series Import ----

/** Import other books in the same series from OL/GR */
export async function importSeriesBooks(d1: D1Database, kv: KVNamespace, bookId: number): Promise<number> {
  const book = await db.getBook(d1, bookId)
  if (!book) return 0

  // Already imported?
  if (db.hasEnriched(book.enriched as number, ENRICH.BOOK_SERIES)) return 0

  const series = book.series as string
  if (!series) {
    await db.setEnriched(d1, 'books', bookId, ENRICH.BOOK_SERIES)
    return 0
  }

  // Check if we already have other books in this series
  const existing = await db.getBooksBySeries(d1, series, bookId)
  if (existing.length >= 3) {
    await db.setEnriched(d1, 'books', bookId, ENRICH.BOOK_SERIES)
    return 0
  }

  // Search OL for the series name to find related works
  const olData = await fetchOLSubjects(kv, series, 20).catch(() => null)
  let imported = 0
  const deadline = Date.now() + 10_000

  if (olData?.works) {
    for (const work of olData.works) {
      if (Date.now() > deadline) break
      if (!work.key || !work.title) continue

      const exists = await db.getBookByOLKey(d1, work.key)
      if (exists) continue

      const authorNames = work.authors?.map(a => a.name).join(', ') || ''
      const newBook = await db.createBook(d1, {
        ol_key: work.key,
        title: work.title,
        cover_url: work.cover_id ? coverURL(work.cover_id) : '',
        author_names: authorNames,
        series,
        subjects: mergeGenres(work.subject?.slice(0, 10)),
      })

      for (const a of (work.authors || [])) {
        if (!a.name) continue
        const author = await db.getOrCreateAuthor(d1, a.name)
        await db.linkBookAuthor(d1, newBook.id as number, author.id as number)
      }
      imported++
    }
  }

  await db.setEnriched(d1, 'books', bookId, ENRICH.BOOK_SERIES)
  return imported
}

// ---- Editions Import ----

/** Import edition data from OL (ISBNs, page counts, publishers) */
export async function importEditions(d1: D1Database, kv: KVNamespace, bookId: number): Promise<number> {
  const book = await db.getBook(d1, bookId)
  if (!book) return 0

  // Already imported?
  if (db.hasEnriched(book.enriched as number, ENRICH.BOOK_EDITIONS)) return 0

  const olKey = book.ol_key as string
  if (!olKey) {
    await db.setEnriched(d1, 'books', bookId, ENRICH.BOOK_EDITIONS)
    return 0
  }

  const editions = await fetchOLEditions(kv, olKey, 20).catch(() => [])
  if (editions.length === 0) {
    await db.setEnriched(d1, 'books', bookId, ENRICH.BOOK_EDITIONS)
    return 0
  }

  // Use first edition with data to fill missing fields on the book
  const updates: Record<string, unknown> = {}
  if (!book.isbn10) {
    for (const ed of editions) {
      if (ed.isbn_10?.length) { updates.isbn10 = ed.isbn_10[0]; break }
    }
  }
  if (!book.isbn13) {
    for (const ed of editions) {
      if (ed.isbn_13?.length) { updates.isbn13 = ed.isbn_13[0]; break }
    }
  }
  if (!book.page_count) {
    for (const ed of editions) {
      if (ed.number_of_pages) { updates.page_count = ed.number_of_pages; break }
    }
  }
  if (!book.publisher) {
    for (const ed of editions) {
      if (ed.publishers?.length) { updates.publisher = ed.publishers[0]; break }
    }
  }
  if (!book.cover_url) {
    for (const ed of editions) {
      if (ed.covers?.length) { updates.cover_url = coverURL(ed.covers[0]); break }
    }
  }
  if (!book.editions_count || (book.editions_count as number) < editions.length) {
    updates.editions_count = editions.length
  }

  if (Object.keys(updates).length > 0) {
    await db.updateBook(d1, bookId, updates)
  }

  await db.setEnriched(d1, 'books', bookId, ENRICH.BOOK_EDITIONS)
  return editions.length
}

// ---- Similar Books Import ----

/** Auto-import similar books based on overlapping subjects */
export async function importSimilarBooks(d1: D1Database, kv: KVNamespace, bookId: number): Promise<number> {
  const book = await db.getBook(d1, bookId)
  if (!book) return 0

  // Already imported?
  if (db.hasEnriched(book.enriched as number, ENRICH.BOOK_SIMILAR)) return 0

  const subjects = (book.subjects as string[]) || []
  if (subjects.length === 0) {
    await db.setEnriched(d1, 'books', bookId, ENRICH.BOOK_SIMILAR)
    return 0
  }

  // Pick top 2 subjects to search OL
  const topSubjects = subjects.slice(0, 2)
  let imported = 0
  const deadline = Date.now() + 10_000

  for (const subject of topSubjects) {
    if (Date.now() > deadline) break

    const olData = await fetchOLSubjects(kv, subject, 10).catch(() => null)
    if (!olData?.works) continue

    for (const work of olData.works) {
      if (Date.now() > deadline) break
      if (!work.key || !work.title) continue

      const exists = await db.getBookByOLKey(d1, work.key)
      if (exists) continue

      const authorNames = work.authors?.map(a => a.name).join(', ') || ''
      const newBook = await db.createBook(d1, {
        ol_key: work.key,
        title: work.title,
        cover_url: work.cover_id ? coverURL(work.cover_id) : '',
        author_names: authorNames,
        subjects: mergeGenres(work.subject?.slice(0, 10), [subject]),
      })

      for (const a of (work.authors || [])) {
        if (!a.name) continue
        const author = await db.getOrCreateAuthor(d1, a.name)
        await db.linkBookAuthor(d1, newBook.id as number, author.id as number)
      }
      imported++
      if (imported >= 10) break
    }
    if (imported >= 10) break
  }

  await db.setEnriched(d1, 'books', bookId, ENRICH.BOOK_SIMILAR)
  return imported
}

// ---- Genre Page Enrichment ----

/** Auto-import books for a genre from OL subjects API when genre page is sparse */
export async function enrichGenre(d1: D1Database, kv: KVNamespace, genre: string, limit: number = 20): Promise<number> {
  const olData = await fetchOLSubjects(kv, genre, limit).catch(() => null)
  if (!olData || olData.works.length === 0) return 0

  let imported = 0
  for (const work of olData.works) {
    if (!work.key || !work.title) continue
    const existing = await db.getBookByOLKey(d1, work.key)
    if (existing) continue

    const authorNames = work.authors?.map(a => a.name).join(', ') || ''
    const book = await db.createBook(d1, {
      ol_key: work.key,
      title: work.title,
      cover_url: work.cover_id ? coverURL(work.cover_id) : '',
      author_names: authorNames,
      subjects: mergeGenres(work.subject?.slice(0, 10), [genre]),
    })

    // Link authors
    for (const a of (work.authors || [])) {
      if (!a.name) continue
      const author = await db.getOrCreateAuthor(d1, a.name)
      await db.linkBookAuthor(d1, book.id as number, author.id as number)
    }
    imported++
  }
  return imported
}

// ---- Seed Popular Lists ----

/** Curated popular Goodreads lists to pre-import on first access */
const SEED_LISTS: { title: string; url: string; voter_count: number; tag: string; description: string }[] = [
  // Fiction
  { title: 'Best Books Ever', url: 'https://www.goodreads.com/list/show/1', voter_count: 250000, tag: 'fiction', description: 'The best books of all time, as voted by Goodreads members.' },
  { title: 'Best Books of the 21st Century', url: 'https://www.goodreads.com/list/show/5', voter_count: 80000, tag: 'fiction', description: 'The best books published since 2000.' },
  { title: 'Books That Everyone Should Read at Least Once', url: 'https://www.goodreads.com/list/show/264', voter_count: 120000, tag: 'fiction', description: 'Essential reading for every book lover.' },
  { title: '1001 Books You Must Read Before You Die', url: 'https://www.goodreads.com/list/show/952', voter_count: 30000, tag: 'fiction', description: 'The classic literary bucket list.' },
  // Genre
  { title: 'Best Science Fiction & Fantasy Books', url: 'https://www.goodreads.com/list/show/3', voter_count: 90000, tag: 'sci-fi & fantasy', description: 'Top-rated science fiction and fantasy.' },
  { title: 'Best Mystery & Thriller Books', url: 'https://www.goodreads.com/list/show/11', voter_count: 40000, tag: 'mystery & thriller', description: 'The most gripping mysteries and thrillers.' },
  { title: 'Best Romance Novels', url: 'https://www.goodreads.com/list/show/30', voter_count: 50000, tag: 'romance', description: 'The best love stories ever written.' },
  { title: 'Best Historical Fiction', url: 'https://www.goodreads.com/list/show/15', voter_count: 35000, tag: 'historical fiction', description: 'Fiction that brings history alive.' },
  { title: 'Best Horror Novels', url: 'https://www.goodreads.com/list/show/32', voter_count: 25000, tag: 'horror', description: 'The scariest books ever written.' },
  // Nonfiction
  { title: 'Best Nonfiction Books', url: 'https://www.goodreads.com/list/show/10', voter_count: 30000, tag: 'nonfiction', description: 'The best nonfiction across all subjects.' },
  { title: 'Best Memoirs and Autobiographies', url: 'https://www.goodreads.com/list/show/24', voter_count: 20000, tag: 'nonfiction', description: 'The most compelling life stories.' },
  { title: 'Best Science Books', url: 'https://www.goodreads.com/list/show/38', voter_count: 15000, tag: 'nonfiction', description: 'Popular science books that enlighten and inspire.' },
  // Classics & Literary
  { title: 'Best Classics', url: 'https://www.goodreads.com/list/show/6', voter_count: 40000, tag: 'classics', description: 'The greatest classic literature.' },
  { title: 'Best Dystopian and Post-Apocalyptic Fiction', url: 'https://www.goodreads.com/list/show/47', voter_count: 60000, tag: 'sci-fi & fantasy', description: 'Dark visions of the future.' },
  // Young Adult
  { title: 'Best Young Adult Books', url: 'https://www.goodreads.com/list/show/7', voter_count: 55000, tag: 'young adult', description: 'The best books for young adult readers.' },
  // Contemporary
  { title: 'Best Literary Fiction', url: 'https://www.goodreads.com/list/show/44', voter_count: 15000, tag: 'literary fiction', description: 'Beautifully written, thought-provoking literary works.' },
  { title: 'Best Books of the 2020s', url: 'https://www.goodreads.com/list/show/171084', voter_count: 10000, tag: 'fiction', description: 'The best books published in the 2020s.' },
]

/** Seed popular lists into D1 on first access. Skips duplicates by source_url. */
export async function seedPopularLists(d1: D1Database): Promise<number> {
  let seeded = 0
  for (const seed of SEED_LISTS) {
    const existing = await db.getListBySourceURL(d1, seed.url)
    if (existing) continue
    await db.createList(d1, {
      title: seed.title,
      description: seed.description,
      source_url: seed.url,
      voter_count: seed.voter_count,
      tag: seed.tag,
    })
    seeded++
  }
  return seeded
}

// ---- List Auto-Import ----

/** Auto-discover and import popular GR lists. Returns count of lists imported. */
export async function discoverLists(d1: D1Database, kv: KVNamespace, tag: string = ''): Promise<Record<string, unknown>[]> {
  const grLists = await getPopularLists(kv, tag).catch(() => [])
  if (grLists.length === 0) return []

  const imported: Record<string, unknown>[] = []
  for (const grSummary of grLists.slice(0, 10)) {
    // Dedup: check if already imported by source_url
    const sourceUrl = grSummary.url
    const existing = await db.getListBySourceURL(d1, sourceUrl)
    if (existing) {
      imported.push(existing)
      continue
    }

    const list = await db.createList(d1, {
      title: grSummary.title,
      source_url: sourceUrl,
      voter_count: grSummary.voter_count,
    })
    imported.push(list)
  }
  return imported
}

/** Import books into a list from its GR source. Time-bounded (15s). */
export async function importListBooks(d1: D1Database, kv: KVNamespace, listId: number): Promise<number> {
  const list = await db.getList(d1, listId)
  if (!list) return 0

  // Already imported? Check flag
  if (db.hasEnriched(list.enriched as number, ENRICH.LIST_BOOKS)) return 0

  const sourceUrl = list.source_url as string
  if (!sourceUrl || !sourceUrl.includes('goodreads.com')) {
    await db.setEnriched(d1, 'book_lists', listId, ENRICH.LIST_BOOKS)
    return 0
  }

  const grListId = extractGRListId(sourceUrl)
  if (!grListId) {
    await db.setEnriched(d1, 'book_lists', listId, ENRICH.LIST_BOOKS)
    return 0
  }

  const deadline = Date.now() + 15_000
  const grList = await grGetList(kv, grListId).catch(() => null)
  if (!grList) {
    await db.setEnriched(d1, 'book_lists', listId, ENRICH.LIST_BOOKS)
    return 0
  }

  let imported = 0
  for (const item of grList.books.slice(0, 50)) {
    if (Date.now() > deadline) break
    if (!item.goodreads_id) continue

    // Check if book exists by source_id
    let book = await db.getBookBySourceId(d1, item.goodreads_id)
    if (!book) {
      // Create a lightweight entry from list data
      book = await db.createBook(d1, {
        title: item.title,
        author_names: item.author_name,
        cover_url: item.cover_url,
        average_rating: item.average_rating,
        ratings_count: item.ratings_count,
        source_id: item.goodreads_id,
        source_url: item.url,
      })
      // Link author
      if (item.author_name) {
        const author = await db.getOrCreateAuthor(d1, item.author_name)
        await db.linkBookAuthor(d1, book.id as number, author.id as number)
      }
    }
    await db.addBookToList(d1, listId, book.id as number)
    imported++
  }

  await db.setEnriched(d1, 'book_lists', listId, ENRICH.LIST_BOOKS)
  return imported
}

function extractGRListId(url: string): string {
  const m = url.match(/\/list\/show\/(\d+)/)
  return m ? m[1] : ''
}
