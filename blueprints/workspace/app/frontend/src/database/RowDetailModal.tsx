import { useState, useEffect, useRef, useCallback } from 'react'
import { createPortal } from 'react-dom'
import { X, MessageSquare, Clock, History, MoreHorizontal, Trash2, Copy, ExternalLink, Plus, ChevronDown } from 'lucide-react'
import { api, Database, DatabaseRow, Property, PropertyType } from '../api/client'
import { PropertyCell } from './PropertyCell'
import { format, parseISO } from 'date-fns'

interface RowDetailModalProps {
  row: DatabaseRow
  database: Database
  onClose: () => void
  onUpdate: (row: DatabaseRow) => void
  onDelete?: (rowId: string) => void
  onAddProperty?: (property: Omit<Property, 'id'>) => Promise<Property | void>
}

const PROPERTY_TYPES: { type: PropertyType; label: string; icon: string }[] = [
  { type: 'text', label: 'Text', icon: 'Aa' },
  { type: 'number', label: 'Number', icon: '#' },
  { type: 'select', label: 'Select', icon: '○' },
  { type: 'multi_select', label: 'Multi-select', icon: '◎' },
  { type: 'status', label: 'Status', icon: '●' },
  { type: 'date', label: 'Date', icon: '📅' },
  { type: 'person', label: 'Person', icon: '👤' },
  { type: 'checkbox', label: 'Checkbox', icon: '☑' },
  { type: 'url', label: 'URL', icon: '🔗' },
  { type: 'email', label: 'Email', icon: '✉' },
  { type: 'phone', label: 'Phone', icon: '📞' },
  { type: 'files', label: 'Files & media', icon: '📎' },
  { type: 'relation', label: 'Relation', icon: '↔' },
  { type: 'rollup', label: 'Rollup', icon: '∑' },
  { type: 'formula', label: 'Formula', icon: 'ƒ' },
  { type: 'created_time', label: 'Created time', icon: '⏱' },
  { type: 'created_by', label: 'Created by', icon: '👤' },
  { type: 'last_edited_time', label: 'Last edited time', icon: '⏱' },
  { type: 'last_edited_by', label: 'Last edited by', icon: '👤' },
]

