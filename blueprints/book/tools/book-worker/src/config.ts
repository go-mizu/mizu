export const OL_SEARCH_URL = 'https://openlibrary.org/search.json'
export const OL_WORKS_URL = 'https://openlibrary.org'
export const OL_AUTHORS_URL = 'https://openlibrary.org/authors'
export const OL_COVERS_URL = 'https://covers.openlibrary.org/b/id'

export const CACHE_TTL = {
  OL_SEARCH: 300,      // 5 min
  OL_WORK: 3600,       // 1 hour
  OL_AUTHOR: 3600,     // 1 hour
  TRENDING: 300,       // 5 min
  GENRES: 600,         // 10 min
  STATS: 120,          // 2 min
}

export const DEFAULT_LIMIT = 20
export const MAX_LIMIT = 100

export function coverURL(coverId: number, size: 'S' | 'M' | 'L' = 'M'): string {
  if (!coverId) return ''
  return `${OL_COVERS_URL}/${coverId}-${size}.jpg`
}
