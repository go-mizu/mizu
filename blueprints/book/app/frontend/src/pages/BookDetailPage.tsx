import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { BookOpen, Calendar, Hash, Globe, Building2, FileText, Quote as QuoteIcon, Users, Eye, BookMarked } from 'lucide-react'
import Header from '../components/Header'
import BookCover from '../components/BookCover'
import ShelfButton from '../components/ShelfButton'
import StarRating from '../components/StarRating'
import ReviewCard from '../components/ReviewCard'
import BookGrid from '../components/BookGrid'
import { booksApi } from '../api/books'
import { useBookStore } from '../stores/bookStore'
import type { Book, Review, Quote, ReadingProgress } from '../types'

function RatingDistribution({ book }: { book: Book }) {
  const dist = book.rating_dist || [0, 0, 0, 0, 0]
  const total = dist.reduce((a, b) => a + b, 0) || 1
  const labels = ['5', '4', '3', '2', '1']

  return (
    <div style={{ marginBottom: 24 }}>
      <h3 style={{
        fontFamily: "'Merriweather', Georgia, serif",
        fontSize: 14,
        fontWeight: 700,
        color: 'var(--gr-brown)',
        marginBottom: 12,
      }}>
        Rating distribution
      </h3>
      {dist.map((count, i) => {
        const pct = total > 0 ? (count / total) * 100 : 0
        return (
          <div key={i} style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            marginBottom: 4,
            fontSize: 13,
          }}>
            <span style={{
              width: 16,
              textAlign: 'right',
              color: 'var(--gr-light)',
              fontWeight: 700,
            }}>
              {labels[i]}
            </span>
            <div style={{
              flex: 1,
              height: 10,
              background: '#eee',
              borderRadius: 2,
              overflow: 'hidden',
            }}>
              <div style={{
                height: '100%',
                width: `${pct}%`,
                background: 'var(--gr-star)',
                borderRadius: 2,
                transition: 'width 0.4s ease',
              }} />
            </div>
            <span style={{
              width: 60,
              textAlign: 'right',
              color: 'var(--gr-light)',
              fontSize: 12,
            }}>
              {count.toLocaleString()} ({Math.round(pct)}%)
            </span>
          </div>
        )
      })}
    </div>
  )
}

function CommunityStats({ book }: { book: Book }) {
  if (!book.currently_reading && !book.want_to_read) return null
  return (
    <div style={{
      display: 'flex',
      gap: 20,
      padding: '12px 0',
      borderTop: '1px solid #eee',
      borderBottom: '1px solid #eee',
      marginBottom: 16,
      fontSize: 13,
      color: 'var(--gr-light)',
    }}>
      {book.currently_reading > 0 && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <Eye size={14} />
          <span><strong style={{ color: 'var(--gr-text)' }}>{book.currently_reading.toLocaleString()}</strong> currently reading</span>
        </div>
      )}
      {book.want_to_read > 0 && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <BookMarked size={14} />
          <span><strong style={{ color: 'var(--gr-text)' }}>{book.want_to_read.toLocaleString()}</strong> want to read</span>
        </div>
      )}
    </div>
  )
}

function ExpandableDescription({ text }: { text: string }) {
  const [expanded, setExpanded] = useState(false)
  const threshold = 400

  if (text.length <= threshold) {
    return (
      <div style={{ fontSize: 15, lineHeight: 1.7, color: 'var(--gr-text)', marginBottom: 24 }}>
        {text}
      </div>
    )
  }

  return (
    <div style={{ fontSize: 15, lineHeight: 1.7, color: 'var(--gr-text)', marginBottom: 24 }}>
      {expanded ? text : text.slice(0, threshold) + '...'}
      <button
        onClick={() => setExpanded(!expanded)}
        style={{
          background: 'none',
          border: 'none',
          color: 'var(--gr-teal)',
          cursor: 'pointer',
          fontWeight: 700,
          fontSize: 14,
          marginLeft: 4,
          padding: 0,
        }}
      >
        {expanded ? 'less' : 'more'}
      </button>
    </div>
  )
}

