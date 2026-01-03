import { useState, useCallback, useMemo } from 'react'
import { DatabaseRow, Property } from '../../api/client'
import { PropertyCell } from '../PropertyCell'

interface BoardViewProps {
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

interface Column {
  id: string
  name: string
  color: string
  rows: DatabaseRow[]
}

export function BoardView({
  rows,
  properties,
  groupBy,
  onAddRow,
  onUpdateRow,
  onDeleteRow,
}: BoardViewProps) {
  const [dragging, setDragging] = useState<{ rowId: string; fromColumn: string } | null>(null)
  const [dragOver, setDragOver] = useState<string | null>(null)

  // Find the groupBy property
  const groupProperty = useMemo(() => {
    if (!groupBy) return null
    return properties.find((p) => p.id === groupBy) || null
  }, [groupBy, properties])

  // Group rows into columns
  const columns = useMemo((): Column[] => {
    if (!groupProperty || !groupProperty.options) {
      // Default columns if no groupBy property
      return [
        { id: 'uncategorized', name: 'Uncategorized', color: '#e0e0e0', rows },
      ]
    }

    const columnMap: Record<string, DatabaseRow[]> = {}
    const uncategorized: DatabaseRow[] = []

    // Initialize columns from options
    groupProperty.options.forEach((option) => {
      columnMap[option.id] = []
    })

    // Distribute rows to columns
    rows.forEach((row) => {
      const value = row.properties[groupProperty.id] as string | undefined
      if (value && columnMap[value]) {
        columnMap[value].push(row)
      } else {
        uncategorized.push(row)
      }
    })

    // Build columns array
    const result: Column[] = groupProperty.options.map((option) => ({
      id: option.id,
      name: option.name,
      color: option.color,
      rows: columnMap[option.id],
    }))

    // Add uncategorized column if there are uncategorized rows
    if (uncategorized.length > 0) {
      result.unshift({
        id: 'uncategorized',
        name: 'No Status',
        color: '#e0e0e0',
        rows: uncategorized,
      })
    }

    return result
  }, [rows, groupProperty])

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
      // Update the row's group property
      const newValue = targetColumn === 'uncategorized' ? null : targetColumn
      onUpdateRow(dragging.rowId, { [groupProperty.id]: newValue })
    }

    setDragging(null)
    setDragOver(null)
  }, [dragging, groupProperty, onUpdateRow])

  // Handle add card to column
  const handleAddCard = useCallback(async (columnId: string) => {
    if (!groupProperty) {
      // No grouping, just add a new row
      await onAddRow()
      return
    }

    // Create the row with the column's group value
    const initialProperties: Record<string, unknown> = columnId === 'uncategorized'
      ? {}
      : { [groupProperty.id]: columnId }

    await onAddRow(initialProperties)
  }, [onAddRow, groupProperty])

  // Get title property (first text property or first property)
  const titleProperty = useMemo(() => {
    return properties.find((p) => p.type === 'text') || properties[0]
  }, [properties])

  // Get visible properties for cards (exclude title and group property)
  const cardProperties = useMemo(() => {
    return properties.filter((p) => p.id !== titleProperty?.id && p.id !== groupProperty?.id).slice(0, 3)
  }, [properties, titleProperty, groupProperty])

  return (
    <div className="board-view">
      <div className="board-columns">
        {columns.map((column) => (
          <div
            key={column.id}
            className={`board-column ${dragOver === column.id ? 'drag-over' : ''}`}
            onDragOver={(e) => handleDragOver(e, column.id)}
            onDrop={() => handleDrop(column.id)}
          >
            <div className="column-header">
              <span
                className="column-color"
                style={{ backgroundColor: column.color }}
              />
              <span className="column-name">{column.name}</span>
              <span className="column-count">{column.rows.length}</span>
            </div>

            <div className="column-cards">
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
                >
                  {/* Card title */}
                  {titleProperty && (
                    <div className="card-title">
                      <PropertyCell
                        property={titleProperty}
                        value={row.properties[titleProperty.id]}
                        onChange={(value) => onUpdateRow(row.id, { [titleProperty.id]: value })}
                      />
                    </div>
                  )}

                  {/* Card properties */}
                  {cardProperties.map((property) => (
                    <div key={property.id} className="card-property">
                      <span className="property-name">{property.name}</span>
                      <PropertyCell
                        property={property}
                        value={row.properties[property.id]}
                        onChange={(value) => onUpdateRow(row.id, { [property.id]: value })}
                      />
                    </div>
                  ))}

                  {/* Card actions */}
                  <div className="card-actions">
                    <button
                      className="card-action-btn"
                      onClick={() => onDeleteRow(row.id)}
                    >
                      <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
                        <path d="M2 2l8 8M10 2L2 10" stroke="currentColor" strokeWidth="1.5" />
                      </svg>
                    </button>
                  </div>
                </div>
              ))}
            </div>

            {/* Add card button */}
            <button
              className="add-card-btn"
              onClick={() => handleAddCard(column.id)}
            >
              <svg width="12" height="12" viewBox="0 0 12 12">
                <path d="M6 2v8M2 6h8" stroke="currentColor" strokeWidth="1.5" />
              </svg>
              <span>New</span>
            </button>
          </div>
        ))}

        {/* Add column button (if groupBy property is set) */}
        {groupProperty && (
          <div className="add-column">
            <button className="add-column-btn">
              <svg width="12" height="12" viewBox="0 0 12 12">
                <path d="M6 2v8M2 6h8" stroke="currentColor" strokeWidth="1.5" />
              </svg>
              <span>Add group</span>
            </button>
          </div>
        )}
      </div>

      {/* No groupBy selected message */}
      {!groupBy && (
        <div className="board-empty-state">
          <p>Select a property to group by for Kanban view</p>
        </div>
      )}
    </div>
  )
}
