import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import Header from '../components/Header'
import BookGrid from '../components/BookGrid'
import { booksApi } from '../api/books'
import type { Book } from '../types'

export default function GenrePage() {
  const { genre } = useParams<{ genre: string }>()
  const [books, setBooks] = useState<Book[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)

  useEffect(() => {
    if (!genre) return
    setLoading(true)
    booksApi.getBooksByGenre(genre, page)
      .then((r) => { setBooks(r.books); setTotal(r.total_count) })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [genre, page])

  return (
    <>
      <Header />
      <div className="page-container">
        <Link to="/browse" className="back-link">
          <ArrowLeft size={14} /> Back to Browse
        </Link>
        <h1 className="page-title page-title-lg">{decodeURIComponent(genre || '')}</h1>

        {loading ? (
          <div className="loading-spinner"><div className="spinner" /></div>
        ) : books.length === 0 ? (
          <div className="empty-state">
            <h3>No books found</h3>
            <p>No books in this genre yet.</p>
          </div>
        ) : (
          <>
            <p className="page-subtitle">{total} books</p>
            <BookGrid books={books} />
            {total > 20 && (
              <div className="pagination-row">
                <button className="btn btn-secondary" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                  Previous
                </button>
                <span className="pagination-text">Page {page}</span>
                <button className="btn btn-secondary" disabled={books.length < 20} onClick={() => setPage((p) => p + 1)}>
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