export default function BookDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [book, setBook] = useState<Book | null>(null)
  const [reviews, setReviews] = useState<Review[]>([])
  const [similar, setSimilar] = useState<Book[]>([])
  const [quotes, setQuotes] = useState<Quote[]>([])
  const [progress, setProgress] = useState<ReadingProgress[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'reviews' | 'similar' | 'quotes'>('reviews')
  const [showReviewForm, setShowReviewForm] = useState(false)
  const [reviewText, setReviewText] = useState('')
  const [reviewRating, setReviewRating] = useState(0)
  const [submitting, setSubmitting] = useState(false)
  const [progressPage, setProgressPage] = useState('')
  const [progressNote, setProgressNote] = useState('')
  const [updatingProgress, setUpdatingProgress] = useState(false)
  const addRecentBook = useBookStore((s) => s.addRecentBook)
  const setCurrentBook = useBookStore((s) => s.setCurrentBook)

  const bookId = Number(id)

  useEffect(() => {
    if (!id) return

    const fetchData = async () => {
      setLoading(true)
      setError(null)
      try {
        const [bookData, reviewsData, similarData] = await Promise.all([
          booksApi.getBook(bookId),
          booksApi.getReviews(bookId),
          booksApi.getSimilar(bookId),
        ])
        setBook(bookData)
        setReviews(reviewsData)
        setSimilar(similarData)
        addRecentBook(bookData)
        setCurrentBook(bookData)

        // Fetch quotes and progress separately (non-blocking)
        booksApi.getBookQuotes(bookId).then(setQuotes).catch(() => {})
        booksApi.getProgress(bookId).then(setProgress).catch(() => {})
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load book')
      } finally {
        setLoading(false)
      }
    }
    fetchData()

    return () => setCurrentBook(null)
  }, [id, bookId, addRecentBook, setCurrentBook])

  const handleRating = async (rating: number) => {
    if (!book) return
    try {
      await booksApi.createReview(book.id, { rating })
      setBook({ ...book, user_rating: rating })
    } catch {
      // Silently fail rating
    }
  }

  const handleSubmitReview = async () => {
    if (!book || reviewRating === 0) return
    setSubmitting(true)
    try {
      const review = await booksApi.createReview(book.id, {
        rating: reviewRating,
        text: reviewText,
      })
      setReviews([review, ...reviews])
      setShowReviewForm(false)
      setReviewText('')
      setReviewRating(0)
      setBook({ ...book, user_rating: reviewRating })
    } catch {
      // Handle error silently
    } finally {
      setSubmitting(false)
    }
  }

  const handleUpdateProgress = async () => {
    if (!book) return
    const page = parseInt(progressPage)
    if (!page || page <= 0) return
    setUpdatingProgress(true)
    try {
      const percent = book.page_count > 0 ? (page / book.page_count) * 100 : 0
      const p = await booksApi.updateProgress(book.id, {
        page,
        percent: Math.min(100, percent),
        note: progressNote || undefined,
      })
      setProgress([p, ...progress])
      setProgressPage('')
      setProgressNote('')
    } catch {
      // Handle error silently
    } finally {
      setUpdatingProgress(false)
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

  if (error || !book) {
    return (
      <>
        <Header />
        <div className="page-container">
          <div className="empty-state">
            <h3>Book not found</h3>
            <p>{error || 'This book does not exist.'}</p>
            <Link to="/browse" className="btn btn-secondary">
              Browse books
            </Link>
          </div>
        </div>
      </>
    )
  }

  const genres = book.subjects || []
  const latestProgress = progress.length > 0 ? progress[0] : null
  const hasRatingDist = book.rating_dist && book.rating_dist.some(n => n > 0)

  return (
    <>
      <Header />
      <div className="page-container fade-in">
        {/* Book Detail Layout */}
        <div style={{ display: 'grid', gridTemplateColumns: '200px 1fr', gap: 32, marginBottom: 40 }}>
          {/* Left Column */}
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 16 }}>
            <BookCover book={book} size="lg" />
            <ShelfButton book={book} />
            <div style={{ textAlign: 'center' }}>
              <div style={{ fontSize: 13, color: 'var(--gr-light)', marginBottom: 4 }}>
                Rate this book
              </div>
              <StarRating
                rating={book.user_rating || 0}
                interactive
                onChange={handleRating}
              />
            </div>

            {/* Reading Progress */}
            {book.page_count > 0 && (
              <div style={{ width: '100%', padding: 12, background: 'var(--gr-cream)', borderRadius: 8 }}>
                <div style={{ fontSize: 12, fontWeight: 700, color: 'var(--gr-brown)', marginBottom: 8 }}>
                  Track Progress
                </div>
                {latestProgress && (
                  <div style={{ marginBottom: 8 }}>
                    <div className="progress-bar" style={{ height: 6 }}>
                      <div
                        className="progress-fill"
                        style={{ width: `${Math.min(100, latestProgress.percent)}%` }}
                      />
                    </div>
                    <div style={{ fontSize: 11, color: 'var(--gr-light)', marginTop: 4 }}>
                      Page {latestProgress.page} of {book.page_count} ({Math.round(latestProgress.percent)}%)
                    </div>
                  </div>
                )}
                <div style={{ display: 'flex', gap: 4 }}>
                  <input
                    type="number"
                    min="1"
                    max={book.page_count}
                    value={progressPage}
                    onChange={(e) => setProgressPage(e.target.value)}
                    placeholder="Page #"
                    className="form-input"
                    style={{ fontSize: 12, padding: '4px 8px', flex: 1 }}
                  />
                  <button
                    className="btn btn-secondary btn-sm"
                    onClick={handleUpdateProgress}
                    disabled={updatingProgress || !progressPage}
                    style={{ fontSize: 11, padding: '4px 8px' }}
                  >
                    {updatingProgress ? '...' : 'Update'}
                  </button>
                </div>
                <input
                  type="text"
                  value={progressNote}
                  onChange={(e) => setProgressNote(e.target.value)}
                  placeholder="Add a note (optional)"
                  className="form-input"
                  style={{ fontSize: 11, padding: '4px 8px', marginTop: 4, width: '100%' }}
                />
              </div>
            )}
          </div>

          {/* Right Column */}
          <div>
            <h1
              style={{
                fontFamily: "'Merriweather', Georgia, serif",
                fontSize: 28,
                fontWeight: 900,
                color: 'var(--gr-brown)',
                margin: '0 0 4px',
                lineHeight: 1.3,
              }}
            >
              {book.title}
            </h1>

            {book.series && (
              <div style={{ fontSize: 14, color: 'var(--gr-light)', marginBottom: 8, fontStyle: 'italic' }}>
                {book.series}
              </div>
            )}

            <p className="book-author" style={{ fontSize: 16, marginBottom: 12 }}>
              by{' '}
              <span style={{ color: 'var(--gr-text)' }}>
                {book.author_names}
              </span>
            </p>

            {/* Average rating */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
              <StarRating rating={book.average_rating} />
              <span style={{ fontWeight: 700, fontSize: 16 }}>
                {book.average_rating?.toFixed(2)}
              </span>
            </div>
            <div style={{ fontSize: 13, color: 'var(--gr-light)', marginBottom: 16, display: 'flex', gap: 12 }}>
              <span>{book.ratings_count?.toLocaleString()} ratings</span>
              {book.reviews_count > 0 && <span>{book.reviews_count.toLocaleString()} reviews</span>}
            </div>

            {/* Community stats */}
            <CommunityStats book={book} />

            {/* Description */}
            {book.description && <ExpandableDescription text={book.description} />}

            {/* Genres */}
            {genres.length > 0 && (
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 20 }}>
                {genres.map((genre) => (
                  <Link key={genre} to={`/genre/${encodeURIComponent(genre)}`} className="genre-tag">
                    {genre}
                  </Link>
                ))}
              </div>
            )}

            {/* Rating Distribution */}
            {hasRatingDist && <RatingDistribution book={book} />}

            {/* Book Details */}
            <div style={{
              padding: 16,
              background: 'var(--gr-cream)',
              borderRadius: 8,
              marginBottom: 24,
            }}>
              <h3 style={{
                fontFamily: "'Merriweather', Georgia, serif",
                fontSize: 14,
                fontWeight: 700,
                color: 'var(--gr-brown)',
                marginBottom: 12,
              }}>
                Book details & editions
              </h3>
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
                  gap: 10,
                  fontSize: 14,
                }}
              >
                {book.format && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--gr-light)' }}>
                    <BookOpen size={14} />
                    <span>{book.format}{book.page_count > 0 ? `, ${book.page_count} pages` : ''}</span>
                  </div>
                )}
                {!book.format && book.page_count > 0 && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--gr-light)' }}>
                    <BookOpen size={14} />
                    <span>{book.page_count} pages</span>
                  </div>
                )}
                {book.publisher && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--gr-light)' }}>
                    <Building2 size={14} />
                    <span>{book.publisher}</span>
                  </div>
                )}
                {book.first_published && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--gr-light)' }}>
                    <Calendar size={14} />
                    <span>First published {book.first_published}</span>
                  </div>
                )}
                {!book.first_published && book.publish_year > 0 && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--gr-light)' }}>
                    <Calendar size={14} />
                    <span>Published {book.publish_year}</span>
                  </div>
                )}
                {book.isbn13 && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--gr-light)' }}>
                    <Hash size={14} />
                    <span>ISBN {book.isbn13}</span>
                  </div>
                )}
                {book.language && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--gr-light)' }}>
                    <Globe size={14} />
                    <span>{book.language}</span>
                  </div>
                )}
                {book.asin && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--gr-light)' }}>
                    <FileText size={14} />
                    <span>ASIN {book.asin}</span>
                  </div>
                )}
                {book.goodreads_id && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--gr-light)' }}>
                    <Users size={14} />
                    <a
                      href={`https://www.goodreads.com/book/show/${book.goodreads_id}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      style={{ color: 'var(--gr-teal)', textDecoration: 'none' }}
                    >
                      View on Goodreads
                    </a>
                  </div>
                )}
                {book.ol_key && !book.goodreads_id && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--gr-light)' }}>
                    <FileText size={14} />
                    <span>{book.ol_key}</span>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Tabs: Reviews / Similar / Quotes */}
        <div className="tabs">
          <button
            className={`tab ${activeTab === 'reviews' ? 'active' : ''}`}
            onClick={() => setActiveTab('reviews')}
          >
            Community Reviews ({book.reviews_count || reviews.length})
          </button>
          <button
            className={`tab ${activeTab === 'similar' ? 'active' : ''}`}
            onClick={() => setActiveTab('similar')}
          >
            Similar Books ({similar.length})
          </button>
          <button
            className={`tab ${activeTab === 'quotes' ? 'active' : ''}`}
            onClick={() => setActiveTab('quotes')}
          >
            Quotes ({quotes.length})
          </button>
        </div>

        {/* Reviews Tab */}
        {activeTab === 'reviews' && (
          <div>
            {!showReviewForm && (
              <button
                className="btn btn-primary"
                style={{ marginBottom: 20 }}
                onClick={() => setShowReviewForm(true)}
              >
                Write a review
              </button>
            )}

            {showReviewForm && (
              <div
                style={{
                  padding: 20,
                  background: 'var(--gr-cream)',
                  borderRadius: 8,
                  marginBottom: 24,
                }}
              >
                <h3
                  style={{
                    fontFamily: "'Merriweather', Georgia, serif",
                    fontSize: 16,
                    color: 'var(--gr-brown)',
                    marginBottom: 12,
                  }}
                >
                  Write your review
                </h3>
                <div className="form-group">
                  <label className="form-label">Your rating</label>
                  <StarRating
                    rating={reviewRating}
                    interactive
                    onChange={setReviewRating}
                    size={24}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Review (optional)</label>
                  <textarea
                    className="form-input"
                    placeholder="What did you think?"
                    value={reviewText}
                    onChange={(e) => setReviewText(e.target.value)}
                  />
                </div>
                <div style={{ display: 'flex', gap: 8 }}>
                  <button
                    className="btn btn-primary"
                    onClick={handleSubmitReview}
                    disabled={submitting || reviewRating === 0}
                  >
                    {submitting ? 'Posting...' : 'Post review'}
                  </button>
                  <button
                    className="btn btn-secondary"
                    onClick={() => {
                      setShowReviewForm(false)
                      setReviewText('')
                      setReviewRating(0)
                    }}
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}

            {reviews.length > 0 ? (
              reviews.map((review) => (
                <ReviewCard key={review.id} review={review} />
              ))
            ) : (
              <div className="empty-state">
                <h3>No reviews yet</h3>
                <p>Be the first to review this book!</p>
              </div>
            )}
          </div>
        )}

        {/* Similar Books Tab */}
        {activeTab === 'similar' && (
          <div>
            {similar.length > 0 ? (
              <BookGrid books={similar} />
            ) : (
              <div className="empty-state">
                <h3>No similar books found</h3>
                <p>Check back later for recommendations.</p>
              </div>
            )}
          </div>
        )}

        {/* Quotes Tab */}
        {activeTab === 'quotes' && (
          <div>
            {quotes.length > 0 ? (
              quotes.map((quote) => (
                <div
                  key={quote.id}
                  style={{
                    padding: 20,
                    borderLeft: '3px solid var(--gr-orange)',
                    background: 'var(--gr-cream)',
                    borderRadius: '0 8px 8px 0',
                    marginBottom: 16,
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'flex-start', gap: 8 }}>
                    <QuoteIcon size={16} style={{ color: 'var(--gr-orange)', flexShrink: 0, marginTop: 2 }} />
                    <div>
                      <p style={{
                        fontSize: 15,
                        lineHeight: 1.7,
                        color: 'var(--gr-text)',
                        fontStyle: 'italic',
                        margin: '0 0 8px',
                      }}>
                        &ldquo;{quote.text}&rdquo;
                      </p>
                      <p style={{ fontSize: 13, color: 'var(--gr-light)', margin: 0 }}>
                        &mdash; {quote.author_name}
                      </p>
                      {quote.likes_count > 0 && (
                        <p style={{ fontSize: 12, color: 'var(--gr-light)', margin: '4px 0 0' }}>
                          {quote.likes_count} likes
                        </p>
                      )}
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="empty-state">
                <h3>No quotes yet</h3>
                <p>Add memorable quotes from this book.</p>
              </div>
            )}
          </div>
        )}
      </div>
    </>
  )
}
