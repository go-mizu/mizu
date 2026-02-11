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
      .then(r => { setBooks(r.books); setTotal(r.total_count) })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [genre, page])

  return (
    <>
      <Header />
      <div className="page-container">
        <Link to="/browse" className="inline-flex items-center gap-1 text-sm text-gr-teal mb-4 hover:underline">
          <ArrowLeft size={14} /> Back to Browse
        </Link>
        <h1 className="font-serif text-2xl font-bold text-gr-brown mb-6">
          {decodeURIComponent(genre || '')}
        </h1>

        {loading ? (
          <div className="loading-spinner"><div className="spinner" /></div>
        ) : books.length === 0 ? (
          <div className="empty-state">
            <h3>No books found</h3>
            <p>No books in this genre yet.</p>
          </div>
        ) : (
          <>
            <p className="text-sm text-gr-light mb-4">{total} books</p>
            <BookGrid books={books} />
            {total > 20 && (
              <div className="flex justify-center gap-4 mt-8">
                <button className="btn btn-secondary" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>
                  Previous
                </button>
                <span className="flex items-center text-sm text-gr-light">Page {page}</span>
                <button className="btn btn-secondary" disabled={books.length < 20} onClick={() => setPage(p => p + 1)}>
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
