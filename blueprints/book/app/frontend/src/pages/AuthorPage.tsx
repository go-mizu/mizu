import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { Calendar, Users, BookOpen, Download, Globe } from 'lucide-react'
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
  const [grId, setGrId] = useState('')
  const [importing, setImporting] = useState(false)

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

  const handleImportAuthorData = async () => {
    if (!grId.trim()) return
    setImporting(true)
    try {
      const imported = await booksApi.importSourceAuthor(grId.trim())
      setAuthor(prev => prev ? { ...prev, ...imported } : imported)
      setGrId('')
    } catch {
      // Silently fail
    } finally {
      setImporting(false)
    }
  }

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

  const genres = author.genres ? author.genres.split(', ').filter(Boolean) : []
  const influences = author.influences ? author.influences.split(', ').filter(Boolean) : []

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

            {/* Stats row */}
            <div
              style={{
                display: 'flex',
                gap: 20,
                marginTop: 16,
                fontSize: 14,
                color: 'var(--gr-light)',
              }}
            >
              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <BookOpen size={14} />
                {author.works_count} book{author.works_count !== 1 ? 's' : ''}
              </span>
              {author.followers > 0 && (
                <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                  <Users size={14} />
                  {author.followers.toLocaleString()} followers
                </span>
              )}
            </div>

            {/* Genres */}
            {genres.length > 0 && (
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 12 }}>
                {genres.map((genre) => (
                  <Link key={genre} to={`/genre/${encodeURIComponent(genre)}`} className="genre-tag">
                    {genre}
                  </Link>
                ))}
              </div>
            )}

            {influences.length > 0 && (
              <div style={{ marginTop: 10, fontSize: 13, color: 'var(--gr-light)' }}>
                <strong style={{ color: 'var(--gr-text)' }}>Influences:</strong> {influences.join(', ')}
              </div>
            )}

            {author.website && (
              <div style={{ marginTop: 10, fontSize: 13 }}>
                <a
                  href={author.website}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{ color: 'var(--gr-teal)', textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: 6 }}
                >
                  <Globe size={13} />
                  Author website
                </a>
              </div>
            )}

            {/* External profile link */}
            {author.source_id && (
              <div style={{ marginTop: 12, fontSize: 13 }}>
                <a
                  href={`https://www.goodreads.com/author/show/${author.source_id}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{ color: 'var(--gr-teal)', textDecoration: 'none' }}
                >
                  View source profile
                </a>
              </div>
            )}

            {/* Import from external source */}
            {!author.source_id && (
              <div style={{ marginTop: 16, display: 'flex', gap: 8, alignItems: 'center' }}>
                <input
                  type="text"
                  value={grId}
                  onChange={(e) => setGrId(e.target.value)}
                  placeholder="Source author ID"
                  className="form-input"
                  style={{ width: 200, fontSize: 13, padding: '6px 10px' }}
                />
                <button
                  className="btn btn-secondary btn-sm"
                  onClick={handleImportAuthorData}
                  disabled={importing || !grId.trim()}
                >
                  <Download size={14} />
                  {importing ? 'Importing...' : 'Import'}
                </button>
              </div>
            )}
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
