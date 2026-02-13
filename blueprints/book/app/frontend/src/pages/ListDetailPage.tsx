import { useState, useEffect } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, Users, ExternalLink, Tag, Trash2, Star } from 'lucide-react'
import Header from '../components/Header'
import BookCover from '../components/BookCover'
import { booksApi } from '../api/books'
import type { BookList, BookListItem } from '../types'

export default function ListDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [list, setList] = useState<BookList | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    booksApi.getList(Number(id))
      .then(setList)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [id])

  const handleDelete = () => {
    if (!list || !confirm('Delete this list?')) return
    booksApi.deleteList(list.id)
      .then(() => navigate('/lists'))
      .catch(() => {})
  }

  if (loading) {
    return (
      <>
        <Header />
        <div className="loading-spinner"><div className="spinner" /></div>
      </>
    )
  }

  if (!list) {
    return (
      <>
        <Header />
        <div className="empty-state"><h3>List not found</h3></div>
      </>
    )
  }

  const items = list.items || []

  return (
    <>
      <Header />
      <div className="page-container">
        <div className="list-detail-top-bar">
          <Link to="/lists" className="back-link">
            <ArrowLeft size={14} /> Back to Lists
          </Link>
          <button className="btn btn-secondary btn-sm btn-danger-text" onClick={handleDelete}>
            <Trash2 size={14} /> Delete
          </button>
        </div>

        <div className="list-detail-header">
          <h1 className="list-detail-title">{list.title}</h1>
          {list.description && (
            <p className="page-subtitle">{list.description}</p>
          )}
          <div className="list-detail-meta">
            <span>{list.item_count} books</span>
            {list.voter_count > 0 && (
              <span className="meta-with-icon">
                <Users size={13} /> {list.voter_count.toLocaleString()} voters
              </span>
            )}
            {list.tag && (
              <span className="meta-with-icon list-tag-badge">
                <Tag size={12} /> {list.tag}
              </span>
            )}
            {list.source_url && (
              <a
                href={list.source_url}
                target="_blank"
                rel="noopener noreferrer"
                className="meta-link"
              >
                <ExternalLink size={13} /> Source
              </a>
            )}
          </div>
        </div>

        {items.length > 0 ? (
          <div className="list-book-table">
            {items.map((item) => item.book && (
              <ListBookRow key={item.id} item={item} />
            ))}
          </div>
        ) : (
          <div className="empty-state">
            <h3>No books in this list yet</h3>
            <p>Books will appear here once they are imported.</p>
          </div>
        )}
      </div>
    </>
  )
}

function ListBookRow({ item }: { item: BookListItem }) {
  const book = item.book!
  const rating = book.average_rating || 0

  return (
    <div className="list-book-row">
      <div className="list-book-rank">{item.position}</div>

      <Link to={`/book/${book.id}`} className="list-book-cover-link">
        <BookCover src={book.cover_url} title={book.title} size="sm" />
      </Link>

      <div className="list-book-info">
        <Link to={`/book/${book.id}`} className="list-book-title">
          {book.title}
        </Link>
        {book.series && (
          <span className="list-book-series">({book.series})</span>
        )}
        <div className="list-book-author">
          by {book.author_names || 'Unknown'}
        </div>
      </div>

      <div className="list-book-rating">
        <div className="list-book-stars">
          {[1, 2, 3, 4, 5].map((s) => (
            <Star
              key={s}
              size={13}
              fill={s <= Math.round(rating) ? 'var(--gr-star)' : 'none'}
              color={s <= Math.round(rating) ? 'var(--gr-star)' : '#ccc'}
            />
          ))}
        </div>
        {rating > 0 && (
          <span className="list-book-avg">{rating.toFixed(2)} avg</span>
        )}
        {book.ratings_count > 0 && (
          <span className="list-book-ratings-count">
            {book.ratings_count.toLocaleString()} ratings
          </span>
        )}
      </div>

      {item.votes > 0 && (
        <div className="list-book-score">
          <span className="list-book-score-num">{item.votes.toLocaleString()}</span>
          <span className="list-book-score-label">score</span>
        </div>
      )}
    </div>
  )
}
