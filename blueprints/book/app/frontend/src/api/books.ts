import { api } from './client'
import type {
  Book, Author, Shelf, Review, ReadingProgress,
  ReadingChallenge, BookList, Quote, FeedItem,
  ReadingStats, SearchResult, Genre,
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
  getTrending: (limit = 20) => api.get<Book[]>(`/api/books/trending?limit=${limit}`),

  // Authors
  searchAuthors: (q: string) =>
    api.get<Author[]>(`/api/authors/search?q=${encodeURIComponent(q)}`),
  getAuthor: (id: number) => api.get<Author>(`/api/authors/${id}`),
  getAuthorBooks: (id: number) => api.get<Book[]>(`/api/authors/${id}/books`),

  // Shelves
  getShelves: () => api.get<Shelf[]>('/api/shelves'),
  createShelf: (shelf: Partial<Shelf>) => api.post<Shelf>('/api/shelves', shelf),
  updateShelf: (id: number, shelf: Partial<Shelf>) => api.put<Shelf>(`/api/shelves/${id}`, shelf),
  deleteShelf: (id: number) => api.del<void>(`/api/shelves/${id}`),
  getShelfBooks: async (id: number, page = 1, limit = 20) => {
    const data = await api.get<{ books: Book[]; total: number; page: number }>(`/api/shelves/${id}/books?page=${page}&limit=${limit}`)
    return { books: data.books || [], total_count: data.total, page: data.page, page_size: limit } as SearchResult
  },
  addToShelf: (shelfId: number, bookId: number) =>
    api.post<void>(`/api/shelves/${shelfId}/books`, { book_id: bookId }),
  removeFromShelf: (shelfId: number, bookId: number) =>
    api.del<void>(`/api/shelves/${shelfId}/books/${bookId}`),

  // Reviews
  getReviews: async (bookId: number) => {
    const data = await api.get<{ reviews: Review[]; total: number }>(`/api/books/${bookId}/reviews`)
    return data.reviews || []
  },
  createReview: (bookId: number, review: Partial<Review>) =>
    api.post<Review>(`/api/books/${bookId}/reviews`, review),
  updateReview: (id: number, review: Partial<Review>) =>
    api.put<Review>(`/api/reviews/${id}`, review),
  deleteReview: (id: number) => api.del<void>(`/api/reviews/${id}`),

  // Reading Progress
  getProgress: (bookId: number) =>
    api.get<ReadingProgress[]>(`/api/books/${bookId}/progress`),
  updateProgress: (bookId: number, progress: Partial<ReadingProgress>) =>
    api.post<ReadingProgress>(`/api/books/${bookId}/progress`, progress),

  // Browse
  getGenres: () => api.get<Genre[]>('/api/genres'),
  getBooksByGenre: (genre: string, page = 1) =>
    api.get<SearchResult>(`/api/genres/${encodeURIComponent(genre)}/books?page=${page}`),
  getNewReleases: (limit = 20) => api.get<Book[]>(`/api/browse/new-releases?limit=${limit}`),
  getPopular: (limit = 20) => api.get<Book[]>(`/api/browse/popular?limit=${limit}`),

  // Challenge
  getChallenge: (year?: number) => {
    const y = year || new Date().getFullYear()
    return api.get<ReadingChallenge>(`/api/challenge/${y}`)
  },
  setChallenge: (year: number, goal: number) =>
    api.post<ReadingChallenge>('/api/challenge', { year, goal }),

  // Lists
  getLists: async () => {
    const data = await api.get<{ lists: BookList[]; total: number }>('/api/lists')
    return data.lists || []
  },
  createList: (list: Partial<BookList>) => api.post<BookList>('/api/lists', list),
  getList: (id: number) => api.get<BookList>(`/api/lists/${id}`),
  addToList: (listId: number, bookId: number) =>
    api.post<void>(`/api/lists/${listId}/books`, { book_id: bookId }),
  voteList: (listId: number, bookId: number) =>
    api.post<void>(`/api/lists/${listId}/vote/${bookId}`),

  // Quotes
  getQuotes: (page = 1) => api.get<Quote[]>(`/api/quotes?page=${page}`),
  createQuote: (quote: Partial<Quote>) => api.post<Quote>('/api/quotes', quote),
  getBookQuotes: (bookId: number) => api.get<Quote[]>(`/api/books/${bookId}/quotes`),

  // Stats
  getStats: () => api.get<ReadingStats>('/api/stats'),
  getStatsByYear: (year: number) => api.get<ReadingStats>(`/api/stats/${year}`),

  // Feed
  getFeed: (limit = 20) => api.get<FeedItem[]>(`/api/feed?limit=${limit}`),

  // Goodreads
  importGoodreads: (url: string) => api.post<Book>('/api/import-goodreads', { url }),
  getGoodreadsBook: (id: string) => api.get<Book>(`/api/goodreads/${id}`),

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
