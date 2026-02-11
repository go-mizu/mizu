import { useState } from 'react'
import { Star } from 'lucide-react'

interface StarRatingProps {
  rating: number
  count?: number
  interactive?: boolean
  size?: number
  onChange?: (rating: number) => void
}

export default function StarRating({
  rating,
  count,
  interactive = false,
  size = 16,
  onChange,
}: StarRatingProps) {
  const [hovered, setHovered] = useState(0)

  const displayRating = hovered || rating

  return (
    <span className="stars">
      {[1, 2, 3, 4, 5].map((value) => (
        <span
          key={value}
          className={`star${value <= displayRating ? ' filled' : ''}${interactive ? ' interactive' : ''}`}
          onMouseEnter={interactive ? () => setHovered(value) : undefined}
          onMouseLeave={interactive ? () => setHovered(0) : undefined}
          onClick={interactive ? () => onChange?.(value) : undefined}
          role={interactive ? 'button' : undefined}
          aria-label={interactive ? `Rate ${value} stars` : undefined}
        >
          <Star
            size={size}
            fill={value <= displayRating ? 'currentColor' : 'none'}
            strokeWidth={1.5}
          />
        </span>
      ))}
      {count !== undefined && (
        <span className="rating-text">
          {rating.toFixed(2)} avg — {count.toLocaleString()} ratings
        </span>
      )}
    </span>
  )
}
