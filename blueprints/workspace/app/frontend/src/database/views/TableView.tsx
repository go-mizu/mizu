import { useState, useCallback, useRef, useEffect } from 'react'
import { DatabaseRow, Property, PropertyType } from '../../api/client'
import { PropertyCell } from '../PropertyCell'
import { PropertyHeader } from '../PropertyHeader'

interface TableViewProps {
  rows: DatabaseRow[]
  properties: Property[]
  groupBy: string | null
  onAddRow: (initialProperties?: Record<string, unknown>) => Promise<DatabaseRow | null>
  onUpdateRow: (rowId: string, updates: Record<string, unknown>) => void
  onDeleteRow: (rowId: string) => void
  onAddProperty: (property: Omit<Property, 'id'>) => void
  onUpdateProperty: (propertyId: string, updates: Partial<Property>) => void
  onDeleteProperty: (propertyId: string) => void
}

export function TableView({
  rows,
  properties,
  onAddRow,
  onUpdateRow,
  onDeleteRow,
  onAddProperty,
  onUpdateProperty,
  onDeleteProperty,
}: TableViewProps) {
  const [columnWidths, setColumnWidths] = useState<Record<string, number>>({})
  const [resizing, setResizing] = useState<{ id: string; startX: number; startWidth: number } | null>(null)
  const [showAddProperty, setShowAddProperty] = useState(false)
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set())
  const tableRef = useRef<HTMLTableElement>(null)

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

  // Handle row selection
  const handleSelectRow = useCallback((rowId: string, e: React.MouseEvent) => {
    if (e.shiftKey && selectedRows.size > 0) {
      // Range selection
      const rowIds = rows.map((r) => r.id)
      const lastSelected = Array.from(selectedRows).pop()!
      const start = rowIds.indexOf(lastSelected)
      const end = rowIds.indexOf(rowId)
      const range = rowIds.slice(Math.min(start, end), Math.max(start, end) + 1)
      setSelectedRows(new Set(range))
    } else if (e.metaKey || e.ctrlKey) {
      // Toggle selection
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
      // Single selection
      setSelectedRows(new Set([rowId]))
    }
  }, [rows, selectedRows])

  // Handle select all
  const handleSelectAll = useCallback((checked: boolean) => {
    if (checked) {
      setSelectedRows(new Set(rows.map((r) => r.id)))
    } else {
      setSelectedRows(new Set())
    }
  }, [rows])

  // Handle delete selected
  const handleDeleteSelected = useCallback(() => {
    selectedRows.forEach((id) => onDeleteRow(id))
    setSelectedRows(new Set())
  }, [selectedRows, onDeleteRow])

  // Add new property
  const handleAddProperty = useCallback((type: PropertyType) => {
    onAddProperty({
      name: 'New Property',
      type,
      options: type === 'select' || type === 'multi_select' ? [] : undefined,
    })
    setShowAddProperty(false)
  }, [onAddProperty])

  const getColumnWidth = (propertyId: string) => columnWidths[propertyId] || 200

  return (
    <div className="table-view">
      {/* Bulk actions */}
      {selectedRows.size > 0 && (
        <div className="bulk-actions">
          <span>{selectedRows.size} selected</span>
          <button className="btn btn-sm btn-danger" onClick={handleDeleteSelected}>
            Delete
          </button>
          <button className="btn btn-sm" onClick={() => setSelectedRows(new Set())}>
            Clear
          </button>
        </div>
      )}

      <div className="table-container">
        <table className="data-table" ref={tableRef}>
          <thead>
            <tr>
              <th className="checkbox-cell">
                <input
                  type="checkbox"
                  checked={selectedRows.size === rows.length && rows.length > 0}
                  onChange={(e) => handleSelectAll(e.target.checked)}
                />
              </th>
              <th className="row-handle"></th>
              {properties.map((property) => (
                <th
                  key={property.id}
                  style={{ width: getColumnWidth(property.id), minWidth: 100 }}
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
                  />
                </th>
              ))}
              <th className="add-property-cell">
                <button
                  className="add-property-btn"
                  onClick={() => setShowAddProperty(!showAddProperty)}
                >
                  +
                </button>
                {showAddProperty && (
                  <PropertyTypeMenu onSelect={handleAddProperty} onClose={() => setShowAddProperty(false)} />
                )}
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr
                key={row.id}
                className={selectedRows.has(row.id) ? 'selected' : ''}
                onClick={(e) => handleSelectRow(row.id, e)}
              >
                <td className="checkbox-cell">
                  <input
                    type="checkbox"
                    checked={selectedRows.has(row.id)}
                    onChange={() => {}}
                  />
                </td>
                <td className="row-handle">
                  <div className="drag-handle">
                    <svg width="10" height="14" viewBox="0 0 10 14">
                      <circle cx="2" cy="2" r="1.5" fill="currentColor" />
                      <circle cx="8" cy="2" r="1.5" fill="currentColor" />
                      <circle cx="2" cy="7" r="1.5" fill="currentColor" />
                      <circle cx="8" cy="7" r="1.5" fill="currentColor" />
                      <circle cx="2" cy="12" r="1.5" fill="currentColor" />
                      <circle cx="8" cy="12" r="1.5" fill="currentColor" />
                    </svg>
                  </div>
                </td>
                {properties.map((property) => (
                  <td key={property.id} style={{ width: getColumnWidth(property.id) }}>
                    <PropertyCell
                      property={property}
                      value={row.properties[property.id]}
                      onChange={(value) => onUpdateRow(row.id, { [property.id]: value })}
                    />
                  </td>
                ))}
                <td className="row-actions">
                  <button
                    className="delete-row-btn"
                    onClick={(e) => {
                      e.stopPropagation()
                      onDeleteRow(row.id)
                    }}
                  >
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                      <path d="M2 4h10M5 4V2h4v2M3 4v8a1 1 0 001 1h6a1 1 0 001-1V4" stroke="currentColor" strokeWidth="1.5" />
                    </svg>
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Add row button */}
      <button className="add-row-btn" onClick={() => onAddRow()}>
        <svg width="12" height="12" viewBox="0 0 12 12">
          <path d="M6 2v8M2 6h8" stroke="currentColor" strokeWidth="1.5" />
        </svg>
        <span>New</span>
      </button>
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
    { type: 'text', label: 'Text', icon: 'T' },
    { type: 'number', label: 'Number', icon: '#' },
    { type: 'select', label: 'Select', icon: '▼' },
    { type: 'multi_select', label: 'Multi-select', icon: '▣' },
    { type: 'date', label: 'Date', icon: '📅' },
    { type: 'person', label: 'Person', icon: '👤' },
    { type: 'checkbox', label: 'Checkbox', icon: '☑' },
    { type: 'url', label: 'URL', icon: '🔗' },
    { type: 'email', label: 'Email', icon: '✉' },
    { type: 'phone', label: 'Phone', icon: '📞' },
    { type: 'files', label: 'Files', icon: '📎' },
    { type: 'status', label: 'Status', icon: '●' },
  ]

  return (
    <div className="property-type-menu" ref={menuRef}>
      <div className="menu-header">Property Type</div>
      {types.map(({ type, label, icon }) => (
        <button key={type} className="menu-item" onClick={() => onSelect(type)}>
          <span className="menu-icon">{icon}</span>
          <span>{label}</span>
        </button>
      ))}
    </div>
  )
}
