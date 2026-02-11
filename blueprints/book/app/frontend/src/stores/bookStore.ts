import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { Book, Shelf, SearchResult } from '../types'

interface BookState {
  // Search
  query: string
  results: SearchResult | null
  isLoading: boolean
  error: string | null

  // Shelves
  shelves: Shelf[]

  // Current book
  currentBook: Book | null

  // Recently viewed
  recentBooks: Book[]

  // Actions
  setQuery: (q: string) => void
  setResults: (r: SearchResult | null) => void
  setLoading: (v: boolean) => void
  setError: (e: string | null) => void
  setShelves: (s: Shelf[]) => void
  setCurrentBook: (b: Book | null) => void
  addRecentBook: (b: Book) => void
}

export const useBookStore = create<BookState>()(
  persist(
    (set) => ({
      query: '',
      results: null,
      isLoading: false,
      error: null,
      shelves: [],
      currentBook: null,
      recentBooks: [],

      setQuery: (query) => set({ query }),
      setResults: (results) => set({ results }),
      setLoading: (isLoading) => set({ isLoading }),
      setError: (error) => set({ error }),
      setShelves: (shelves) => set({ shelves }),
      setCurrentBook: (currentBook) => set({ currentBook }),
      addRecentBook: (book) =>
        set((state) => ({
          recentBooks: [book, ...state.recentBooks.filter((b) => b.id !== book.id)].slice(0, 20),
        })),
    }),
    {
      name: 'book-store',
      partialize: (state) => ({ recentBooks: state.recentBooks }),
    }
  )
)
