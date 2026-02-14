import { Link } from 'react-router-dom'
import { Heart } from 'lucide-react'
import type { Quote } from '../types'

interface QuoteCardProps {
  quote: Quote
}

export default function QuoteCard({ quote }: QuoteCardProps) {
  return (
    <div className="quote-card">
      <div className="quote-text">
        &ldquo;{quote.text}&rdquo;
      </div>
      <div className="quote-attr">
        &mdash; {quote.author_name}
        {quote.book && (
          <>
            ,{' '}
            <Link
              to={`/book/${quote.book_id}`}
              className="quote-book-link"
            >
              {quote.book.title}
            </Link>
          </>
        )}
      </div>
      {quote.likes_count > 0 && (
        <div className="quote-likes">
          <Heart size={12} />
          {quote.likes_count.toLocaleString()} likes
        </div>
      )}
    </div>
  )
}
