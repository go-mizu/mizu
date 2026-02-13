import { api } from './client'
import type {
  Book, Author, Shelf, ShelfBook, Review, ReadingProgress,
  ReadingChallenge, BookList, Quote, FeedItem,
  ReadingStats, SearchResult, Genre, SourceListSummary, ReviewQuery, ReviewComment,
  BookNote,
} from '../types'

export const booksApi = {
  // Books
  search: (q: string, page = 1, limit = 20) =>
    api.get<SearchResult>(`/api/books/search?q=${encodeURIComponent(q)}&page=${page}&limit=${limit}`),
  getBook: (id: number) => api.get<Book>(`/api/books/${id}`),
  createBook: (book: Partial<Book>) => api.post<Book>('/api/books', book),
  getSimilar: async (id: number, limit = 10) => {
    const data = await api.get<Book[] | null>(`/api/books/${id}/similar?limit=${limit}`)
    return data || []
  },
  getTrending: async (limit = 20) => {
    const data = await api.get<Book[] | null>(`/api/books/trending?limit=${limit}`)
    return data || []
  },

  // Authors
  searchAuthors: (q: string) =>
    api.get<Author[]>(`/api/authors/search?q=${encodeURIComponent(q)}`),
  getAuthor: (id: number) => api.get<Author>(`/api/authors/${id}`),
  getAuthorBooks: async (id: number) => {
    const data = await api.get<{ books: Book[]; has_more: boolean; total: number } | Book[] | null>(`/api/authors/${id}/books`)
    if (!data) return { books: [], has_more: false, total: 0 }
    if (Array.isArray(data)) return { books: data, has_more: false, total: data.length }
    return { books: data.books || [], has_more: data.has_more || false, total: data.total || 0 }
  },

  // Shelves
  getShelves: () => api.get<Shelf[]>('/api/shelves'),
  createShelf: (shelf: Partial<Shelf>) => api.post<Shelf>('/api/shelves', shelf),
  updateShelf: (id: number, shelf: Partial<Shelf>) => api.put<Shelf>(`/api/shelves/${id}`, shelf),
  deleteShelf: (id: number) => api.del<void>(`/api/shelves/${id}`),
  getShelfBooks: async (id: number, page = 1, limit = 20) => {
    const data = await api.get<{ books: ShelfBook[]; total: number; page: number }>(`/api/shelves/${id}/books?page=${page}&limit=${limit}`)
    const shelfBooks = data.books || []
    return {
      books: shelfBooks.map(sb => sb.book!),
      shelfBooks,
      total_count: data.total,
      page: data.page,
      page_size: limit,
    }
  },
  addToShelf: (shelfId: number, bookId: number) =>
    api.post<void>(`/api/shelves/${shelfId}/books`, { book_id: bookId }),
  removeFromShelf: (shelfId: number, bookId: number) =>
    api.del<void>(`/api/shelves/${shelfId}/books/${bookId}`),
  updateShelfBook: (shelfId: number, bookId: number, data: { date_started?: string; date_read?: string; read_count?: number }) =>
    api.put<void>(`/api/shelves/${shelfId}/books/${bookId}`, data),

  // Reviews
  getReviews: async (bookId: number, query?: Partial<ReviewQuery>) => {
    const params = new URLSearchParams()
    if (query?.page) params.set('page', String(query.page))
    if (query?.limit) params.set('limit', String(query.limit))
    if (query?.sort) params.set('sort', query.sort)
    if (query?.rating) params.set('rating', String(query.rating))
    if (query?.source) params.set('source', query.source)
    if (typeof query?.has_text === 'boolean') params.set('has_text', String(query.has_text))
    if (query?.q) params.set('q', query.q)
    if (query?.include_spoilers) params.set('include_spoilers', String(query.include_spoilers))
    const qs = params.toString()
    return api.get<{ reviews: Review[]; total: number }>(`/api/books/${bookId}/reviews${qs ? `?${qs}` : ''}`)
  },
  createReview: (bookId: number, review: Partial<Review>) =>
    api.post<Review>(`/api/books/${bookId}/reviews`, review),
  updateReview: (id: number, review: Partial<Review>) =>
    api.put<Review>(`/api/reviews/${id}`, review),
  deleteReview: (id: number) => api.del<void>(`/api/reviews/${id}`),
  likeReview: (id: number) =>
    api.post<{ likes_count: number }>(`/api/reviews/${id}/like`, {}),
  getReviewComments: (reviewId: number, page = 1, limit = 20) =>
    api.get<{ comments: ReviewComment[]; total: number }>(`/api/reviews/${reviewId}/comments?page=${page}&limit=${limit}`),
  createReviewComment: (reviewId: number, comment: Partial<ReviewComment>) =>
    api.post<ReviewComment>(`/api/reviews/${reviewId}/comments`, comment),
  deleteReviewComment: (reviewId: number, commentId: number) =>
    api.del<void>(`/api/reviews/${reviewId}/comments/${commentId}`),

  // Reading Progress
  getProgress: (bookId: number) =>
    api.get<ReadingProgress[]>(`/api/books/${bookId}/progress`),
  updateProgress: (bookId: number, progress: Partial<ReadingProgress>) =>
    api.post<ReadingProgress>(`/api/books/${bookId}/progress`, progress),

  // Browse
  getGenres: async () => {
    const data = await api.get<Genre[] | null>('/api/genres')
    return data || []
  },
  getBooksByGenre: (genre: string, page = 1) =>
    api.get<SearchResult>(`/api/genres/${encodeURIComponent(genre)}/books?page=${page}`),
  getNewReleases: async (limit = 20) => {
    const data = await api.get<Book[] | null>(`/api/browse/new-releases?limit=${limit}`)
    return data || []
  },
  getPopular: async (limit = 20) => {
    const data = await api.get<Book[] | null>(`/api/browse/popular?limit=${limit}`)
    return data || []
  },

  // Challenge
  getChallenge: (year?: number) => {
    const y = year || new Date().getFullYear()
    return api.get<ReadingChallenge>(`/api/challenge/${y}`)
  },
  setChallenge: (year: number, goal: number) =>
    api.post<ReadingChallenge>('/api/challenge', { year, goal }),

  // Lists
  getLists: async (tag?: string) => {
    const q = tag?.trim() ? `?tag=${encodeURIComponent(tag.trim())}` : ''
    const data = await api.get<{ lists: BookList[]; total: number; tags: string[] }>(`/api/lists${q}`)
    return { lists: data.lists || [], tags: data.tags || [] }
  },
  createList: (list: Partial<BookList>) => api.post<BookList>('/api/lists', list),
  getList: (id: number) => api.get<BookList>(`/api/lists/${id}`),
  deleteList: (id: number) => api.del<void>(`/api/lists/${id}`),
  searchLists: async (q: string) => {
    const data = await api.get<BookList[] | null>(`/api/lists/search?q=${encodeURIComponent(q)}`)
    return data || []
  },
  addToList: (listId: number, bookId: number) =>
    api.post<void>(`/api/lists/${listId}/books`, { book_id: bookId }),
  voteList: (listId: number, bookId: number) =>
    api.post<void>(`/api/lists/${listId}/vote/${bookId}`),

  // Quotes
  getQuotes: async (page = 1) => {
    const data = await api.get<Quote[] | null>(`/api/quotes?page=${page}`)
    return data || []
  },
  createQuote: (quote: Partial<Quote>) => api.post<Quote>('/api/quotes', quote),
  getBookQuotes: async (bookId: number) => {
    const data = await api.get<Quote[] | null>(`/api/books/${bookId}/quotes`)
    return data || []
  },

  // Stats
  getStats: () => api.get<ReadingStats>('/api/stats'),
  getStatsByYear: (year: number) => api.get<ReadingStats>(`/api/stats/${year}`),

  // Feed
  getFeed: async (limit = 20) => {
    const data = await api.get<FeedItem[] | null>(`/api/feed?limit=${limit}`)
    return data || []
  },

  // External source sync
  importSourceBook: (url: string) => api.post<Book>('/api/import-source', { url }),
  getSourceBook: (id: string) => api.get<Book>(`/api/source/${id}`),
  importSourceAuthor: (id: string) => api.get<Author>(`/api/source/author/${id}`),
  browseSourceLists: (tag?: string) => {
    const q = tag?.trim() ? `?tag=${encodeURIComponent(tag.trim())}` : ''
    return api.get<SourceListSummary[]>(`/api/source/lists${q}`)
  },
  importSourceList: (url: string) => api.post<BookList>('/api/import-source-list', { url }),
  enrichBook: (id: number) => api.post<Book>(`/api/books/${id}/enrich`, {}),

  // Notes
  getNote: async (bookId: number) => {
    const data = await api.get<BookNote>(`/api/books/${bookId}/notes`)
    return data
  },
  saveNote: (bookId: number, text: string) =>
    api.post<BookNote>(`/api/books/${bookId}/notes`, { text }),
  deleteNote: (bookId: number) =>
    api.del<void>(`/api/books/${bookId}/notes`),

  // Import/Export
  importCSV: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return fetch('/api/import/csv', { method: 'POST', body: form }).then(r => r.json())
  },
  exportCSV: () => {
    window.location.href = '/api/export/csv'
  },
}
