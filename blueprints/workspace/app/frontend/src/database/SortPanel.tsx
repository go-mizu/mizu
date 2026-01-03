import { Property, Sort } from '../api/client'

interface SortPanelProps {
  properties: Property[]
  sorts: Sort[]
  onSortsChange: (sorts: Sort[]) => void
  onClose: () => void
}

export function SortPanel({ properties, sorts, onSortsChange, onClose }: SortPanelProps) {
  const addSort = () => {
    if (properties.length === 0) return
    const newSort: Sort = {
      property: properties[0].id,
      direction: 'asc',
    }
    onSortsChange([...sorts, newSort])
  }

  const updateSort = (index: number, updates: Partial<Sort>) => {
    const newSorts = [...sorts]
    newSorts[index] = { ...newSorts[index], ...updates }
    onSortsChange(newSorts)
  }

  const removeSort = (index: number) => {
    onSortsChange(sorts.filter((_, i) => i !== index))
  }

  const moveSort = (index: number, direction: 'up' | 'down') => {
    const newIndex = direction === 'up' ? index - 1 : index + 1
    if (newIndex < 0 || newIndex >= sorts.length) return

    const newSorts = [...sorts]
    const [removed] = newSorts.splice(index, 1)
    newSorts.splice(newIndex, 0, removed)
    onSortsChange(newSorts)
  }

  const clearAll = () => {
    onSortsChange([])
  }

  return (
    <div className="sort-panel">
      <div className="panel-header">
        <h3>Sort</h3>
        <button className="close-btn" onClick={onClose}>×</button>
      </div>

      <div className="sorts-list">
        {sorts.map((sort, index) => {
          const property = properties.find((p) => p.id === sort.property)

          return (
            <div key={index} className="sort-row">
              {/* Drag handle / reorder buttons */}
              <div className="sort-order-controls">
                <button
                  className="order-btn"
                  disabled={index === 0}
                  onClick={() => moveSort(index, 'up')}
                >
                  <svg width="10" height="10" viewBox="0 0 10 10">
                    <path d="M5 2L2 6h6L5 2z" fill="currentColor" />
                  </svg>
                </button>
                <button
                  className="order-btn"
                  disabled={index === sorts.length - 1}
                  onClick={() => moveSort(index, 'down')}
                >
                  <svg width="10" height="10" viewBox="0 0 10 10">
                    <path d="M5 8L2 4h6L5 8z" fill="currentColor" />
                  </svg>
                </button>
              </div>

              {/* Property select */}
              <select
                value={sort.property}
                onChange={(e) => updateSort(index, { property: e.target.value })}
                className="sort-select"
              >
                {properties.map((prop) => (
                  <option key={prop.id} value={prop.id}>
                    {prop.name}
                  </option>
                ))}
              </select>

              {/* Direction select */}
              <select
                value={sort.direction}
                onChange={(e) => updateSort(index, { direction: e.target.value as 'asc' | 'desc' })}
                className="sort-select direction"
              >
                <option value="asc">
                  {property?.type === 'text' ? 'A → Z' :
                    property?.type === 'number' ? '1 → 9' :
                    property?.type === 'date' ? 'Oldest first' :
                    'Ascending'}
                </option>
                <option value="desc">
                  {property?.type === 'text' ? 'Z → A' :
                    property?.type === 'number' ? '9 → 1' :
                    property?.type === 'date' ? 'Newest first' :
                    'Descending'}
                </option>
              </select>

              {/* Remove button */}
              <button className="remove-sort-btn" onClick={() => removeSort(index)}>
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                  <path d="M3 3l8 8M11 3L3 11" stroke="currentColor" strokeWidth="1.5" />
                </svg>
              </button>
            </div>
          )
        })}
      </div>

      <div className="panel-footer">
        <button className="add-sort-btn" onClick={addSort}>
          <svg width="12" height="12" viewBox="0 0 12 12">
            <path d="M6 2v8M2 6h8" stroke="currentColor" strokeWidth="1.5" />
          </svg>
          Add sort
        </button>
        {sorts.length > 0 && (
          <button className="clear-btn" onClick={clearAll}>
            Clear all
          </button>
        )}
      </div>
    </div>
  )
}
