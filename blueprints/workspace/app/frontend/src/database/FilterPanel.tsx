import { useState, useRef, useEffect } from 'react'
import { Property, Filter, FilterGroup, RelativeDateValue } from '../api/client'
import { Plus, X, ChevronDown } from 'lucide-react'

interface FilterPanelProps {
  properties: Property[]
  filters: Filter[]
  filterGroups?: FilterGroup
  onFiltersChange: (filters: Filter[]) => void
  onFilterGroupsChange?: (groups: FilterGroup) => void
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
    { label: 'is within', types: ['date', 'created_time', 'last_edited_time'] },
    { label: 'is empty', types: ['date', 'created_time', 'last_edited_time'] },
    { label: 'is not empty', types: ['date', 'created_time', 'last_edited_time'] },
  ],
  checkbox: [
    { label: 'is', types: ['checkbox'] },
  ],
  person: [
    { label: 'is', types: ['person', 'created_by', 'last_edited_by'] },
    { label: 'is not', types: ['person', 'created_by', 'last_edited_by'] },
    { label: 'contains', types: ['person'] },
    { label: 'does not contain', types: ['person'] },
    { label: 'is empty', types: ['person', 'created_by', 'last_edited_by'] },
    { label: 'is not empty', types: ['person', 'created_by', 'last_edited_by'] },
  ],
  relation: [
    { label: 'contains', types: ['relation'] },
    { label: 'does not contain', types: ['relation'] },
    { label: 'is empty', types: ['relation'] },
    { label: 'is not empty', types: ['relation'] },
  ],
  files: [
    { label: 'is empty', types: ['files'] },
    { label: 'is not empty', types: ['files'] },
  ],
}

const RELATIVE_DATE_OPTIONS: { value: RelativeDateValue; label: string; group: string }[] = [
  { value: 'today', label: 'Today', group: 'Days' },
  { value: 'tomorrow', label: 'Tomorrow', group: 'Days' },
  { value: 'yesterday', label: 'Yesterday', group: 'Days' },
  { value: 'one_week_ago', label: 'One week ago', group: 'Days' },
  { value: 'one_week_from_now', label: 'One week from now', group: 'Days' },
  { value: 'one_month_ago', label: 'One month ago', group: 'Days' },
  { value: 'one_month_from_now', label: 'One month from now', group: 'Days' },
  { value: 'this_week', label: 'This week', group: 'Weeks' },
  { value: 'last_week', label: 'Last week', group: 'Weeks' },
  { value: 'next_week', label: 'Next week', group: 'Weeks' },
  { value: 'this_month', label: 'This month', group: 'Months' },
  { value: 'last_month', label: 'Last month', group: 'Months' },
  { value: 'next_month', label: 'Next month', group: 'Months' },
  { value: 'this_year', label: 'This year', group: 'Years' },
  { value: 'last_year', label: 'Last year', group: 'Years' },
  { value: 'next_year', label: 'Next year', group: 'Years' },
]

function getOperatorsForType(type: string): string[] {
  for (const [, operators] of Object.entries(OPERATORS)) {
    const matching = operators.filter((op) => op.types.includes(type))
    if (matching.length > 0) {
      return matching.map((op) => op.label)
    }
  }
  return ['is', 'is not', 'is empty', 'is not empty']
}

