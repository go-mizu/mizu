import { useState, useCallback, useRef, useEffect, useMemo } from 'react'
import { DatabaseRow, Property, PropertyType, Database } from '../../api/client'
import { PropertyCell } from '../PropertyCell'
import { PropertyHeader } from '../PropertyHeader'
import { RowDetailModal } from '../RowDetailModal'
import { GripVertical, Eye, EyeOff, ChevronDown, Maximize2, Plus, Trash2, Search } from 'lucide-react'

interface TableViewProps {
  rows: DatabaseRow[]
  properties: Property[]
  groupBy: string | null
  database?: Database
  hiddenProperties?: string[]
  onAddRow: (initialProperties?: Record<string, unknown>) => Promise<DatabaseRow | null>
  onUpdateRow: (rowId: string, updates: Record<string, unknown>) => void
  onDeleteRow: (rowId: string) => void
  onAddProperty: (property: Omit<Property, 'id'>) => void
  onUpdateProperty: (propertyId: string, updates: Partial<Property>) => void
  onDeleteProperty: (propertyId: string) => void
  onHiddenPropertiesChange?: (hiddenProperties: string[]) => void
  onRowsReorder?: (rowIds: string[]) => void
}

export function TableView({
  rows,
  properties,
  database,
  hiddenProperties = [],
  onAddRow,
  onUpdateRow,
  onDeleteRow,
  onAddProperty,
  onUpdateProperty,
  onDeleteProperty,
  onHiddenPropertiesChange,
  onRowsReorder,
}: TableViewProps) {
  const [columnWidths, setColumnWidths] = useState<Record<string, number>>({})
  const [resizing, setResizing] = useState<{ id: string; startX: number; startWidth: number } | null>(null)
  const [showAddProperty, setShowAddProperty] = useState(false)
  const [showColumnVisibility, setShowColumnVisibility] = useState(false)
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set())
  const [selectedCell, setSelectedCell] = useState<{ rowIndex: number; colIndex: number } | null>(null)
  const [detailRow, setDetailRow] = useState<DatabaseRow | null>(null)
  const [draggedRowId, setDraggedRowId] = useState<string | null>(null)
  const [dragOverRowId, setDragOverRowId] = useState<string | null>(null)
  const [localRows, setLocalRows] = useState(rows)
  const [searchQuery, setSearchQuery] = useState('')
  const tableRef = useRef<HTMLTableElement>(null)
  const columnVisibilityRef = useRef<HTMLDivElement>(null)

  // Sync local rows with prop
  useEffect(() => {
    setLocalRows(rows)
  }, [rows])

  // Get visible properties
  const visibleProperties = useMemo(() => {
    return properties.filter(p => !hiddenProperties.includes(p.id))
  }, [properties, hiddenProperties])

  // Filter rows by search query
  const filteredRows = useMemo(() => {
    if (!searchQuery.trim()) return localRows
    const query = searchQuery.toLowerCase()
    return localRows.filter(row => {
      return properties.some(prop => {
        const value = row.properties[prop.id]
        if (value === null || value === undefined) return false
        return String(value).toLowerCase().includes(query)
      })
    })
  }, [localRows, searchQuery, properties])

  // Handle column resize
  const handleResizeStart = useCallback((e: React.MouseEvent, propertyId: string, currentWidth: number) => {
    e.preventDefault()
    setResizing({ id: propertyId, startX: e.clientX, startWidth: currentWidth })
  }, [])

  useEffect(() => {
    if (!resizing) return

    const handleMouseMove = (e: MouseEvent) => {
      const diff = e.clientX - resizing.startX
      const newWidth = Math.max(100, resizing.startWidth + diff)
      setColumnWidths((prev) => ({ ...prev, [resizing.id]: newWidth }))
    }

    const handleMouseUp = () => {
      setResizing(null)
    }

    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)

    return () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }, [resizing])

  // Close column visibility menu on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (columnVisibilityRef.current && !columnVisibilityRef.current.contains(e.target as Node)) {
        setShowColumnVisibility(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Handle keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!selectedCell) return

      const { rowIndex, colIndex } = selectedCell
      let newRowIndex = rowIndex
      let newColIndex = colIndex

      switch (e.key) {
        case 'ArrowUp':
          e.preventDefault()
          newRowIndex = Math.max(0, rowIndex - 1)
          break
        case 'ArrowDown':
          e.preventDefault()
          newRowIndex = Math.min(filteredRows.length - 1, rowIndex + 1)
          break
        case 'ArrowLeft':
          e.preventDefault()
          newColIndex = Math.max(0, colIndex - 1)
          break
        case 'ArrowRight':
          e.preventDefault()
          newColIndex = Math.min(visibleProperties.length - 1, colIndex + 1)
          break
        case 'Tab':
          e.preventDefault()
          if (e.shiftKey) {
            // Move left or to previous row
            if (colIndex > 0) {
              newColIndex = colIndex - 1
            } else if (rowIndex > 0) {
              newRowIndex = rowIndex - 1
              newColIndex = visibleProperties.length - 1
            }
          } else {
            // Move right or to next row
            if (colIndex < visibleProperties.length - 1) {
              newColIndex = colIndex + 1
            } else if (rowIndex < filteredRows.length - 1) {
              newRowIndex = rowIndex + 1
              newColIndex = 0
            }
          }
          break
        case 'Enter':
          e.preventDefault()
          // Open row detail modal
          setDetailRow(filteredRows[rowIndex])
          return
        case 'Escape':
          e.preventDefault()
          setSelectedCell(null)
          return
        default:
          return
      }

      setSelectedCell({ rowIndex: newRowIndex, colIndex: newColIndex })
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [selectedCell, filteredRows, visibleProperties])

  // Handle row selection
  const handleSelectRow = useCallback((rowId: string, e: React.MouseEvent) => {
    if (e.shiftKey && selectedRows.size > 0) {
      const rowIds = filteredRows.map((r) => r.id)
      const lastSelected = Array.from(selectedRows).pop()!
      const start = rowIds.indexOf(lastSelected)
      const end = rowIds.indexOf(rowId)
      const range = rowIds.slice(Math.min(start, end), Math.max(start, end) + 1)
      setSelectedRows(new Set(range))
    } else if (e.metaKey || e.ctrlKey) {
      setSelectedRows((prev) => {
        const next = new Set(prev)
        if (next.has(rowId)) {
          next.delete(rowId)
        } else {
          next.add(rowId)
        }
        return next
      })
    } else {
      setSelectedRows(new Set([rowId]))
    }
  }, [filteredRows, selectedRows])

  // Handle select all
  const handleSelectAll = useCallback((checked: boolean) => {
    if (checked) {
      setSelectedRows(new Set(filteredRows.map((r) => r.id)))
    } else {
      setSelectedRows(new Set())
    }
  }, [filteredRows])

  // Handle delete selected
  const handleDeleteSelected = useCallback(() => {
    if (!confirm(`Delete ${selectedRows.size} row(s)?`)) return
    selectedRows.forEach((id) => onDeleteRow(id))
    setSelectedRows(new Set())
  }, [selectedRows, onDeleteRow])

  // Handle row drag and drop
  const handleDragStart = useCallback((e: React.DragEvent, rowId: string) => {
    setDraggedRowId(rowId)
    e.dataTransfer.effectAllowed = 'move'
  }, [])

  const handleDragOver = useCallback((e: React.DragEvent, rowId: string) => {
    e.preventDefault()
    if (draggedRowId && draggedRowId !== rowId) {
      setDragOverRowId(rowId)
    }
  }, [draggedRowId])

  const handleDragEnd = useCallback(() => {
    if (draggedRowId && dragOverRowId && draggedRowId !== dragOverRowId) {
      const newRows = [...localRows]
      const draggedIndex = newRows.findIndex(r => r.id === draggedRowId)
      const targetIndex = newRows.findIndex(r => r.id === dragOverRowId)

      if (draggedIndex !== -1 && targetIndex !== -1) {
        const [removed] = newRows.splice(draggedIndex, 1)
        newRows.splice(targetIndex, 0, removed)
        setLocalRows(newRows)
        onRowsReorder?.(newRows.map(r => r.id))
      }
    }
    setDraggedRowId(null)
    setDragOverRowId(null)
  }, [draggedRowId, dragOverRowId, localRows, onRowsReorder])

  // Toggle property visibility
  const togglePropertyVisibility = useCallback((propertyId: string) => {
    const newHidden = hiddenProperties.includes(propertyId)
      ? hiddenProperties.filter(id => id !== propertyId)
      : [...hiddenProperties, propertyId]
    onHiddenPropertiesChange?.(newHidden)
  }, [hiddenProperties, onHiddenPropertiesChange])

  // Add new property
  const handleAddProperty = useCallback((type: PropertyType) => {
    onAddProperty({
      name: 'New Property',
      type,
      options: type === 'select' || type === 'multi_select' || type === 'status' ? [] : undefined,
    })
    setShowAddProperty(false)
  }, [onAddProperty])

  const getColumnWidth = (propertyId: string) => columnWidths[propertyId] || 200

  // Handle cell click for keyboard navigation
  const handleCellClick = useCallback((rowIndex: number, colIndex: number) => {
    setSelectedCell({ rowIndex, colIndex })
  }, [])

  // Handle row detail update
  const handleDetailUpdate = useCallback((updatedRow: DatabaseRow) => {
    setLocalRows(prev => prev.map(r => r.id === updatedRow.id ? updatedRow : r))
  }, [])

  return (
    <div className="table-view">
      {/* Toolbar */}
      <div className="table-toolbar" style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        padding: '8px 0',
        borderBottom: '1px solid var(--border-color)',
        marginBottom: 8,
      }}>
        {/* Search */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '6px 12px',
          background: 'var(--bg-secondary)',
          borderRadius: 'var(--radius-md)',
          flex: '0 0 250px',
        }}>
          <Search size={14} style={{ color: 'var(--text-tertiary)' }} />
          <input
            type="text"
            placeholder="Search in table..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            style={{
              flex: 1,
              border: 'none',
              background: 'none',
              outline: 'none',
              fontSize: 13,
              color: 'var(--text-primary)',
            }}
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              style={{
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                padding: 0,
                color: 'var(--text-tertiary)',
              }}
            >
              ×
            </button>
          )}
        </div>

        {/* Column visibility toggle */}
        <div ref={columnVisibilityRef} style={{ position: 'relative' }}>
          <button
            onClick={() => setShowColumnVisibility(!showColumnVisibility)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              padding: '6px 12px',
              background: 'none',
              border: '1px solid var(--border-color)',
              borderRadius: 'var(--radius-md)',
              cursor: 'pointer',
              fontSize: 13,
              color: 'var(--text-secondary)',
            }}
          >
            <Eye size={14} />
            Properties
            <ChevronDown size={14} />
          </button>
          {showColumnVisibility && (
            <div style={{
              position: 'absolute',
              top: '100%',
              left: 0,
              marginTop: 4,
              background: 'var(--bg-primary)',
              border: '1px solid var(--border-color)',
              borderRadius: 'var(--radius-md)',
              boxShadow: 'var(--shadow-lg)',
              minWidth: 220,
              maxHeight: 300,
              overflowY: 'auto',
              zIndex: 100,
            }}>
              <div style={{
                padding: '8px 12px',
                borderBottom: '1px solid var(--border-color)',
                fontSize: 12,
                fontWeight: 600,
                color: 'var(--text-secondary)',
              }}>
                Toggle columns
              </div>
              {properties.map((prop) => (
                <button
                  key={prop.id}
                  onClick={() => togglePropertyVisibility(prop.id)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    width: '100%',
                    padding: '8px 12px',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    textAlign: 'left',
                    fontSize: 13,
                  }}
                >
                  {hiddenProperties.includes(prop.id) ? (
                    <EyeOff size={14} style={{ color: 'var(--text-tertiary)' }} />
                  ) : (
                    <Eye size={14} style={{ color: 'var(--accent-color)' }} />
                  )}
                  <span style={{
                    color: hiddenProperties.includes(prop.id) ? 'var(--text-tertiary)' : 'var(--text-primary)',
                    textDecoration: hiddenProperties.includes(prop.id) ? 'line-through' : 'none',
                  }}>
                    {prop.name}
                  </span>
                </button>
              ))}
              {hiddenProperties.length > 0 && (
                <button
                  onClick={() => onHiddenPropertiesChange?.([])}
                  style={{
                    display: 'block',
                    width: '100%',
                    padding: '8px 12px',
                    background: 'none',
                    border: 'none',
                    borderTop: '1px solid var(--border-color)',
                    cursor: 'pointer',
                    textAlign: 'center',
                    fontSize: 12,
                    color: 'var(--accent-color)',
                  }}
                >
                  Show all
                </button>
              )}
            </div>
          )}
        </div>

        <div style={{ flex: 1 }} />

        {/* Row count */}
        <span style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>
          {filteredRows.length} {filteredRows.length === 1 ? 'row' : 'rows'}
          {searchQuery && ` (filtered)`}
        </span>
      </div>

      {/* Bulk actions */}
      {selectedRows.size > 0 && (
        <div className="bulk-actions" style={{
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          padding: '8px 12px',
          background: 'var(--accent-bg)',
          borderRadius: 'var(--radius-md)',
          marginBottom: 8,
        }}>
          <span style={{ fontSize: 13, fontWeight: 500 }}>{selectedRows.size} selected</span>
          <button
            className="btn btn-sm btn-danger"
            onClick={handleDeleteSelected}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 4,
              padding: '4px 12px',
              background: 'var(--error-color)',
              color: 'white',
              border: 'none',
              borderRadius: 'var(--radius-sm)',
              fontSize: 12,
              cursor: 'pointer',
            }}
          >
            <Trash2 size={12} />
            Delete
          </button>
          <button
            className="btn btn-sm"
            onClick={() => setSelectedRows(new Set())}
            style={{
              padding: '4px 12px',
              background: 'none',
              border: '1px solid var(--border-color)',
              borderRadius: 'var(--radius-sm)',
              fontSize: 12,
              cursor: 'pointer',
            }}
          >
            Clear
          </button>
        </div>
      )}

      <div className="table-container" style={{ overflowX: 'auto' }}>
        <table className="data-table" ref={tableRef} style={{
          width: '100%',
          borderCollapse: 'collapse',
          fontSize: 14,
        }}>
          <thead>
            <tr>
              <th className="checkbox-cell" style={{
                width: 32,
                padding: '8px 4px',
                borderBottom: '1px solid var(--border-color)',
              }}>
                <input
                  type="checkbox"
                  checked={selectedRows.size === filteredRows.length && filteredRows.length > 0}
                  onChange={(e) => handleSelectAll(e.target.checked)}
                  style={{ cursor: 'pointer' }}
                />
              </th>
              <th className="row-handle" style={{
                width: 28,
                padding: '8px 4px',
                borderBottom: '1px solid var(--border-color)',
              }} />
              {visibleProperties.map((property) => (
                <th
                  key={property.id}
                  style={{
                    width: getColumnWidth(property.id),
                    minWidth: 100,
                    padding: '8px 12px',
                    borderBottom: '1px solid var(--border-color)',
                    textAlign: 'left',
                    position: 'relative',
                  }}
                  className="property-header-cell"
                >
                  <PropertyHeader
                    property={property}
                    onUpdate={(updates) => onUpdateProperty(property.id, updates)}
                    onDelete={() => onDeleteProperty(property.id)}
                  />
                  <div
                    className="resize-handle"
                    onMouseDown={(e) => handleResizeStart(e, property.id, getColumnWidth(property.id))}
                    style={{
                      position: 'absolute',
                      right: 0,
                      top: 0,
                      bottom: 0,
                      width: 4,
                      cursor: 'col-resize',
                      background: resizing?.id === property.id ? 'var(--accent-color)' : 'transparent',
                    }}
                  />
                </th>
              ))}
              <th className="add-property-cell" style={{
                width: 40,
                padding: '8px 4px',
                borderBottom: '1px solid var(--border-color)',
                position: 'relative',
              }}>
                <button
                  className="add-property-btn"
                  onClick={() => setShowAddProperty(!showAddProperty)}
                  style={{
                    width: 28,
                    height: 28,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: 'none',
                    border: '1px dashed var(--border-color)',
                    borderRadius: 'var(--radius-sm)',
                    cursor: 'pointer',
                    color: 'var(--text-tertiary)',
                  }}
                >
                  <Plus size={14} />
                </button>
                {showAddProperty && (
                  <PropertyTypeMenu onSelect={handleAddProperty} onClose={() => setShowAddProperty(false)} />
                )}
              </th>
            </tr>
          </thead>
          <tbody>
            {filteredRows.map((row, rowIndex) => (
              <tr
                key={row.id}
                className={`
                  ${selectedRows.has(row.id) ? 'selected' : ''}
                  ${draggedRowId === row.id ? 'dragging' : ''}
                  ${dragOverRowId === row.id ? 'drag-over' : ''}
                `}
                style={{
                  background: selectedRows.has(row.id) ? 'var(--accent-bg)' :
                              dragOverRowId === row.id ? 'var(--bg-secondary)' : 'transparent',
                  opacity: draggedRowId === row.id ? 0.5 : 1,
                }}
                onClick={(e) => handleSelectRow(row.id, e)}
                draggable
                onDragStart={(e) => handleDragStart(e, row.id)}
                onDragOver={(e) => handleDragOver(e, row.id)}
                onDragEnd={handleDragEnd}
              >
                <td className="checkbox-cell" style={{ padding: '8px 4px' }}>
                  <input
                    type="checkbox"
                    checked={selectedRows.has(row.id)}
                    onChange={() => {}}
                    style={{ cursor: 'pointer' }}
                  />
                </td>
                <td className="row-handle" style={{ padding: '8px 4px' }}>
                  <div
                    className="drag-handle"
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      cursor: 'grab',
                      color: 'var(--text-tertiary)',
                    }}
                  >
                    <GripVertical size={14} />
                  </div>
                </td>
                {visibleProperties.map((property, colIndex) => (
                  <td
                    key={property.id}
                    style={{
                      width: getColumnWidth(property.id),
                      padding: '4px 12px',
                      borderBottom: '1px solid var(--border-color-light)',
                      outline: selectedCell?.rowIndex === rowIndex && selectedCell?.colIndex === colIndex
                        ? '2px solid var(--accent-color)'
                        : 'none',
                    }}
                    onClick={(e) => {
                      e.stopPropagation()
                      handleCellClick(rowIndex, colIndex)
                    }}
                  >
                    <PropertyCell
                      property={property}
                      value={row.properties[property.id]}
                      onChange={(value) => onUpdateRow(row.id, { [property.id]: value })}
                    />
                  </td>
                ))}
                <td className="row-actions" style={{ padding: '4px 8px' }}>
                  <div style={{ display: 'flex', gap: 4 }}>
                    <button
                      className="open-row-btn"
                      onClick={(e) => {
                        e.stopPropagation()
                        setDetailRow(row)
                      }}
                      title="Open"
                      style={{
                        width: 24,
                        height: 24,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        background: 'none',
                        border: 'none',
                        borderRadius: 'var(--radius-sm)',
                        cursor: 'pointer',
                        color: 'var(--text-tertiary)',
                        opacity: 0,
                        transition: 'opacity 0.15s',
                      }}
                      onMouseEnter={(e) => (e.currentTarget.style.opacity = '1')}
                      onMouseLeave={(e) => (e.currentTarget.style.opacity = '0')}
                    >
                      <Maximize2 size={14} />
                    </button>
                    <button
                      className="delete-row-btn"
                      onClick={(e) => {
                        e.stopPropagation()
                        if (confirm('Delete this row?')) {
                          onDeleteRow(row.id)
                        }
                      }}
                      title="Delete"
                      style={{
                        width: 24,
                        height: 24,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        background: 'none',
                        border: 'none',
                        borderRadius: 'var(--radius-sm)',
                        cursor: 'pointer',
                        color: 'var(--text-tertiary)',
                        opacity: 0,
                        transition: 'opacity 0.15s',
                      }}
                      onMouseEnter={(e) => (e.currentTarget.style.opacity = '1')}
                      onMouseLeave={(e) => (e.currentTarget.style.opacity = '0')}
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Add row button */}
      <button
        className="add-row-btn"
        onClick={() => onAddRow()}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '8px 12px',
          marginTop: 8,
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          color: 'var(--text-tertiary)',
          fontSize: 13,
          width: '100%',
          textAlign: 'left',
        }}
      >
        <Plus size={14} />
        <span>New</span>
      </button>

      {/* Row Detail Modal */}
      {detailRow && database && (
        <RowDetailModal
          row={detailRow}
          database={database}
          onClose={() => setDetailRow(null)}
          onUpdate={handleDetailUpdate}
          onDelete={(rowId) => {
            onDeleteRow(rowId)
            setDetailRow(null)
          }}
        />
      )}

      {/* Keyboard shortcuts hint */}
      {selectedCell && (
        <div style={{
          position: 'fixed',
          bottom: 16,
          right: 16,
          padding: '8px 12px',
          background: 'var(--bg-primary)',
          border: '1px solid var(--border-color)',
          borderRadius: 'var(--radius-md)',
          boxShadow: 'var(--shadow-md)',
          fontSize: 11,
          color: 'var(--text-secondary)',
          display: 'flex',
          gap: 16,
        }}>
          <span>↑↓←→ Navigate</span>
          <span>Tab Next</span>
          <span>Enter Open</span>
          <span>Esc Deselect</span>
        </div>
      )}
    </div>
  )
}

