// Auto-enrichment module for authors and books
// On-demand: fetch from OL + GR, update D1, return enriched data

import { searchOLAuthors, fetchOLAuthor, fetchOLAuthorWorks, fetchOLWork, fetchOLSubjects } from './openlibrary'
import { authorPhotoURL, coverURL } from './config'
import {
  getAuthor as grGetAuthor, searchBook as grSearchBook, getBook as grGetBook,
  getPopularLists, getList as grGetList, parseGoodreadsURL,
  type GoodreadsAuthor, type GoodreadsBook,
} from './goodreads'
import * as db from './db'

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

  // Already enriched? (has bio AND photo)
  if (author.bio && author.photo_url) return author

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

  // Step 4: Persist
  if (Object.keys(updates).length > 0) {
    return db.updateAuthor(d1, authorId, updates)
  }
  return author
}

// ---- Author Works Import ----

export async function importAuthorWorks(d1: D1Database, kv: KVNamespace, authorId: number, limit: number = 50): Promise<number> {
  const author = await db.getAuthor(d1, authorId)
  if (!author) return 0

  const olKey = author.ol_key as string
  if (!olKey) return 0

  const deadline = Date.now() + 15_000 // 15s time budget

  const works = await fetchOLAuthorWorks(kv, olKey, limit).catch(() => [])
  let imported = 0

  for (const work of works) {
    if (Date.now() > deadline) break

    const workOLKey = work.key || ''
    if (!workOLKey || !work.title) continue

    // Dedup: check if already in D1
    const existing = await db.getBookByOLKey(d1, workOLKey)
    if (existing) continue

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
      author_names: author.name as string,
      subjects: olWork?.subjects?.slice(0, 10) || [],
      first_published: olWork?.first_publish_date || '',
    })

    await db.linkBookAuthor(d1, book.id as number, authorId)
    imported++
  }

  return imported
}

// ---- Book Enrichment ----

export async function enrichBook(d1: D1Database, kv: KVNamespace, bookId: number, force: boolean = false): Promise<Record<string, unknown> | null> {
  let book = await db.getBook(d1, bookId)
  if (!book) return null

  // Already enriched? (has description AND source_id)
  if (!force && book.description && book.source_id) return book

  // Step 1: Find on Goodreads by title
  let sourceId = book.source_id as string
  if (!sourceId) {
    sourceId = await grSearchBook(kv, book.title as string).catch(() => '')
    if (!sourceId) return book // Can't find on GR, return as-is
  }

  // Step 2: Fetch GR book
  const gr = await grGetBook(kv, sourceId).catch(() => null)
  if (!gr) return book

  // Step 3: Update book with full GR data
  const enrichData = grBookToData(gr)
  book = await db.updateBook(d1, bookId, enrichData)

  // Step 4: Create and link authors from GR data
  if (gr.author_name) {
    const names = gr.author_name.split(',').map((n: string) => n.trim()).filter(Boolean)
    const authorSourceId = extractAuthorID(gr.author_url)
    for (const name of names) {
      const author = await db.getOrCreateAuthor(d1, name, authorSourceId)
      await db.linkBookAuthor(d1, bookId, author.id as number)
    }
  }

  // Step 5: Import reviews if none exist
  const reviews = await db.getBookReviews(d1, bookId, { page: 1, limit: 1 })
  if (reviews.total === 0) {
    for (const r of gr.reviews.slice(0, 20)) {
      await db.createReview(d1, bookId, {
        rating: r.rating, text: r.text, reviewer_name: r.reviewer_name,
        is_spoiler: r.is_spoiler, likes_count: r.likes_count, source: 'imported',
      })
    }
  }

  // Step 6: Import quotes if none exist
  const quotes = await db.getBookQuotes(d1, bookId)
  if (quotes.length === 0) {
    for (const q of gr.quotes.slice(0, 20)) {
      await db.createQuote(d1, { book_id: bookId, author_name: q.author_name, text: q.text, likes_count: q.likes_count })
    }
  }

  return db.getBook(d1, bookId)
}
