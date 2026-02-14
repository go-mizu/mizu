import { useState, useEffect } from 'react'
import { useSearchParams, Link } from 'react-router-dom'
import { Search } from 'lucide-react'
import Header from '../components/Header'
import BookCard from '../components/BookCard'
import { booksApi } from '../api/books'
import { useBookStore } from '../stores/bookStore'
import type { Book, SearchResult } from '../types'

export default function SearchPage() {
  const [searchParams] = useSearchParams()
  const q = searchParams.get('q') || ''
  const [results, setResults] = useState<SearchResult | null>(null)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const setQuery = useBookStore((s) => s.setQuery)

  const limit = 20
  const totalPages = results ? Math.ceil(results.total_count / limit) : 0

  useEffect(() => {
    if (!q) {
      setResults(null)
      return
    }
    setQuery(q)
    setPage(1)
  }, [q, setQuery])

  useEffect(() => {
    if (!q) return

    const fetchResults = async () => {
      setLoading(true)
      setError(null)
      try {
        const data = await booksApi.search(q, page, limit)
        setResults(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Search failed')
      } finally {
        setLoading(false)
      }
    }
    fetchResults()
  }, [q, page])

  return (
    <>
      <Header />
      <div className="page-container fade-in">
        {q && (
          <div className="page-title-block">
            <h1 className="page-title">Search results for "{q}"</h1>
            {results && !loading && (
              <p className="page-subtitle">
                {results.total_count} result{results.total_count !== 1 ? 's' : ''} found
              </p>
            )}
          </div>
        )}

        {!q && (
          <div className="empty-state">
            <Search size={48} className="empty-state-icon" />
            <h3>Search for books</h3>
            <p>Use the search bar above to find books by title, author, or ISBN.</p>
          </div>
        )}

        {loading && (
          <div className="loading-spinner">
            <div className="spinner" />
          </div>
        )}

        {error && (
          <div className="empty-state">
            <h3>Search error</h3>
            <p>{error}</p>
          </div>
        )}

        {!loading && !error && results && results.books.length === 0 && (
          <div className="empty-state">
            <h3>No results found</h3>
            <p>Try a different search term or check your spelling.</p>
            <Link to="/browse" className="btn btn-secondary">
              Browse books
            </Link>
          </div>
        )}

        {!loading && !error && results && results.books.length > 0 && (
          <>
            <div>
              {results.books.map((book: Book) => (
                <BookCard key={book.id} book={book} />
              ))}
            </div>

            {totalPages > 1 && (
              <div className="pagination-row">
                <button
                  className="btn btn-secondary btn-sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  Previous
                </button>
                <span className="pagination-text">
                  Page {page} of {totalPages}
                </span>
                <button
                  className="btn btn-secondary btn-sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </>
  )
}
