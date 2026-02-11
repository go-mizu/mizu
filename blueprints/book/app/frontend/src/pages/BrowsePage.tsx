import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Flame, Sparkles, BookMarked } from 'lucide-react'
import Header from '../components/Header'
import BookGrid from '../components/BookGrid'
import { booksApi } from '../api/books'
import type { Book, Genre } from '../types'

export default function BrowsePage() {
  const [popular, setPopular] = useState<Book[]>([])
  const [newReleases, setNewReleases] = useState<Book[]>([])
  const [genres, setGenres] = useState<Genre[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      setError(null)
      try {
        const [popularData, newData, genresData] = await Promise.all([
          booksApi.getPopular(12),
          booksApi.getNewReleases(12),
          booksApi.getGenres(),
        ])
        setPopular(popularData)
        setNewReleases(newData)
        setGenres(genresData)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load browse data')
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [])

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

  if (error) {
    return (
      <>
        <Header />
        <div className="page-container">
          <div className="empty-state">
            <h3>Something went wrong</h3>
            <p>{error}</p>
          </div>
        </div>
      </>
    )
  }

  return (
    <>
      <Header />
      <div className="page-container fade-in">
        {/* Popular Section */}
        {popular.length > 0 && (
          <section style={{ marginBottom: 40 }}>
            <div className="section-header">
              <span className="section-title">
                <Flame size={18} style={{ marginRight: 8, verticalAlign: 'text-bottom' }} />
                Popular
              </span>
            </div>
            <BookGrid books={popular} />
          </section>
        )}

        {/* New Releases Section */}
        {newReleases.length > 0 && (
          <section style={{ marginBottom: 40 }}>
            <div className="section-header">
              <span className="section-title">
                <Sparkles size={18} style={{ marginRight: 8, verticalAlign: 'text-bottom' }} />
                New Releases
              </span>
            </div>
            <BookGrid books={newReleases} />
          </section>
        )}

        {/* Genres Section */}
        {genres.length > 0 && (
          <section style={{ marginBottom: 40 }}>
            <div className="section-header">
              <span className="section-title">
                <BookMarked size={18} style={{ marginRight: 8, verticalAlign: 'text-bottom' }} />
                Genres
              </span>
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
              {genres.map((genre) => (
                <Link
                  key={genre.name}
                  to={`/genre/${encodeURIComponent(genre.name)}`}
                  className="genre-tag"
                  style={{ fontSize: 14, padding: '8px 16px' }}
                >
                  {genre.name}
                  {genre.count > 0 && (
                    <span
                      style={{
                        marginLeft: 6,
                        fontSize: 12,
                        color: 'var(--gr-light)',
                      }}
                    >
                      ({genre.count})
                    </span>
                  )}
                </Link>
              ))}
            </div>
          </section>
        )}

        {popular.length === 0 && newReleases.length === 0 && genres.length === 0 && (
          <div className="empty-state">
            <h3>Nothing to browse yet</h3>
            <p>Add some books to get started!</p>
          </div>
        )}
      </div>
    </>
  )
}
