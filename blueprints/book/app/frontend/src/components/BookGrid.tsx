import { Link } from 'react-router-dom'
import type { Book } from '../types'
import BookCover from './BookCover'

interface BookGridProps {
  books: Book[]
}

export default function BookGrid({ books }: BookGridProps) {
  return (
    <div className="book-grid">
      {books.map((book) => (
        <div key={book.id} className="book-grid-item">
          <Link to={`/book/${book.id}`} className="book-grid-link">
            <BookCover src={book.cover_url} title={book.title} />
          </Link>
          <div className="title">
            <Link to={`/book/${book.id}`} className="book-grid-title-link">
              {book.title}
            </Link>
          </div>
          <div className="author">
            <span>{book.author_names}</span>
          </div>
        </div>
      ))}
    </div>
  )
}
