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
  const maxBooks = stats?.books_by_month
    ? Math.max(1, ...Object.values(stats.books_by_month))
    : 1

  return (
    <>
      <Header />
      <div className="page-container">
        <div className="section-header">
          <h1 className="section-title">
            <BarChart3 size={24} className="inline mr-2" />
            Year in Books
          </h1>
        </div>

        <div className="tabs mb-6">
          {years.map(y => (
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
            {/* Summary boxes */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
              <div className="stat-box">
                <div className="stat-number">{stats.total_books}</div>
                <div className="stat-label">Books Read</div>
              </div>
              <div className="stat-box">
                <div className="stat-number">{stats.total_pages.toLocaleString()}</div>
                <div className="stat-label">Pages Read</div>
              </div>
              <div className="stat-box">
                <div className="stat-number">{stats.avg_rating.toFixed(1)}</div>
                <div className="stat-label">Avg Rating</div>
              </div>
              <div className="stat-box">
                <div className="stat-number">{Math.round(stats.avg_pages)}</div>
                <div className="stat-label">Avg Pages</div>
              </div>
            </div>

            {/* Books by month chart */}
            <div className="mb-8">
              <h2 className="section-title mb-4">Books by Month</h2>
              <div className="bar-chart" style={{ paddingBottom: 24 }}>
                {MONTHS.map((m, i) => {
                  const key = String(i + 1)
                  const count = stats.books_by_month?.[key] || 0
                  const height = maxBooks > 0 ? (count / maxBooks) * 100 : 0
                  return (
                    <div key={m} className="bar" style={{ height: `${Math.max(height, 2)}%` }}>
                      <span className="bar-label">{m}</span>
                      {count > 0 && (
                        <span className="absolute -top-5 left-1/2 -translate-x-1/2 text-xs font-bold text-gr-brown">
                          {count}
                        </span>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>

            {/* Rating distribution */}
            <div className="mb-8">
              <h2 className="section-title mb-4">Rating Distribution</h2>
              <div className="space-y-2">
                {[5, 4, 3, 2, 1].map(r => {
                  const count = stats.rating_distribution?.[String(r)] || 0
                  const maxRating = Math.max(1, ...Object.values(stats.rating_distribution || {}))
                  const width = maxRating > 0 ? (count / maxRating) * 100 : 0
                  return (
                    <div key={r} className="flex items-center gap-3">
                      <div className="flex items-center gap-1 w-24">
                        <StarRating rating={r} size={14} />
                      </div>
                      <div className="flex-1 h-5 bg-gray-100 rounded">
                        <div
                          className="h-full bg-gr-orange rounded"
                          style={{ width: `${width}%` }}
                        />
                      </div>
                      <span className="text-sm text-gr-light w-8 text-right">{count}</span>
                    </div>
                  )
                })}
              </div>
            </div>

            {/* Top authors */}
            {stats.top_authors && stats.top_authors.length > 0 && (
              <div className="mb-8">
                <h2 className="section-title mb-4">Top Authors</h2>
                <div className="space-y-2">
                  {stats.top_authors.map((a, i) => (
                    <div key={i} className="flex items-center justify-between py-2 border-b border-gray-100">
                      <span className="text-sm">{a.name}</span>
                      <span className="text-sm text-gr-light">{a.count} books</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Genres */}
            {stats.genres && Object.keys(stats.genres).length > 0 && (
              <div className="mb-8">
                <h2 className="section-title mb-4">Genres</h2>
                <div className="flex flex-wrap gap-2">
                  {Object.entries(stats.genres)
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

            {/* Notable books */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
              {stats.shortest_book && (
                <div>
                  <h3 className="text-sm font-bold text-gr-light uppercase mb-2">Shortest Book</h3>
                  <BookCard book={stats.shortest_book} />
                </div>
              )}
              {stats.longest_book && (
                <div>
                  <h3 className="text-sm font-bold text-gr-light uppercase mb-2">Longest Book</h3>
                  <BookCard book={stats.longest_book} />
                </div>
              )}
              {stats.highest_rated && (
                <div>
                  <h3 className="text-sm font-bold text-gr-light uppercase mb-2">Highest Rated</h3>
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
