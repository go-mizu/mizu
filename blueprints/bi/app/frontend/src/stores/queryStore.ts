import { create } from 'zustand'
import type { Filter, Join, Aggregation, OrderBy, VisualizationType, VisualizationSettings } from '../api/types'

interface SelectedColumn {
  id: string
  table: string
  column: string
  alias?: string
}

interface GroupByColumn {
  id: string
  column: string
  temporalBucket?: 'minute' | 'hour' | 'day' | 'week' | 'month' | 'quarter' | 'year'
}

interface CustomColumn {
  id: string
  name: string
  expression: string
}

interface QueryState {
  // Mode
  mode: 'query' | 'native'

  // Source selection
  datasourceId: string | null
  sourceTable: string | null

  // Native SQL
  nativeSql: string

  // Query builder state
  columns: SelectedColumn[]
  joins: Join[]
  filters: Filter[]
  aggregations: Aggregation[]
  groupBy: GroupByColumn[]
  customColumns: CustomColumn[]
  orderBy: OrderBy[]
  limit: number | null

  // Visualization
  visualization: VisualizationSettings

  // Results
  isExecuting: boolean
  lastExecuted: Date | null

  // Actions
  setMode: (mode: 'query' | 'native') => void
  setDatasource: (id: string | null) => void
  setSourceTable: (table: string | null) => void
  setNativeSql: (sql: string) => void

  // Column actions
  addColumn: (column: SelectedColumn) => void
  removeColumn: (id: string) => void
  updateColumn: (id: string, updates: Partial<SelectedColumn>) => void
  reorderColumns: (columns: SelectedColumn[]) => void
  clearColumns: () => void

  // Join actions
  addJoin: (join: Join) => void
  removeJoin: (id: string) => void
  updateJoin: (id: string, updates: Partial<Join>) => void

  // Filter actions
  addFilter: (filter: Filter) => void
  removeFilter: (id: string) => void
  updateFilter: (id: string, updates: Partial<Filter>) => void
  clearFilters: () => void

  // Aggregation actions
  addAggregation: (aggregation: Aggregation) => void
  removeAggregation: (id: string) => void
  updateAggregation: (id: string, updates: Partial<Aggregation>) => void

  // Group by actions
  addGroupBy: (groupBy: GroupByColumn) => void
  removeGroupBy: (id: string) => void
  updateGroupBy: (id: string, updates: Partial<GroupByColumn>) => void

  // Custom column actions
  addCustomColumn: (column: CustomColumn) => void
  removeCustomColumn: (id: string) => void
  updateCustomColumn: (id: string, updates: Partial<CustomColumn>) => void

  // Order by actions
  addOrderBy: (orderBy: OrderBy) => void
  removeOrderBy: (column: string) => void
  updateOrderBy: (column: string, direction: 'asc' | 'desc') => void

  // Limit actions
  setLimit: (limit: number | null) => void

  // Visualization actions
  setVisualizationType: (type: VisualizationType) => void
  setVisualizationSettings: (settings: Record<string, any>) => void
  setVisualization: (visualization: VisualizationSettings) => void

  // Execution state
  setIsExecuting: (isExecuting: boolean) => void
  setLastExecuted: (date: Date | null) => void

  // Reset
  reset: () => void
  loadQuestion: (question: {
    mode: 'query' | 'native'
    datasourceId: string
    query: any
    visualization: VisualizationSettings
  }) => void
}

const generateId = () => Math.random().toString(36).substring(2, 9)

const initialState = {
  mode: 'query' as const,
  datasourceId: null,
  sourceTable: null,
  nativeSql: '',
  columns: [],
  joins: [],
  filters: [],
  aggregations: [],
  groupBy: [],
  customColumns: [],
  orderBy: [],
  limit: null,
  visualization: { type: 'table' as const },
  isExecuting: false,
  lastExecuted: null,
}

