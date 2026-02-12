import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, ThumbsUp, Users, ExternalLink } from 'lucide-react'
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
        <Link to="/lists" className="back-link">
          <ArrowLeft size={14} /> Back to Lists
        </Link>

        <div className="list-detail-header">
          <h1 className="list-detail-title">{list.title}</h1>
          {list.description && (
            <p className="page-subtitle">{list.description}</p>
          )}
          <div className="list-detail-meta">
            <span>{list.item_count} books</span>
            {list.voter_count > 0 && (
              <span className="meta-with-icon">
                <Users size={13} /> {list.voter_count.toLocaleString()} voters
              </span>
            )}
            {list.source_url && (
              <a
                href={list.source_url}
                target="_blank"
                rel="noopener noreferrer"
                className="meta-link"
              >
                <ExternalLink size={13} /> View source list
              </a>
            )}
          </div>
        </div>

        {items.length > 0 ? (
          <div>
            {items.map((item) => item.book && (
              <div key={item.id} className="list-detail-item">
                <div className="list-rank-badge">
                  {item.position}
                </div>
                <div className="list-detail-book">
                  <BookCard book={item.book} />
                  {item.votes > 0 && (
                    <div className="list-source-score">
                      Source score: {item.votes.toLocaleString()}
                    </div>
                  )}
                </div>
                <button
                  className="btn btn-secondary btn-sm"
                  onClick={() => handleVote(item.book_id)}
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
