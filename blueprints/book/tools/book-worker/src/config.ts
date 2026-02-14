export const OL_SEARCH_URL = 'https://openlibrary.org/search.json'
export const OL_AUTHORS_URL = 'https://openlibrary.org/search/authors.json'
export const OL_WORKS_URL = 'https://openlibrary.org'
export const OL_COVERS_URL = 'https://covers.openlibrary.org/b/id'
export const OL_AUTHOR_COVERS_URL = 'https://covers.openlibrary.org/a/olid'

export const DEFAULT_LIMIT = 20
export const MAX_LIMIT = 100

export function coverURL(coverId: number, size: 'S' | 'M' | 'L' = 'M'): string {
  if (!coverId) return ''
  return `${OL_COVERS_URL}/${coverId}-${size}.jpg`
}

export function authorPhotoURL(olKey: string, size: 'S' | 'M' | 'L' = 'M'): string {
  if (!olKey) return ''
  const cleanKey = olKey.replace('/authors/', '')
  return `${OL_AUTHOR_COVERS_URL}/${cleanKey}-${size}.jpg`
}
