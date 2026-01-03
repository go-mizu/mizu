import { createReactBlockSpec } from '@blocknote/react'
import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Database,
  Table,
  LayoutGrid,
  Calendar,
  List,
  Kanban,
  BarChart3,
  Clock,
  ExternalLink,
  ChevronDown,
  Plus,
  Settings,
  Filter,
  ArrowUpDown,
  Search,
  Loader2,
  X,
  ArrowUp,
  ArrowDown,
} from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { api } from '../../api/client'

interface DatabaseRecord {
  id: string
  properties: Record<string, unknown>
}

interface DatabaseView {
  id: string
  name: string
  type: 'table' | 'board' | 'calendar' | 'list' | 'gallery' | 'timeline' | 'chart'
}

interface DatabaseData {
  id: string
  name: string
  icon?: string
  views: DatabaseView[]
  records: DatabaseRecord[]
  properties: Record<string, { type: string; name: string }>
}

// Filter operators
type FilterOperator = 'equals' | 'not_equals' | 'contains' | 'not_contains' | 'is_empty' | 'is_not_empty'

interface FilterItem {
  id: string
  property: string
  operator: FilterOperator
  value: string
}

interface SortItem {
  id: string
  property: string
  direction: 'asc' | 'desc'
}

// Apply filters to records
function applyFilters(records: DatabaseRecord[], filters: FilterItem[]): DatabaseRecord[] {
  if (filters.length === 0) return records

  return records.filter((record) =>
    filters.every((filter) => {
      const value = record.properties[filter.property]
      const strValue = value != null ? String(value).toLowerCase() : ''
      const filterValue = filter.value.toLowerCase()

      switch (filter.operator) {
        case 'equals':
          return strValue === filterValue
        case 'not_equals':
          return strValue !== filterValue
        case 'contains':
          return strValue.includes(filterValue)
        case 'not_contains':
          return !strValue.includes(filterValue)
        case 'is_empty':
          return !value || strValue === ''
        case 'is_not_empty':
          return value && strValue !== ''
        default:
          return true
      }
    })
  )
}

// Apply sorts to records
function applySorts(records: DatabaseRecord[], sorts: SortItem[]): DatabaseRecord[] {
  if (sorts.length === 0) return records

  return [...records].sort((a, b) => {
    for (const sort of sorts) {
      const aVal = a.properties[sort.property]
      const bVal = b.properties[sort.property]

      const aStr = aVal != null ? String(aVal) : ''
      const bStr = bVal != null ? String(bVal) : ''

      const comparison = aStr.localeCompare(bStr, undefined, { numeric: true })
      if (comparison !== 0) {
        return sort.direction === 'asc' ? comparison : -comparison
      }
    }
    return 0
  })
}

// View type icons
const viewIcons: Record<string, React.ReactNode> = {
  table: <Table size={14} />,
  board: <Kanban size={14} />,
  calendar: <Calendar size={14} />,
  list: <List size={14} />,
  gallery: <LayoutGrid size={14} />,
  timeline: <Clock size={14} />,
  chart: <BarChart3 size={14} />,
}

