import { useEffect, useState } from 'react'
import type { Review, ReviewComment } from '../types'
import { booksApi } from '../api/books'
import StarRating from './StarRating'

interface ReviewCardProps {
  review: Review
  onEdit?: (review: Review) => void
  onDelete?: (review: Review) => void
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

export default function ReviewCard({ review, onEdit, onDelete }: ReviewCardProps) {
  const [current, setCurrent] = useState(review)
  const [spoilerVisible, setSpoilerVisible] = useState(false)
  const [showComments, setShowComments] = useState(false)
  const [comments, setComments] = useState<ReviewComment[]>([])
  const [commentsLoading, setCommentsLoading] = useState(false)
  const [commentText, setCommentText] = useState('')
  const [liking, setLiking] = useState(false)
  const [liked, setLiked] = useState(false)

  useEffect(() => {
    setCurrent(review)
    setSpoilerVisible(false)
    setLiked(false)
  }, [review])

  const isGoodreads = current.source === 'goodreads'
  const displayName = current.reviewer_name || (current.source === 'user' ? 'You' : (current.book?.title ? current.book.title.charAt(0).toUpperCase() : 'R'))
  const initial = displayName.charAt(0).toUpperCase()

  const handleLike = async () => {
    if (liked || liking) return
    setLiking(true)
    try {
      const data = await booksApi.likeReview(current.id)
      setCurrent({ ...current, likes_count: data.likes_count })
      setLiked(true)
    } catch {
      // ignore
    } finally {
      setLiking(false)
    }
  }

  const loadComments = async () => {
    setCommentsLoading(true)
    try {
      const data = await booksApi.getReviewComments(current.id)
      setComments(data.comments || [])
    } catch {
      // ignore
    } finally {
      setCommentsLoading(false)
    }
  }

  const toggleComments = () => {
    const next = !showComments
    setShowComments(next)
    if (next && comments.length === 0) {
      loadComments()
    }
  }

  const handleAddComment = async () => {
    const text = commentText.trim()
    if (!text) return
    try {
      const comment = await booksApi.createReviewComment(current.id, { text, author_name: 'You' })
      setComments([comment, ...comments])
      setCommentText('')
      setCurrent({ ...current, comments_count: (current.comments_count || 0) + 1 })
    } catch {
      // ignore
    }
  }

  const handleDeleteComment = async (commentId: number) => {
    try {
      await booksApi.deleteReviewComment(current.id, commentId)
      setComments(comments.filter(c => c.id !== commentId))
      setCurrent({ ...current, comments_count: Math.max(0, (current.comments_count || 0) - 1) })
    } catch {
      // ignore
    }
  }

  return (
    <div className="review-card">
      <div className="review-header">
        <div className="review-avatar">
          {initial}
        </div>
        <div style={{ flex: 1 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontSize: 14, fontWeight: 700, color: 'var(--gr-brown)' }}>
              {displayName}
            </span>
            {isGoodreads && (
              <span style={{
                fontSize: 10,
                padding: '2px 6px',
                background: 'var(--gr-tan)',
                borderRadius: 3,
                color: 'var(--gr-light)',
                fontWeight: 700,
                textTransform: 'uppercase',
                letterSpacing: 0.5,
              }}>
                Goodreads
              </span>
            )}
          </div>
          <StarRating rating={current.rating} />
          <div style={{ fontSize: 12, color: 'var(--gr-light)', marginTop: 2 }}>
            {formatDate(current.created_at)}
            {current.likes_count ? ` · ${current.likes_count} likes` : ''}
            {current.comments_count ? ` · ${current.comments_count} comments` : ''}
          </div>
        </div>
        {current.source === 'user' && (onEdit || onDelete) && (
          <div className="review-owner-actions">
            {onEdit && (
              <button className="link-button" onClick={() => onEdit(current)}>Edit</button>
            )}
            {onDelete && (
              <button className="link-button" onClick={() => onDelete(current)}>Delete</button>
            )}
          </div>
        )}
      </div>

      {current.text && current.is_spoiler && !spoilerVisible ? (
        <div className="review-spoiler">
          <div>
            <strong>Spoiler</strong> · This review contains spoilers.
          </div>
          <button className="link-button" onClick={() => setSpoilerVisible(true)}>Reveal</button>
        </div>
      ) : current.text ? (
        <div className="review-text">{current.text}</div>
      ) : null}

      <div className="review-actions">
        <button className="link-button" onClick={handleLike} disabled={liking || liked}>
          {liked ? 'Liked' : 'Like'}
        </button>
        <button className="link-button" onClick={toggleComments}>
          Comment
        </button>
      </div>

      {(current.started_at || current.finished_at) && (
        <div style={{ fontSize: 12, color: 'var(--gr-light)', marginTop: 8 }}>
          {current.started_at && <>Started {formatDate(current.started_at)}</>}
          {current.started_at && current.finished_at && <> &middot; </>}
          {current.finished_at && <>Finished {formatDate(current.finished_at)}</>}
        </div>
      )}

      {showComments && (
        <div className="review-comments">
          <div className="comment-form">
            <input
              className="form-input"
              placeholder="Add a comment..."
              value={commentText}
              onChange={(e) => setCommentText(e.target.value)}
            />
            <button className="btn btn-secondary btn-sm" onClick={handleAddComment} disabled={!commentText.trim()}>
              Post
            </button>
          </div>
          {commentsLoading ? (
            <div className="comment-loading">Loading comments...</div>
          ) : comments.length > 0 ? (
            comments.map((c) => (
              <div key={c.id} className="comment-item">
                <div>
                  <strong>{c.author_name}</strong> {c.text}
                </div>
                <div className="comment-meta">
                  <span>{formatDate(c.created_at)}</span>
                  {c.author_name === 'You' && (
                    <button className="link-button" onClick={() => handleDeleteComment(c.id)}>Delete</button>
                  )}
                </div>
              </div>
            ))
          ) : (
            <div className="comment-empty">No comments yet.</div>
          )}
        </div>
      )}
    </div>
  )
}
