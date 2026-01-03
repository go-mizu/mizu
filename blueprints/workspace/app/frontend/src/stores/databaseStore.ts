import { create } from 'zustand'
import { api, Database, DatabaseRow, Property, View, Filter, Sort } from '../api/client'

interface DatabaseState {
  // Data
  databases: Record<string, Database>
  rows: Record<string, DatabaseRow[]>
  views: Record<string, View[]>

  // Active state
  activeViewId: string | null
  activeFilters: Filter[]
  activeSorts: Sort[]

  // Loading states
  loading: Record<string, boolean>

  // Actions
  fetchDatabase: (databaseId: string) => Promise<Database | null>
  fetchRows: (databaseId: string, filters?: Filter[], sorts?: Sort[]) => Promise<DatabaseRow[]>
  fetchViews: (databaseId: string) => Promise<View[]>

  // Row operations
  createRow: (databaseId: string, properties?: Record<string, unknown>) => Promise<DatabaseRow | null>
  updateRow: (rowId: string, properties: Record<string, unknown>) => Promise<void>
  deleteRow: (rowId: string, databaseId: string) => Promise<void>
  duplicateRow: (rowId: string, databaseId: string) => Promise<DatabaseRow | null>

  // Property operations
  addProperty: (databaseId: string, property: Partial<Property>) => Promise<void>
  updateProperty: (databaseId: string, propertyId: string, property: Partial<Property>) => Promise<void>
  deleteProperty: (databaseId: string, propertyId: string) => Promise<void>

  // View operations
  setActiveView: (viewId: string) => void
  setFilters: (filters: Filter[]) => void
  setSorts: (sorts: Sort[]) => void
  createView: (databaseId: string, view: Partial<View>) => Promise<View | null>
  updateView: (viewId: string, data: Partial<View>) => Promise<void>
  deleteView: (viewId: string) => Promise<void>

  // Cache
  invalidateDatabase: (databaseId: string) => void
}