export function FilterPanel({ properties, filters, filterGroups, onFiltersChange, onFilterGroupsChange, onClose }: FilterPanelProps) {
  const [useAdvancedMode, setUseAdvancedMode] = useState(!!filterGroups)
  const [localGroups, setLocalGroups] = useState<FilterGroup>(
    filterGroups || { operator: 'and', filters: filters.map(f => ({ ...f })) }
  )
  const [groupOperator, setGroupOperator] = useState<'and' | 'or'>('and')

  // Sync local groups when filterGroups prop changes
  useEffect(() => {
    if (filterGroups) {
      setLocalGroups(filterGroups)
    }
  }, [filterGroups])

  const addFilter = () => {
    if (properties.length === 0) return
    const newFilter: Filter = {
      property: properties[0].id,
      operator: 'is',
      value: '',
    }

    if (useAdvancedMode) {
      setLocalGroups({
        ...localGroups,
        filters: [...localGroups.filters, newFilter]
      })
      onFilterGroupsChange?.(localGroups)
    } else {
      onFiltersChange([...filters, newFilter])
    }
  }

  const addFilterGroup = () => {
    const newGroup: FilterGroup = {
      operator: groupOperator === 'and' ? 'or' : 'and',
      filters: []
    }
    setLocalGroups({
      ...localGroups,
      filters: [...localGroups.filters, newGroup]
    })
    onFilterGroupsChange?.(localGroups)
  }

  const updateFilter = (index: number, updates: Partial<Filter>) => {
    if (useAdvancedMode) {
      const newFilters = [...localGroups.filters]
      newFilters[index] = { ...newFilters[index], ...updates } as Filter
      const newGroups = { ...localGroups, filters: newFilters }
      setLocalGroups(newGroups)
      onFilterGroupsChange?.(newGroups)
    } else {
      const newFilters = [...filters]
      newFilters[index] = { ...newFilters[index], ...updates }
      onFiltersChange(newFilters)
    }
  }

  const removeFilter = (index: number) => {
    if (useAdvancedMode) {
      const newGroups = {
        ...localGroups,
        filters: localGroups.filters.filter((_, i) => i !== index)
      }
      setLocalGroups(newGroups)
      onFilterGroupsChange?.(newGroups)
    } else {
      onFiltersChange(filters.filter((_, i) => i !== index))
    }
  }

  const clearAll = () => {
    if (useAdvancedMode) {
      const emptyGroups = { operator: 'and' as const, filters: [] }
      setLocalGroups(emptyGroups)
      onFilterGroupsChange?.(emptyGroups)
    } else {
      onFiltersChange([])
    }
  }

  const toggleAdvancedMode = () => {
    if (!useAdvancedMode) {
      // Switch to advanced mode
      setLocalGroups({ operator: 'and', filters: filters.map(f => ({ ...f })) })
    } else {
      // Switch to simple mode - flatten filters
      const flatFilters = localGroups.filters.filter(f => 'property' in f) as Filter[]
      onFiltersChange(flatFilters)
    }
    setUseAdvancedMode(!useAdvancedMode)
  }

  const toggleGroupOperator = () => {
    const newOp = localGroups.operator === 'and' ? 'or' : 'and'
    const newGroups = { ...localGroups, operator: newOp as 'and' | 'or' }
    setLocalGroups(newGroups)
    onFilterGroupsChange?.(newGroups)
  }

  const currentFilters = useAdvancedMode ? (localGroups.filters.filter(f => 'property' in f) as Filter[]) : filters

  return (
    <div className="filter-panel" style={{
      background: 'var(--bg-primary)',
      borderRadius: 'var(--radius-lg)',
      boxShadow: 'var(--shadow-lg)',
      border: '1px solid var(--border-color)',
      padding: 0,
      minWidth: 400,
      maxWidth: 600,
    }}>
      <div className="panel-header" style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '12px 16px',
        borderBottom: '1px solid var(--border-color)',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <h3 style={{ margin: 0, fontSize: 14, fontWeight: 600 }}>Filters</h3>
          {useAdvancedMode && currentFilters.length > 1 && (
            <button
              onClick={toggleGroupOperator}
              style={{
                padding: '4px 8px',
                background: 'var(--bg-secondary)',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-sm)',
                fontSize: 12,
                fontWeight: 500,
                cursor: 'pointer',
                textTransform: 'uppercase',
                color: 'var(--accent-color)',
              }}
            >
              {localGroups.operator}
            </button>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <button
            onClick={toggleAdvancedMode}
            style={{
              padding: '4px 8px',
              background: 'none',
              border: 'none',
              fontSize: 12,
              cursor: 'pointer',
              color: 'var(--text-secondary)',
            }}
          >
            {useAdvancedMode ? 'Simple' : 'Advanced'}
          </button>
          <button
            className="close-btn"
            onClick={onClose}
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
              color: 'var(--text-secondary)',
            }}
          >
            <X size={16} />
          </button>
        </div>
      </div>

      <div className="filters-list" style={{ padding: '8px 16px', maxHeight: 400, overflowY: 'auto' }}>
        {currentFilters.length === 0 ? (
          <div style={{ padding: '24px 0', textAlign: 'center', color: 'var(--text-tertiary)' }}>
            No filters applied. Click "Add filter" to get started.
          </div>
        ) : (
          currentFilters.map((filter, index) => {
            const property = properties.find((p) => p.id === filter.property)
            const operators = property ? getOperatorsForType(property.type) : []
            const showValue = !['is empty', 'is not empty'].includes(filter.operator)

            return (
              <div key={index} className="filter-row" style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '8px 0',
                borderBottom: index < currentFilters.length - 1 ? '1px solid var(--border-color-light)' : 'none',
              }}>
                {/* Connector label for subsequent filters */}
                {index > 0 && useAdvancedMode && (
                  <span style={{
                    fontSize: 11,
                    color: 'var(--accent-color)',
                    textTransform: 'uppercase',
                    fontWeight: 600,
                    minWidth: 30,
                  }}>
                    {localGroups.operator}
                  </span>
                )}

                {/* Property select */}
                <select
                  value={filter.property}
                  onChange={(e) => updateFilter(index, { property: e.target.value })}
                  className="filter-select"
                  style={{
                    flex: '0 0 140px',
                    padding: '6px 8px',
                    border: '1px solid var(--border-color)',
                    borderRadius: 'var(--radius-sm)',
                    fontSize: 13,
                    background: 'var(--bg-primary)',
                    color: 'var(--text-primary)',
                  }}
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
                  style={{
                    flex: '0 0 120px',
                    padding: '6px 8px',
                    border: '1px solid var(--border-color)',
                    borderRadius: 'var(--radius-sm)',
                    fontSize: 13,
                    background: 'var(--bg-primary)',
                    color: 'var(--text-primary)',
                  }}
                >
                  {operators.map((op) => (
                    <option key={op} value={op}>
                      {op}
                    </option>
                  ))}
                </select>

                {/* Value input */}
                {showValue && property && (
                  <div style={{ flex: 1 }}>
                    <FilterValueInput
                      property={property}
                      operator={filter.operator}
                      value={filter.value}
                      onChange={(value) => updateFilter(index, { value })}
                    />
                  </div>
                )}

                {/* Remove button */}
                <button
                  className="remove-filter-btn"
                  onClick={() => removeFilter(index)}
                  style={{
                    width: 28,
                    height: 28,
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
                  <X size={14} />
                </button>
              </div>
            )
          })
        )}
      </div>

      <div className="panel-footer" style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '12px 16px',
        borderTop: '1px solid var(--border-color)',
      }}>
        <div style={{ display: 'flex', gap: 8 }}>
          <button
            className="add-filter-btn"
            onClick={addFilter}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              padding: '6px 12px',
              background: 'none',
              border: 'none',
              borderRadius: 'var(--radius-sm)',
              cursor: 'pointer',
              fontSize: 13,
              color: 'var(--accent-color)',
              fontWeight: 500,
            }}
          >
            <Plus size={14} />
            Add filter
          </button>
          {useAdvancedMode && (
            <button
              className="add-group-btn"
              onClick={addFilterGroup}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '6px 12px',
                background: 'none',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
                fontSize: 13,
                color: 'var(--text-secondary)',
              }}
            >
              <Plus size={14} />
              Add filter group
            </button>
          )}
        </div>
        {currentFilters.length > 0 && (
          <button
            className="clear-btn"
            onClick={clearAll}
            style={{
              padding: '6px 12px',
              background: 'none',
              border: 'none',
              borderRadius: 'var(--radius-sm)',
              cursor: 'pointer',
              fontSize: 13,
              color: 'var(--text-secondary)',
            }}
          >
            Clear all
          </button>
        )}
      </div>
    </div>
  )
}

