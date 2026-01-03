import { Property, Filter } from '../api/client'

interface FilterPanelProps {
  properties: Property[]
  filters: Filter[]
  onFiltersChange: (filters: Filter[]) => void
  onClose: () => void
}

const OPERATORS: Record<string, { label: string; types: string[] }[]> = {
  text: [
    { label: 'contains', types: ['text', 'url', 'email', 'phone'] },
    { label: 'does not contain', types: ['text', 'url', 'email', 'phone'] },
    { label: 'is', types: ['text', 'url', 'email', 'phone'] },
    { label: 'is not', types: ['text', 'url', 'email', 'phone'] },
    { label: 'starts with', types: ['text', 'url', 'email', 'phone'] },
    { label: 'ends with', types: ['text', 'url', 'email', 'phone'] },
    { label: 'is empty', types: ['text', 'url', 'email', 'phone'] },
    { label: 'is not empty', types: ['text', 'url', 'email', 'phone'] },
  ],
  number: [
    { label: '=', types: ['number'] },
    { label: '≠', types: ['number'] },
    { label: '>', types: ['number'] },
    { label: '<', types: ['number'] },
    { label: '≥', types: ['number'] },
    { label: '≤', types: ['number'] },
    { label: 'is empty', types: ['number'] },
    { label: 'is not empty', types: ['number'] },
  ],
  select: [
    { label: 'is', types: ['select', 'status'] },
    { label: 'is not', types: ['select', 'status'] },
    { label: 'is empty', types: ['select', 'status'] },
    { label: 'is not empty', types: ['select', 'status'] },
  ],
  multi_select: [
    { label: 'contains', types: ['multi_select'] },
    { label: 'does not contain', types: ['multi_select'] },
    { label: 'is empty', types: ['multi_select'] },
    { label: 'is not empty', types: ['multi_select'] },
  ],
  date: [
    { label: 'is', types: ['date', 'created_time', 'last_edited_time'] },
    { label: 'is before', types: ['date', 'created_time', 'last_edited_time'] },
    { label: 'is after', types: ['date', 'created_time', 'last_edited_time'] },
    { label: 'is on or before', types: ['date', 'created_time', 'last_edited_time'] },
    { label: 'is on or after', types: ['date', 'created_time', 'last_edited_time'] },
    { label: 'is empty', types: ['date', 'created_time', 'last_edited_time'] },
    { label: 'is not empty', types: ['date', 'created_time', 'last_edited_time'] },
  ],
  checkbox: [
    { label: 'is', types: ['checkbox'] },
  ],
  person: [
    { label: 'is', types: ['person', 'created_by', 'last_edited_by'] },
    { label: 'is not', types: ['person', 'created_by', 'last_edited_by'] },
    { label: 'is empty', types: ['person', 'created_by', 'last_edited_by'] },
    { label: 'is not empty', types: ['person', 'created_by', 'last_edited_by'] },
  ],
}

function getOperatorsForType(type: string): string[] {
  for (const [, operators] of Object.entries(OPERATORS)) {
    const matching = operators.filter((op) => op.types.includes(type))
    if (matching.length > 0) {
      return matching.map((op) => op.label)
    }
  }
  return ['is', 'is not', 'is empty', 'is not empty']
}

export function FilterPanel({ properties, filters, onFiltersChange, onClose }: FilterPanelProps) {
  const addFilter = () => {
    if (properties.length === 0) return
    const newFilter: Filter = {
      property: properties[0].id,
      operator: 'is',
      value: '',
    }
    onFiltersChange([...filters, newFilter])
  }

  const updateFilter = (index: number, updates: Partial<Filter>) => {
    const newFilters = [...filters]
    newFilters[index] = { ...newFilters[index], ...updates }
    onFiltersChange(newFilters)
  }

  const removeFilter = (index: number) => {
    onFiltersChange(filters.filter((_, i) => i !== index))
  }

  const clearAll = () => {
    onFiltersChange([])
  }

  return (
    <div className="filter-panel">
      <div className="panel-header">
        <h3>Filters</h3>
        <button className="close-btn" onClick={onClose}>×</button>
      </div>

      <div className="filters-list">
        {filters.map((filter, index) => {
          const property = properties.find((p) => p.id === filter.property)
          const operators = property ? getOperatorsForType(property.type) : []
          const showValue = !['is empty', 'is not empty'].includes(filter.operator)

          return (
            <div key={index} className="filter-row">
              {/* Property select */}
              <select
                value={filter.property}
                onChange={(e) => updateFilter(index, { property: e.target.value })}
                className="filter-select"
              >
                {properties.map((prop) => (
                  <option key={prop.id} value={prop.id}>
                    {prop.name}
                  </option>
                ))}
              </select>

              {/* Operator select */}
              <select
                value={filter.operator}
                onChange={(e) => updateFilter(index, { operator: e.target.value })}
                className="filter-select"
              >
                {operators.map((op) => (
                  <option key={op} value={op}>
                    {op}
                  </option>
                ))}
              </select>

              {/* Value input */}
              {showValue && property && (
                <FilterValueInput
                  property={property}
                  value={filter.value}
                  onChange={(value) => updateFilter(index, { value })}
                />
              )}

              {/* Remove button */}
              <button className="remove-filter-btn" onClick={() => removeFilter(index)}>
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                  <path d="M3 3l8 8M11 3L3 11" stroke="currentColor" strokeWidth="1.5" />
                </svg>
              </button>
            </div>
          )
        })}
      </div>

      <div className="panel-footer">
        <button className="add-filter-btn" onClick={addFilter}>
          <svg width="12" height="12" viewBox="0 0 12 12">
            <path d="M6 2v8M2 6h8" stroke="currentColor" strokeWidth="1.5" />
          </svg>
          Add filter
        </button>
        {filters.length > 0 && (
          <button className="clear-btn" onClick={clearAll}>
            Clear all
          </button>
        )}
      </div>
    </div>
  )
}

function FilterValueInput({
  property,
  value,
  onChange,
}: {
  property: Property
  value: unknown
  onChange: (value: unknown) => void
}) {
  switch (property.type) {
    case 'select':
    case 'status':
      return (
        <select
          value={value as string || ''}
          onChange={(e) => onChange(e.target.value)}
          className="filter-select"
        >
          <option value="">Select...</option>
          {property.options?.map((opt) => (
            <option key={opt.id} value={opt.id}>
              {opt.name}
            </option>
          ))}
        </select>
      )

    case 'multi_select':
      return (
        <select
          value={value as string || ''}
          onChange={(e) => onChange(e.target.value)}
          className="filter-select"
        >
          <option value="">Select...</option>
          {property.options?.map((opt) => (
            <option key={opt.id} value={opt.id}>
              {opt.name}
            </option>
          ))}
        </select>
      )

    case 'checkbox':
      return (
        <select
          value={String(value || false)}
          onChange={(e) => onChange(e.target.value === 'true')}
          className="filter-select"
        >
          <option value="true">Checked</option>
          <option value="false">Unchecked</option>
        </select>
      )

    case 'date':
    case 'created_time':
    case 'last_edited_time':
      return (
        <input
          type="date"
          value={value as string || ''}
          onChange={(e) => onChange(e.target.value)}
          className="filter-input"
        />
      )

    case 'number':
      return (
        <input
          type="number"
          value={value as number || ''}
          onChange={(e) => onChange(parseFloat(e.target.value))}
          className="filter-input"
          placeholder="Value"
        />
      )

    default:
      return (
        <input
          type="text"
          value={value as string || ''}
          onChange={(e) => onChange(e.target.value)}
          className="filter-input"
          placeholder="Value"
        />
      )
  }
}
