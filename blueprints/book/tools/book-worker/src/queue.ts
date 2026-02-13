// Queue-based enrichment processor
// Replaces inline/waitUntil enrichment with durable Cloudflare Queue jobs

import type { Env } from './types'
import {
  ENRICH, enrichAuthor, enrichBook,
  importEditions, importSeriesBooks, importSimilarBooks,
  enrichGenre, importListBooks,
} from './enrich'
import { fetchOLAuthorWorks, fetchOLWork } from './openlibrary'
import { coverURL } from './config'
import * as db from './db'

// ---- Job Types ----

export type QueueJob =
  | { type: 'enrich-author'; authorId: number }
  | { type: 'import-author-works'; authorId: number; offset: number; batchSize: number }
  | { type: 'enrich-book'; bookId: number; force?: boolean }
  | { type: 'import-editions'; bookId: number }
  | { type: 'import-series'; bookId: number }
  | { type: 'import-similar'; bookId: number }
  | { type: 'enrich-genre'; genre: string }
  | { type: 'import-list-books'; listId: number }

// ---- Send Helper ----

export async function sendJob(queue: Queue, job: QueueJob): Promise<void> {
  await queue.send(job)
}

export async function sendJobs(queue: Queue, jobs: QueueJob[]): Promise<void> {
  if (jobs.length === 0) return
  await queue.sendBatch(jobs.map(body => ({ body })))
}

// ---- Queue Consumer ----

export async function handleQueue(batch: MessageBatch<QueueJob>, env: Env): Promise<void> {
  for (const msg of batch.messages) {
    try {
      await processJob(msg.body, env)
      msg.ack()
    } catch (err) {
      console.error(`[Queue] Job failed: ${msg.body.type}`, err)
      msg.retry()
    }
  }
}

// ---- Job Dispatcher ----

async function processJob(job: QueueJob, env: Env): Promise<void> {
  const { DB: d1, KV: kv, ENRICH_QUEUE: queue } = env
  console.log(`[Queue] Processing: ${job.type}`, JSON.stringify(job))

  switch (job.type) {
    case 'enrich-author': {
      const author = await db.getAuthor(d1, job.authorId)
      if (!author) break
      if (db.hasEnriched(author.enriched as number, ENRICH.AUTHOR_CORE)) break

      await enrichAuthor(d1, kv, job.authorId)

      // Chain: start importing works if author has ol_key
      const updated = await db.getAuthor(d1, job.authorId)
      if (updated && !db.hasEnriched(updated.enriched as number, ENRICH.AUTHOR_WORKS)) {
        await sendJob(queue, { type: 'import-author-works', authorId: job.authorId, offset: 0, batchSize: 200 })
      }
      break
    }

    case 'import-author-works': {
      await handleImportAuthorWorks(d1, kv, queue, job.authorId, job.offset, job.batchSize)
      break
    }

    case 'enrich-book': {
      const book = await db.getBook(d1, job.bookId)
      if (!book) break
      if (!job.force && db.hasEnriched(book.enriched as number, ENRICH.BOOK_CORE)) break

      await enrichBook(d1, kv, job.bookId, job.force)

      // Chain: enqueue editions + series if not yet done
      const updated = await db.getBook(d1, job.bookId)
      if (!updated) break
      const enriched = (updated.enriched as number) || 0
      const followUp: QueueJob[] = []
      if (!db.hasEnriched(enriched, ENRICH.BOOK_EDITIONS)) {
        followUp.push({ type: 'import-editions', bookId: job.bookId })
      }
      if (!db.hasEnriched(enriched, ENRICH.BOOK_SERIES)) {
        followUp.push({ type: 'import-series', bookId: job.bookId })
      }
      await sendJobs(queue, followUp)
      break
    }

    case 'import-editions': {
      const book = await db.getBook(d1, job.bookId)
      if (book && !db.hasEnriched(book.enriched as number, ENRICH.BOOK_EDITIONS)) {
        await importEditions(d1, kv, job.bookId)
      }
      break
    }

    case 'import-series': {
      const book = await db.getBook(d1, job.bookId)
      if (book && !db.hasEnriched(book.enriched as number, ENRICH.BOOK_SERIES)) {
        await importSeriesBooks(d1, kv, job.bookId)
      }
      break
    }

    case 'import-similar': {
      const book = await db.getBook(d1, job.bookId)
      if (book && !db.hasEnriched(book.enriched as number, ENRICH.BOOK_SIMILAR)) {
        await importSimilarBooks(d1, kv, job.bookId)
      }
      break
    }

    case 'enrich-genre': {
      const genreKey = `enriched:genre:${job.genre.toLowerCase().replace(/\s+/g, '_')}`
      const alreadyEnriched = await kv.get(genreKey)
      if (!alreadyEnriched) {
        const imported = await enrichGenre(d1, kv, job.genre)
        await kv.put(genreKey, String(imported))
      }
      break
    }

    case 'import-list-books': {
      const list = await db.getList(d1, job.listId)
      if (list && !db.hasEnriched(list.enriched as number, ENRICH.LIST_BOOKS)) {
        await importListBooks(d1, kv, job.listId)
      }
      break
    }
  }
}

// ---- Author Works Paginated Import ----

async function handleImportAuthorWorks(
  d1: D1Database, kv: KVNamespace, queue: Queue,
  authorId: number, offset: number, batchSize: number
): Promise<void> {
  const author = await db.getAuthor(d1, authorId)
  if (!author) return

  const olKey = author.ol_key as string
  if (!olKey) {
    // No OL key — mark as done
    if (!db.hasEnriched(author.enriched as number, ENRICH.AUTHOR_WORKS)) {
      await db.setEnriched(d1, 'authors', authorId, ENRICH.AUTHOR_WORKS)
    }
    return
  }

  const works = await fetchOLAuthorWorks(kv, olKey, batchSize, offset).catch(() => [])
  if (works.length === 0) {
    // Update KV progress as done
    await kv.put(`author-works-progress:${authorId}`, JSON.stringify({ offset, total: (author.works_count as number) || 0, done: true }))
    return
  }

  let imported = 0
  for (const work of works) {
    const workOLKey = work.key || ''
    if (!workOLKey || !work.title) continue

    // Dedup
    const existing = await db.getBookByOLKey(d1, workOLKey)
    if (existing) continue

    // Fetch OL work detail
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

  // Mark AUTHOR_WORKS flag after first batch so GET /authors/:id/books knows import started
  if (offset === 0 && !db.hasEnriched(author.enriched as number, ENRICH.AUTHOR_WORKS)) {
    await db.setEnriched(d1, 'authors', authorId, ENRICH.AUTHOR_WORKS)
  }

  const newOffset = offset + works.length
  const totalWorks = (author.works_count as number) || 0
  const done = works.length < batchSize || newOffset >= totalWorks

  // Update progress in KV
  await kv.put(`author-works-progress:${authorId}`, JSON.stringify({
    offset: newOffset,
    total: totalWorks,
    done,
    imported,
  }))

  // Self-chain: enqueue next batch if not done
  if (!done) {
    await sendJob(queue, {
      type: 'import-author-works',
      authorId,
      offset: newOffset,
      batchSize,
    })
  }
}
