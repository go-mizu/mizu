import type { ReactNode } from 'react'
import type { Shelf } from '../types'

interface SidebarProps {
  shelves?: Shelf[]
  activeShelf?: number
  onSelect?: (id: number) => void
  children?: ReactNode
}

export default function Sidebar({ shelves, activeShelf, onSelect, children }: SidebarProps) {
  if (children) {
    return <aside className="sidebar">{children}</aside>
  }

  const totalBooks = (shelves || []).reduce((sum, s) => sum + s.book_count, 0)

  return (
    <aside className="sidebar">
      <h3>Bookshelves</h3>

      <button
        className={`sidebar-link${activeShelf === undefined ? ' active' : ''}`}
        onClick={() => onSelect?.(0)}
      >
        <span>All</span>
        <span className="sidebar-count">{totalBooks}</span>
      </button>

      {(shelves || []).map((shelf) => (
        <button
          key={shelf.id}
          className={`sidebar-link${activeShelf === shelf.id ? ' active' : ''}`}
          onClick={() => onSelect?.(shelf.id)}
        >
          <span>{shelf.name}</span>
          <span className="sidebar-count">{shelf.book_count}</span>
        </button>
      ))}

      <button
        className="sidebar-link"
        style={{ color: 'var(--gr-teal)', marginTop: 8, fontWeight: 700, fontSize: 13 }}
        onClick={() => onSelect?.(-1)}
      >
        <span>+ Add shelf</span>
      </button>
    </aside>
  )
}
