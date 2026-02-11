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
              style={{ color: 'inherit', textDecoration: 'underline' }}
            >
              {quote.book.title}
            </Link>
          </>
        )}
      </div>
      {quote.likes_count > 0 && (
        <div style={{ marginTop: 8, fontSize: 12, color: 'var(--gr-light)', display: 'flex', alignItems: 'center', gap: 4 }}>
          <Heart size={12} />
          {quote.likes_count.toLocaleString()} likes
        </div>
      )}
    </div>
  )
}
