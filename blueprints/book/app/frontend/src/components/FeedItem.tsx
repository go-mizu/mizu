import { Link } from 'react-router-dom'
import type { FeedItem as FeedItemType } from '../types'
import BookCover from './BookCover'
import StarRating from './StarRating'

interface FeedItemProps {
  item: FeedItemType
}

function timeAgo(dateStr?: string): string {
  if (!dateStr) return ''
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000)
  if (seconds < 60) return 'just now'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  const months = Math.floor(days / 30)
  return `${months}mo ago`
}

function actionLabel(action: string): string {
  switch (action) {
    case 'rated': return 'rated'
    case 'shelved': return 'wants to read'
    case 'reviewed': return 'reviewed'
    case 'finished': return 'finished reading'
    case 'started': return 'started reading'
    default: return action
  }
}

export default function FeedItemComponent({ item }: FeedItemProps) {
  return (
    <div className="feed-item">
      <Link to={`/book/${item.book_id}`}>
        <BookCover
          src={item.book_cover}
          title={item.book_title}
          className="book-cover-sm"
        />
      </Link>

      <div style={{ flex: 1 }}>
        <div className="action-text">
          <span>{actionLabel(item.action)} </span>
          <Link to={`/book/${item.book_id}`}>{item.book_title}</Link>
          {item.author_name && (
            <span> by {item.author_name}</span>
          )}
          {item.shelf_name && item.action === 'shelved' && (
            <span> ({item.shelf_name})</span>
          )}
        </div>

        {item.rating > 0 && (
          <div style={{ marginTop: 4 }}>
            <StarRating rating={item.rating} size={14} />
          </div>
        )}

        {item.review_text && (
          <div style={{ fontSize: 13, color: 'var(--gr-text)', marginTop: 4, lineHeight: 1.5 }}>
            {item.review_text.length > 200
              ? item.review_text.slice(0, 200) + '...'
              : item.review_text}
          </div>
        )}

        <div className="feed-time">{timeAgo(item.created_at)}</div>
      </div>
    </div>
  )
}
