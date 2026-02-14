import { useState, useEffect } from 'react'
import { BarChart3 } from 'lucide-react'
import Header from '../components/Header'
import BookCard from '../components/BookCard'
import StarRating from '../components/StarRating'
import { booksApi } from '../api/books'
import type { ReadingStats } from '../types'

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

export default function StatsPage() {
  const currentYear = new Date().getFullYear()
  const [year, setYear] = useState(currentYear)
  const [stats, setStats] = useState<ReadingStats | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    booksApi.getStatsByYear(year)
      .then(setStats)
      .catch(() => setStats(null))
      .finally(() => setLoading(false))
  }, [year])

  const years = Array.from({ length: 5 }, (_, i) => currentYear - i)
  const maxBooks = stats?.books_per_month
    ? Math.max(1, ...Object.values(stats.books_per_month))
    : 1

  return (
    <>
      <Header />
      <div className="page-container">
        <div className="section-header">
          <h1 className="section-title section-title-with-icon">
            <BarChart3 size={24} />
            Year in Books
          </h1>
        </div>

        <div className="tabs tabs-spaced">
          {years.map((y) => (
            <button
              key={y}
              className={`tab ${y === year ? 'active' : ''}`}
              onClick={() => setYear(y)}
            >
              {y}
            </button>
          ))}
        </div>

        {loading ? (
          <div className="loading-spinner"><div className="spinner" /></div>
        ) : !stats ? (
          <div className="empty-state">
            <h3>No reading data for {year}</h3>
            <p>Start reading and tracking books to see your stats.</p>
          </div>
        ) : (
          <div className="fade-in">
            <div className="stats-summary-grid">
              <div className="stat-box">
                <div className="stat-number">{stats.total_books}</div>
                <div className="stat-label">Books Read</div>
              </div>
              <div className="stat-box">
                <div className="stat-number">{stats.total_pages.toLocaleString()}</div>
                <div className="stat-label">Pages Read</div>
              </div>
              <div className="stat-box">
                <div className="stat-number">{stats.average_rating.toFixed(1)}</div>
                <div className="stat-label">Avg Rating</div>
              </div>
              <div className="stat-box">
                <div className="stat-number">{stats.total_books > 0 ? Math.round(stats.total_pages / stats.total_books) : 0}</div>
                <div className="stat-label">Avg Pages</div>
              </div>
            </div>

            <div className="stats-section">
              <h2 className="section-title">Books by Month</h2>
              <div className="bar-chart bar-chart-spaced">
                {MONTHS.map((m, i) => {
                  const key = String(i + 1)
                  const count = stats.books_per_month?.[key] || 0
                  const height = maxBooks > 0 ? (count / maxBooks) * 100 : 0
                  return (
                    <div key={m} className="bar" style={{ height: `${Math.max(height, 2)}%` }}>
                      <span className="bar-label">{m}</span>
                      {count > 0 && (
                        <span className="bar-value">
                          {count}
                        </span>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>

            {stats.pages_per_month && Object.values(stats.pages_per_month).some(v => v > 0) && (
              <div className="stats-section">
                <h2 className="section-title">Pages by Month</h2>
                <div className="bar-chart bar-chart-spaced">
                  {MONTHS.map((m, i) => {
                    const key = String(i + 1)
                    const count = stats.pages_per_month?.[key] || 0
                    const maxPages = Math.max(1, ...Object.values(stats.pages_per_month))
                    const height = maxPages > 0 ? (count / maxPages) * 100 : 0
                    return (
                      <div key={m} className="bar" style={{ height: `${Math.max(height, 2)}%` }}>
                        <span className="bar-label">{m}</span>
                        {count > 0 && (
                          <span className="bar-value">
                            {count.toLocaleString()}
                          </span>
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            )}

            <div className="stats-section">
              <h2 className="section-title">Rating Distribution</h2>
              <div className="stats-rating-list">
                {[5, 4, 3, 2, 1].map((r) => {
                  const count = stats.rating_distribution?.[String(r)] || 0
                  const maxRating = Math.max(1, ...Object.values(stats.rating_distribution || {}))
                  const width = maxRating > 0 ? (count / maxRating) * 100 : 0
                  return (
                    <div key={r} className="stats-rating-row">
                      <div className="stats-rating-stars">
                        <StarRating rating={r} size={14} />
                      </div>
                      <div className="stats-rating-track">
                        <div
                          className="stats-rating-fill"
                          style={{ width: `${width}%` }}
                        />
                      </div>
                      <span className="stats-rating-count">{count}</span>
                    </div>
                  )
                })}
              </div>
            </div>

            {stats.genre_breakdown && Object.keys(stats.genre_breakdown).length > 0 && (
              <div className="stats-section">
                <h2 className="section-title">Genres</h2>
                <div className="genre-list">
                  {Object.entries(stats.genre_breakdown)
                    .sort(([, a], [, b]) => b - a)
                    .slice(0, 20)
                    .map(([genre, count]) => (
                      <span key={genre} className="genre-tag">
                        {genre} ({count})
                      </span>
                    ))}
                </div>
              </div>
            )}

            <div className="stats-notable-grid">
              {stats.shortest_book && (
                <div>
                  <h3 className="stats-notable-label">Shortest Book</h3>
                  <BookCard book={stats.shortest_book} />
                </div>
              )}
              {stats.longest_book && (
                <div>
                  <h3 className="stats-notable-label">Longest Book</h3>
                  <BookCard book={stats.longest_book} />
                </div>
              )}
              {stats.highest_rated && (
                <div>
                  <h3 className="stats-notable-label">Highest Rated</h3>
                  <BookCard book={stats.highest_rated} />
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </>
  )
}
