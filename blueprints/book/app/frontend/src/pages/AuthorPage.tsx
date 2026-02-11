import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { Calendar } from 'lucide-react'
import Header from '../components/Header'
import BookCard from '../components/BookCard'
import { booksApi } from '../api/books'
import type { Author, Book } from '../types'

export default function AuthorPage() {
  const { id } = useParams<{ id: string }>()
  const [author, setAuthor] = useState<Author | null>(null)
  const [books, setBooks] = useState<Book[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const authorId = Number(id)

  useEffect(() => {
    if (!id) return

    const fetchData = async () => {
      setLoading(true)
      setError(null)
      try {
        const [authorData, booksData] = await Promise.all([
          booksApi.getAuthor(authorId),
          booksApi.getAuthorBooks(authorId),
        ])
        setAuthor(authorData)
        setBooks(booksData)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load author')
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [id, authorId])

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

  if (error || !author) {
    return (
      <>
        <Header />
        <div className="page-container">
          <div className="empty-state">
            <h3>Author not found</h3>
            <p>{error || 'This author does not exist.'}</p>
            <Link to="/browse" className="btn btn-secondary">
              Browse books
            </Link>
          </div>
        </div>
      </>
    )
  }

  const formatDates = () => {
    const parts: string[] = []
    if (author.birth_date) parts.push(`Born ${author.birth_date}`)
    if (author.death_date) parts.push(`Died ${author.death_date}`)
    return parts.join(' \u2022 ')
  }

  return (
    <>
      <Header />
      <div className="page-container fade-in">
        {/* Author Profile */}
        <div style={{ display: 'flex', gap: 24, marginBottom: 40 }}>
          {/* Photo */}
          <div style={{ flexShrink: 0 }}>
            {author.photo_url ? (
              <img
                src={author.photo_url}
                alt={author.name}
                style={{
                  width: 150,
                  height: 200,
                  objectFit: 'cover',
                  borderRadius: 8,
                  boxShadow: '0 2px 8px rgba(0,0,0,0.12)',
                }}
              />
            ) : (
              <div
                style={{
                  width: 150,
                  height: 200,
                  background: 'var(--gr-tan)',
                  borderRadius: 8,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 48,
                  color: 'var(--gr-light)',
                }}
              >
                {author.name.charAt(0)}
              </div>
            )}
          </div>

          {/* Info */}
          <div style={{ flex: 1, minWidth: 0 }}>
            <h1
              style={{
                fontFamily: "'Merriweather', Georgia, serif",
                fontSize: 28,
                fontWeight: 900,
                color: 'var(--gr-brown)',
                margin: '0 0 8px',
              }}
            >
              {author.name}
            </h1>

            {(author.birth_date || author.death_date) && (
              <p
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  fontSize: 14,
                  color: 'var(--gr-light)',
                  marginBottom: 12,
                }}
              >
                <Calendar size={14} />
                {formatDates()}
              </p>
            )}

            {author.bio && (
              <div
                style={{
                  fontSize: 15,
                  lineHeight: 1.7,
                  color: 'var(--gr-text)',
                  maxWidth: 700,
                }}
              >
                {author.bio}
              </div>
            )}

            <div
              style={{
                marginTop: 16,
                fontSize: 14,
                color: 'var(--gr-light)',
              }}
            >
              {author.book_count} book{author.book_count !== 1 ? 's' : ''}
            </div>
          </div>
        </div>

        {/* Books by Author */}
        <section>
          <div className="section-header">
            <span className="section-title">Books by {author.name}</span>
          </div>

          {books.length > 0 ? (
            <div>
              {books.map((book) => (
                <BookCard key={book.id} book={book} />
              ))}
            </div>
          ) : (
            <div className="empty-state">
              <h3>No books found</h3>
              <p>No books by this author are in the library yet.</p>
            </div>
          )}
        </section>
      </div>
    </>
  )
}
