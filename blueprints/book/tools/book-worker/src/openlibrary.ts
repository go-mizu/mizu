import { OL_SEARCH_URL, OL_WORKS_URL, OL_AUTHORS_URL, CACHE_TTL, coverURL } from './config'
import type { OLSearchResult, Book, Author } from './types'

async function cached<T>(kv: KVNamespace, key: string, ttl: number, fn: () => Promise<T>): Promise<T> {
  const hit = await kv.get(key)
  if (hit) return JSON.parse(hit) as T
  const result = await fn()
  await kv.put(key, JSON.stringify(result), { expirationTtl: ttl })
  return result
}

export async function searchOL(kv: KVNamespace, query: string, limit: number = 20, offset: number = 0): Promise<{ docs: OLSearchResult[], numFound: number }> {
  return cached(kv, `ol:search:${query}:${limit}:${offset}`, CACHE_TTL.OL_SEARCH, async () => {
    const url = `${OL_SEARCH_URL}?q=${encodeURIComponent(query)}&limit=${limit}&offset=${offset}&fields=key,title,author_name,first_publish_year,cover_i,isbn,subject,publisher,language,ratings_average,ratings_count,edition_count`
    const resp = await fetch(url, { headers: { 'User-Agent': 'BookWorker/1.0' } })
    if (!resp.ok) return { docs: [], numFound: 0 }
    const data = await resp.json() as any
    return { docs: data.docs || [], numFound: data.numFound || 0 }
  })
}

export async function fetchOLWork(kv: KVNamespace, olKey: string): Promise<any> {
  return cached(kv, `ol:work:${olKey}`, CACHE_TTL.OL_WORK, async () => {
    const resp = await fetch(`${OL_WORKS_URL}${olKey}.json`, { headers: { 'User-Agent': 'BookWorker/1.0' } })
    if (!resp.ok) return null
    return await resp.json()
  })
}

export async function fetchOLAuthor(kv: KVNamespace, authorKey: string): Promise<any> {
  return cached(kv, `ol:author:${authorKey}`, CACHE_TTL.OL_AUTHOR, async () => {
    const resp = await fetch(`${OL_AUTHORS_URL}/${authorKey}.json`, { headers: { 'User-Agent': 'BookWorker/1.0' } })
    if (!resp.ok) return null
    return await resp.json()
  })
}

// Convert OL search result to a partial Book for import
export function olResultToBook(doc: OLSearchResult): Partial<Book> {
  const desc = ''
  return {
    ol_key: doc.key || '',
    title: doc.title || '',
    author_names: (doc.author_name || []).join(', '),
    cover_id: doc.cover_i || 0,
    cover_url: doc.cover_i ? coverURL(doc.cover_i, 'M') : '',
    isbn13: (doc.isbn || []).find(i => i.length === 13) || '',
    isbn10: (doc.isbn || []).find(i => i.length === 10) || '',
    publisher: (doc.publisher || [])[0] || '',
    publish_year: doc.first_publish_year || 0,
    language: (doc.language || [])[0] || 'en',
    subjects_json: JSON.stringify((doc.subject || []).slice(0, 10)),
    average_rating: doc.ratings_average || 0,
    ratings_count: doc.ratings_count || 0,
    editions_count: doc.edition_count || 0,
    description: desc,
  }
}

// Enrich a book with full OL work data
export async function enrichBookFromOL(kv: KVNamespace, olKey: string): Promise<Partial<Book>> {
  const work = await fetchOLWork(kv, olKey)
  if (!work) return {}
  const desc = typeof work.description === 'string' ? work.description : work.description?.value || ''
  const subjects = (work.subjects || []).slice(0, 10)
  const characters = (work.subject_people || []).slice(0, 10)
  const settings = (work.subject_places || []).slice(0, 10)
  return {
    description: desc,
    subjects_json: JSON.stringify(subjects),
    characters_json: JSON.stringify(characters),
    settings_json: JSON.stringify(settings),
    first_published: work.first_publish_date || '',
  }
}
