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
        {/* Hero Section */}
        <section style={{ textAlign: 'center', padding: '48px 0 40px' }}>
          <h1
            style={{
              fontFamily: "'Merriweather', Georgia, serif",
              fontSize: 32,
              fontWeight: 900,
              color: 'var(--gr-brown)',
              marginBottom: 8,
            }}
          >
            What are you reading?
          </h1>
          <p style={{ color: 'var(--gr-light)', fontSize: 16, marginBottom: 24 }}>
            Discover your next favorite book, track your reading, and connect with other readers.
          </p>
          <form
            onSubmit={handleHeroSearch}
            style={{
              maxWidth: 540,
              margin: '0 auto',
              position: 'relative',
            }}
          >
            <input
              type="text"
              placeholder="Search by title, author, or ISBN..."
              value={heroQuery}
              onChange={(e) => setHeroQuery(e.target.value)}
              className="form-input"
              style={{
                padding: '12px 48px 12px 16px',
                fontSize: 16,
                borderRadius: 8,
              }}
            />
            <button
              type="submit"
              style={{
                position: 'absolute',
                right: 12,
                top: '50%',
                transform: 'translateY(-50%)',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                color: 'var(--gr-light)',
              }}
              aria-label="Search"
            >
              <Search size={20} />
            </button>
          </form>
        </section>

        {/* Trending Section */}
        {trending.length > 0 && (
          <section style={{ marginBottom: 40 }}>
            <div className="section-header">
              <span className="section-title">
                <TrendingUp size={18} style={{ marginRight: 8, verticalAlign: 'text-bottom' }} />
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
                  style={{ textDecoration: 'none', width: 120, textAlign: 'center' }}
                >
                  <BookCover book={book} />
                  <div
                    style={{
                      fontSize: 13,
                      fontWeight: 700,
                      color: 'var(--gr-brown)',
                      marginTop: 8,
                      lineHeight: 1.3,
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {book.title}
                  </div>
                  <div style={{ fontSize: 12, color: 'var(--gr-light)' }}>
                    {book.author_names}
                  </div>
                </Link>
              ))}
            </div>
          </section>
        )}

        {/* Content grid: Feed + Challenge */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: challenge ? '1fr 300px' : '1fr',
            gap: 30,
            alignItems: 'start',
          }}
        >
          {/* Recent Updates Feed */}
          <section>
            <div className="section-header">
              <span className="section-title">
                <BookOpen size={18} style={{ marginRight: 8, verticalAlign: 'text-bottom' }} />
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

          {/* Reading Challenge Widget */}
          {challenge && (
            <aside>
              <div className="challenge-card">
                <div className="challenge-year">{challenge.year} Reading Challenge</div>
                <div className="challenge-title">
                  <Target size={20} style={{ marginRight: 6, verticalAlign: 'text-bottom' }} />
                  Reading Challenge
                </div>
                <div className="challenge-progress">
                  {challenge.progress}
                  <span style={{ fontSize: 20, color: 'var(--gr-light)' }}>
                    /{challenge.goal}
                  </span>
                </div>
                <div className="challenge-goal">books read</div>
                <div style={{ marginTop: 16 }}>
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
                  className="btn btn-secondary btn-sm"
                  style={{ marginTop: 16 }}
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
