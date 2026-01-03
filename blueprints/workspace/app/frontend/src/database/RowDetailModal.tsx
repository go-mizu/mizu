import { useState, useEffect, useRef, useCallback } from 'react'
import { createPortal } from 'react-dom'
import { X, MessageSquare, Clock, History, MoreHorizontal, Trash2, Copy, ExternalLink } from 'lucide-react'
import { api, Database, DatabaseRow, Property } from '../api/client'
import { PropertyCell } from './PropertyCell'
import { format, parseISO } from 'date-fns'

interface RowDetailModalProps {
  row: DatabaseRow
  database: Database
  onClose: () => void
  onUpdate: (row: DatabaseRow) => void
  onDelete?: (rowId: string) => void
}

export function RowDetailModal({ row, database, onClose, onUpdate, onDelete }: RowDetailModalProps) {
  const [localRow, setLocalRow] = useState(row)
  const [title, setTitle] = useState(row.properties.title as string || 'Untitled')
  const [isSaving, setIsSaving] = useState(false)
  const [showMenu, setShowMenu] = useState(false)
  const [activeTab, setActiveTab] = useState<'properties' | 'comments' | 'activity'>('properties')
  const modalRef = useRef<HTMLDivElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  // Close on escape
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  // Close menu on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Update title
  const handleTitleChange = useCallback(async (newTitle: string) => {
    setTitle(newTitle)
    const updated = {
      ...localRow,
      properties: { ...localRow.properties, title: newTitle }
    }
    setLocalRow(updated)

    try {
      setIsSaving(true)
      await api.patch(`/rows/${row.id}`, { properties: updated.properties })
      onUpdate(updated)
    } catch (err) {
      console.error('Failed to update row:', err)
    } finally {
      setIsSaving(false)
    }
  }, [localRow, row.id, onUpdate])

  // Update property
  const handlePropertyChange = useCallback(async (propertyId: string, value: unknown) => {
    const updated = {
      ...localRow,
      properties: { ...localRow.properties, [propertyId]: value }
    }
    setLocalRow(updated)

    try {
      setIsSaving(true)
      await api.patch(`/rows/${row.id}`, { properties: updated.properties })
      onUpdate(updated)
    } catch (err) {
      console.error('Failed to update row:', err)
    } finally {
      setIsSaving(false)
    }
  }, [localRow, row.id, onUpdate])

  // Delete row
  const handleDelete = useCallback(async () => {
    if (!confirm('Are you sure you want to delete this row?')) return

    try {
      await api.delete(`/rows/${row.id}`)
      onDelete?.(row.id)
      onClose()
    } catch (err) {
      console.error('Failed to delete row:', err)
    }
  }, [row.id, onDelete, onClose])

  // Duplicate row
  const handleDuplicate = useCallback(async () => {
    try {
      const newRow = await api.post<DatabaseRow>(`/databases/${database.id}/rows`, {
        properties: { ...localRow.properties, title: `${title} (copy)` }
      })
      onUpdate(newRow)
      setShowMenu(false)
    } catch (err) {
      console.error('Failed to duplicate row:', err)
    }
  }, [database.id, localRow.properties, title, onUpdate])

  // Get non-title properties
  const otherProperties = database.properties.filter(p => p.name.toLowerCase() !== 'title')

  // Format dates
  const createdAt = row.created_at ? format(parseISO(row.created_at), 'MMM d, yyyy h:mm a') : ''
  const updatedAt = row.updated_at ? format(parseISO(row.updated_at), 'MMM d, yyyy h:mm a') : ''

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      <div
        ref={modalRef}
        className="row-detail-modal"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="modal-header">
          <div className="modal-header-left">
            <button
              className="icon-btn"
              onClick={() => setActiveTab('comments')}
              title="Comments"
            >
              <MessageSquare size={16} />
            </button>
            <button
              className="icon-btn"
              onClick={() => setActiveTab('activity')}
              title="Activity"
            >
              <History size={16} />
            </button>
          </div>
          <div className="modal-header-right">
            {isSaving && <span className="save-indicator">Saving...</span>}
            <div className="menu-wrapper" ref={menuRef}>
              <button
                className="icon-btn"
                onClick={() => setShowMenu(!showMenu)}
                title="More options"
              >
                <MoreHorizontal size={16} />
              </button>
              {showMenu && (
                <div className="dropdown-menu">
                  <button onClick={handleDuplicate}>
                    <Copy size={14} />
                    <span>Duplicate</span>
                  </button>
                  <button onClick={() => window.open(`/p/${row.id}`, '_blank')}>
                    <ExternalLink size={14} />
                    <span>Open as page</span>
                  </button>
                  <hr />
                  <button className="danger" onClick={handleDelete}>
                    <Trash2 size={14} />
                    <span>Delete</span>
                  </button>
                </div>
              )}
            </div>
            <button className="icon-btn" onClick={onClose} title="Close">
              <X size={16} />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="modal-content">
          {/* Title */}
          <div className="row-title-wrapper">
            <input
              type="text"
              className="row-title-input"
              value={title}
              onChange={e => setTitle(e.target.value)}
              onBlur={e => handleTitleChange(e.target.value)}
              onKeyDown={e => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  handleTitleChange((e.target as HTMLInputElement).value)
                }
              }}
              placeholder="Untitled"
            />
          </div>

          {/* Tabs */}
          <div className="modal-tabs">
            <button
              className={`tab ${activeTab === 'properties' ? 'active' : ''}`}
              onClick={() => setActiveTab('properties')}
            >
              Properties
            </button>
            <button
              className={`tab ${activeTab === 'comments' ? 'active' : ''}`}
              onClick={() => setActiveTab('comments')}
            >
              Comments
            </button>
            <button
              className={`tab ${activeTab === 'activity' ? 'active' : ''}`}
              onClick={() => setActiveTab('activity')}
            >
              Activity
            </button>
          </div>

          {/* Tab content */}
          {activeTab === 'properties' && (
            <div className="properties-panel">
              {otherProperties.map(property => (
                <div key={property.id} className="property-row">
                  <div className="property-label">
                    <PropertyIcon type={property.type} />
                    <span>{property.name}</span>
                  </div>
                  <div className="property-value">
                    <PropertyCell
                      property={property}
                      value={localRow.properties[property.id]}
                      onChange={value => handlePropertyChange(property.id, value)}
                    />
                  </div>
                </div>
              ))}

              {/* Add property button */}
              <button className="add-property-btn">
                + Add a property
              </button>

              {/* Timestamps */}
              <div className="timestamps">
                {createdAt && (
                  <div className="timestamp">
                    <Clock size={12} />
                    <span>Created {createdAt}</span>
                  </div>
                )}
                {updatedAt && updatedAt !== createdAt && (
                  <div className="timestamp">
                    <Clock size={12} />
                    <span>Last edited {updatedAt}</span>
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTab === 'comments' && (
            <div className="comments-panel">
              <div className="empty-state">
                <MessageSquare size={32} />
                <p>No comments yet</p>
                <span>Start a conversation</span>
              </div>
              <div className="comment-input-wrapper">
                <input
                  type="text"
                  className="comment-input"
                  placeholder="Add a comment..."
                />
              </div>
            </div>
          )}

          {activeTab === 'activity' && (
            <div className="activity-panel">
              <div className="empty-state">
                <History size={32} />
                <p>No activity yet</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>,
    document.body
  )
}

// Property type icon component
function PropertyIcon({ type }: { type: string }) {
  const icons: Record<string, string> = {
    text: 'Aa',
    number: '#',
    select: '○',
    multi_select: '◎',
    date: '📅',
    person: '👤',
    checkbox: '☑',
    url: '🔗',
    email: '✉',
    phone: '📞',
    files: '📎',
    relation: '↔',
    rollup: '∑',
    formula: 'ƒ',
    status: '●',
    created_time: '⏱',
    created_by: '👤',
    last_edited_time: '⏱',
    last_edited_by: '👤',
  }

  return <span className="property-icon">{icons[type] || '•'}</span>
}
