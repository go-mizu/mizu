import { useState, useEffect, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Search, TrendingUp, BookOpen, Target } from 'lucide-react'
import Header from '../components/Header'
import BookCover from '../components/BookCover'
import FeedItem from '../components/FeedItem'
import { booksApi } from '../api/books'
import type { Book, FeedItem as FeedItemType, ReadingChallenge } from '../types'

export default function HomePage() {
  const navigate = useNavigate()
  const [heroQuery, setHeroQuery] = useState('')
  const [trending, setTrending] = useState<Book[]>([])
  const [feed, setFeed] = useState<FeedItemType[]>([])
  const [challenge, setChallenge] = useState<ReadingChallenge | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      setError(null)
      try {
        const [trendingData, feedData] = await Promise.all([
          booksApi.getTrending(12),
          booksApi.getFeed(10),
        ])
        setTrending(trendingData)
        setFeed(feedData)

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
