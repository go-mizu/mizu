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
          <Link to={`/book/${book.id}`}>
            <BookCover src={book.cover_url} title={book.title} />
          </Link>
          <div className="title">
            <Link to={`/book/${book.id}`} style={{ color: 'inherit', textDecoration: 'none' }}>
              {book.title}
            </Link>
          </div>
          <div className="author">
            <Link to={`/author/${book.author_id}`} style={{ color: 'inherit', textDecoration: 'none' }}>
              {book.author_names}
            </Link>
          </div>
        </div>
      ))}
    </div>
  )
}