// Property type selection menu
function PropertyTypeMenu({
  onSelect,
  onClose,
}: {
  onSelect: (type: PropertyType) => void
  onClose: () => void
}) {
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose()
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [onClose])

  const types: { type: PropertyType; label: string; icon: string }[] = [
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

  return (
    <div
      className="property-type-menu"
      ref={menuRef}
      style={{
        position: 'absolute',
        top: '100%',
        right: 0,
        marginTop: 4,
        background: 'var(--bg-primary)',
        border: '1px solid var(--border-color)',
        borderRadius: 'var(--radius-md)',
        boxShadow: 'var(--shadow-lg)',
        minWidth: 200,
        maxHeight: 400,
        overflowY: 'auto',
        zIndex: 100,
      }}
    >
      <div className="menu-header" style={{
        padding: '8px 12px',
        borderBottom: '1px solid var(--border-color)',
        fontSize: 12,
        fontWeight: 600,
        color: 'var(--text-secondary)',
      }}>
        Property Type
      </div>
      {types.map(({ type, label, icon }) => (
        <button
          key={type}
          className="menu-item"
          onClick={() => onSelect(type)}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            width: '100%',
            padding: '8px 12px',
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            textAlign: 'left',
            fontSize: 13,
          }}
        >
          <span className="menu-icon" style={{ width: 20, textAlign: 'center' }}>{icon}</span>
          <span>{label}</span>
        </button>
      ))}
    </div>
  )
}