export function RowDetailModal({ row, database, onClose, onUpdate, onDelete, onAddProperty }: RowDetailModalProps) {
  const [localRow, setLocalRow] = useState(row)
  const [localDatabase, setLocalDatabase] = useState(database)
  const [title, setTitle] = useState(row.properties.title as string || 'Untitled')
  const [isSaving, setIsSaving] = useState(false)
  const [showMenu, setShowMenu] = useState(false)
  const [activeTab, setActiveTab] = useState<'properties' | 'comments' | 'activity'>('properties')
  const [showAddProperty, setShowAddProperty] = useState(false)
  const [newPropertyName, setNewPropertyName] = useState('')
  const [newPropertyType, setNewPropertyType] = useState<PropertyType>('text')
  const [showTypeSelect, setShowTypeSelect] = useState(false)
  const modalRef = useRef<HTMLDivElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const addPropertyRef = useRef<HTMLDivElement>(null)

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
      if (addPropertyRef.current && !addPropertyRef.current.contains(e.target as Node)) {
        setShowAddProperty(false)
        setShowTypeSelect(false)
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

  // Add new property
  const handleAddNewProperty = useCallback(async () => {
    if (!newPropertyName.trim()) return

    try {
      setIsSaving(true)

      const newProperty: Omit<Property, 'id'> = {
        name: newPropertyName.trim(),
        type: newPropertyType,
        options: ['select', 'multi_select', 'status'].includes(newPropertyType) ? [] : undefined,
      }

      // Use callback if provided, otherwise call API directly
      if (onAddProperty) {
        await onAddProperty(newProperty)
      } else {
        const addedProperty = await api.post<Property>(`/databases/${database.id}/properties`, newProperty)
        setLocalDatabase({
          ...localDatabase,
          properties: [...localDatabase.properties, addedProperty]
        })
      }

      setNewPropertyName('')
      setNewPropertyType('text')
      setShowAddProperty(false)
      setShowTypeSelect(false)
    } catch (err) {
      console.error('Failed to add property:', err)
    } finally {
      setIsSaving(false)
    }
  }, [newPropertyName, newPropertyType, database.id, localDatabase, onAddProperty])

  // Get non-title properties
  const otherProperties = localDatabase.properties.filter(p => p.name.toLowerCase() !== 'title')

  // Format dates
  const createdAt = row.created_at ? format(parseISO(row.created_at), 'MMM d, yyyy h:mm a') : ''
  const updatedAt = row.updated_at ? format(parseISO(row.updated_at), 'MMM d, yyyy h:mm a') : ''

  return createPortal(
    <div className="modal-overlay" onClick={onClose} style={{
      position: 'fixed',
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      background: 'rgba(0, 0, 0, 0.5)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      zIndex: 1000,
    }}>
      <div
        ref={modalRef}
        className="row-detail-modal"
        onClick={e => e.stopPropagation()}
        style={{
          width: '100%',
          maxWidth: 700,
          maxHeight: '90vh',
          background: 'var(--bg-primary)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-xl)',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        {/* Header */}
        <div className="modal-header" style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 16px',
          borderBottom: '1px solid var(--border-color)',
        }}>
          <div className="modal-header-left" style={{ display: 'flex', gap: 8 }}>
            <button
              className="icon-btn"
              onClick={() => setActiveTab('comments')}
              title="Comments"
              style={{
                width: 32,
                height: 32,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: activeTab === 'comments' ? 'var(--bg-secondary)' : 'none',
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
                color: 'var(--text-secondary)',
              }}
            >
              <MessageSquare size={16} />
            </button>
            <button
              className="icon-btn"
              onClick={() => setActiveTab('activity')}
              title="Activity"
              style={{
                width: 32,
                height: 32,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: activeTab === 'activity' ? 'var(--bg-secondary)' : 'none',
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
                color: 'var(--text-secondary)',
              }}
            >
              <History size={16} />
            </button>
          </div>
          <div className="modal-header-right" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            {isSaving && <span className="save-indicator" style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>Saving...</span>}
            <div className="menu-wrapper" ref={menuRef} style={{ position: 'relative' }}>
              <button
                className="icon-btn"
                onClick={() => setShowMenu(!showMenu)}
                title="More options"
                style={{
                  width: 32,
                  height: 32,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'none',
                  border: 'none',
                  borderRadius: 'var(--radius-sm)',
                  cursor: 'pointer',
                  color: 'var(--text-secondary)',
                }}
              >
                <MoreHorizontal size={16} />
              </button>
              {showMenu && (
                <div className="dropdown-menu" style={{
                  position: 'absolute',
                  top: '100%',
                  right: 0,
                  marginTop: 4,
                  background: 'var(--bg-primary)',
                  border: '1px solid var(--border-color)',
                  borderRadius: 'var(--radius-md)',
                  boxShadow: 'var(--shadow-lg)',
                  minWidth: 180,
                  zIndex: 100,
                }}>
                  <button onClick={handleDuplicate} style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    width: '100%',
                    padding: '8px 12px',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    fontSize: 13,
                    textAlign: 'left',
                  }}>
                    <Copy size={14} />
                    <span>Duplicate</span>
                  </button>
                  <button onClick={() => window.open(`/p/${row.id}`, '_blank')} style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    width: '100%',
                    padding: '8px 12px',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    fontSize: 13,
                    textAlign: 'left',
                  }}>
                    <ExternalLink size={14} />
                    <span>Open as page</span>
                  </button>
                  <hr style={{ margin: '4px 0', border: 'none', borderTop: '1px solid var(--border-color)' }} />
                  <button className="danger" onClick={handleDelete} style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    width: '100%',
                    padding: '8px 12px',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    fontSize: 13,
                    textAlign: 'left',
                    color: 'var(--error-color)',
                  }}>
                    <Trash2 size={14} />
                    <span>Delete</span>
                  </button>
                </div>
              )}
            </div>
            <button className="icon-btn" onClick={onClose} title="Close" style={{
              width: 32,
              height: 32,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'none',
              border: 'none',
              borderRadius: 'var(--radius-sm)',
              cursor: 'pointer',
              color: 'var(--text-secondary)',
            }}>
              <X size={16} />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="modal-content" style={{ flex: 1, overflow: 'auto', padding: 24 }}>
          {/* Title */}
          <div className="row-title-wrapper" style={{ marginBottom: 24 }}>
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
              style={{
                width: '100%',
                fontSize: 24,
                fontWeight: 600,
                border: 'none',
                outline: 'none',
                background: 'transparent',
                padding: 0,
              }}
            />
          </div>

          {/* Tabs */}
          <div className="modal-tabs" style={{
            display: 'flex',
            gap: 4,
            marginBottom: 16,
            borderBottom: '1px solid var(--border-color)',
          }}>
            <button
              className={`tab ${activeTab === 'properties' ? 'active' : ''}`}
              onClick={() => setActiveTab('properties')}
              style={{
                padding: '8px 16px',
                background: 'none',
                border: 'none',
                borderBottom: activeTab === 'properties' ? '2px solid var(--accent-color)' : '2px solid transparent',
                cursor: 'pointer',
                fontSize: 13,
                fontWeight: activeTab === 'properties' ? 600 : 400,
                color: activeTab === 'properties' ? 'var(--text-primary)' : 'var(--text-secondary)',
              }}
            >
              Properties
            </button>
            <button
              className={`tab ${activeTab === 'comments' ? 'active' : ''}`}
              onClick={() => setActiveTab('comments')}
              style={{
                padding: '8px 16px',
                background: 'none',
                border: 'none',
                borderBottom: activeTab === 'comments' ? '2px solid var(--accent-color)' : '2px solid transparent',
                cursor: 'pointer',
                fontSize: 13,
                fontWeight: activeTab === 'comments' ? 600 : 400,
                color: activeTab === 'comments' ? 'var(--text-primary)' : 'var(--text-secondary)',
              }}
            >
              Comments
            </button>
            <button
              className={`tab ${activeTab === 'activity' ? 'active' : ''}`}
              onClick={() => setActiveTab('activity')}
              style={{
                padding: '8px 16px',
                background: 'none',
                border: 'none',
                borderBottom: activeTab === 'activity' ? '2px solid var(--accent-color)' : '2px solid transparent',
                cursor: 'pointer',
                fontSize: 13,
                fontWeight: activeTab === 'activity' ? 600 : 400,
                color: activeTab === 'activity' ? 'var(--text-primary)' : 'var(--text-secondary)',
              }}
            >
              Activity
            </button>
          </div>

          {/* Tab content */}
          {activeTab === 'properties' && (
            <div className="properties-panel">
              {otherProperties.map(property => (
                <div key={property.id} className="property-row" style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '8px 0',
                  borderBottom: '1px solid var(--border-color-light)',
                }}>
                  <div className="property-label" style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    minWidth: 140,
                    color: 'var(--text-secondary)',
                    fontSize: 13,
                  }}>
                    <PropertyIcon type={property.type} />
                    <span>{property.name}</span>
                  </div>
                  <div className="property-value" style={{ flex: 1 }}>
                    <PropertyCell
                      property={property}
                      value={localRow.properties[property.id]}
                      onChange={value => handlePropertyChange(property.id, value)}
                    />
                  </div>
                </div>
              ))}

              {/* Add property button */}
              <div ref={addPropertyRef} style={{ position: 'relative', marginTop: 12 }}>
                {showAddProperty ? (
                  <div style={{
                    background: 'var(--bg-secondary)',
                    borderRadius: 'var(--radius-md)',
                    padding: 12,
                    border: '1px solid var(--border-color)',
                  }}>
                    <input
                      type="text"
                      value={newPropertyName}
                      onChange={(e) => setNewPropertyName(e.target.value)}
                      placeholder="Property name"
                      autoFocus
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' && newPropertyName.trim()) {
                          handleAddNewProperty()
                        }
                        if (e.key === 'Escape') {
                          setShowAddProperty(false)
                          setNewPropertyName('')
                        }
                      }}
                      style={{
                        width: '100%',
                        padding: '8px 12px',
                        border: '1px solid var(--border-color)',
                        borderRadius: 'var(--radius-sm)',
                        fontSize: 13,
                        marginBottom: 8,
                        outline: 'none',
                      }}
                    />

                    {/* Property type selector */}
                    <div style={{ position: 'relative', marginBottom: 12 }}>
                      <button
                        onClick={() => setShowTypeSelect(!showTypeSelect)}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between',
                          width: '100%',
                          padding: '8px 12px',
                          background: 'var(--bg-primary)',
                          border: '1px solid var(--border-color)',
                          borderRadius: 'var(--radius-sm)',
                          cursor: 'pointer',
                          fontSize: 13,
                        }}
                      >
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                          <span>{PROPERTY_TYPES.find(t => t.type === newPropertyType)?.icon}</span>
                          <span>{PROPERTY_TYPES.find(t => t.type === newPropertyType)?.label}</span>
                        </div>
                        <ChevronDown size={14} />
                      </button>
                      {showTypeSelect && (
                        <div style={{
                          position: 'absolute',
                          top: '100%',
                          left: 0,
                          right: 0,
                          marginTop: 4,
                          background: 'var(--bg-primary)',
                          border: '1px solid var(--border-color)',
                          borderRadius: 'var(--radius-md)',
                          boxShadow: 'var(--shadow-lg)',
                          maxHeight: 300,
                          overflowY: 'auto',
                          zIndex: 100,
                        }}>
                          {PROPERTY_TYPES.map(({ type, label, icon }) => (
                            <button
                              key={type}
                              onClick={() => {
                                setNewPropertyType(type)
                                setShowTypeSelect(false)
                              }}
                              style={{
                                display: 'flex',
                                alignItems: 'center',
                                gap: 8,
                                width: '100%',
                                padding: '8px 12px',
                                background: type === newPropertyType ? 'var(--bg-secondary)' : 'none',
                                border: 'none',
                                cursor: 'pointer',
                                fontSize: 13,
                                textAlign: 'left',
                              }}
                            >
                              <span style={{ width: 20, textAlign: 'center' }}>{icon}</span>
                              <span>{label}</span>
                            </button>
                          ))}
                        </div>
                      )}
                    </div>

                    <div style={{ display: 'flex', gap: 8 }}>
                      <button
                        onClick={handleAddNewProperty}
                        disabled={!newPropertyName.trim()}
                        style={{
                          flex: 1,
                          padding: '8px 12px',
                          background: 'var(--accent-color)',
                          color: 'white',
                          border: 'none',
                          borderRadius: 'var(--radius-sm)',
                          cursor: newPropertyName.trim() ? 'pointer' : 'not-allowed',
                          fontSize: 13,
                          opacity: newPropertyName.trim() ? 1 : 0.5,
                        }}
                      >
                        Add property
                      </button>
                      <button
                        onClick={() => {
                          setShowAddProperty(false)
                          setNewPropertyName('')
                        }}
                        style={{
                          padding: '8px 12px',
                          background: 'none',
                          border: '1px solid var(--border-color)',
                          borderRadius: 'var(--radius-sm)',
                          cursor: 'pointer',
                          fontSize: 13,
                        }}
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                ) : (
                  <button
                    className="add-property-btn"
                    onClick={() => setShowAddProperty(true)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      padding: '8px 12px',
                      background: 'none',
                      border: 'none',
                      cursor: 'pointer',
                      fontSize: 13,
                      color: 'var(--text-tertiary)',
                    }}
                  >
                    <Plus size={14} />
                    Add a property
                  </button>
                )}
              </div>

              {/* Timestamps */}
              <div className="timestamps" style={{ marginTop: 24, paddingTop: 16, borderTop: '1px solid var(--border-color)' }}>
                {createdAt && (
                  <div className="timestamp" style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    fontSize: 12,
                    color: 'var(--text-tertiary)',
                    marginBottom: 4,
                  }}>
                    <Clock size={12} />
                    <span>Created {createdAt}</span>
                  </div>
                )}
                {updatedAt && updatedAt !== createdAt && (
                  <div className="timestamp" style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    fontSize: 12,
                    color: 'var(--text-tertiary)',
                  }}>
                    <Clock size={12} />
                    <span>Last edited {updatedAt}</span>
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTab === 'comments' && (
            <div className="comments-panel">
              <div className="empty-state" style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                padding: 40,
                color: 'var(--text-tertiary)',
                textAlign: 'center',
              }}>
                <MessageSquare size={32} style={{ marginBottom: 12, opacity: 0.5 }} />
                <p style={{ margin: 0, fontSize: 14 }}>No comments yet</p>
                <span style={{ fontSize: 12 }}>Start a conversation</span>
              </div>
              <div className="comment-input-wrapper" style={{ marginTop: 16 }}>
                <input
                  type="text"
                  className="comment-input"
                  placeholder="Add a comment..."
                  style={{
                    width: '100%',
                    padding: '12px',
                    border: '1px solid var(--border-color)',
                    borderRadius: 'var(--radius-md)',
                    fontSize: 13,
                    outline: 'none',
                  }}
                />
              </div>
            </div>
          )}

          {activeTab === 'activity' && (
            <div className="activity-panel">
              <div className="empty-state" style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                padding: 40,
                color: 'var(--text-tertiary)',
                textAlign: 'center',
              }}>
                <History size={32} style={{ marginBottom: 12, opacity: 0.5 }} />
                <p style={{ margin: 0, fontSize: 14 }}>No activity yet</p>
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

  return <span className="property-icon" style={{ fontSize: 12 }}>{icons[type] || '•'}</span>
}
