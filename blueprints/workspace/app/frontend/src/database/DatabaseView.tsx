import { useState, useCallback, useEffect, useMemo, useRef } from 'react'
import { TableView } from './views/TableView'
import { BoardView } from './views/BoardView'
import { ListView } from './views/ListView'
import { CalendarView } from './views/CalendarView'
import { GalleryView } from './views/GalleryView'
import { ChartView } from './views/ChartView'
import { TimelineView } from './views/TimelineView'
import { FilterPanel } from './FilterPanel'
import { SortPanel } from './SortPanel'
import { api, Database, DatabaseRow, Property, Filter, Sort, View } from '../api/client'
import { useDatabaseStore } from '../stores/databaseStore'
import { motion, AnimatePresence } from 'framer-motion'
import { ConfirmDialog, AlertDialog } from '../components/ConfirmDialog'
import toast from 'react-hot-toast'
import {
  Plus,
  MoreHorizontal,
  Edit2,
  Copy,
  Trash2,
} from 'lucide-react'

export type ViewType = 'table' | 'board' | 'list' | 'calendar' | 'gallery' | 'timeline' | 'chart'

interface DatabaseViewProps {
  databaseId: string
  viewId?: string
  viewType: ViewType
  initialData: {
    database?: Database
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
    deleteView,
    setActiveView,
    activeViewId,
  } = useDatabaseStore()

  const [viewType, setViewType] = useState<ViewType>(initialViewType)
  const [rows, setRows] = useState<DatabaseRow[]>(initialData.rows)
  const [properties, setProperties] = useState<Property[]>(initialData.properties)
  const [filters, setFilters] = useState<Filter[]>([])
  const [sorts, setSorts] = useState<Sort[]>([])
  const [groupBy, setGroupBy] = useState<string | null>(null)
  const [hiddenProperties, setHiddenProperties] = useState<string[]>([])
  const [showFilterPanel, setShowFilterPanel] = useState(false)
  const [showSortPanel, setShowSortPanel] = useState(false)
  const [showViewMenu, setShowViewMenu] = useState<string | null>(null)
  const [showAddViewMenu, setShowAddViewMenu] = useState(false)
  const [editingViewName, setEditingViewName] = useState<string | null>(null)
  const [newViewName, setNewViewName] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [currentViewId, setCurrentViewId] = useState<string | undefined>(initialViewId)
  const [searchQuery, setSearchQuery] = useState('')
  const viewMenuRef = useRef<HTMLDivElement>(null)
  const addViewMenuRef = useRef<HTMLDivElement>(null)
  const [viewMenuPosition, setViewMenuPosition] = useState<{ x: number; y: number } | null>(null)
  const [addViewMenuPosition, setAddViewMenuPosition] = useState<{ x: number; y: number } | null>(null)

