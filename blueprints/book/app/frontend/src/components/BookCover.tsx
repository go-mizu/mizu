import type { Book } from '../types'

interface BookCoverProps {
  src?: string
  title?: string
  book?: Book
  size?: 'sm' | 'lg'
  className?: string
}

export default function BookCover({ src, title, book, size, className = '' }: BookCoverProps) {
  const imgSrc = src ?? book?.cover_url
  const imgTitle = title ?? book?.title ?? ''
  const sizeClass = size === 'sm' ? 'book-cover-sm' : size === 'lg' ? 'book-cover-lg' : ''
  const classes = `book-cover ${sizeClass} ${className}`.replace(/\s+/g, ' ').trim()

  if (imgSrc) {
    return (
      <img
        src={imgSrc}
        alt={imgTitle}
        className={classes}
        loading="lazy"
      />
    )
  }

  return (
    <div className={`${classes} book-cover-placeholder`.trim()}>
      {imgTitle}
    </div>
  )
}
