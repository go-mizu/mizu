import type { Review } from '../types'
import StarRating from './StarRating'

interface ReviewCardProps {
  review: Review
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

export default function ReviewCard({ review }: ReviewCardProps) {
  const initial = review.book_title ? review.book_title.charAt(0).toUpperCase() : 'R'

  return (
    <div className="review-card">
      <div className="review-header">
        <div className="review-avatar">
          {initial}
        </div>
        <div>
          <StarRating rating={review.rating} />
          <div style={{ fontSize: 12, color: 'var(--gr-light)', marginTop: 2 }}>
            {formatDate(review.created_at)}
          </div>
        </div>
      </div>

      {review.text && (
        <div className="review-text">{review.text}</div>
      )}

      {(review.started_at || review.finished_at) && (
        <div style={{ fontSize: 12, color: 'var(--gr-light)', marginTop: 8 }}>
          {review.started_at && <>Started {formatDate(review.started_at)}</>}
          {review.started_at && review.finished_at && <> &middot; </>}
          {review.finished_at && <>Finished {formatDate(review.finished_at)}</>}
        </div>
      )}
    </div>
  )
}