export const useQueryStore = create<QueryState>((set, get) => ({
  ...initialState,

  // Mode
  setMode: (mode) => set({ mode }),

  // Source
  setDatasource: (datasourceId) => set({
    datasourceId,
    sourceTable: null,
    columns: [],
    joins: [],
    filters: [],
    aggregations: [],
    groupBy: [],
  }),

  setSourceTable: (sourceTable) => set({
    sourceTable,
    columns: [],
    joins: [],
    filters: [],
    aggregations: [],
    groupBy: [],
  }),

  setNativeSql: (nativeSql) => set({ nativeSql }),

  // Columns
  addColumn: (column) => set((state) => ({
    columns: [...state.columns, { ...column, id: column.id || generateId() }],
  })),

  removeColumn: (id) => set((state) => ({
    columns: state.columns.filter((c) => c.id !== id),
  })),

  updateColumn: (id, updates) => set((state) => ({
    columns: state.columns.map((c) => (c.id === id ? { ...c, ...updates } : c)),
  })),

  reorderColumns: (columns) => set({ columns }),

  clearColumns: () => set({ columns: [] }),

  // Joins
  addJoin: (join) => set((state) => ({
    joins: [...state.joins, { ...join, id: join.id || generateId() }],
  })),

  removeJoin: (id) => set((state) => ({
    joins: state.joins.filter((j) => j.id !== id),
  })),

  updateJoin: (id, updates) => set((state) => ({
    joins: state.joins.map((j) => (j.id === id ? { ...j, ...updates } : j)),
  })),

  // Filters
  addFilter: (filter) => set((state) => ({
    filters: [...state.filters, { ...filter, id: filter.id || generateId() }],
  })),

  removeFilter: (id) => set((state) => ({
    filters: state.filters.filter((f) => f.id !== id),
  })),

  updateFilter: (id, updates) => set((state) => ({
    filters: state.filters.map((f) => (f.id === id ? { ...f, ...updates } : f)),
  })),

  clearFilters: () => set({ filters: [] }),

  // Aggregations
  addAggregation: (aggregation) => set((state) => ({
    aggregations: [...state.aggregations, { ...aggregation, id: aggregation.id || generateId() }],
  })),

  removeAggregation: (id) => set((state) => ({
    aggregations: state.aggregations.filter((a) => a.id !== id),
  })),

  updateAggregation: (id, updates) => set((state) => ({
    aggregations: state.aggregations.map((a) => (a.id === id ? { ...a, ...updates } : a)),
  })),

  // Group by
  addGroupBy: (groupBy) => set((state) => ({
    groupBy: [...state.groupBy, { ...groupBy, id: groupBy.id || generateId() }],
  })),

  removeGroupBy: (id) => set((state) => ({
    groupBy: state.groupBy.filter((g) => g.id !== id),
  })),

  updateGroupBy: (id, updates) => set((state) => ({
    groupBy: state.groupBy.map((g) => (g.id === id ? { ...g, ...updates } : g)),
  })),

  // Custom columns
  addCustomColumn: (column) => set((state) => ({
    customColumns: [...state.customColumns, { ...column, id: column.id || generateId() }],
  })),

  removeCustomColumn: (id) => set((state) => ({
    customColumns: state.customColumns.filter((c) => c.id !== id),
  })),

  updateCustomColumn: (id, updates) => set((state) => ({
    customColumns: state.customColumns.map((c) => (c.id === id ? { ...c, ...updates } : c)),
  })),

  // Order by
  addOrderBy: (orderBy) => set((state) => ({
    orderBy: [...state.orderBy.filter((o) => o.column !== orderBy.column), orderBy],
  })),

  removeOrderBy: (column) => set((state) => ({
    orderBy: state.orderBy.filter((o) => o.column !== column),
  })),

  updateOrderBy: (column, direction) => set((state) => ({
    orderBy: state.orderBy.map((o) => (o.column === column ? { ...o, direction } : o)),
  })),

  // Limit
  setLimit: (limit) => set({ limit }),

  // Visualization
  setVisualizationType: (type) => set((state) => ({
    visualization: { ...state.visualization, type },
  })),

  setVisualizationSettings: (settings) => set((state) => ({
    visualization: { ...state.visualization, settings },
  })),

  setVisualization: (visualization) => set({ visualization }),

  // Execution
  setIsExecuting: (isExecuting) => set({ isExecuting }),
  setLastExecuted: (lastExecuted) => set({ lastExecuted }),

  // Reset
  reset: () => set(initialState),

  loadQuestion: (question) => set({
    mode: question.mode,
    datasourceId: question.datasourceId,
    sourceTable: question.query?.table || null,
    nativeSql: question.query?.sql || '',
    columns: question.query?.columns?.map((c: string, i: number) => ({
      id: generateId(),
      table: question.query?.table || '',
      column: c,
    })) || [],
    filters: question.query?.filters || [],
    joins: question.query?.joins || [],
    aggregations: question.query?.aggregations || [],
    groupBy: question.query?.group_by?.map((c: string) => ({
      id: generateId(),
      column: c,
    })) || [],
    orderBy: question.query?.order_by || [],
    limit: question.query?.limit || null,
    visualization: question.visualization || { type: 'table' },
  }),
}))
