import { useState, useCallback, useMemo } from 'react'
import { DatabaseRow, Property } from '../../api/client'
import { PropertyCell } from '../PropertyCell'

interface ListViewProps {
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

interface Group {
  id: string
  name: string
  color?: string
  rows: DatabaseRow[]
}

export function ListView({
  rows,
  properties,
  groupBy,
  onAddRow,
  onUpdateRow,
  onDeleteRow,
}: ListViewProps) {
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set())
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set(['all', 'uncategorized']))

  // Find the groupBy property
  const groupProperty = useMemo(() => {
    if (!groupBy) return null
    return properties.find((p) => p.id === groupBy) || null
  }, [groupBy, properties])

  // Group rows
  const groups = useMemo((): Group[] => {
    if (!groupProperty || !groupProperty.options) {
      return [{ id: 'all', name: 'All items', rows }]
    }

    const groupMap: Record<string, DatabaseRow[]> = {}
    const uncategorized: DatabaseRow[] = []

    groupProperty.options.forEach((option) => {
      groupMap[option.id] = []
    })

    rows.forEach((row) => {
      const value = row.properties[groupProperty.id] as string | undefined
      if (value && groupMap[value]) {
        groupMap[value].push(row)
      } else {
        uncategorized.push(row)
      }
    })

    const result: Group[] = groupProperty.options.map((option) => ({
      id: option.id,
      name: option.name,
      color: option.color,
      rows: groupMap[option.id],
    }))

    if (uncategorized.length > 0) {
      result.unshift({
        id: 'uncategorized',
        name: 'No Status',
        rows: uncategorized,
      })
    }

    return result
  }, [rows, groupProperty])

  // Get title property
  const titleProperty = useMemo(() => {
    return properties.find((p) => p.type === 'text') || properties[0]
  }, [properties])

  // Get preview properties (exclude title and group)
  const previewProperties = useMemo(() => {
    return properties.filter((p) => p.id !== titleProperty?.id && p.id !== groupProperty?.id).slice(0, 2)
  }, [properties, titleProperty, groupProperty])

  // Toggle row expansion
  const toggleRow = useCallback((rowId: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev)
      if (next.has(rowId)) {
        next.delete(rowId)
      } else {
        next.add(rowId)
      }
      return next
    })
  }, [])

  // Toggle group expansion
  const toggleGroup = useCallback((groupId: string) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(groupId)) {
        next.delete(groupId)
      } else {
        next.add(groupId)
      }
      return next
    })
  }, [])

  return (
    <div className="list-view">
      {groups.map((group) => (
        <div key={group.id} className="list-group">
          {/* Group header */}
          <div
            className="group-header"
            onClick={() => toggleGroup(group.id)}
          >
            <span className={`expand-icon ${expandedGroups.has(group.id) ? 'expanded' : ''}`}>
              <svg width="10" height="10" viewBox="0 0 10 10">
                <path d="M3 2l4 3-4 3" fill="none" stroke="currentColor" strokeWidth="1.5" />
              </svg>
            </span>
            {group.color && (
              <span className="group-color" style={{ backgroundColor: group.color }} />
            )}
            <span className="group-name">{group.name}</span>
            <span className="group-count">{group.rows.length}</span>
          </div>

          {/* Group items */}
          {expandedGroups.has(group.id) && (
            <div className="group-items">
              {group.rows.map((row) => (
                <div
                  key={row.id}
                  className={`list-item ${expandedRows.has(row.id) ? 'expanded' : ''}`}
                >
                  {/* Item header */}
                  <div
                    className="item-header"
                    onClick={() => toggleRow(row.id)}
                  >
                    <span className={`expand-icon ${expandedRows.has(row.id) ? 'expanded' : ''}`}>
                      <svg width="10" height="10" viewBox="0 0 10 10">
                        <path d="M3 2l4 3-4 3" fill="none" stroke="currentColor" strokeWidth="1.5" />
                      </svg>
                    </span>

                    {/* Title */}
                    <div className="item-title" onClick={(e) => e.stopPropagation()}>
                      {titleProperty && (
                        <PropertyCell
                          property={titleProperty}
                          value={row.properties[titleProperty.id]}
                          onChange={(value) => onUpdateRow(row.id, { [titleProperty.id]: value })}
                        />
                      )}
                    </div>

                    {/* Preview properties */}
                    <div className="item-preview" onClick={(e) => e.stopPropagation()}>
                      {previewProperties.map((property) => (
                        <div key={property.id} className="preview-property">
                          <PropertyCell
                            property={property}
                            value={row.properties[property.id]}
                            onChange={(value) => onUpdateRow(row.id, { [property.id]: value })}
                          />
                        </div>
                      ))}
                    </div>

                    {/* Delete button */}
                    <button
                      className="item-delete"
                      onClick={(e) => {
                        e.stopPropagation()
                        onDeleteRow(row.id)
                      }}
                    >
                      <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
                        <path d="M2 2l8 8M10 2L2 10" stroke="currentColor" strokeWidth="1.5" />
                      </svg>
                    </button>
                  </div>

                  {/* Expanded content */}
                  {expandedRows.has(row.id) && (
                    <div className="item-expanded">
                      {properties.map((property) => (
                        <div key={property.id} className="expanded-property">
                          <span className="property-label">{property.name}</span>
                          <div className="property-value">
                            <PropertyCell
                              property={property}
                              value={row.properties[property.id]}
                              onChange={(value) => onUpdateRow(row.id, { [property.id]: value })}
                            />
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      ))}

      {/* Add item button */}
      <button className="add-item-btn" onClick={() => onAddRow()}>
        <svg width="12" height="12" viewBox="0 0 12 12">
          <path d="M6 2v8M2 6h8" stroke="currentColor" strokeWidth="1.5" />
        </svg>
        <span>New</span>
      </button>
    </div>
  )
}
