import { useState, useRef, useEffect } from 'react'
import { Property, PropertyType } from '../api/client'

interface PropertyHeaderProps {
  property: Property
  onUpdate: (updates: Partial<Property>) => void
  onDelete: () => void
}

const TYPE_ICONS: Record<PropertyType, string> = {
  text: 'T',
  number: '#',
  select: '▼',
  multi_select: '▣',
  date: '📅',
  person: '👤',
  checkbox: '☑',
  url: '🔗',
  email: '✉',
  phone: '📞',
  files: '📎',
  relation: '↗',
  rollup: '∑',
  formula: 'ƒ',
  created_time: '🕐',
  created_by: '👤',
  last_edited_time: '🕐',
  last_edited_by: '👤',
  status: '●',
}

export function PropertyHeader({ property, onUpdate, onDelete }: PropertyHeaderProps) {
  const [editing, setEditing] = useState(false)
  const [showMenu, setShowMenu] = useState(false)
  const [name, setName] = useState(property.name)
  const inputRef = useRef<HTMLInputElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [editing])

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleSave = () => {
    if (name !== property.name) {
      onUpdate({ name })
    }
    setEditing(false)
  }

  const handleTypeChange = (type: PropertyType) => {
    onUpdate({ type })
    setShowMenu(false)
  }

  return (
    <div className="property-header">
      <span className="property-type-icon">{TYPE_ICONS[property.type]}</span>

      {editing ? (
        <input
          ref={inputRef}
          className="property-name-input"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onBlur={handleSave}
          onKeyDown={(e) => {
            if (e.key === 'Enter') handleSave()
            if (e.key === 'Escape') {
              setName(property.name)
              setEditing(false)
            }
          }}
        />
      ) : (
        <span className="property-name" onDoubleClick={() => setEditing(true)}>
          {property.name}
        </span>
      )}

      <button className="property-menu-btn" onClick={() => setShowMenu(!showMenu)}>
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <circle cx="7" cy="3" r="1.5" fill="currentColor" />
          <circle cx="7" cy="7" r="1.5" fill="currentColor" />
          <circle cx="7" cy="11" r="1.5" fill="currentColor" />
        </svg>
      </button>

      {showMenu && (
        <div className="property-menu" ref={menuRef}>
          <div className="menu-section">
            <button className="menu-item" onClick={() => setEditing(true)}>
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path d="M10 2l2 2M2 10l6-6 2 2-6 6H2v-2z" stroke="currentColor" strokeWidth="1.5" />
              </svg>
              Rename
            </button>
          </div>

          <div className="menu-section">
            <div className="menu-header">Property Type</div>
            {(Object.keys(TYPE_ICONS) as PropertyType[]).map((type) => (
              <button
                key={type}
                className={`menu-item ${property.type === type ? 'active' : ''}`}
                onClick={() => handleTypeChange(type)}
              >
                <span className="menu-icon">{TYPE_ICONS[type]}</span>
                <span>{formatTypeName(type)}</span>
              </button>
            ))}
          </div>

          <div className="menu-section">
            <button className="menu-item danger" onClick={onDelete}>
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path d="M2 4h10M5 4V2h4v2M3 4v8a1 1 0 001 1h6a1 1 0 001-1V4" stroke="currentColor" strokeWidth="1.5" />
              </svg>
              Delete property
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function formatTypeName(type: PropertyType): string {
  return type
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}
