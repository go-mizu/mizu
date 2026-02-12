import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Grid3X3, List, Table, Upload, Download } from 'lucide-react'
import Header from '../components/Header'
import Sidebar from '../components/Sidebar'
import BookCard from '../components/BookCard'
import BookGrid from '../components/BookGrid'
import StarRating from '../components/StarRating'
import BookCover from '../components/BookCover'
import { booksApi } from '../api/books'
import { useBookStore } from '../stores/bookStore'
import { useUIStore } from '../stores/uiStore'
import type { Shelf, SearchResult } from '../types'

export default function MyBooksPage() {
  const [shelves, setShelves] = useState<Shelf[]>([])
  const [selectedShelf, setSelectedShelf] = useState<number | null>(null)
  const [results, setResults] = useState<SearchResult | null>(null)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [booksLoading, setBooksLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const shelfView = useUIStore((s) => s.shelfView)
  const setShelfView = useUIStore((s) => s.setShelfView)
  const setStoreShelves = useBookStore((s) => s.setShelves)

  const limit = 20
  const totalPages = results ? Math.ceil(results.total_count / limit) : 0

  useEffect(() => {
    const fetchShelves = async () => {
      setLoading(true)
      try {
        const data = await booksApi.getShelves()
        setShelves(data)
        setStoreShelves(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load shelves')
      } finally {
        setLoading(false)
      }
    }
    fetchShelves()
  }, [setStoreShelves])

  useEffect(() => {
    const fetchBooks = async () => {
      setBooksLoading(true)
      try {
        if (selectedShelf === null) {
          const data = await booksApi.search('', page, limit)
          setResults(data)
        } else {
          const data = await booksApi.getShelfBooks(selectedShelf, page, limit)
          setResults(data)
        }
      } catch {
        setResults({ books: [], total_count: 0, page: 1, page_size: limit })
      } finally {
        setBooksLoading(false)
      }
    }
    if (!loading) fetchBooks()
  }, [selectedShelf, page, loading])

  const handleShelfSelect = (shelfId: number | null) => {
    setSelectedShelf(shelfId)
    setPage(1)
  }

  const handleImport = () => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.csv'
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (file) {
        try {
          await booksApi.importCSV(file)
          window.location.reload()
        } catch {
          // Handle error
        }
      }
    }
    input.click()
  }

  const handleExport = () => {
    booksApi.exportCSV()
  }

  const selectedShelfName = selectedShelf === null
    ? 'All'
    : shelves.find((s) => s.id === selectedShelf)?.name || 'Shelf'

  const books = results?.books || []

  if (loading) {
    return (
      <>
        <Header />
        <div className="loading-spinner">
          <div className="spinner" />
        </div>
      </>
    )
  }

  return (
    <>
      <Header />
      <div className="page-with-sidebar fade-in">
        <Sidebar>
          <h3>Bookshelves</h3>
          <button
            type="button"
            className={`sidebar-link ${selectedShelf === null ? 'active' : ''}`}
            onClick={() => handleShelfSelect(null)}
          >
            <span>All</span>
            <span className="sidebar-count">
              {shelves.reduce((sum, s) => sum + s.book_count, 0)}
            </span>
          </button>
          {shelves.map((shelf) => (
            <button
              type="button"
              key={shelf.id}
              className={`sidebar-link ${selectedShelf === shelf.id ? 'active' : ''}`}
              onClick={() => handleShelfSelect(shelf.id)}
            >
              <span>{shelf.name}</span>
              <span className="sidebar-count">{shelf.book_count}</span>
            </button>
          ))}
        </Sidebar>

        <main>
          <div className="mybooks-header">
            <h1 className="page-title mybooks-title">
              {selectedShelfName}
              {results && (
                <span className="page-count">({results.total_count})</span>
              )}
            </h1>

            <div className="mybooks-controls">
              <button
                className="btn btn-secondary btn-sm"
                onClick={handleImport}
                title="Import CSV"
              >
                <Upload size={14} />
                Import
              </button>
              <button
                className="btn btn-secondary btn-sm"
                onClick={handleExport}
                title="Export CSV"
              >
                <Download size={14} />
                Export
              </button>

              <div className="view-toggle" role="group" aria-label="View mode">
                <button
                  type="button"
                  onClick={() => setShelfView('grid')}
                  className={shelfView === 'grid' ? 'active' : ''}
                  title="Grid view"
                >
                  <Grid3X3 size={16} />
                </button>
                <button
                  type="button"
                  onClick={() => setShelfView('list')}
                  className={shelfView === 'list' ? 'active' : ''}
                  title="List view"
                >
                  <List size={16} />
                </button>
                <button
                  type="button"
                  onClick={() => setShelfView('table')}
                  className={shelfView === 'table' ? 'active' : ''}
                  title="Table view"
                >
                  <Table size={16} />
                </button>
              </div>
            </div>
          </div>

          {error && (
            <div className="empty-state">
              <p>{error}</p>
            </div>
          )}

          {booksLoading && (
            <div className="loading-spinner">
              <div className="spinner" />
            </div>
          )}

          {!booksLoading && books.length === 0 && (
            <div className="empty-state">
              <h3>No books on this shelf</h3>
              <p>Search for books and add them to your shelves.</p>
              <Link to="/browse" className="btn btn-primary">
                Browse books
              </Link>
            </div>
          )}

          {!booksLoading && books.length > 0 && shelfView === 'grid' && (
            <BookGrid books={books} />
          )}

          {!booksLoading && books.length > 0 && shelfView === 'list' && (
            <div>
              {books.map((book) => (
                <div key={book.id} className="mybooks-list-row">
                  <div className="mybooks-list-main">
                    <BookCard book={book} />
                  </div>
                  <div className="mybooks-list-meta">
                    {book.user_shelf && (
                      <span className="genre-tag mybooks-shelf-tag">
                        {book.user_shelf}
                      </span>
                    )}
                    {book.user_rating != null && book.user_rating > 0 && (
                      <div className="mybooks-list-rating">
                        <StarRating rating={book.user_rating} size={14} />
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}

          {!booksLoading && books.length > 0 && shelfView === 'table' && (
            <table className="book-table">
              <thead>
                <tr>
                  <th>Cover</th>
                  <th>Title</th>
                  <th>Author</th>
                  <th>Rating</th>
                  <th>Shelf</th>
                  <th>Pages</th>
                </tr>
              </thead>
              <tbody>
                {books.map((book) => (
                  <tr key={book.id}>
                    <td>
                      <BookCover book={book} size="sm" />
                    </td>
                    <td>
                      <Link to={`/book/${book.id}`} className="table-book-link">
                        {book.title}
                      </Link>
                    </td>
                    <td>
                      <span>{book.author_names}</span>
                    </td>
                    <td>
                      {book.user_rating != null && book.user_rating > 0 ? (
                        <StarRating rating={book.user_rating} size={12} />
                      ) : (
                        <span className="table-muted">--</span>
                      )}
                    </td>
                    <td>
                      {book.user_shelf && (
                        <span className="genre-tag">{book.user_shelf}</span>
                      )}
                    </td>
                    <td className="table-muted">{book.page_count || '--'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {!booksLoading && totalPages > 1 && (
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
        </main>
      </div>
    </>
  )
}
