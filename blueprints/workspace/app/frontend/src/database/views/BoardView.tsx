import { useState, useCallback, useMemo, useRef, useEffect } from 'react'
import { DatabaseRow, Property, PropertyOption, Database } from '../../api/client'
import { PropertyCell } from '../PropertyCell'
import { RowDetailModal } from '../RowDetailModal'
import { Plus, MoreHorizontal, Trash2, Edit2, Maximize2, EyeOff, X } from 'lucide-react'

interface BoardViewProps {
  rows: DatabaseRow[]
  properties: Property[]
  groupBy: string | null
  database?: Database
  onAddRow: (initialProperties?: Record<string, unknown>) => Promise<DatabaseRow | null>
  onUpdateRow: (rowId: string, updates: Record<string, unknown>) => void
  onDeleteRow: (rowId: string) => void
  onAddProperty: (property: Omit<Property, 'id'>) => void
  onUpdateProperty: (propertyId: string, updates: Partial<Property>) => void
  onDeleteProperty: (propertyId: string) => void
}

interface Column {
  id: string
  name: string
  color: string
  rows: DatabaseRow[]
}

const COLORS = [
  'gray', 'brown', 'orange', 'yellow', 'green', 'blue', 'purple', 'pink', 'red'
]

export function BoardView({
  rows,
  properties,
  groupBy,
  database,
  onAddRow,
  onUpdateRow,
  onDeleteRow,
  onUpdateProperty,
}: BoardViewProps) {
  const [dragging, setDragging] = useState<{ rowId: string; fromColumn: string } | null>(null)
  const [dragOver, setDragOver] = useState<string | null>(null)
  const [showAddGroup, setShowAddGroup] = useState(false)
  const [newGroupName, setNewGroupName] = useState('')
  const [newGroupColor, setNewGroupColor] = useState('gray')
  const [editingGroup, setEditingGroup] = useState<string | null>(null)
  const [editGroupName, setEditGroupName] = useState('')
  const [hiddenColumns, setHiddenColumns] = useState<Set<string>>(new Set())
  const [columnMenu, setColumnMenu] = useState<string | null>(null)
  const [detailRow, setDetailRow] = useState<DatabaseRow | null>(null)
  const addGroupRef = useRef<HTMLDivElement>(null)
  const columnMenuRef = useRef<HTMLDivElement>(null)

  // Find the groupBy property
  const groupProperty = useMemo(() => {
    if (!groupBy) return null
    return properties.find((p) => p.id === groupBy) || null
  }, [groupBy, properties])

  // Group rows into columns
  const columns = useMemo((): Column[] => {
    if (!groupProperty || !groupProperty.options) {
      return [
        { id: 'uncategorized', name: 'Uncategorized', color: 'gray', rows },
      ]
    }

    const columnMap: Record<string, DatabaseRow[]> = {}
    const uncategorized: DatabaseRow[] = []

    groupProperty.options.forEach((option) => {
      columnMap[option.id] = []
    })

    rows.forEach((row) => {
      const value = row.properties[groupProperty.id] as string | undefined
      if (value && columnMap[value]) {
        columnMap[value].push(row)
      } else {
        uncategorized.push(row)
      }
    })

    const result: Column[] = groupProperty.options.map((option) => ({
      id: option.id,
      name: option.name,
      color: option.color,
      rows: columnMap[option.id],
    }))

    if (uncategorized.length > 0 || result.length === 0) {
      result.unshift({
        id: 'uncategorized',
        name: 'No Status',
        color: 'gray',
        rows: uncategorized,
      })
    }

    return result
  }, [rows, groupProperty])

  // Close menus on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (addGroupRef.current && !addGroupRef.current.contains(e.target as Node)) {
        setShowAddGroup(false)
        setNewGroupName('')
      }
      if (columnMenuRef.current && !columnMenuRef.current.contains(e.target as Node)) {
        setColumnMenu(null)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Handle drag start
  const handleDragStart = useCallback((rowId: string, fromColumn: string) => {
    setDragging({ rowId, fromColumn })
  }, [])

  // Handle drag over
  const handleDragOver = useCallback((e: React.DragEvent, columnId: string) => {
    e.preventDefault()
    setDragOver(columnId)
  }, [])

  // Handle drop
  const handleDrop = useCallback((targetColumn: string) => {
    if (!dragging || !groupProperty) return

    if (dragging.fromColumn !== targetColumn) {
      const newValue = targetColumn === 'uncategorized' ? null : targetColumn
      onUpdateRow(dragging.rowId, { [groupProperty.id]: newValue })
    }

    setDragging(null)
    setDragOver(null)
  }, [dragging, groupProperty, onUpdateRow])

  // Handle add card to column
  const handleAddCard = useCallback(async (columnId: string) => {
    if (!groupProperty) {
      await onAddRow()
      return
    }

    const initialProperties: Record<string, unknown> = columnId === 'uncategorized'
      ? {}
      : { [groupProperty.id]: columnId }

    await onAddRow(initialProperties)
  }, [onAddRow, groupProperty])

  // Handle add group (new column)
  const handleAddGroup = useCallback(async () => {
    if (!groupProperty || !newGroupName.trim()) return

    const newOption: PropertyOption = {
      id: `option-${Date.now()}`,
      name: newGroupName.trim(),
      color: newGroupColor,
    }

    const updatedOptions = [...(groupProperty.options || []), newOption]
    onUpdateProperty(groupProperty.id, { options: updatedOptions })

    setNewGroupName('')
    setNewGroupColor('gray')
    setShowAddGroup(false)
  }, [groupProperty, newGroupName, newGroupColor, onUpdateProperty])

  // Handle rename group
  const handleRenameGroup = useCallback((columnId: string) => {
    if (!groupProperty || columnId === 'uncategorized' || !editGroupName.trim()) return

    const updatedOptions = groupProperty.options?.map(opt =>
      opt.id === columnId ? { ...opt, name: editGroupName.trim() } : opt
    )

    onUpdateProperty(groupProperty.id, { options: updatedOptions })
    setEditingGroup(null)
    setEditGroupName('')
  }, [groupProperty, editGroupName, onUpdateProperty])

  // Handle delete group
  const handleDeleteGroup = useCallback((columnId: string) => {
    if (!groupProperty || columnId === 'uncategorized') return

    if (!confirm('Delete this group? Cards will be moved to "No Status".')) return

    // Move all cards in this column to uncategorized
    const column = columns.find(c => c.id === columnId)
    column?.rows.forEach(row => {
      onUpdateRow(row.id, { [groupProperty.id]: null })
    })

    // Remove the option
    const updatedOptions = groupProperty.options?.filter(opt => opt.id !== columnId)
    onUpdateProperty(groupProperty.id, { options: updatedOptions })
    setColumnMenu(null)
  }, [groupProperty, columns, onUpdateRow, onUpdateProperty])

  // Handle change group color
  const handleChangeColor = useCallback((columnId: string, color: string) => {
    if (!groupProperty || columnId === 'uncategorized') return

    const updatedOptions = groupProperty.options?.map(opt =>
      opt.id === columnId ? { ...opt, color } : opt
    )

    onUpdateProperty(groupProperty.id, { options: updatedOptions })
    setColumnMenu(null)
  }, [groupProperty, onUpdateProperty])

  // Toggle column visibility
  const toggleColumnVisibility = useCallback((columnId: string) => {
    setHiddenColumns(prev => {
      const next = new Set(prev)
      if (next.has(columnId)) {
        next.delete(columnId)
      } else {
        next.add(columnId)
      }
      return next
    })
    setColumnMenu(null)
  }, [])

  // Get title property
  const titleProperty = useMemo(() => {
    return properties.find((p) => p.type === 'text') || properties[0]
  }, [properties])

  // Get visible properties for cards
  const cardProperties = useMemo(() => {
    return properties.filter((p) => p.id !== titleProperty?.id && p.id !== groupProperty?.id).slice(0, 3)
  }, [properties, titleProperty, groupProperty])

  // Visible columns
  const visibleColumns = columns.filter(c => !hiddenColumns.has(c.id))
  const hiddenColumnsArray = columns.filter(c => hiddenColumns.has(c.id))

  return (
    <div className="board-view" style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Hidden columns indicator */}
      {hiddenColumnsArray.length > 0 && (
        <div style={{
          display: 'flex',
          gap: 8,
          padding: '8px 0',
          borderBottom: '1px solid var(--border-color)',
          marginBottom: 8,
        }}>
          <span style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>Hidden columns:</span>
          {hiddenColumnsArray.map(col => (
            <button
              key={col.id}
              onClick={() => toggleColumnVisibility(col.id)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 4,
                padding: '2px 8px',
                background: `var(--tag-${col.color})`,
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                fontSize: 12,
                cursor: 'pointer',
              }}
            >
              {col.name}
              <EyeOff size={10} />
            </button>
          ))}
        </div>
      )}

      <div className="board-columns" style={{
        display: 'flex',
        gap: 12,
        overflowX: 'auto',
        padding: '8px 0',
        flex: 1,
      }}>
        {visibleColumns.map((column) => (
          <div
            key={column.id}
            className={`board-column ${dragOver === column.id ? 'drag-over' : ''}`}
            onDragOver={(e) => handleDragOver(e, column.id)}
            onDrop={() => handleDrop(column.id)}
            style={{
              flex: '0 0 280px',
              display: 'flex',
              flexDirection: 'column',
              background: dragOver === column.id ? 'var(--bg-secondary)' : 'transparent',
              borderRadius: 'var(--radius-md)',
              maxHeight: '100%',
            }}
          >
            <div className="column-header" style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '8px 12px',
              position: 'relative',
            }}>
              {editingGroup === column.id ? (
                <input
                  type="text"
                  value={editGroupName}
                  onChange={(e) => setEditGroupName(e.target.value)}
                  onBlur={() => handleRenameGroup(column.id)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleRenameGroup(column.id)
                    if (e.key === 'Escape') {
                      setEditingGroup(null)
                      setEditGroupName('')
                    }
                  }}
                  autoFocus
                  style={{
                    flex: 1,
                    padding: '4px 8px',
                    border: '1px solid var(--accent-color)',
                    borderRadius: 'var(--radius-sm)',
                    fontSize: 13,
                    fontWeight: 600,
                    outline: 'none',
                  }}
                />
              ) : (
                <>
                  <span
                    className="column-color"
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: '50%',
                      backgroundColor: `var(--tag-${column.color})`,
                    }}
                  />
                  <span
                    className="column-name"
                    style={{
                      flex: 1,
                      fontSize: 13,
                      fontWeight: 600,
                      cursor: column.id !== 'uncategorized' ? 'pointer' : 'default',
                    }}
                    onDoubleClick={() => {
                      if (column.id !== 'uncategorized') {
                        setEditingGroup(column.id)
                        setEditGroupName(column.name)
                      }
                    }}
                  >
                    {column.name}
                  </span>
                  <span className="column-count" style={{
                    fontSize: 12,
                    color: 'var(--text-tertiary)',
                  }}>
                    {column.rows.length}
                  </span>
                  {column.id !== 'uncategorized' && (
                    <button
                      onClick={() => setColumnMenu(columnMenu === column.id ? null : column.id)}
                      style={{
                        width: 24,
                        height: 24,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        background: 'none',
                        border: 'none',
                        cursor: 'pointer',
                        color: 'var(--text-tertiary)',
                        borderRadius: 'var(--radius-sm)',
                      }}
                    >
                      <MoreHorizontal size={14} />
                    </button>
                  )}
                </>
              )}

              {/* Column menu */}
              {columnMenu === column.id && (
                <div
                  ref={columnMenuRef}
                  style={{
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
                  }}
                >
                  <button
                    onClick={() => {
                      setEditingGroup(column.id)
                      setEditGroupName(column.name)
                      setColumnMenu(null)
                    }}
                    style={{
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
                    }}
                  >
                    <Edit2 size={14} />
                    Rename
                  </button>
                  <button
                    onClick={() => toggleColumnVisibility(column.id)}
                    style={{
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
                    }}
                  >
                    <EyeOff size={14} />
                    Hide column
                  </button>
                  <div style={{
                    padding: '8px 12px',
                    borderTop: '1px solid var(--border-color)',
                  }}>
                    <div style={{ fontSize: 11, color: 'var(--text-tertiary)', marginBottom: 6 }}>Color</div>
                    <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                      {COLORS.map(color => (
                        <button
                          key={color}
                          onClick={() => handleChangeColor(column.id, color)}
                          style={{
                            width: 20,
                            height: 20,
                            borderRadius: '50%',
                            border: column.color === color ? '2px solid var(--accent-color)' : '2px solid transparent',
                            background: `var(--tag-${color})`,
                            cursor: 'pointer',
                          }}
                        />
                      ))}
                    </div>
                  </div>
                  <button
                    onClick={() => handleDeleteGroup(column.id)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      width: '100%',
                      padding: '8px 12px',
                      background: 'none',
                      border: 'none',
                      borderTop: '1px solid var(--border-color)',
                      cursor: 'pointer',
                      fontSize: 13,
                      textAlign: 'left',
                      color: 'var(--error-color)',
                    }}
                  >
                    <Trash2 size={14} />
                    Delete group
                  </button>
                </div>
              )}
            </div>

            <div className="column-cards" style={{
              flex: 1,
              overflowY: 'auto',
              padding: '0 8px',
              display: 'flex',
              flexDirection: 'column',
              gap: 8,
            }}>
              {column.rows.map((row) => (
                <div
                  key={row.id}
                  className={`board-card ${dragging?.rowId === row.id ? 'dragging' : ''}`}
                  draggable
                  onDragStart={() => handleDragStart(row.id, column.id)}
                  onDragEnd={() => {
                    setDragging(null)
                    setDragOver(null)
                  }}
                  style={{
                    background: 'var(--bg-primary)',
                    border: '1px solid var(--border-color)',
                    borderRadius: 'var(--radius-md)',
                    padding: 12,
                    cursor: 'grab',
                    opacity: dragging?.rowId === row.id ? 0.5 : 1,
                    position: 'relative',
                  }}
                >
                  {/* Open button */}
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      setDetailRow(row)
                    }}
                    className="card-open-btn"
                    style={{
                      position: 'absolute',
                      top: 8,
                      right: 8,
                      width: 24,
                      height: 24,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: 'var(--bg-secondary)',
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
                    <Maximize2 size={12} />
                  </button>

                  {/* Card title */}
                  {titleProperty && (
                    <div className="card-title" style={{ marginBottom: 8 }}>
                      <PropertyCell
                        property={titleProperty}
                        value={row.properties[titleProperty.id]}
                        onChange={(value) => onUpdateRow(row.id, { [titleProperty.id]: value })}
                      />
                    </div>
                  )}

                  {/* Card properties */}
                  {cardProperties.map((property) => (
                    <div
                      key={property.id}
                      className="card-property"
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                        marginBottom: 4,
                        fontSize: 12,
                      }}
                    >
                      <span
                        className="property-name"
                        style={{
                          color: 'var(--text-tertiary)',
                          minWidth: 60,
                        }}
                      >
                        {property.name}
                      </span>
                      <div style={{ flex: 1 }}>
                        <PropertyCell
                          property={property}
                          value={row.properties[property.id]}
                          onChange={(value) => onUpdateRow(row.id, { [property.id]: value })}
                        />
                      </div>
                    </div>
                  ))}

                  {/* Card actions */}
                  <div
                    className="card-actions"
                    style={{
                      display: 'flex',
                      justifyContent: 'flex-end',
                      marginTop: 8,
                      paddingTop: 8,
                      borderTop: '1px solid var(--border-color-light)',
                    }}
                  >
                    <button
                      className="card-action-btn"
                      onClick={(e) => {
                        e.stopPropagation()
                        if (confirm('Delete this card?')) {
                          onDeleteRow(row.id)
                        }
                      }}
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
                      }}
                    >
                      <Trash2 size={12} />
                    </button>
                  </div>
                </div>
              ))}
            </div>

            {/* Add card button */}
            <button
              className="add-card-btn"
              onClick={() => handleAddCard(column.id)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '8px 12px',
                margin: 8,
                background: 'none',
                border: '1px dashed var(--border-color)',
                borderRadius: 'var(--radius-md)',
                cursor: 'pointer',
                color: 'var(--text-tertiary)',
                fontSize: 13,
              }}
            >
              <Plus size={14} />
              <span>New</span>
            </button>
          </div>
        ))}

        {/* Add column button */}
        {groupProperty && (
          <div className="add-column" style={{ flex: '0 0 280px' }} ref={addGroupRef}>
            {showAddGroup ? (
              <div style={{
                background: 'var(--bg-primary)',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-md)',
                padding: 12,
              }}>
                <input
                  type="text"
                  value={newGroupName}
                  onChange={(e) => setNewGroupName(e.target.value)}
                  placeholder="Group name"
                  autoFocus
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleAddGroup()
                    if (e.key === 'Escape') {
                      setShowAddGroup(false)
                      setNewGroupName('')
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
                <div style={{ marginBottom: 12 }}>
                  <div style={{ fontSize: 11, color: 'var(--text-tertiary)', marginBottom: 6 }}>Color</div>
                  <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                    {COLORS.map(color => (
                      <button
                        key={color}
                        onClick={() => setNewGroupColor(color)}
                        style={{
                          width: 24,
                          height: 24,
                          borderRadius: '50%',
                          border: newGroupColor === color ? '2px solid var(--accent-color)' : '2px solid transparent',
                          background: `var(--tag-${color})`,
                          cursor: 'pointer',
                        }}
                      />
                    ))}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 8 }}>
                  <button
                    onClick={handleAddGroup}
                    disabled={!newGroupName.trim()}
                    style={{
                      flex: 1,
                      padding: '8px 12px',
                      background: 'var(--accent-color)',
                      color: 'white',
                      border: 'none',
                      borderRadius: 'var(--radius-sm)',
                      cursor: newGroupName.trim() ? 'pointer' : 'not-allowed',
                      fontSize: 13,
                      opacity: newGroupName.trim() ? 1 : 0.5,
                    }}
                  >
                    Add group
                  </button>
                  <button
                    onClick={() => {
                      setShowAddGroup(false)
                      setNewGroupName('')
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
                className="add-column-btn"
                onClick={() => setShowAddGroup(true)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: 8,
                  width: '100%',
                  padding: '12px',
                  background: 'none',
                  border: '2px dashed var(--border-color)',
                  borderRadius: 'var(--radius-md)',
                  cursor: 'pointer',
                  color: 'var(--text-tertiary)',
                  fontSize: 13,
                }}
              >
                <Plus size={14} />
                <span>Add group</span>
              </button>
            )}
          </div>
        )}
      </div>

      {/* No groupBy selected message */}
      {!groupBy && (
        <div className="board-empty-state" style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: 40,
          color: 'var(--text-tertiary)',
        }}>
          <p>Select a Status or Select property to group by for Kanban view</p>
        </div>
      )}

      {/* Row Detail Modal */}
      {detailRow && database && (
        <RowDetailModal
          row={detailRow}
          database={database}
          onClose={() => setDetailRow(null)}
          onUpdate={(updatedRow) => {
            onUpdateRow(updatedRow.id, updatedRow.properties)
          }}
          onDelete={(rowId) => {
            onDeleteRow(rowId)
            setDetailRow(null)
          }}
        />
      )}
    </div>
  )
}
