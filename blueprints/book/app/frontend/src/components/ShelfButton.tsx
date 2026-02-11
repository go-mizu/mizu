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
    <div ref={ref} style={{ position: 'relative', display: 'inline-block' }}>
      <button
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
        <div
          style={{
            position: 'absolute',
            top: '100%',
            left: 0,
            marginTop: 4,
            background: '#fff',
            border: '1px solid var(--gr-border)',
            borderRadius: 4,
            boxShadow: '0 4px 12px rgba(0,0,0,0.12)',
            minWidth: 180,
            zIndex: 20,
          }}
        >
          {shelves.map((shelf) => (
            <button
              key={shelf.id}
              onClick={() => handleSelect(shelf)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                width: '100%',
                padding: '8px 12px',
                border: 'none',
                background: 'none',
                cursor: 'pointer',
                fontSize: 14,
                fontFamily: 'inherit',
                color: 'var(--gr-text)',
                textAlign: 'left',
              }}
              onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--gr-hover)')}
              onMouseLeave={(e) => (e.currentTarget.style.background = 'none')}
            >
              {currentShelf === shelf.name && <Check size={14} style={{ color: 'var(--gr-green)' }} />}
              <span>{shelf.name}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
