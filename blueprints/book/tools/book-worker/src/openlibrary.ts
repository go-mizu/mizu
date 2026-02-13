import { OL_SEARCH_URL, OL_WORKS_URL, OL_AUTHORS_URL } from './config'
import type { OLSearchResult } from './types'

async function kvGet<T>(kv: KVNamespace, key: string): Promise<T | null> {
  const val = await kv.get(key)
  if (!val) return null
  return JSON.parse(val) as T
}

async function kvPut(kv: KVNamespace, key: string, data: unknown): Promise<void> {
  await kv.put(key, JSON.stringify(data))
}

export async function searchOL(kv: KVNamespace, query: string, limit: number = 20, offset: number = 0): Promise<{ docs: OLSearchResult[], numFound: number }> {
  const key = `ol:search:${query}:${limit}:${offset}`
  const cached = await kvGet<{ docs: OLSearchResult[], numFound: number }>(kv, key)
  if (cached) return cached

  const url = `${OL_SEARCH_URL}?q=${encodeURIComponent(query)}&limit=${limit}&offset=${offset}&fields=key,title,author_name,first_publish_year,cover_i,isbn,subject,publisher,language,ratings_average,ratings_count,edition_count`
  const resp = await fetch(url, { headers: { 'User-Agent': 'BookWorker/1.0' } })
  if (!resp.ok) return { docs: [], numFound: 0 }
  const data = await resp.json() as any
  const result = { docs: data.docs || [], numFound: data.numFound || 0 }

  await kvPut(kv, key, result)
  return result
}

export async function searchOLAuthors(kv: KVNamespace, name: string): Promise<{ key: string; name: string; birth_date?: string; death_date?: string; work_count?: number } | null> {
  const cacheKey = `ol:author-search:${name}`
  const cached = await kvGet<{ key: string; name: string; birth_date?: string; death_date?: string; work_count?: number }>(kv, cacheKey)
  if (cached) return cached

  const url = `${OL_AUTHORS_URL}?q=${encodeURIComponent(name)}&limit=5`
  const resp = await fetch(url, { headers: { 'User-Agent': 'BookWorker/1.0' } })
  if (!resp.ok) return null
  const data = await resp.json() as any
  const docs = data.docs || []
  if (docs.length === 0) return null

  const match = docs[0]
  const result = {
    key: match.key || '',
    name: match.name || '',
    birth_date: match.birth_date || '',
    death_date: match.death_date || '',
    work_count: match.work_count || 0,
  }

  await kvPut(kv, cacheKey, result)
  return result
}

export async function fetchOLAuthor(kv: KVNamespace, olKey: string): Promise<{
  bio: string; birth_date: string; death_date: string; photos: number[];
  remote_ids: { goodreads?: string }
} | null> {
  const cleanKey = olKey.replace('/authors/', '')
  const cacheKey = `ol:author:${cleanKey}`
  const cached = await kvGet<any>(kv, cacheKey)
  if (cached) return cached

  const resp = await fetch(`${OL_WORKS_URL}/authors/${cleanKey}.json`, { headers: { 'User-Agent': 'BookWorker/1.0' } })
  if (!resp.ok) return null
  const data = await resp.json() as any

  // OL returns bio as string OR { value: string }
  let bio = ''
  if (typeof data.bio === 'string') bio = data.bio
  else if (data.bio?.value) bio = data.bio.value

  const result = {
    bio,
    birth_date: data.birth_date || '',
    death_date: data.death_date || '',
    photos: data.photos || [],
    remote_ids: data.remote_ids || {},
  }

  await kvPut(kv, cacheKey, result)
  return result
}

export async function fetchOLAuthorWorks(kv: KVNamespace, olKey: string, limit: number = 50, offset: number = 0): Promise<{ key: string; title: string; size: number }[]> {
  const cleanKey = olKey.replace('/authors/', '')
  const cacheKey = `ol:author-works:${cleanKey}:${limit}:${offset}`
  const cached = await kvGet<{ key: string; title: string; size: number }[]>(kv, cacheKey)
  if (cached) return cached

  const resp = await fetch(`${OL_WORKS_URL}/authors/${cleanKey}/works.json?limit=${limit}&offset=${offset}`, { headers: { 'User-Agent': 'BookWorker/1.0' } })
  if (!resp.ok) return []
  const data = await resp.json() as any
  const totalSize = data.size || 0
  const entries = (data.entries || []).map((e: any) => ({
    key: e.key || '',
    title: e.title || '',
    size: totalSize,
  }))

  await kvPut(kv, cacheKey, entries)
  return entries
}

export async function fetchOLSubjects(kv: KVNamespace, subject: string, limit: number = 20): Promise<{
  name: string; work_count: number;
  works: { key: string; title: string; cover_id?: number; authors: { name: string }[]; subject?: string[] }[]
} | null> {
  const slug = subject.toLowerCase().replace(/\s+/g, '_')
  const cacheKey = `ol:subject:${slug}:${limit}`
  const cached = await kvGet<any>(kv, cacheKey)
  if (cached) return cached

  const url = `${OL_WORKS_URL}/subjects/${slug}.json?limit=${limit}`
  const resp = await fetch(url, { headers: { 'User-Agent': 'BookWorker/1.0' } })
  if (!resp.ok) return null
  const data = await resp.json() as any

  const result = {
    name: data.name || subject,
    work_count: data.work_count || 0,
    works: (data.works || []).map((w: any) => ({
      key: w.key || '',
      title: w.title || '',
      cover_id: w.cover_id || 0,
      authors: (w.authors || []).map((a: any) => ({ name: a.name || '' })),
      subject: w.subject || [],
    })),
  }

  await kvPut(kv, cacheKey, result)
  return result
}

export async function fetchOLEditions(kv: KVNamespace, olKey: string, limit: number = 20): Promise<{
  key: string; title: string; isbn_10?: string[]; isbn_13?: string[];
  number_of_pages?: number; covers?: number[]; publishers?: string[]; publish_date?: string;
}[]> {
  const cleanKey = olKey.replace('/works/', '')
  const cacheKey = `ol:editions:${cleanKey}:${limit}`
  const cached = await kvGet<any[]>(kv, cacheKey)
  if (cached) return cached

  const resp = await fetch(`${OL_WORKS_URL}/works/${cleanKey}/editions.json?limit=${limit}`, { headers: { 'User-Agent': 'BookWorker/1.0' } })
  if (!resp.ok) return []
  const data = await resp.json() as any
  const entries = (data.entries || []).map((e: any) => ({
    key: e.key || '',
    title: e.title || '',
    isbn_10: e.isbn_10 || [],
    isbn_13: e.isbn_13 || [],
    number_of_pages: e.number_of_pages || 0,
    covers: e.covers || [],
    publishers: e.publishers || [],
    publish_date: e.publish_date || '',
  }))

  await kvPut(kv, cacheKey, entries)
  return entries
}

export async function fetchOLWork(kv: KVNamespace, olKey: string): Promise<any> {
  const key = `ol:work:${olKey}`
  const cached = await kvGet<any>(kv, key)
  if (cached) return cached

  const resp = await fetch(`${OL_WORKS_URL}${olKey}.json`, { headers: { 'User-Agent': 'BookWorker/1.0' } })
  if (!resp.ok) return null
  const data = await resp.json()

  await kvPut(kv, key, data)
  return data
}