export const InlineDatabaseBlock = createReactBlockSpec(
  {
    type: 'inlineDatabase',
    propSchema: {
      databaseId: {
        default: '',
      },
      viewId: {
        default: '',
      },
      showTitle: {
        default: true,
      },
      maxRows: {
        default: 5,
      },
      filters: {
        default: '[]', // JSON string of filters
      },
      sorts: {
        default: '[]', // JSON string of sorts
      },
    },
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [database, setDatabase] = useState<DatabaseData | null>(null)
      const [isLoading, setIsLoading] = useState(true)
      const [error, setError] = useState<string | null>(null)
      const [isHovered, setIsHovered] = useState(false)
      const [showViewPicker, setShowViewPicker] = useState(false)
      const [showDatabasePicker, setShowDatabasePicker] = useState(false)
      const [showFilterMenu, setShowFilterMenu] = useState(false)
      const [showSortMenu, setShowSortMenu] = useState(false)
      const [availableDatabases, setAvailableDatabases] = useState<Array<{ id: string; name: string; icon?: string }>>([])
      const [searchQuery, setSearchQuery] = useState('')

      const databaseId = block.props.databaseId as string
      const viewId = block.props.viewId as string
      const showTitle = block.props.showTitle as boolean
      const maxRows = block.props.maxRows as number

      // Parse filters and sorts from props
      const filters: FilterItem[] = useMemo(() => {
        try {
          return JSON.parse(block.props.filters as string) || []
        } catch {
          return []
        }
      }, [block.props.filters])

      const sorts: SortItem[] = useMemo(() => {
        try {
          return JSON.parse(block.props.sorts as string) || []
        } catch {
          return []
        }
      }, [block.props.sorts])

      // Update filters in props
      const updateFilters = useCallback((newFilters: FilterItem[]) => {
        editor.updateBlock(block, {
          props: { ...block.props, filters: JSON.stringify(newFilters) },
        })
      }, [block, editor])

      // Update sorts in props
      const updateSorts = useCallback((newSorts: SortItem[]) => {
        editor.updateBlock(block, {
          props: { ...block.props, sorts: JSON.stringify(newSorts) },
        })
      }, [block, editor])

      // Add a new filter
      const addFilter = useCallback((property: string) => {
        const newFilter: FilterItem = {
          id: Date.now().toString(),
          property,
          operator: 'contains',
          value: '',
        }
        updateFilters([...filters, newFilter])
      }, [filters, updateFilters])

      // Remove a filter
      const removeFilter = useCallback((id: string) => {
        updateFilters(filters.filter((f) => f.id !== id))
      }, [filters, updateFilters])

      // Update a filter
      const updateFilter = useCallback((id: string, updates: Partial<FilterItem>) => {
        updateFilters(filters.map((f) => (f.id === id ? { ...f, ...updates } : f)))
      }, [filters, updateFilters])

      // Add a sort
      const addSort = useCallback((property: string) => {
        const newSort: SortItem = {
          id: Date.now().toString(),
          property,
          direction: 'asc',
        }
        updateSorts([...sorts, newSort])
      }, [sorts, updateSorts])

      // Remove a sort
      const removeSort = useCallback((id: string) => {
        updateSorts(sorts.filter((s) => s.id !== id))
      }, [sorts, updateSorts])

      // Toggle sort direction
      const toggleSortDirection = useCallback((id: string) => {
        updateSorts(sorts.map((s) =>
          s.id === id ? { ...s, direction: s.direction === 'asc' ? 'desc' : 'asc' } : s
        ))
      }, [sorts, updateSorts])

      // Filter and sort records
      const processedRecords = useMemo(() => {
        if (!database?.records) return []
        let records = database.records
        records = applyFilters(records, filters)
        records = applySorts(records, sorts)
        return records
      }, [database?.records, filters, sorts])

      // Fetch database data
      const fetchDatabase = useCallback(async () => {
        if (!databaseId) {
          setIsLoading(false)
          return
        }

        setIsLoading(true)
        setError(null)

        try {
          const data = await api.get<DatabaseData>(`/databases/${databaseId}`)
          setDatabase(data)
        } catch (err) {
          console.error('Failed to fetch database:', err)
          setError('Unable to load database')
        } finally {
          setIsLoading(false)
        }
      }, [databaseId])

      // Fetch available databases for picker
      const fetchAvailableDatabases = useCallback(async () => {
        try {
          const response = await api.get<{ databases: Array<{ id: string; name: string; icon?: string }> }>(
            `/search?q=${encodeURIComponent(searchQuery)}&type=database&limit=10`
          )
          setAvailableDatabases(response.databases || [])
        } catch (err) {
          console.error('Failed to fetch databases:', err)
        }
      }, [searchQuery])

      useEffect(() => {
        fetchDatabase()
      }, [fetchDatabase])

      useEffect(() => {
        if (showDatabasePicker) {
          fetchAvailableDatabases()
        }
      }, [showDatabasePicker, fetchAvailableDatabases])

      // Update view
      const handleViewChange = useCallback((newViewId: string) => {
        editor.updateBlock(block, {
          props: { ...block.props, viewId: newViewId },
        })
        setShowViewPicker(false)
      }, [block, editor])

      // Select database
      const handleSelectDatabase = useCallback((newDbId: string) => {
        editor.updateBlock(block, {
          props: { ...block.props, databaseId: newDbId, viewId: '' },
        })
        setShowDatabasePicker(false)
      }, [block, editor])

      // Navigate to full database
      const handleOpenFull = useCallback(() => {
        if (databaseId) {
          window.location.href = `/databases/${databaseId}`
        }
      }, [databaseId])

      // Get current view
      const currentView = database?.views.find(v => v.id === viewId) || database?.views[0]

      // Empty state - no database selected
      if (!databaseId) {
        return (
          <div
            className="inline-database-block empty"
            style={{
              position: 'relative',
              margin: '8px 0',
            }}
          >
            <button
              onClick={() => setShowDatabasePicker(true)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                width: '100%',
                padding: '16px',
                background: 'var(--bg-secondary)',
                border: '1px dashed var(--border-color)',
                borderRadius: '8px',
                fontSize: '14px',
                color: 'var(--text-secondary)',
                cursor: 'pointer',
                transition: 'all 0.15s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.borderColor = 'var(--accent-color)'
                e.currentTarget.style.background = 'var(--bg-hover)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.borderColor = 'var(--border-color)'
                e.currentTarget.style.background = 'var(--bg-secondary)'
              }}
            >
              <Database size={20} style={{ color: 'var(--text-tertiary)' }} />
              <span>Select a database to embed</span>
            </button>

            {/* Database picker dropdown */}
            <AnimatePresence>
              {showDatabasePicker && (
                <motion.div
                  initial={{ opacity: 0, y: -8, scale: 0.95 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, y: -8, scale: 0.95 }}
                  transition={{ duration: 0.15 }}
                  style={{
                    position: 'absolute',
                    top: '100%',
                    left: 0,
                    right: 0,
                    marginTop: '4px',
                    background: 'var(--bg-primary)',
                    borderRadius: '8px',
                    boxShadow: '0 4px 24px rgba(0, 0, 0, 0.15)',
                    border: '1px solid var(--border-color)',
                    zIndex: 100,
                    overflow: 'hidden',
                  }}
                >
                  {/* Search input */}
                  <div style={{ padding: '8px', borderBottom: '1px solid var(--border-color)' }}>
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '8px',
                        padding: '8px 12px',
                        background: 'var(--bg-secondary)',
                        borderRadius: '6px',
                      }}
                    >
                      <Search size={14} style={{ color: 'var(--text-tertiary)' }} />
                      <input
                        type="text"
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        placeholder="Search databases..."
                        style={{
                          flex: 1,
                          border: 'none',
                          background: 'none',
                          fontSize: '14px',
                          color: 'var(--text-primary)',
                          outline: 'none',
                        }}
                        autoFocus
                      />
                    </div>
                  </div>

                  {/* Database list */}
                  <div style={{ maxHeight: '240px', overflowY: 'auto', padding: '4px' }}>
                    {availableDatabases.length === 0 ? (
                      <div
                        style={{
                          padding: '24px',
                          textAlign: 'center',
                          color: 'var(--text-tertiary)',
                          fontSize: '13px',
                        }}
                      >
                        No databases found
                      </div>
                    ) : (
                      availableDatabases.map((db) => (
                        <button
                          key={db.id}
                          onClick={() => handleSelectDatabase(db.id)}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: '10px',
                            width: '100%',
                            padding: '8px 12px',
                            border: 'none',
                            background: 'none',
                            borderRadius: '4px',
                            fontSize: '14px',
                            color: 'var(--text-primary)',
                            cursor: 'pointer',
                            textAlign: 'left',
                            transition: 'background 0.1s',
                          }}
                          onMouseEnter={(e) => {
                            e.currentTarget.style.background = 'var(--bg-hover)'
                          }}
                          onMouseLeave={(e) => {
                            e.currentTarget.style.background = 'none'
                          }}
                        >
                          <span style={{ fontSize: '16px' }}>{db.icon || '📊'}</span>
                          <span>{db.name}</span>
                        </button>
                      ))
                    )}
                  </div>

                  {/* Create new database option */}
                  <div
                    style={{
                      padding: '8px',
                      borderTop: '1px solid var(--border-color)',
                    }}
                  >
                    <button
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '8px',
                        width: '100%',
                        padding: '8px 12px',
                        border: 'none',
                        background: 'none',
                        borderRadius: '4px',
                        fontSize: '13px',
                        color: 'var(--accent-color)',
                        cursor: 'pointer',
                        textAlign: 'left',
                      }}
                    >
                      <Plus size={14} />
                      Create new database
                    </button>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        )
      }

      // Loading state
      if (isLoading) {
        return (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              padding: '32px',
              background: 'var(--bg-secondary)',
              borderRadius: '8px',
              margin: '8px 0',
            }}
          >
            <Loader2
              size={20}
              style={{ color: 'var(--accent-color)', animation: 'spin 1s linear infinite' }}
            />
            <span style={{ marginLeft: '12px', color: 'var(--text-secondary)', fontSize: '14px' }}>
              Loading database...
            </span>
          </div>
        )
      }

      // Error state
      if (error) {
        return (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
              padding: '16px',
              background: 'var(--danger-bg)',
              borderRadius: '8px',
              border: '1px solid var(--danger-color)',
              margin: '8px 0',
            }}
          >
            <Database size={20} style={{ color: 'var(--danger-color)' }} />
            <span style={{ color: 'var(--text-secondary)', fontSize: '14px' }}>{error}</span>
            <button
              onClick={fetchDatabase}
              style={{
                marginLeft: 'auto',
                padding: '6px 12px',
                background: 'var(--bg-primary)',
                border: '1px solid var(--border-color)',
                borderRadius: '4px',
                fontSize: '13px',
                cursor: 'pointer',
              }}
            >
              Retry
            </button>
          </div>
        )
      }

      if (!database) return null

      // Render database preview
      return (
        <div
          className="inline-database-block"
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => {
            setIsHovered(false)
            setShowViewPicker(false)
          }}
          style={{
            position: 'relative',
            margin: '12px 0',
            border: isHovered ? '1px solid var(--accent-color)' : '1px solid var(--border-color)',
            borderRadius: '8px',
            overflow: 'hidden',
            transition: 'border-color 0.15s',
          }}
        >
          {/* Header */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '12px 16px',
              background: 'var(--bg-secondary)',
              borderBottom: '1px solid var(--border-color)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              {showTitle && (
                <>
                  <span style={{ fontSize: '16px' }}>{database.icon || '📊'}</span>
                  <span style={{ fontSize: '14px', fontWeight: 500, color: 'var(--text-primary)' }}>
                    {database.name}
                  </span>
                </>
              )}

              {/* View switcher */}
              {database.views.length > 1 && (
                <div style={{ position: 'relative' }}>
                  <button
                    onClick={() => setShowViewPicker(!showViewPicker)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '4px',
                      padding: '4px 8px',
                      background: 'var(--bg-primary)',
                      border: '1px solid var(--border-color)',
                      borderRadius: '4px',
                      fontSize: '12px',
                      color: 'var(--text-secondary)',
                      cursor: 'pointer',
                    }}
                  >
                    {currentView && viewIcons[currentView.type]}
                    {currentView?.name || 'View'}
                    <ChevronDown size={12} />
                  </button>

                  {/* View picker dropdown */}
                  <AnimatePresence>
                    {showViewPicker && (
                      <motion.div
                        initial={{ opacity: 0, y: -4 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: -4 }}
                        style={{
                          position: 'absolute',
                          top: '100%',
                          left: 0,
                          marginTop: '4px',
                          background: 'var(--bg-primary)',
                          borderRadius: '6px',
                          boxShadow: '0 4px 12px rgba(0, 0, 0, 0.15)',
                          border: '1px solid var(--border-color)',
                          minWidth: '150px',
                          zIndex: 100,
                          overflow: 'hidden',
                        }}
                      >
                        {database.views.map((view) => (
                          <button
                            key={view.id}
                            onClick={() => handleViewChange(view.id)}
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: '8px',
                              width: '100%',
                              padding: '8px 12px',
                              border: 'none',
                              background: view.id === viewId ? 'var(--accent-bg)' : 'none',
                              color: view.id === viewId ? 'var(--accent-color)' : 'var(--text-primary)',
                              fontSize: '13px',
                              cursor: 'pointer',
                              textAlign: 'left',
                            }}
                          >
                            {viewIcons[view.type]}
                            {view.name}
                          </button>
                        ))}
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>
              )}
            </div>

            {/* Actions */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
              {/* Filter button with active indicator */}
              <div style={{ position: 'relative' }}>
                <button
                  onClick={() => {
                    setShowFilterMenu(!showFilterMenu)
                    setShowSortMenu(false)
                  }}
                  title="Filter"
                  style={{
                    padding: '4px 8px',
                    background: filters.length > 0 ? 'var(--accent-bg)' : 'none',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    color: filters.length > 0 ? 'var(--accent-color)' : 'var(--text-tertiary)',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    fontSize: '12px',
                  }}
                >
                  <Filter size={14} />
                  {filters.length > 0 && <span>{filters.length}</span>}
                </button>

                {/* Filter menu */}
                <AnimatePresence>
                  {showFilterMenu && (
                    <motion.div
                      initial={{ opacity: 0, y: -4 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, y: -4 }}
                      style={{
                        position: 'absolute',
                        top: '100%',
                        right: 0,
                        marginTop: '4px',
                        background: 'var(--bg-primary)',
                        borderRadius: '8px',
                        boxShadow: '0 4px 16px rgba(0, 0, 0, 0.15)',
                        border: '1px solid var(--border-color)',
                        minWidth: '300px',
                        zIndex: 100,
                        padding: '8px',
                      }}
                      onClick={(e) => e.stopPropagation()}
                    >
                      <div style={{ fontSize: '12px', fontWeight: 500, color: 'var(--text-tertiary)', marginBottom: '8px' }}>
                        Filters
                      </div>

                      {/* Existing filters */}
                      {filters.map((filter) => (
                        <div key={filter.id} style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                          <select
                            value={filter.property}
                            onChange={(e) => updateFilter(filter.id, { property: e.target.value })}
                            style={{ flex: 1, padding: '6px 8px', borderRadius: '4px', border: '1px solid var(--border-color)', fontSize: '12px', background: 'var(--bg-primary)' }}
                          >
                            {Object.entries(database?.properties || {}).map(([key, prop]) => (
                              <option key={key} value={key}>{prop.name}</option>
                            ))}
                          </select>
                          <select
                            value={filter.operator}
                            onChange={(e) => updateFilter(filter.id, { operator: e.target.value as FilterOperator })}
                            style={{ padding: '6px 8px', borderRadius: '4px', border: '1px solid var(--border-color)', fontSize: '12px', background: 'var(--bg-primary)' }}
                          >
                            <option value="contains">contains</option>
                            <option value="not_contains">does not contain</option>
                            <option value="equals">equals</option>
                            <option value="not_equals">does not equal</option>
                            <option value="is_empty">is empty</option>
                            <option value="is_not_empty">is not empty</option>
                          </select>
                          {!['is_empty', 'is_not_empty'].includes(filter.operator) && (
                            <input
                              type="text"
                              value={filter.value}
                              onChange={(e) => updateFilter(filter.id, { value: e.target.value })}
                              placeholder="Value..."
                              style={{ flex: 1, padding: '6px 8px', borderRadius: '4px', border: '1px solid var(--border-color)', fontSize: '12px', background: 'var(--bg-primary)' }}
                            />
                          )}
                          <button
                            onClick={() => removeFilter(filter.id)}
                            style={{ padding: '4px', background: 'none', border: 'none', color: 'var(--text-tertiary)', cursor: 'pointer' }}
                          >
                            <X size={14} />
                          </button>
                        </div>
                      ))}

                      {/* Add filter button */}
                      <div style={{ marginTop: '8px' }}>
                        <select
                          onChange={(e) => {
                            if (e.target.value) {
                              addFilter(e.target.value)
                              e.target.value = ''
                            }
                          }}
                          value=""
                          style={{ width: '100%', padding: '6px 8px', borderRadius: '4px', border: '1px dashed var(--border-color)', fontSize: '12px', background: 'var(--bg-secondary)', color: 'var(--text-tertiary)', cursor: 'pointer' }}
                        >
                          <option value="">+ Add filter...</option>
                          {Object.entries(database?.properties || {}).map(([key, prop]) => (
                            <option key={key} value={key}>{prop.name}</option>
                          ))}
                        </select>
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>

              {/* Sort button with active indicator */}
              <div style={{ position: 'relative' }}>
                <button
                  onClick={() => {
                    setShowSortMenu(!showSortMenu)
                    setShowFilterMenu(false)
                  }}
                  title="Sort"
                  style={{
                    padding: '4px 8px',
                    background: sorts.length > 0 ? 'var(--accent-bg)' : 'none',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    color: sorts.length > 0 ? 'var(--accent-color)' : 'var(--text-tertiary)',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    fontSize: '12px',
                  }}
                >
                  <ArrowUpDown size={14} />
                  {sorts.length > 0 && <span>{sorts.length}</span>}
                </button>

                {/* Sort menu */}
                <AnimatePresence>
                  {showSortMenu && (
                    <motion.div
                      initial={{ opacity: 0, y: -4 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, y: -4 }}
                      style={{
                        position: 'absolute',
                        top: '100%',
                        right: 0,
                        marginTop: '4px',
                        background: 'var(--bg-primary)',
                        borderRadius: '8px',
                        boxShadow: '0 4px 16px rgba(0, 0, 0, 0.15)',
                        border: '1px solid var(--border-color)',
                        minWidth: '250px',
                        zIndex: 100,
                        padding: '8px',
                      }}
                      onClick={(e) => e.stopPropagation()}
                    >
                      <div style={{ fontSize: '12px', fontWeight: 500, color: 'var(--text-tertiary)', marginBottom: '8px' }}>
                        Sort
                      </div>

                      {/* Existing sorts */}
                      {sorts.map((sort) => (
                        <div key={sort.id} style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                          <select
                            value={sort.property}
                            onChange={(e) => updateSorts(sorts.map((s) => s.id === sort.id ? { ...s, property: e.target.value } : s))}
                            style={{ flex: 1, padding: '6px 8px', borderRadius: '4px', border: '1px solid var(--border-color)', fontSize: '12px', background: 'var(--bg-primary)' }}
                          >
                            {Object.entries(database?.properties || {}).map(([key, prop]) => (
                              <option key={key} value={key}>{prop.name}</option>
                            ))}
                          </select>
                          <button
                            onClick={() => toggleSortDirection(sort.id)}
                            style={{ padding: '6px 8px', borderRadius: '4px', border: '1px solid var(--border-color)', background: 'var(--bg-primary)', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '4px', fontSize: '12px' }}
                          >
                            {sort.direction === 'asc' ? <ArrowUp size={12} /> : <ArrowDown size={12} />}
                            {sort.direction === 'asc' ? 'Asc' : 'Desc'}
                          </button>
                          <button
                            onClick={() => removeSort(sort.id)}
                            style={{ padding: '4px', background: 'none', border: 'none', color: 'var(--text-tertiary)', cursor: 'pointer' }}
                          >
                            <X size={14} />
                          </button>
                        </div>
                      ))}

                      {/* Add sort button */}
                      <div style={{ marginTop: '8px' }}>
                        <select
                          onChange={(e) => {
                            if (e.target.value) {
                              addSort(e.target.value)
                              e.target.value = ''
                            }
                          }}
                          value=""
                          style={{ width: '100%', padding: '6px 8px', borderRadius: '4px', border: '1px dashed var(--border-color)', fontSize: '12px', background: 'var(--bg-secondary)', color: 'var(--text-tertiary)', cursor: 'pointer' }}
                        >
                          <option value="">+ Add sort...</option>
                          {Object.entries(database?.properties || {}).map(([key, prop]) => (
                            <option key={key} value={key}>{prop.name}</option>
                          ))}
                        </select>
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>

              <button
                onClick={handleOpenFull}
                title="Open full database"
                style={{
                  padding: '4px',
                  background: 'none',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  color: 'var(--text-tertiary)',
                }}
              >
                <ExternalLink size={14} />
              </button>
            </div>
          </div>

          {/* Content - Simple table preview */}
          <div style={{ padding: '8px 0', maxHeight: '300px', overflowY: 'auto' }}>
            {processedRecords.length === 0 ? (
              <div
                style={{
                  padding: '32px',
                  textAlign: 'center',
                  color: 'var(--text-tertiary)',
                  fontSize: '13px',
                }}
              >
                <Database size={24} style={{ marginBottom: '8px', opacity: 0.5 }} />
                <p>{filters.length > 0 ? 'No matching records' : 'No records in this database'}</p>
              </div>
            ) : (
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                <thead>
                  <tr>
                    {Object.entries(database.properties).slice(0, 4).map(([key, prop]) => (
                      <th
                        key={key}
                        style={{
                          padding: '8px 16px',
                          textAlign: 'left',
                          fontWeight: 500,
                          color: 'var(--text-secondary)',
                          borderBottom: '1px solid var(--border-color)',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {prop.name}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {processedRecords.slice(0, maxRows).map((record) => (
                    <tr
                      key={record.id}
                      style={{ cursor: 'pointer' }}
                      onMouseEnter={(e) => {
                        e.currentTarget.style.background = 'var(--bg-hover)'
                      }}
                      onMouseLeave={(e) => {
                        e.currentTarget.style.background = ''
                      }}
                    >
                      {Object.keys(database.properties).slice(0, 4).map((key) => (
                        <td
                          key={key}
                          style={{
                            padding: '8px 16px',
                            color: 'var(--text-primary)',
                            borderBottom: '1px solid var(--border-color)',
                            maxWidth: '200px',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {renderPropertyValue(record.properties[key])}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            )}

            {/* Show more indicator */}
            {processedRecords.length > maxRows && (
              <div
                style={{
                  padding: '8px 16px',
                  fontSize: '12px',
                  color: 'var(--text-tertiary)',
                  borderTop: '1px solid var(--border-color)',
                }}
              >
                + {processedRecords.length - maxRows} more records
                {filters.length > 0 && ` (${database.records.length} total)`}
              </div>
            )}
          </div>

          {/* CSS for loading animation */}
          <style>{`
            @keyframes spin {
              from { transform: rotate(0deg); }
              to { transform: rotate(360deg); }
            }
          `}</style>
        </div>
      )
    },
    // Parse HTML to recreate block when pasting or drag-dropping
    parse: (element: HTMLElement) => {
      if (element.classList.contains('inline-database-block') || element.hasAttribute('data-database-id')) {
        return {
          databaseId: element.getAttribute('data-database-id') || '',
          viewId: element.getAttribute('data-view-id') || '',
          showTitle: element.getAttribute('data-show-title') !== 'false',
          maxRows: parseInt(element.getAttribute('data-max-rows') || '5', 10),
          filters: element.getAttribute('data-filters') || '[]',
          sorts: element.getAttribute('data-sorts') || '[]',
        }
      }
      return undefined
    },
    // Convert to external HTML for clipboard/export
    toExternalHTML: ({ block }) => {
      const { databaseId, viewId, showTitle, maxRows, filters, sorts } = block.props as Record<string, unknown>
      return (
        <div
          className="inline-database-block"
          data-database-id={databaseId as string}
          data-view-id={viewId as string}
          data-show-title={String(showTitle)}
          data-max-rows={String(maxRows)}
          data-filters={filters as string}
          data-sorts={sorts as string}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            padding: '12px 16px',
            border: '1px solid rgba(55, 53, 47, 0.16)',
            borderRadius: '8px',
          }}
        >
          <span style={{ fontSize: '20px' }}>📊</span>
          <span>Linked Database</span>
        </div>
      )
    },
  }
)

// Helper to render property values
function renderPropertyValue(value: unknown): React.ReactNode {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'string') return value
  if (typeof value === 'number') return value.toString()
  if (typeof value === 'boolean') return value ? '✓' : '✗'
  if (Array.isArray(value)) {
    return value.map((v, i) => (
      <span
        key={i}
        style={{
          display: 'inline-block',
          padding: '2px 6px',
          background: 'var(--tag-default)',
          borderRadius: '4px',
          marginRight: '4px',
          fontSize: '12px',
        }}
      >
        {String(v)}
      </span>
    ))
  }
  if (typeof value === 'object') {
    // Handle specific property types
    const obj = value as Record<string, unknown>
    if (obj.name) return obj.name as string
    if (obj.title) return obj.title as string
    if (obj.email) return obj.email as string
    return JSON.stringify(value)
  }
  return String(value)
}
