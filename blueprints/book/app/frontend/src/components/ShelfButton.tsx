import { useState, useRef, useEffect } from 'react'
import { ChevronDown, Check } from 'lucide-react'
import type { Book, Shelf } from '../types'
import { booksApi } from '../api/books'
import { useBookStore } from '../stores/bookStore'

interface ShelfButtonProps {
  book: Book
  shelves?: Shelf[]
}

export default function ShelfButton({ book, shelves: shelvesProp }: ShelfButtonProps) {
  const storeShelves = useBookStore((s) => s.shelves)
  const shelves = shelvesProp ?? storeShelves
  const [open, setOpen] = useState(false)
  const [currentShelf, setCurrentShelf] = useState(book.user_shelf || '')
  const [loading, setLoading] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleSelect = async (shelf: Shelf) => {
    setLoading(true)
    try {
      await booksApi.addToShelf(shelf.id, book.id)
      setCurrentShelf(shelf.name)
    } catch {
      // silently fail
    } finally {
      setLoading(false)
      setOpen(false)
    }
  }

  const label = currentShelf || 'Want to Read'
  const isShelved = !!currentShelf

  return (
    <div ref={ref} className="shelf-dropdown">
      <button
        type="button"
        className={`shelf-btn${isShelved ? ' shelved' : ''}`}
        onClick={() => setOpen(!open)}
        disabled={loading}
      >
        {isShelved && <Check size={14} />}
        <span>{label}</span>
        <span className="dropdown-arrow">
          <ChevronDown size={14} />
        </span>
      </button>

      {open && (
        <div className="shelf-menu">
          {shelves.map((shelf) => (
            <button
              type="button"
              key={shelf.id}
              onClick={() => handleSelect(shelf)}
              className="shelf-menu-item"
            >
              {currentShelf === shelf.name && <Check size={14} className="shelf-menu-check" />}
              <span>{shelf.name}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
