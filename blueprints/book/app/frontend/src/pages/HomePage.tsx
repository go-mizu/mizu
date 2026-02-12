import { useState, useEffect, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Search, TrendingUp, BookOpen, Target, Eye } from 'lucide-react'
import Header from '../components/Header'
import BookCover from '../components/BookCover'
import FeedItem from '../components/FeedItem'
import { booksApi } from '../api/books'
import type { Book, Shelf, FeedItem as FeedItemType, ReadingChallenge, ReadingProgress } from '../types'

interface CurrentlyReadingBook {
  book: Book
  progress?: ReadingProgress
}

export default function HomePage() {
  const navigate = useNavigate()
  const [heroQuery, setHeroQuery] = useState('')
  const [trending, setTrending] = useState<Book[]>([])
  const [feed, setFeed] = useState<FeedItemType[]>([])
  const [challenge, setChallenge] = useState<ReadingChallenge | null>(null)
  const [currentlyReading, setCurrentlyReading] = useState<CurrentlyReadingBook[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      setError(null)
      try {
        const [trendingData, feedData, shelvesData] = await Promise.all([
          booksApi.getTrending(12),
          booksApi.getFeed(10),
          booksApi.getShelves(),
        ])
        setTrending(trendingData)
        setFeed(feedData)

        // Load currently reading books
        const crShelf = shelvesData.find((s: Shelf) => s.slug === 'currently-reading')
        if (crShelf && crShelf.book_count > 0) {
          const crBooks = await booksApi.getShelfBooks(crShelf.id, 1, 10)
          const withProgress: CurrentlyReadingBook[] = await Promise.all(
            (crBooks.books || []).map(async (book: Book) => {
              try {
                const progressList = await booksApi.getProgress(book.id)
                return { book, progress: progressList?.[0] }
              } catch {
                return { book }
              }
            })
          )
          setCurrentlyReading(withProgress)
        }

        try {
          const challengeData = await booksApi.getChallenge()
          setChallenge(challengeData)
        } catch {
          // No challenge set, that's fine
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load data')
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [])

  const handleHeroSearch = (e: FormEvent) => {
    e.preventDefault()
    const q = heroQuery.trim()
    if (q) {
      navigate(`/search?q=${encodeURIComponent(q)}`)
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

  const challengePercent = challenge
    ? Math.min(100, Math.round((challenge.progress / challenge.goal) * 100))
    : 0

  return (
    <>
      <Header />
      <div className="page-container fade-in">
        <section className="home-hero">
          <h1 className="home-hero-title">What are you reading?</h1>
          <p className="home-hero-subtitle">
            Discover your next favorite book, track your reading, and connect with other readers.
          </p>
          <form onSubmit={handleHeroSearch} className="home-hero-search">
            <input
              type="text"
              placeholder="Search by title, author, or ISBN..."
              value={heroQuery}
              onChange={(e) => setHeroQuery(e.target.value)}
              className="form-input home-hero-search-input"
            />
            <button
              type="submit"
              className="home-hero-search-button"
              aria-label="Search"
            >
              <Search size={20} />
            </button>
          </form>
        </section>

        {currentlyReading.length > 0 && (
          <section className="page-section">
            <div className="section-header">
              <span className="section-title section-title-with-icon">
                <Eye size={18} />
                Currently Reading
              </span>
              <Link to="/my-books" className="section-link">
                My Books
              </Link>
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 16 }}>
              {currentlyReading.map(({ book, progress }) => (
                <Link
                  key={book.id}
                  to={`/book/${book.id}`}
                  style={{
                    display: 'flex',
                    gap: 12,
                    padding: 12,
                    background: 'var(--gr-cream)',
                    borderRadius: 8,
                    textDecoration: 'none',
                    color: 'inherit',
                    transition: 'box-shadow 0.2s',
                  }}
                >
                  <div style={{ flexShrink: 0, width: 50 }}>
                    <BookCover book={book} size="sm" />
                  </div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{
                      fontFamily: "'Merriweather', Georgia, serif",
                      fontSize: 13,
                      fontWeight: 700,
                      color: 'var(--gr-brown)',
                      marginBottom: 2,
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}>
                      {book.title}
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--gr-light)', marginBottom: 8 }}>
                      {book.author_names}
                    </div>
                    {progress && (
                      <div>
                        <div className="progress-bar" style={{ height: 5, marginBottom: 4 }}>
                          <div className="progress-fill" style={{ width: `${Math.min(100, progress.percent)}%` }} />
                        </div>
                        <div style={{ fontSize: 11, color: 'var(--gr-light)' }}>
                          {Math.round(progress.percent)}% done
                          {book.page_count > 0 && ` (p. ${progress.page}/${book.page_count})`}
                        </div>
                      </div>
                    )}
                    {!progress && book.page_count > 0 && (
                      <div style={{ fontSize: 11, color: 'var(--gr-light)' }}>
                        {book.page_count} pages
                      </div>
                    )}
                  </div>
                </Link>
              ))}
            </div>
          </section>
        )}

        {trending.length > 0 && (
          <section className="page-section">
            <div className="section-header">
              <span className="section-title section-title-with-icon">
                <TrendingUp size={18} />
                Trending Books
              </span>
              <Link to="/browse" className="section-link">
                Browse all
              </Link>
            </div>
            <div className="book-scroll">
              {trending.map((book) => (
                <Link
                  key={book.id}
                  to={`/book/${book.id}`}
                  className="book-scroll-item"
                >
                  <BookCover book={book} />
                  <div className="book-scroll-title">{book.title}</div>
                  <div className="book-scroll-author">{book.author_names}</div>
                </Link>
              ))}
            </div>
          </section>
        )}

        <div className={`home-content-grid${challenge ? ' has-challenge' : ''}`}>
          <section>
            <div className="section-header">
              <span className="section-title section-title-with-icon">
                <BookOpen size={18} />
                Recent Updates
              </span>
            </div>
            {feed.length > 0 ? (
              <div>
                {feed.map((item) => (
                  <FeedItem key={item.id} item={item} />
                ))}
              </div>
            ) : (
              <div className="empty-state">
                <p>No recent updates yet. Start by adding books to your shelves!</p>
              </div>
            )}
          </section>

          {challenge && (
            <aside>
              <div className="challenge-card">
                <div className="challenge-year">{challenge.year} Reading Challenge</div>
                <div className="challenge-title challenge-title-with-icon">
                  <Target size={20} />
                  Reading Challenge
                </div>
                <div className="challenge-progress">
                  {challenge.progress}
                  <span className="challenge-progress-total">/{challenge.goal}</span>
                </div>
                <div className="challenge-goal">books read</div>
                <div className="challenge-progress-wrap">
                  <div className="progress-bar">
                    <div
                      className="progress-fill"
                      style={{ width: `${challengePercent}%` }}
                    />
                  </div>
                  <div className="progress-label">{challengePercent}% complete</div>
                </div>
                <Link
                  to="/challenge"
                  className="btn btn-secondary btn-sm challenge-link"
                >
                  View Challenge
                </Link>
              </div>
            </aside>
          )}
        </div>
      </div>
    </>
  )
}