function FilterValueInput({
  property,
  operator,
  value,
  onChange,
}: {
  property: Property
  operator: string
  value: unknown
  onChange: (value: unknown) => void
}) {
  const [showRelativePicker, setShowRelativePicker] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setShowRelativePicker(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const inputStyle = {
    width: '100%',
    padding: '6px 8px',
    border: '1px solid var(--border-color)',
    borderRadius: 'var(--radius-sm)',
    fontSize: 13,
    background: 'var(--bg-primary)',
    color: 'var(--text-primary)',
  }

  switch (property.type) {
    case 'select':
    case 'status':
      return (
        <select
          value={value as string || ''}
          onChange={(e) => onChange(e.target.value)}
          className="filter-select"
          style={inputStyle}
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
          style={inputStyle}
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
          style={inputStyle}
        >
          <option value="true">Checked</option>
          <option value="false">Unchecked</option>
        </select>
      )

    case 'date':
    case 'created_time':
    case 'last_edited_time':
      // Show relative date picker for "is within" operator
      if (operator === 'is within') {
        const currentValue = typeof value === 'object' && value !== null && 'type' in value
          ? (value as { type: string; value: RelativeDateValue }).value
          : ''

        return (
          <div ref={ref} style={{ position: 'relative' }}>
            <button
              onClick={() => setShowRelativePicker(!showRelativePicker)}
              style={{
                ...inputStyle,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                cursor: 'pointer',
                textAlign: 'left',
              }}
            >
              <span>
                {currentValue
                  ? RELATIVE_DATE_OPTIONS.find(o => o.value === currentValue)?.label || 'Select...'
                  : 'Select time period...'}
              </span>
              <ChevronDown size={14} />
            </button>
            {showRelativePicker && (
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
                {['Days', 'Weeks', 'Months', 'Years'].map(group => (
                  <div key={group}>
                    <div style={{
                      padding: '8px 12px',
                      fontSize: 11,
                      fontWeight: 600,
                      textTransform: 'uppercase',
                      color: 'var(--text-tertiary)',
                      background: 'var(--bg-secondary)',
                    }}>
                      {group}
                    </div>
                    {RELATIVE_DATE_OPTIONS.filter(o => o.group === group).map(option => (
                      <button
                        key={option.value}
                        onClick={() => {
                          onChange({ type: 'relative', value: option.value })
                          setShowRelativePicker(false)
                        }}
                        style={{
                          display: 'block',
                          width: '100%',
                          padding: '8px 12px',
                          background: currentValue === option.value ? 'var(--accent-bg)' : 'none',
                          border: 'none',
                          textAlign: 'left',
                          cursor: 'pointer',
                          fontSize: 13,
                        }}
                      >
                        {option.label}
                      </button>
                    ))}
                  </div>
                ))}
              </div>
            )}
          </div>
        )
      }

      // Default date input
      return (
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <input
            type="date"
            value={typeof value === 'string' ? value : ''}
            onChange={(e) => onChange(e.target.value)}
            className="filter-input"
            style={{ ...inputStyle, flex: 1 }}
          />
          <button
            onClick={() => {
              const today = new Date().toISOString().split('T')[0]
              onChange(today)
            }}
            style={{
              padding: '6px 8px',
              background: 'var(--bg-secondary)',
              border: '1px solid var(--border-color)',
              borderRadius: 'var(--radius-sm)',
              fontSize: 11,
              cursor: 'pointer',
              whiteSpace: 'nowrap',
            }}
          >
            Today
          </button>
        </div>
      )

    case 'number':
      return (
        <input
          type="number"
          value={value as number || ''}
          onChange={(e) => onChange(parseFloat(e.target.value))}
          className="filter-input"
          placeholder="Value"
          style={inputStyle}
        />
      )

    case 'person':
    case 'created_by':
    case 'last_edited_by':
      return (
        <select
          value={value as string || ''}
          onChange={(e) => onChange(e.target.value)}
          className="filter-select"
          style={inputStyle}
        >
          <option value="">Select person...</option>
          <option value="me">Me</option>
          {/* In real implementation, fetch workspace members */}
        </select>
      )

    default:
      return (
        <input
          type="text"
          value={value as string || ''}
          onChange={(e) => onChange(e.target.value)}
          className="filter-input"
          placeholder="Value"
          style={inputStyle}
        />
      )
  }
}
