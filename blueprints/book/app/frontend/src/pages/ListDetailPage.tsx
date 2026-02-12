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
        <Link to="/lists" style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 4,
          fontSize: 13,
          color: 'var(--gr-teal)',
          textDecoration: 'none',
          marginBottom: 16,
        }}>
          <ArrowLeft size={14} /> Back to Lists
        </Link>

        <div style={{ marginBottom: 24 }}>
          <h1 style={{
            fontFamily: "'Merriweather', Georgia, serif",
            fontSize: 24,
            fontWeight: 900,
            color: 'var(--gr-brown)',
            margin: '0 0 8px',
          }}>
            {list.title}
          </h1>
          {list.description && (
            <p style={{ color: 'var(--gr-light)', fontSize: 14, marginBottom: 8 }}>{list.description}</p>
          )}
          <div style={{ display: 'flex', alignItems: 'center', gap: 16, fontSize: 13, color: 'var(--gr-light)' }}>
            <span>{list.item_count} books</span>
            {list.voter_count > 0 && (
              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <Users size={13} /> {list.voter_count.toLocaleString()} voters
              </span>
            )}
            {list.goodreads_url && (
              <a
                href={list.goodreads_url}
                target="_blank"
                rel="noopener noreferrer"
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 4,
                  color: 'var(--gr-teal)',
                  textDecoration: 'none',
                }}
              >
                <ExternalLink size={13} /> View source list
              </a>
            )}
          </div>
        </div>

        {items.length > 0 ? (
          <div>
            {items.map(item => item.book && (
              <div key={item.id} style={{ display: 'flex', alignItems: 'start', gap: 8, marginBottom: 12 }}>
                <div style={{
                  width: 34,
                  height: 34,
                  borderRadius: '999px',
                  background: 'var(--gr-cream)',
                  border: '1px solid var(--gr-border)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 13,
                  fontWeight: 700,
                  color: 'var(--gr-brown)',
                  marginTop: 8,
                  flexShrink: 0,
                }}>
                  {item.position}
                </div>
                <div style={{ flex: 1 }}>
                  <BookCard book={item.book} />
                  {item.votes > 0 && (
                    <div style={{ marginTop: -4, marginBottom: 10, fontSize: 12, color: 'var(--gr-light)' }}>
                      Source score: {item.votes.toLocaleString()}
                    </div>
                  )}
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
