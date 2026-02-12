import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ensureSchema } from '../db'
import * as db from '../db'
import { renderLayout, renderImportPage } from '../html'
import { searchOL, olResultToBook, enrichBookFromOL } from '../openlibrary'

const app = new Hono<HonoEnv>()

// Import/export page
app.get('/', async (c) => {
  await ensureSchema(c.env.DB)

  const message = c.req.query('message') || ''
  return c.html(renderLayout('Import / Export - Books', renderImportPage(), { flash: message }))
})

// Import from Open Library
app.post('/ol-import', async (c) => {
  await ensureSchema(c.env.DB)

  const body = await c.req.parseBody()
  const olKey = (body.ol_key as string || '').trim()
  const isbn = (body.isbn as string || '').trim()

  if (!olKey && !isbn) return c.redirect('/import?message=Please+provide+an+OL+key+or+ISBN', 302)

  try {
    let query = olKey || isbn
    const olResults = await searchOL(c.env.KV, query, 1)

    if (!olResults.docs || olResults.docs.length === 0) {
      return c.redirect('/import?message=No+results+found+on+Open+Library', 302)
    }

    const doc = olResults.docs[0]
    const partial = olResultToBook(doc)

    // Check if already exists
    if (partial.ol_key) {
      const existing = await db.getBookByOLKey(c.env.DB, partial.ol_key)
      if (existing) {
        return c.redirect(`/book/${existing.id}`, 302)
      }
    }

    // Enrich with full work data
    if (partial.ol_key) {
      const enriched = await enrichBookFromOL(c.env.KV, partial.ol_key)
      Object.assign(partial, enriched)
    }

    const bookId = await db.createBook(c.env.DB, partial)
    return c.redirect(`/book/${bookId}`, 302)
  } catch {
    return c.redirect('/import?message=Error+importing+from+Open+Library', 302)
  }
})

// Import CSV
app.post('/csv', async (c) => {
  await ensureSchema(c.env.DB)

  try {
    const body = await c.req.parseBody()
    const file = body.file

    if (!file || typeof file === 'string') {
      return c.redirect('/import?message=Please+upload+a+CSV+file', 302)
    }

    const csvText = await (file as File).text()
    const lines = csvText.split('\n').filter(l => l.trim())
    if (lines.length < 2) {
      return c.redirect('/import?message=CSV+file+is+empty+or+invalid', 302)
    }

    // Parse header
    const headers = parseCSVRow(lines[0]).map(h => h.toLowerCase().trim())
    const titleIdx = headers.indexOf('title')
    const authorIdx = headers.indexOf('author')
    const isbnIdx = headers.indexOf('isbn13') !== -1 ? headers.indexOf('isbn13') : headers.indexOf('isbn')
    const ratingIdx = headers.indexOf('my rating') !== -1 ? headers.indexOf('my rating') : headers.indexOf('rating')
    const shelfIdx = headers.indexOf('exclusive shelf') !== -1 ? headers.indexOf('exclusive shelf') : headers.indexOf('shelf')
    const pagesIdx = headers.indexOf('number of pages') !== -1 ? headers.indexOf('number of pages') : headers.indexOf('pages')
    const yearIdx = headers.indexOf('year published') !== -1 ? headers.indexOf('year published') : headers.indexOf('publish_year')
    const dateReadIdx = headers.indexOf('date read')
    const dateAddedIdx = headers.indexOf('date added')

    if (titleIdx === -1) {
      return c.redirect('/import?message=CSV+must+have+a+Title+column', 302)
    }

    let imported = 0
    const shelves = await db.getShelves(c.env.DB)

    for (let i = 1; i < lines.length; i++) {
      const cols = parseCSVRow(lines[i])
      const title = cols[titleIdx]?.trim()
      if (!title) continue

      const bookId = await db.createBook(c.env.DB, {
        title,
        author_names: authorIdx >= 0 ? (cols[authorIdx] || '').trim() : '',
        isbn13: isbnIdx >= 0 ? (cols[isbnIdx] || '').replace(/[="]/g, '').trim() : '',
        page_count: pagesIdx >= 0 ? parseInt(cols[pagesIdx], 10) || 0 : 0,
        publish_year: yearIdx >= 0 ? parseInt(cols[yearIdx], 10) || 0 : 0,
      })

      // Add to shelf if specified
      if (shelfIdx >= 0 && cols[shelfIdx]) {
        const shelfSlug = cols[shelfIdx].trim().toLowerCase().replace(/\s+/g, '-')
        const shelf = shelves.find(s => s.slug === shelfSlug)
        if (shelf) {
          await db.addBookToShelf(c.env.DB, shelf.id, bookId)
        }
      }

      // Add rating as review
      if (ratingIdx >= 0) {
        const rating = parseInt(cols[ratingIdx], 10)
        if (rating > 0 && rating <= 5) {
          await db.createReview(c.env.DB, {
            book_id: bookId,
            rating,
            source: 'csv-import',
          })
        }
      }

      imported++
    }

    return c.redirect(`/import?message=Successfully+imported+${imported}+books`, 302)
  } catch {
    return c.redirect('/import?message=Error+processing+CSV+file', 302)
  }
})

// Simple CSV row parser that handles quoted fields
function parseCSVRow(row: string): string[] {
  const result: string[] = []
  let current = ''
  let inQuotes = false

  for (let i = 0; i < row.length; i++) {
    const ch = row[i]
    if (ch === '"') {
      if (inQuotes && row[i + 1] === '"') {
        current += '"'
        i++
      } else {
        inQuotes = !inQuotes
      }
    } else if (ch === ',' && !inQuotes) {
      result.push(current)
      current = ''
    } else {
      current += ch
    }
  }
  result.push(current)
  return result
}

export default app