  // Dialog states
  const [deleteViewDialog, setDeleteViewDialog] = useState<{ isOpen: boolean; viewId: string | null }>({
    isOpen: false,
    viewId: null,
  })
  const [alertDialog, setAlertDialog] = useState<{ isOpen: boolean; title: string; message: string }>({
    isOpen: false,
    title: '',
    message: '',
  })

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
        if (currentView.config.hiddenProperties) setHiddenProperties(currentView.config.hiddenProperties)
      }
    }
  }, [currentView])

  // Close menus on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (viewMenuRef.current && !viewMenuRef.current.contains(e.target as Node)) {
        setShowViewMenu(null)
        setViewMenuPosition(null)
      }
      if (addViewMenuRef.current && !addViewMenuRef.current.contains(e.target as Node)) {
        setShowAddViewMenu(false)
        setAddViewMenuPosition(null)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Handle view selection
  const handleSelectView = useCallback((view: View) => {
    setCurrentViewId(view.id)
    setActiveView(view.id)
  }, [setActiveView])

  // Handle view type change with persistence
  const handleAddView = useCallback(async (type: ViewType) => {
    const existingCount = views.filter(v => v.type === type).length
    const viewName = existingCount > 0
      ? `${type.charAt(0).toUpperCase() + type.slice(1)} view ${existingCount + 1}`
      : `${type.charAt(0).toUpperCase() + type.slice(1)} view`

    const newView = await createView(databaseId, {
      name: viewName,
      type,
      config: {},
    })

    if (newView) {
      setCurrentViewId(newView.id)
      setActiveView(newView.id)
    }

    setShowAddViewMenu(false)
  }, [views, databaseId, createView, setActiveView])

  // Handle rename view
  const handleRenameView = useCallback(async (viewId: string, name: string) => {
    if (!name.trim()) {
      setEditingViewName(null)
      return
    }

    await updateView(viewId, { name: name.trim() })
    setEditingViewName(null)
    setNewViewName('')
    setShowViewMenu(null)
  }, [updateView])

  // Handle duplicate view
  const handleDuplicateView = useCallback(async (view: View) => {
    const newView = await createView(databaseId, {
      name: `${view.name} (copy)`,
      type: view.type,
      config: { ...view.config },
    })

    if (newView) {
      setCurrentViewId(newView.id)
      setActiveView(newView.id)
    }

    setShowViewMenu(null)
  }, [databaseId, createView, setActiveView])

  // Handle delete view - open confirmation dialog
  const handleDeleteView = useCallback((viewId: string) => {
    if (views.length <= 1) {
      setAlertDialog({
        isOpen: true,
        title: 'Cannot delete view',
        message: 'This is the only view in this database. Create another view before deleting this one.',
      })
      return
    }

    setDeleteViewDialog({ isOpen: true, viewId })
    setShowViewMenu(null)
  }, [views.length])

  // Confirm delete view
  const handleConfirmDeleteView = useCallback(async () => {
    if (!deleteViewDialog.viewId) return

    try {
      await deleteView(deleteViewDialog.viewId)

      // Switch to first remaining view
      const remaining = views.find(v => v.id !== deleteViewDialog.viewId)
      if (remaining) {
        setCurrentViewId(remaining.id)
        setActiveView(remaining.id)
      }

      toast.success('View deleted')
    } catch (err) {
      console.error('Failed to delete view:', err)
      toast.error('Failed to delete view')
    }
  }, [deleteViewDialog.viewId, views, deleteView, setActiveView])

  // Save view config when filters/sorts change
  const saveViewConfig = useCallback(async () => {
    if (!currentViewId) return

    try {
      await updateView(currentViewId, {
        config: {
          filters,
          sorts,
          groupBy,
          hiddenProperties,
        }
      })
    } catch (err) {
      console.error('Failed to save view config:', err)
    }
  }, [currentViewId, filters, sorts, groupBy, hiddenProperties, updateView])

  // Debounced save on filter/sort change
  useEffect(() => {
    const timeout = setTimeout(saveViewConfig, 1000)
    return () => clearTimeout(timeout)
  }, [filters, sorts, groupBy, hiddenProperties, saveViewConfig])

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
      database: initialData.database,
      hiddenProperties,
      onAddRow: handleAddRow,
      onUpdateRow: handleUpdateRow,
      onDeleteRow: handleDeleteRow,
      onAddProperty: handleAddProperty,
      onUpdateProperty: handleUpdateProperty,
      onDeleteProperty: handleDeleteProperty,
      onHiddenPropertiesChange: setHiddenProperties,
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
      {/* View tabs */}
      <div className="database-view-tabs" style={{
        display: 'flex',
        alignItems: 'center',
        gap: 4,
        padding: '8px 0',
        borderBottom: '1px solid var(--border-color)',
        overflowX: 'auto',
      }}>
        {views.map((view) => (
          <div
            key={view.id}
            className={`view-tab ${currentView?.id === view.id ? 'active' : ''}`}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              padding: '6px 12px',
              background: currentView?.id === view.id ? 'var(--bg-secondary)' : 'transparent',
              borderRadius: 'var(--radius-md)',
              cursor: 'pointer',
              position: 'relative',
            }}
          >
            {editingViewName === view.id ? (
              <input
                type="text"
                value={newViewName}
                onChange={(e) => setNewViewName(e.target.value)}
                onBlur={() => handleRenameView(view.id, newViewName)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleRenameView(view.id, newViewName)
                  if (e.key === 'Escape') {
                    setEditingViewName(null)
                    setNewViewName('')
                  }
                }}
                autoFocus
                onClick={(e) => e.stopPropagation()}
                style={{
                  padding: '2px 6px',
                  border: '1px solid var(--accent-color)',
                  borderRadius: 'var(--radius-sm)',
                  fontSize: 13,
                  outline: 'none',
                  width: 120,
                }}
              />
            ) : (
              <>
                <div
                  onClick={() => handleSelectView(view)}
                  style={{ display: 'flex', alignItems: 'center', gap: 6 }}
                >
                  <ViewIcon type={view.type as ViewType} />
                  <span style={{ fontSize: 13, fontWeight: currentView?.id === view.id ? 500 : 400 }}>
                    {view.name}
                  </span>
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    if (showViewMenu === view.id) {
                      setShowViewMenu(null)
                      setViewMenuPosition(null)
                    } else {
                      const rect = e.currentTarget.getBoundingClientRect()
                      setViewMenuPosition({ x: rect.left, y: rect.bottom + 4 })
                      setShowViewMenu(view.id)
                    }
                  }}
                  style={{
                    width: 20,
                    height: 20,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    color: 'var(--text-tertiary)',
                    borderRadius: 'var(--radius-sm)',
                    opacity: currentView?.id === view.id ? 1 : 0,
                    transition: 'opacity 0.15s',
                  }}
                  className="view-menu-btn"
                >
                  <MoreHorizontal size={14} />
                </button>
              </>
            )}

            {/* View menu - rendered via portal below */}
            {showViewMenu === view.id && viewMenuPosition && (
              <div
                ref={viewMenuRef}
                style={{
                  position: 'fixed',
                  top: viewMenuPosition.y,
                  left: viewMenuPosition.x,
                  background: 'var(--bg-primary)',
                  border: '1px solid var(--border-color)',
                  borderRadius: 'var(--radius-md)',
                  boxShadow: 'var(--shadow-lg)',
                  minWidth: 180,
                  zIndex: 10000,
                }}
              >
                <button
                  onClick={() => {
                    setEditingViewName(view.id)
                    setNewViewName(view.name)
                    setShowViewMenu(null)
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
                  onClick={() => handleDuplicateView(view)}
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
                  <Copy size={14} />
                  Duplicate
                </button>
                <button
                  onClick={() => handleDeleteView(view.id)}
                  disabled={views.length <= 1}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    width: '100%',
                    padding: '8px 12px',
                    background: 'none',
                    border: 'none',
                    borderTop: '1px solid var(--border-color)',
                    cursor: views.length <= 1 ? 'not-allowed' : 'pointer',
                    fontSize: 13,
                    textAlign: 'left',
                    color: views.length <= 1 ? 'var(--text-tertiary)' : 'var(--error-color)',
                    opacity: views.length <= 1 ? 0.5 : 1,
                  }}
                >
                  <Trash2 size={14} />
                  Delete
                </button>
              </div>
            )}
          </div>
        ))}

        {/* Add view button */}
        <div ref={addViewMenuRef} style={{ position: 'relative' }}>
          <button
            onClick={(e) => {
              if (showAddViewMenu) {
                setShowAddViewMenu(false)
                setAddViewMenuPosition(null)
              } else {
                const rect = e.currentTarget.getBoundingClientRect()
                setAddViewMenuPosition({ x: rect.left, y: rect.bottom + 4 })
                setShowAddViewMenu(true)
              }
            }}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 4,
              padding: '6px 12px',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              color: 'var(--text-tertiary)',
              borderRadius: 'var(--radius-md)',
              fontSize: 13,
            }}
          >
            <Plus size={14} />
            Add view
          </button>
          {showAddViewMenu && addViewMenuPosition && (
            <div style={{
              position: 'fixed',
              top: addViewMenuPosition.y,
              left: addViewMenuPosition.x,
              background: 'var(--bg-primary)',
              border: '1px solid var(--border-color)',
              borderRadius: 'var(--radius-md)',
              boxShadow: 'var(--shadow-lg)',
              minWidth: 180,
              zIndex: 10000,
            }}>
              {(['table', 'board', 'list', 'calendar', 'gallery', 'timeline', 'chart'] as ViewType[]).map((type) => (
                <button
                  key={type}
                  onClick={() => handleAddView(type)}
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
                  <ViewIcon type={type} />
                  <span>{type.charAt(0).toUpperCase() + type.slice(1)}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Toolbar */}
      <div className="database-toolbar" style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        padding: '8px 0',
      }}>
        <button
          className={`toolbar-btn ${showFilterPanel ? 'active' : ''}`}
          onClick={() => setShowFilterPanel(!showFilterPanel)}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            padding: '6px 12px',
            background: showFilterPanel ? 'var(--accent-bg)' : 'none',
            border: '1px solid var(--border-color)',
            borderRadius: 'var(--radius-md)',
            cursor: 'pointer',
            fontSize: 13,
          }}
        >
          <FilterIcon />
          <span>Filter</span>
          {filters.length > 0 && (
            <span style={{
              padding: '2px 6px',
              background: 'var(--accent-color)',
              color: 'white',
              borderRadius: 'var(--radius-sm)',
              fontSize: 11,
              fontWeight: 600,
            }}>
              {filters.length}
            </span>
          )}
        </button>
        <button
          className={`toolbar-btn ${showSortPanel ? 'active' : ''}`}
          onClick={() => setShowSortPanel(!showSortPanel)}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            padding: '6px 12px',
            background: showSortPanel ? 'var(--accent-bg)' : 'none',
            border: '1px solid var(--border-color)',
            borderRadius: 'var(--radius-md)',
            cursor: 'pointer',
            fontSize: 13,
          }}
        >
          <SortIcon />
          <span>Sort</span>
          {sorts.length > 0 && (
            <span style={{
              padding: '2px 6px',
              background: 'var(--accent-color)',
              color: 'white',
              borderRadius: 'var(--radius-sm)',
              fontSize: 11,
              fontWeight: 600,
            }}>
              {sorts.length}
            </span>
          )}
        </button>
        {(viewType === 'board' || viewType === 'list') && (
          <select
            className="group-select"
            value={groupBy || ''}
            onChange={(e) => setGroupBy(e.target.value || null)}
            style={{
              padding: '6px 12px',
              border: '1px solid var(--border-color)',
              borderRadius: 'var(--radius-md)',
              fontSize: 13,
              background: 'var(--bg-primary)',
              cursor: 'pointer',
            }}
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
      <AnimatePresence>
        {showFilterPanel && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            style={{ overflow: 'hidden', marginBottom: 8 }}
          >
            <FilterPanel
              properties={properties}
              filters={filters}
              onFiltersChange={setFilters}
              onClose={() => setShowFilterPanel(false)}
            />
          </motion.div>
        )}
      </AnimatePresence>

      {/* Sort panel */}
      <AnimatePresence>
        {showSortPanel && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            style={{ overflow: 'hidden', marginBottom: 8 }}
          >
            <SortPanel
              properties={properties}
              sorts={sorts}
              onSortsChange={setSorts}
              onClose={() => setShowSortPanel(false)}
            />
          </motion.div>
        )}
      </AnimatePresence>

      {/* Loading overlay */}
      {isLoading && (
        <div className="loading-overlay" style={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          background: 'rgba(255,255,255,0.7)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 50,
        }}>
          Loading...
        </div>
      )}

      {/* View content */}
      <div className="database-content" style={{ flex: 1, overflow: 'auto' }}>
        {renderView()}
      </div>

      {/* Delete view confirmation dialog */}
      <ConfirmDialog
        isOpen={deleteViewDialog.isOpen}
        onClose={() => setDeleteViewDialog({ isOpen: false, viewId: null })}
        onConfirm={handleConfirmDeleteView}
        title="Delete view?"
        message="This action cannot be undone. The view configuration will be permanently removed."
        confirmText="Delete"
        cancelText="Cancel"
        variant="danger"
      />

      {/* Alert dialog */}
      <AlertDialog
        isOpen={alertDialog.isOpen}
        onClose={() => setAlertDialog({ isOpen: false, title: '', message: '' })}
        title={alertDialog.title}
        message={alertDialog.message}
        variant="warning"
      />
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
