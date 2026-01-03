import { useState, useCallback, useEffect, useMemo } from 'react'
import { TableView } from './views/TableView'
import { BoardView } from './views/BoardView'
import { ListView } from './views/ListView'
import { CalendarView } from './views/CalendarView'
import { GalleryView } from './views/GalleryView'
import { ChartView } from './views/ChartView'
import { TimelineView } from './views/TimelineView'
import { FilterPanel } from './FilterPanel'
import { SortPanel } from './SortPanel'
import { api, DatabaseRow, Property, Filter, Sort, View } from '../api/client'
import { useDatabaseStore } from '../stores/databaseStore'
import { motion, AnimatePresence } from 'framer-motion'

export type ViewType = 'table' | 'board' | 'list' | 'calendar' | 'gallery' | 'timeline' | 'chart'

interface DatabaseViewProps {
  databaseId: string
  viewId?: string
  viewType: ViewType
  initialData: {
    rows: DatabaseRow[]
    properties: Property[]
    views?: View[]
  }
}

export function DatabaseView({ databaseId, viewId: initialViewId, viewType: initialViewType, initialData }: DatabaseViewProps) {
  const {
    views: storedViews,
    fetchViews,
    createView,
    updateView,
    setActiveView,
    activeViewId,
  } = useDatabaseStore()

  const [viewType, setViewType] = useState<ViewType>(initialViewType)
  const [rows, setRows] = useState<DatabaseRow[]>(initialData.rows)
  const [properties, setProperties] = useState<Property[]>(initialData.properties)
  const [filters, setFilters] = useState<Filter[]>([])
  const [sorts, setSorts] = useState<Sort[]>([])
  const [groupBy, setGroupBy] = useState<string | null>(null)
  const [showFilterPanel, setShowFilterPanel] = useState(false)
  const [showSortPanel, setShowSortPanel] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [currentViewId, setCurrentViewId] = useState<string | undefined>(initialViewId)

  // Get views for this database
  const views = storedViews[databaseId] || initialData.views || []

  // Load views on mount
  useEffect(() => {
    fetchViews(databaseId)
  }, [databaseId, fetchViews])

  // Find current view
  const currentView = useMemo(() => {
    if (currentViewId) {
      return views.find(v => v.id === currentViewId)
    }
    return views.find(v => v.type === viewType) || views[0]
  }, [views, currentViewId, viewType])

  // Apply view settings when view changes
  useEffect(() => {
    if (currentView) {
      setViewType(currentView.type as ViewType)
      if (currentView.config) {
        if (currentView.config.filters) setFilters(currentView.config.filters)
        if (currentView.config.sorts) setSorts(currentView.config.sorts)
        if (currentView.config.groupBy) setGroupBy(currentView.config.groupBy)
      }
    }
  }, [currentView])

  // Handle view type change with persistence
  const handleViewTypeChange = useCallback(async (newType: ViewType) => {
    setViewType(newType)

    // Find existing view of this type or create one
    const existingView = views.find(v => v.type === newType)

    if (existingView) {
      setCurrentViewId(existingView.id)
      setActiveView(existingView.id)
    } else {
      // Create a new view
      const newView = await createView(databaseId, {
        name: `${newType.charAt(0).toUpperCase() + newType.slice(1)} view`,
        type: newType,
        config: {},
      })
      if (newView) {
        setCurrentViewId(newView.id)
        setActiveView(newView.id)
      }
    }
  }, [views, databaseId, createView, setActiveView])

  // Save view config when filters/sorts change
  const saveViewConfig = useCallback(async () => {
    if (!currentViewId) return

    try {
      await updateView(currentViewId, {
        config: {
          filters,
          sorts,
          groupBy,
        }
      })
    } catch (err) {
      console.error('Failed to save view config:', err)
    }
  }, [currentViewId, filters, sorts, groupBy, updateView])

  // Debounced save on filter/sort change
  useEffect(() => {
    const timeout = setTimeout(saveViewConfig, 1000)
    return () => clearTimeout(timeout)
  }, [filters, sorts, groupBy, saveViewConfig])

  // Fetch rows when filters/sorts change
  const fetchRows = useCallback(async () => {
    setIsLoading(true)
    try {
      const params = new URLSearchParams()
      if (filters.length) params.set('filters', JSON.stringify(filters))
      if (sorts.length) params.set('sorts', JSON.stringify(sorts))

      const data = await api.get<{ rows: DatabaseRow[] }>(
        `/databases/${databaseId}/rows?${params.toString()}`
      )
      setRows(data.rows)
    } catch (err) {
      console.error('Failed to fetch rows:', err)
    } finally {
      setIsLoading(false)
    }
  }, [databaseId, filters, sorts])

  useEffect(() => {
    if (filters.length || sorts.length) {
      fetchRows()
    }
  }, [fetchRows, filters, sorts])

  // Add new row with optional initial properties
  const handleAddRow = useCallback(async (initialProperties?: Record<string, unknown>) => {
    try {
      const newRow = await api.post<DatabaseRow>(`/databases/${databaseId}/rows`, {
        properties: initialProperties || {},
      })
      setRows((prev) => [...prev, newRow])
      return newRow
    } catch (err) {
      console.error('Failed to add row:', err)
      return null
    }
  }, [databaseId])

  // Update row
  const handleUpdateRow = useCallback(async (rowId: string, updates: Record<string, unknown>) => {
    try {
      await api.patch(`/rows/${rowId}`, { properties: updates })
      setRows((prev) =>
        prev.map((row) =>
          row.id === rowId ? { ...row, properties: { ...row.properties, ...updates } } : row
        )
      )
    } catch (err) {
      console.error('Failed to update row:', err)
    }
  }, [])

  // Delete row
  const handleDeleteRow = useCallback(async (rowId: string) => {
    try {
      await api.delete(`/rows/${rowId}`)
      setRows((prev) => prev.filter((row) => row.id !== rowId))
    } catch (err) {
      console.error('Failed to delete row:', err)
    }
  }, [])

  // Add property
  const handleAddProperty = useCallback(async (property: Omit<Property, 'id'>) => {
    try {
      const newProp = await api.post<Property>(`/databases/${databaseId}/properties`, property)
      setProperties((prev) => [...prev, newProp])
    } catch (err) {
      console.error('Failed to add property:', err)
    }
  }, [databaseId])

  // Update property
  const handleUpdateProperty = useCallback(async (propertyId: string, updates: Partial<Property>) => {
    try {
      await api.patch(`/properties/${propertyId}`, updates)
      setProperties((prev) =>
        prev.map((prop) => (prop.id === propertyId ? { ...prop, ...updates } : prop))
      )
    } catch (err) {
      console.error('Failed to update property:', err)
    }
  }, [])

  // Delete property
  const handleDeleteProperty = useCallback(async (propertyId: string) => {
    try {
      await api.delete(`/properties/${propertyId}`)
      setProperties((prev) => prev.filter((prop) => prop.id !== propertyId))
    } catch (err) {
      console.error('Failed to delete property:', err)
    }
  }, [])

  // Render the appropriate view
  const renderView = () => {
    const commonProps = {
      rows,
      properties,
      groupBy,
      onAddRow: handleAddRow,
      onUpdateRow: handleUpdateRow,
      onDeleteRow: handleDeleteRow,
      onAddProperty: handleAddProperty,
      onUpdateProperty: handleUpdateProperty,
      onDeleteProperty: handleDeleteProperty,
    }

    switch (viewType) {
      case 'table':
        return <TableView {...commonProps} />
      case 'board':
        return <BoardView {...commonProps} />
      case 'list':
        return <ListView {...commonProps} />
      case 'calendar':
        return <CalendarView {...commonProps} />
      case 'gallery':
        return <GalleryView {...commonProps} />
      case 'timeline':
        return <TimelineView {...commonProps} />
      case 'chart':
        return <ChartView {...commonProps} />
      default:
        return <TableView {...commonProps} />
    }
  }

  return (
    <div className="database-view-container">
      {/* View type tabs */}
      <div className="database-view-tabs">
        {(['table', 'board', 'list', 'calendar', 'gallery', 'timeline', 'chart'] as ViewType[]).map((type) => (
          <button
            key={type}
            className={`view-tab ${viewType === type ? 'active' : ''}`}
            onClick={() => handleViewTypeChange(type)}
          >
            <ViewIcon type={type} />
            <span>{type.charAt(0).toUpperCase() + type.slice(1)}</span>
          </button>
        ))}
      </div>

      {/* Toolbar */}
      <div className="database-toolbar">
        <button
          className={`toolbar-btn ${showFilterPanel ? 'active' : ''}`}
          onClick={() => setShowFilterPanel(!showFilterPanel)}
        >
          <FilterIcon />
          <span>Filter</span>
          {filters.length > 0 && <span className="badge">{filters.length}</span>}
        </button>
        <button
          className={`toolbar-btn ${showSortPanel ? 'active' : ''}`}
          onClick={() => setShowSortPanel(!showSortPanel)}
        >
          <SortIcon />
          <span>Sort</span>
          {sorts.length > 0 && <span className="badge">{sorts.length}</span>}
        </button>
        {(viewType === 'board' || viewType === 'list') && (
          <select
            className="group-select"
            value={groupBy || ''}
            onChange={(e) => setGroupBy(e.target.value || null)}
          >
            <option value="">No grouping</option>
            {properties
              .filter((p) => p.type === 'select' || p.type === 'status')
              .map((prop) => (
                <option key={prop.id} value={prop.id}>
                  Group by {prop.name}
                </option>
              ))}
          </select>
        )}
      </div>

      {/* Filter panel */}
      {showFilterPanel && (
        <FilterPanel
          properties={properties}
          filters={filters}
          onFiltersChange={setFilters}
          onClose={() => setShowFilterPanel(false)}
        />
      )}

      {/* Sort panel */}
      {showSortPanel && (
        <SortPanel
          properties={properties}
          sorts={sorts}
          onSortsChange={setSorts}
          onClose={() => setShowSortPanel(false)}
        />
      )}

      {/* Loading overlay */}
      {isLoading && <div className="loading-overlay">Loading...</div>}

      {/* View content */}
      <div className="database-content">{renderView()}</div>
    </div>
  )
}

