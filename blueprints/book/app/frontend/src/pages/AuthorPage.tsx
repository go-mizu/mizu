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
  const [hasMore, setHasMore] = useState(false)
  const [totalWorks, setTotalWorks] = useState(0)

  const authorId = Number(id)

  useEffect(() => {
    if (!id) return

    const fetchData = async () => {
      setLoading(true)
      setError(null)
      try {
        const [authorData, booksResult] = await Promise.all([
          booksApi.getAuthor(authorId),
          booksApi.getAuthorBooks(authorId),
        ])
        setAuthor(authorData)
        setBooks(booksResult.books)
        setHasMore(booksResult.has_more)
        setTotalWorks(booksResult.total)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load author')
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [id, authorId])

  // Auto-poll for more books when background import is running
  useEffect(() => {
    if (!hasMore || loading) return
    const timer = setInterval(async () => {
      try {
        const result = await booksApi.getAuthorBooks(authorId)
        setBooks(result.books)
        setHasMore(result.has_more)
        setTotalWorks(result.total)
        if (!result.has_more) clearInterval(timer)
      } catch { /* ignore */ }
    }, 8000) // Poll every 8s
    return () => clearInterval(timer)
  }, [hasMore, loading, authorId])

  const handleImportAuthorData = async () => {
    if (!grId.trim()) return
    setImporting(true)
    try {
      const imported = await booksApi.importSourceAuthor(grId.trim())
      setAuthor((prev) => (prev ? { ...prev, ...imported } : imported))
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
    return parts.join(' • ')
  }

  const genres = author.genres ? author.genres.split(', ').filter(Boolean) : []
  const influences = author.influences ? author.influences.split(', ').filter(Boolean) : []

  return (
    <>
      <Header />
      <div className="page-container fade-in">
        <div className="author-layout">
          <div className="author-photo-wrap">
            {author.photo_url ? (
              <img src={author.photo_url} alt={author.name} className="author-photo" />
            ) : (
              <div className="author-photo-placeholder">{author.name.charAt(0)}</div>
            )}
          </div>

          <div className="author-content">
            <h1 className="author-name">{author.name}</h1>

            {(author.birth_date || author.death_date) && (
              <p className="author-date-row">
                <Calendar size={14} />
                {formatDates()}
              </p>
            )}

            {author.bio && (
              <div className="author-bio">{author.bio}</div>
            )}

            <div className="author-stats-row">
              <span className="meta-with-icon">
                <BookOpen size={14} />
                {author.works_count} book{author.works_count !== 1 ? 's' : ''}
              </span>
              {author.followers > 0 && (
                <span className="meta-with-icon">
                  <Users size={14} />
                  {author.followers.toLocaleString()} followers
                </span>
              )}
            </div>

            {genres.length > 0 && (
              <div className="genre-list">
                {genres.map((genre) => (
                  <Link key={genre} to={`/genre/${encodeURIComponent(genre)}`} className="genre-tag">
                    {genre}
                  </Link>
                ))}
              </div>
            )}

            {influences.length > 0 && (
              <div className="author-note">
                <strong>Influences:</strong> {influences.join(', ')}
              </div>
            )}

            {author.website && (
              <div className="author-link-row">
                <a
                  href={author.website}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="meta-link"
                >
                  <Globe size={13} />
                  Author website
                </a>
              </div>
            )}

            {author.source_id && (
              <div className="author-link-row">
                <a
                  href={`https://www.goodreads.com/author/show/${author.source_id}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="meta-link"
                >
                  View source profile
                </a>
              </div>
            )}

            {!author.source_id && (
              <div className="author-import-row">
                <input
                  type="text"
                  value={grId}
                  onChange={(e) => setGrId(e.target.value)}
                  placeholder="Source author ID"
                  className="form-input author-import-input"
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

        <section>
          <div className="section-header">
            <span className="section-title">
              Books by {author.name}
              {books.length > 0 && (
                <span className="section-count">
                  {books.length}{totalWorks > books.length ? ` of ${totalWorks}` : ''}
                </span>
              )}
            </span>
          </div>

          {books.length > 0 ? (
            <div>
              {books.map((book) => (
                <BookCard key={book.id} book={book} />
              ))}
              {hasMore && (
                <div className="loading-more-bar">
                  <div className="spinner spinner-sm" />
                  <span>Loading more books ({books.length} of {totalWorks})...</span>
                </div>
              )}
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
