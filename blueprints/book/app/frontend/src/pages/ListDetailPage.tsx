import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, ThumbsUp } from 'lucide-react'
import Header from '../components/Header'
import BookCard from '../components/BookCard'
import { booksApi } from '../api/books'
import type { Book, BookList } from '../types'

export default function ListDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [list, setList] = useState<(BookList & { items: Book[] }) | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    booksApi.getList(Number(id))
      .then(setList)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [id])

  const handleVote = () => {
    if (!list) return
    booksApi.voteList(list.id)
      .then(() => setList(prev => prev ? { ...prev, vote_count: prev.vote_count + 1 } : null))
      .catch(() => {})
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

  return (
    <>
      <Header />
      <div className="page-container">
        <Link to="/lists" className="inline-flex items-center gap-1 text-sm text-gr-teal mb-4 hover:underline">
          <ArrowLeft size={14} /> Back to Lists
        </Link>

        <div className="flex items-start justify-between mb-6">
          <div>
            <h1 className="font-serif text-2xl font-bold text-gr-brown">{list.title}</h1>
            {list.description && <p className="text-gr-light mt-2">{list.description}</p>}
            <p className="text-sm text-gr-light mt-2">{list.book_count} books</p>
          </div>
          <button className="btn btn-secondary" onClick={handleVote}>
            <ThumbsUp size={16} /> {list.vote_count}
          </button>
        </div>

        {list.items && list.items.length > 0 ? (
          <div>
            {list.items.map(book => (
              <BookCard key={book.id} book={book} />
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
