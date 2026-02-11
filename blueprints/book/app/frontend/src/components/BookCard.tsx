import { Link } from 'react-router-dom'
import type { Book } from '../types'
import BookCover from './BookCover'
import StarRating from './StarRating'
import ShelfButton from './ShelfButton'
import { useBookStore } from '../stores/bookStore'

interface BookCardProps {
  book: Book
  showShelf?: boolean
}

export default function BookCard({ book, showShelf = true }: BookCardProps) {
  const shelves = useBookStore((s) => s.shelves)

  return (
    <div className="book-card">
      <Link to={`/book/${book.id}`}>
        <BookCover src={book.cover_url} title={book.title} />
      </Link>

      <div className="book-info">
        <h3 className="book-title">
          <Link to={`/book/${book.id}`}>{book.title}</Link>
        </h3>

        <p className="book-author">
          by{' '}
          <span>{book.author_names}</span>
        </p>

        <div className="book-meta">
          <StarRating
            rating={book.average_rating}
            count={book.ratings_count}
          />
        </div>

        {book.publish_year > 0 && (
          <p className="book-meta">
            Published {book.publish_year}
            {book.page_count > 0 && <> &middot; {book.page_count} pages</>}
          </p>
        )}

        {showShelf && shelves.length > 0 && (
          <div style={{ marginTop: 8 }}>
            <ShelfButton book={book} shelves={shelves} />
          </div>
        )}
      </div>
    </div>
  )
}
