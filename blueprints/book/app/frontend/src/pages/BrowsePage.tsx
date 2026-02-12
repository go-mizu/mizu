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
        {popular.length > 0 && (
          <section className="page-section">
            <div className="section-header">
              <span className="section-title section-title-with-icon">
                <Flame size={18} />
                Popular
              </span>
            </div>
            <BookGrid books={popular} />
          </section>
        )}

        {newReleases.length > 0 && (
          <section className="page-section">
            <div className="section-header">
              <span className="section-title section-title-with-icon">
                <Sparkles size={18} />
                New Releases
              </span>
            </div>
            <BookGrid books={newReleases} />
          </section>
        )}

        {genres.length > 0 && (
          <section className="page-section">
            <div className="section-header">
              <span className="section-title section-title-with-icon">
                <BookMarked size={18} />
                Genres
              </span>
            </div>
            <div className="genre-list">
              {genres.map((genre) => (
                <Link
                  key={genre.name}
                  to={`/genre/${encodeURIComponent(genre.name)}`}
                  className="genre-tag genre-pill"
                >
                  {genre.name}
                  {genre.book_count > 0 && (
                    <span className="genre-count">({genre.book_count})</span>
                  )}
                </Link>
              ))}
            </div>
          </section>
        )}

        {popular.length === 0 && newReleases.length === 0 && genres.length === 0 && (
          <div className="empty-state">
            <h3>Nothing to browse yet</h3>
            <p>Add some books to get started.</p>
          </div>
        )}
      </div>
    </>
  )
}
