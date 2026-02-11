import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { BookOpen, Calendar, Hash, Globe, Building2, FileText } from 'lucide-react'
import Header from '../components/Header'
import BookCover from '../components/BookCover'
import ShelfButton from '../components/ShelfButton'
import StarRating from '../components/StarRating'
import ReviewCard from '../components/ReviewCard'
import BookGrid from '../components/BookGrid'
import { booksApi } from '../api/books'
import { useBookStore } from '../stores/bookStore'
import type { Book, Review } from '../types'

export default function BookDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [book, setBook] = useState<Book | null>(null)
  const [reviews, setReviews] = useState<Review[]>([])
  const [similar, setSimilar] = useState<Book[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'reviews' | 'similar'>('reviews')
  const [showReviewForm, setShowReviewForm] = useState(false)
  const [reviewText, setReviewText] = useState('')
  const [reviewRating, setReviewRating] = useState(0)
  const [submitting, setSubmitting] = useState(false)
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
      await booksApi.createReview({ book_id: book.id, rating })
      setBook({ ...book, user_rating: rating })
    } catch {
      // Silently fail rating
    }
  }

  const handleSubmitReview = async () => {
    if (!book || reviewRating === 0) return
    setSubmitting(true)
    try {
      const review = await booksApi.createReview({
        book_id: book.id,
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

  const genres = book.genres
    ? book.genres.split(',').map((g) => g.trim()).filter(Boolean)
    : []

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
          </div>

          {/* Right Column */}
          <div>
            <h1
              style={{
                fontFamily: "'Merriweather', Georgia, serif",
                fontSize: 28,
                fontWeight: 900,
                color: 'var(--gr-brown)',
                margin: '0 0 8px',
                lineHeight: 1.3,
              }}
            >
              {book.title}
            </h1>

            <p className="book-author" style={{ fontSize: 16, marginBottom: 12 }}>
              by{' '}
              <Link to={`/author/${book.author_id}`} style={{ color: 'var(--gr-text)' }}>
                {book.author_names}
              </Link>
            </p>

            {/* Average rating */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 16 }}>
              <StarRating rating={book.average_rating} />
              <span style={{ fontWeight: 700, fontSize: 16 }}>
                {book.average_rating?.toFixed(2)}
              </span>
              <span style={{ color: 'var(--gr-light)', fontSize: 14 }}>
                ({book.ratings_count?.toLocaleString()} ratings)
              </span>
            </div>

            {/* Description */}
            {book.description && (
              <div
                style={{
                  fontSize: 15,
                  lineHeight: 1.7,
                  color: 'var(--gr-text)',
                  marginBottom: 24,
                }}
              >
                {book.description}
              </div>
            )}

            {/* Metadata */}
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
                gap: 12,
                marginBottom: 24,
                fontSize: 14,
              }}
            >
              {book.page_count > 0 && (
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
              {book.publish_year > 0 && (
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
              {book.ol_key && (
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--gr-light)' }}>
                  <FileText size={14} />
                  <span>{book.ol_key}</span>
                </div>
              )}
            </div>

            {/* Genres */}
            {genres.length > 0 && (
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 16 }}>
                {genres.map((genre) => (
                  <Link key={genre} to={`/genre/${encodeURIComponent(genre)}`} className="genre-tag">
                    {genre}
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Tabs: Reviews / Similar */}
        <div className="tabs">
          <button
            className={`tab ${activeTab === 'reviews' ? 'active' : ''}`}
            onClick={() => setActiveTab('reviews')}
          >
            Reviews ({reviews.length})
          </button>
          <button
            className={`tab ${activeTab === 'similar' ? 'active' : ''}`}
            onClick={() => setActiveTab('similar')}
          >
            Similar Books ({similar.length})
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
      </div>
    </>
  )
}