export const useDatabaseStore = create<DatabaseState>((set, get) => ({
  databases: {},
  rows: {},
  views: {},
  activeViewId: null,
  activeFilters: [],
  activeSorts: [],
  loading: {},

  fetchDatabase: async (databaseId) => {
    const { databases } = get()
    if (databases[databaseId]) {
      return databases[databaseId]
    }

    set({ loading: { ...get().loading, [databaseId]: true } })

    try {
      const database = await api.get<Database>(`/databases/${databaseId}`)
      set({
        databases: { ...get().databases, [databaseId]: database },
        loading: { ...get().loading, [databaseId]: false },
      })
      return database
    } catch (err) {
      set({ loading: { ...get().loading, [databaseId]: false } })
      return null
    }
  },

  fetchRows: async (databaseId, filters, sorts) => {
    try {
      const params = new URLSearchParams()
      if (filters?.length) {
        params.set('filters', JSON.stringify(filters))
      }
      if (sorts?.length) {
        params.set('sorts', JSON.stringify(sorts))
      }

      const url = `/databases/${databaseId}/rows${params.toString() ? '?' + params.toString() : ''}`
      const result = await api.get<{ rows: DatabaseRow[] }>(url)
      set({ rows: { ...get().rows, [databaseId]: result.rows } })
      return result.rows
    } catch (err) {
      return []
    }
  },

  fetchViews: async (databaseId) => {
    try {
      const result = await api.get<{ views: View[] }>(`/databases/${databaseId}/views`)
      set({ views: { ...get().views, [databaseId]: result.views } })
      return result.views
    } catch (err) {
      return []
    }
  },

  createRow: async (databaseId, properties = {}) => {
    try {
      const row = await api.post<DatabaseRow>(`/databases/${databaseId}/rows`, {
        properties: { title: 'New row', ...properties },
      })

      const { rows } = get()
      set({ rows: { ...rows, [databaseId]: [...(rows[databaseId] || []), row] } })
      return row
    } catch (err) {
      return null
    }
  },

  updateRow: async (rowId, properties) => {
    const { rows } = get()

    // Find the row and its database
    let databaseId: string | null = null
    let rowIndex = -1

    for (const [dbId, dbRows] of Object.entries(rows)) {
      const idx = dbRows.findIndex(r => r.id === rowId)
      if (idx !== -1) {
        databaseId = dbId
        rowIndex = idx
        break
      }
    }

    if (!databaseId || rowIndex === -1) return

    const currentRows = rows[databaseId]
    const currentRow = currentRows[rowIndex]

    // Optimistic update
    const updatedRow = { ...currentRow, properties: { ...currentRow.properties, ...properties } }
    const newRows = [...currentRows]
    newRows[rowIndex] = updatedRow
    set({ rows: { ...rows, [databaseId]: newRows } })

    try {
      await api.patch(`/rows/${rowId}`, { properties })
    } catch (err) {
      // Revert on error
      set({ rows: { ...get().rows, [databaseId]: currentRows } })
      throw err
    }
  },

  deleteRow: async (rowId, databaseId) => {
    const { rows } = get()
    const currentRows = rows[databaseId] || []

    // Optimistic delete
    set({ rows: { ...rows, [databaseId]: currentRows.filter(r => r.id !== rowId) } })

    try {
      await api.delete(`/rows/${rowId}`)
    } catch (err) {
      // Revert on error
      set({ rows: { ...get().rows, [databaseId]: currentRows } })
      throw err
    }
  },

  duplicateRow: async (rowId, databaseId) => {
    const { rows } = get()
    const currentRows = rows[databaseId] || []
    const originalRow = currentRows.find(r => r.id === rowId)

    if (!originalRow) return null

    try {
      const newRow = await api.post<DatabaseRow>(`/databases/${databaseId}/rows`, {
        properties: {
          ...originalRow.properties,
          title: `${originalRow.properties.title || 'Untitled'} (copy)`,
        },
      })

      set({ rows: { ...get().rows, [databaseId]: [...currentRows, newRow] } })
      return newRow
    } catch (err) {
      return null
    }
  },

  addProperty: async (databaseId, property) => {
    try {
      const result = await api.post<Database>(`/databases/${databaseId}/properties`, property)
      set({ databases: { ...get().databases, [databaseId]: result } })
    } catch (err) {
      throw err
    }
  },

  updateProperty: async (databaseId, propertyId, property) => {
    const { databases } = get()
    const currentDb = databases[databaseId]

    if (!currentDb) return

    // Optimistic update
    const updatedProps = currentDb.properties.map(p =>
      p.id === propertyId ? { ...p, ...property } : p
    )
    set({
      databases: {
        ...databases,
        [databaseId]: { ...currentDb, properties: updatedProps },
      },
    })

    try {
      await api.put(`/properties/${propertyId}`, property)
    } catch (err) {
      // Revert on error
      set({ databases: { ...get().databases, [databaseId]: currentDb } })
      throw err
    }
  },

  deleteProperty: async (databaseId, propertyId) => {
    const { databases } = get()
    const currentDb = databases[databaseId]

    if (!currentDb) return

    // Optimistic delete
    const updatedProps = currentDb.properties.filter(p => p.id !== propertyId)
    set({
      databases: {
        ...databases,
        [databaseId]: { ...currentDb, properties: updatedProps },
      },
    })

    try {
      await api.delete(`/properties/${propertyId}`)
    } catch (err) {
      // Revert on error
      set({ databases: { ...get().databases, [databaseId]: currentDb } })
      throw err
    }
  },

  setActiveView: (viewId) => {
    set({ activeViewId: viewId })
  },

  setFilters: (filters) => {
    set({ activeFilters: filters })
  },

  setSorts: (sorts) => {
    set({ activeSorts: sorts })
  },

  createView: async (databaseId, view) => {
    try {
      const result = await api.post<View>(`/databases/${databaseId}/views`, view)
      const { views } = get()
      set({ views: { ...views, [databaseId]: [...(views[databaseId] || []), result] } })
      return result
    } catch (err) {
      return null
    }
  },

  updateView: async (viewId, data) => {
    try {
      await api.put(`/views/${viewId}`, data)
    } catch (err) {
      throw err
    }
  },

  deleteView: async (viewId) => {
    try {
      await api.delete(`/views/${viewId}`)
      // Remove from local state
      const { views } = get()
      const newViews: Record<string, View[]> = {}
      for (const [dbId, dbViews] of Object.entries(views)) {
        newViews[dbId] = dbViews.filter(v => v.id !== viewId)
      }
      set({ views: newViews })
    } catch (err) {
      throw err
    }
  },

  invalidateDatabase: (databaseId) => {
    const { databases, rows, views } = get()
    const newDatabases = { ...databases }
    const newRows = { ...rows }
    const newViews = { ...views }
    delete newDatabases[databaseId]
    delete newRows[databaseId]
    delete newViews[databaseId]
    set({ databases: newDatabases, rows: newRows, views: newViews })
  },
}))
