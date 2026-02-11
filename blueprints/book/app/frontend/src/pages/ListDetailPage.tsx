import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, ThumbsUp } from 'lucide-react'
import Header from '../components/Header'
import BookCard from '../components/BookCard'
import { booksApi } from '../api/books'
import type { BookList } from '../types'

export default function ListDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [list, setList] = useState<BookList | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    booksApi.getList(Number(id))
      .then(setList)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [id])

  const handleVote = (bookId: number) => {
    if (!list) return
    booksApi.voteList(list.id, bookId).catch(() => {})
  }

  if (loading) {
    return (
      <>
        <Header />
        <div className="loading-spinner"><div className="spinner" /></div>
      </>
    )
  }

  if (!list) {
    return (
      <>
        <Header />
        <div className="empty-state"><h3>List not found</h3></div>
      </>
    )
  }

  const items = list.items || []

  return (
    <>
      <Header />
      <div className="page-container">
        <Link to="/lists" className="inline-flex items-center gap-1 text-sm text-gr-teal mb-4 hover:underline">
          <ArrowLeft size={14} /> Back to Lists
        </Link>

        <div className="mb-6">
          <h1 className="font-serif text-2xl font-bold text-gr-brown">{list.title}</h1>
          {list.description && <p className="text-gr-light mt-2">{list.description}</p>}
          <p className="text-sm text-gr-light mt-2">{list.item_count} books</p>
        </div>

        {items.length > 0 ? (
          <div>
            {items.map(item => item.book && (
              <div key={item.id} style={{ display: 'flex', alignItems: 'start', gap: 8, marginBottom: 12 }}>
                <div style={{ flex: 1 }}>
                  <BookCard book={item.book} />
                </div>
                <button
                  className="btn btn-secondary btn-sm"
                  onClick={() => handleVote(item.book_id)}
                  style={{ flexShrink: 0, marginTop: 8 }}
                >
                  <ThumbsUp size={14} /> {item.votes > 0 ? item.votes : ''}
                </button>
              </div>
            ))}
          </div>
        ) : (
          <div className="empty-state">
            <h3>No books in this list</h3>
            <p>Add books to get started.</p>
          </div>
        )}
      </div>
    </>
  )
}
