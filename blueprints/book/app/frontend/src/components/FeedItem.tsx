import { Link } from 'react-router-dom'
import type { FeedItem as FeedItemType } from '../types'
import StarRating from './StarRating'

interface FeedItemProps {
  item: FeedItemType
}

interface FeedData {
  rating?: number
  text?: string
  shelf_name?: string
  page?: number
  percent?: number
  goal?: number
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

function actionLabel(type: string): string {
  switch (type) {
    case 'rating': return 'rated'
    case 'shelve': return 'shelved'
    case 'review': return 'reviewed'
    case 'progress': return 'is reading'
    case 'challenge': return 'set a reading challenge'
    default: return type
  }
}

export default function FeedItemComponent({ item }: FeedItemProps) {
  let data: FeedData = {}
  try {
    data = JSON.parse(item.data || '{}')
  } catch {
    // ignore parse errors
  }

  return (
    <div className="feed-item">
      <div style={{ flex: 1 }}>
        <div className="action-text">
          <span>{actionLabel(item.type)} </span>
          {item.book_id ? (
            <Link to={`/book/${item.book_id}`}>{item.book_title}</Link>
          ) : (
            <span>{item.book_title}</span>
          )}
          {data.shelf_name && item.type === 'shelve' && (
            <span> ({data.shelf_name})</span>
          )}
        </div>

        {data.rating && data.rating > 0 && (
          <div style={{ marginTop: 4 }}>
            <StarRating rating={data.rating} size={14} />
          </div>
        )}

        {data.text && (
          <div style={{ fontSize: 13, color: 'var(--gr-text)', marginTop: 4, lineHeight: 1.5 }}>
            {data.text.length > 200 ? data.text.slice(0, 200) + '...' : data.text}
          </div>
        )}

        {item.type === 'progress' && data.percent !== undefined && (
          <div style={{ fontSize: 13, color: 'var(--gr-light)', marginTop: 4 }}>
            {data.page ? `Page ${data.page} · ` : ''}{Math.round(data.percent)}% complete
          </div>
        )}

        <div className="feed-time">{timeAgo(item.created_at)}</div>
      </div>
    </div>
  )
}