// View type icons
function ViewIcon({ type }: { type: ViewType }) {
  switch (type) {
    case 'table':
      return (
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <rect x="1" y="1" width="12" height="12" rx="1" stroke="currentColor" fill="none" />
          <line x1="1" y1="5" x2="13" y2="5" stroke="currentColor" />
          <line x1="5" y1="5" x2="5" y2="13" stroke="currentColor" />
        </svg>
      )
    case 'board':
      return (
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <rect x="1" y="1" width="3.5" height="12" rx="0.5" stroke="currentColor" fill="none" />
          <rect x="5.25" y="1" width="3.5" height="8" rx="0.5" stroke="currentColor" fill="none" />
          <rect x="9.5" y="1" width="3.5" height="10" rx="0.5" stroke="currentColor" fill="none" />
        </svg>
      )
    case 'list':
      return (
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <line x1="1" y1="3" x2="13" y2="3" stroke="currentColor" strokeWidth="2" />
          <line x1="1" y1="7" x2="13" y2="7" stroke="currentColor" strokeWidth="2" />
          <line x1="1" y1="11" x2="13" y2="11" stroke="currentColor" strokeWidth="2" />
        </svg>
      )
    case 'calendar':
      return (
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <rect x="1" y="2" width="12" height="11" rx="1" stroke="currentColor" fill="none" />
          <line x1="1" y1="5" x2="13" y2="5" stroke="currentColor" />
          <line x1="4" y1="1" x2="4" y2="3" stroke="currentColor" />
          <line x1="10" y1="1" x2="10" y2="3" stroke="currentColor" />
        </svg>
      )
    case 'gallery':
      return (
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <rect x="1" y="1" width="5" height="5" rx="0.5" stroke="currentColor" fill="none" />
          <rect x="8" y="1" width="5" height="5" rx="0.5" stroke="currentColor" fill="none" />
          <rect x="1" y="8" width="5" height="5" rx="0.5" stroke="currentColor" fill="none" />
          <rect x="8" y="8" width="5" height="5" rx="0.5" stroke="currentColor" fill="none" />
        </svg>
      )
    case 'timeline':
      return (
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <line x1="1" y1="7" x2="13" y2="7" stroke="currentColor" strokeWidth="1.5" />
          <rect x="2" y="5" width="4" height="4" rx="0.5" stroke="currentColor" fill="none" />
          <rect x="7" y="5" width="5" height="4" rx="0.5" stroke="currentColor" fill="none" />
        </svg>
      )
    case 'chart':
      return (
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <rect x="1" y="8" width="2" height="5" rx="0.5" stroke="currentColor" fill="none" />
          <rect x="4.5" y="5" width="2" height="8" rx="0.5" stroke="currentColor" fill="none" />
          <rect x="8" y="2" width="2" height="11" rx="0.5" stroke="currentColor" fill="none" />
          <rect x="11.5" y="6" width="2" height="7" rx="0.5" stroke="currentColor" fill="none" />
        </svg>
      )
  }
}

function FilterIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
      <path d="M2 3h10M4 7h6M6 11h2" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
}

function SortIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
      <path d="M4 2v10M4 2L2 4M4 2l2 2M10 12V2M10 12l-2-2M10 12l2-2" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}
