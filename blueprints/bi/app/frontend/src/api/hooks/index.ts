import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../client'
import type {
  DataSource, DataSourceStatus, SyncResult, SyncLog, CacheStats,
  Table, Column, ColumnScanResult, Question, Dashboard, DashboardCard,
  Collection, Model, Metric, Alert, Subscription, User, QueryResult, Settings
} from '../types'

// Query keys
export const queryKeys = {
  datasources: ['datasources'] as const,
  datasource: (id: string) => ['datasources', id] as const,
  datasourceStatus: (id: string) => ['datasources', id, 'status'] as const,
  datasourceSchemas: (id: string) => ['datasources', id, 'schemas'] as const,
  datasourceSyncLog: (id: string) => ['datasources', id, 'sync-log'] as const,
  datasourceCacheStats: (id: string) => ['datasources', id, 'cache-stats'] as const,
  tables: (datasourceId: string) => ['tables', datasourceId] as const,
  table: (tableId: string) => ['table', tableId] as const,
  columns: (tableId: string) => ['columns', tableId] as const,
  questions: ['questions'] as const,
  question: (id: string) => ['questions', id] as const,
  dashboards: ['dashboards'] as const,
  dashboard: (id: string) => ['dashboards', id] as const,
  dashboardCards: (dashboardId: string) => ['dashboards', dashboardId, 'cards'] as const,
  collections: ['collections'] as const,
  collection: (id: string) => ['collections', id] as const,
  collectionItems: (id: string) => ['collections', id, 'items'] as const,
  models: ['models'] as const,
  model: (id: string) => ['models', id] as const,
  metrics: ['metrics'] as const,
  metric: (id: string) => ['metrics', id] as const,
  alerts: ['alerts'] as const,
  alert: (id: string) => ['alerts', id] as const,
  subscriptions: ['subscriptions'] as const,
  subscription: (id: string) => ['subscriptions', id] as const,
  users: ['users'] as const,
  user: (id: string) => ['users', id] as const,
  currentUser: ['currentUser'] as const,
  settings: ['settings'] as const,
  queryHistory: ['queryHistory'] as const,
}

// Data Sources
export function useDataSources() {
  return useQuery({
    queryKey: queryKeys.datasources,
    queryFn: () => api.get<DataSource[]>('/datasources'),
  })
}

export function useDataSource(id: string) {
  return useQuery({
    queryKey: queryKeys.datasource(id),
    queryFn: () => api.get<DataSource>(`/datasources/${id}`),
    enabled: !!id,
  })
}

export function useCreateDataSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<DataSource>) => api.post<DataSource>('/datasources', data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.datasources }),
  })
}

export function useUpdateDataSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<DataSource> & { id: string }) =>
      api.put<DataSource>(`/datasources/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.datasource(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.datasources })
    },
  })
}

export function useDeleteDataSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete(`/datasources/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.datasources }),
  })
}

export function useTestDataSource() {
  return useMutation({
    mutationFn: (id: string) => api.post<{ success: boolean; error?: string }>(`/datasources/${id}/test`),
  })
}

export function useSyncDataSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { id: string; full_sync?: boolean; scan_field_values?: boolean }) =>
      api.post<SyncResult>(`/datasources/${params.id}/sync`, {
        full_sync: params.full_sync ?? true,
        scan_field_values: params.scan_field_values ?? false,
      }),
    onSuccess: (_, params) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tables(params.id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.datasourceStatus(params.id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.datasourceSyncLog(params.id) })
    },
  })
}

// Data Source Status
export function useDataSourceStatus(id: string) {
  return useQuery({
    queryKey: queryKeys.datasourceStatus(id),
    queryFn: () => api.get<DataSourceStatus>(`/datasources/${id}/status`),
    enabled: !!id,
    refetchInterval: 30000, // Refresh every 30 seconds
  })
}

// Data Source Schemas
export function useDataSourceSchemas(id: string) {
  return useQuery({
    queryKey: queryKeys.datasourceSchemas(id),
    queryFn: () => api.get<string[]>(`/datasources/${id}/schemas`),
    enabled: !!id,
  })
}

// Scan Data Source (field values)
export function useScanDataSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { id: string; table_id?: string; column_id?: string; limit?: number }) =>
      api.post<SyncResult>(`/datasources/${params.id}/scan`, {
        table_id: params.table_id,
        column_id: params.column_id,
        limit: params.limit ?? 1000,
      }),
    onSuccess: (_, params) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tables(params.id) })
      if (params.table_id) {
        queryClient.invalidateQueries({ queryKey: queryKeys.columns(params.table_id) })
      }
    },
  })
}

// Fingerprint Data Source
export function useFingerprintDataSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { id: string; table_id?: string; sample_size?: number }) =>
      api.post<SyncResult>(`/datasources/${params.id}/fingerprint`, {
        table_id: params.table_id,
        sample_size: params.sample_size ?? 10000,
      }),
    onSuccess: (_, params) => {
      if (params.table_id) {
        queryClient.invalidateQueries({ queryKey: queryKeys.columns(params.table_id) })
      }
    },
  })
}

// Sync Log
export function useSyncLog(id: string, limit?: number) {
  return useQuery({
    queryKey: queryKeys.datasourceSyncLog(id),
    queryFn: () => api.get<{ logs: SyncLog[] }>(`/datasources/${id}/sync-log?limit=${limit ?? 10}`),
    enabled: !!id,
  })
}

// Cache Stats
export function useCacheStats(id: string) {
  return useQuery({
    queryKey: queryKeys.datasourceCacheStats(id),
    queryFn: () => api.get<CacheStats>(`/datasources/${id}/cache/stats`),
    enabled: !!id,
  })
}

// Clear Cache
export function useClearCache() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.post<{ status: string; columns_cleared: number }>(`/datasources/${id}/cache/clear`),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.datasourceCacheStats(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.tables(id) })
    },
  })
}

// Tables
export function useTables(datasourceId: string) {
  return useQuery({
    queryKey: queryKeys.tables(datasourceId),
    queryFn: () => api.get<Table[]>(`/datasources/${datasourceId}/tables`),
    enabled: !!datasourceId,
  })
}

// Single Table
export function useTable(datasourceId: string, tableId: string) {
  return useQuery({
    queryKey: queryKeys.table(tableId),
    queryFn: () => api.get<Table>(`/datasources/${datasourceId}/tables/${tableId}`),
    enabled: !!datasourceId && !!tableId,
  })
}

// Update Table
export function useUpdateTable() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: {
      datasourceId: string
      tableId: string
      display_name?: string
      description?: string
      visible?: boolean
      field_order?: string
    }) =>
      api.put<Table>(`/datasources/${params.datasourceId}/tables/${params.tableId}`, {
        display_name: params.display_name,
        description: params.description,
        visible: params.visible,
        field_order: params.field_order,
      }),
    onSuccess: (_, params) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.table(params.tableId) })
      queryClient.invalidateQueries({ queryKey: queryKeys.tables(params.datasourceId) })
    },
  })
}

// Sync Single Table
export function useSyncTable() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { datasourceId: string; tableId: string }) =>
      api.post<SyncResult>(`/datasources/${params.datasourceId}/tables/${params.tableId}/sync`),
    onSuccess: (_, params) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.columns(params.tableId) })
      queryClient.invalidateQueries({ queryKey: queryKeys.table(params.tableId) })
    },
  })
}

// Scan Single Table
export function useScanTable() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { datasourceId: string; tableId: string }) =>
      api.post<SyncResult>(`/datasources/${params.datasourceId}/tables/${params.tableId}/scan`),
    onSuccess: (_, params) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.columns(params.tableId) })
    },
  })
}

// Discard Cached Values for Table
export function useDiscardCachedValues() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { datasourceId: string; tableId: string }) =>
      api.post<{ status: string; columns_cleared: number }>(
        `/datasources/${params.datasourceId}/tables/${params.tableId}/discard-values`
      ),
    onSuccess: (_, params) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.columns(params.tableId) })
    },
  })
}

// Columns
export function useColumns(tableId: string) {
  return useQuery({
    queryKey: queryKeys.columns(tableId),
    queryFn: () => api.get<Column[]>(`/datasources/tables/${tableId}/columns`),
    enabled: !!tableId,
  })
}

// Table Preview - preview table data with pagination, sorting, filtering
export interface TablePreviewParams {
  datasourceId: string
  tableId: string
  page?: number
  pageSize?: number
  orderBy?: Array<{ column: string; direction: 'asc' | 'desc' }>
  filters?: Array<{ column: string; operator: string; value: any }>
}

export function useTablePreview(params: TablePreviewParams) {
  const { datasourceId, tableId, page = 1, pageSize = 100, orderBy = [], filters = [] } = params
  return useQuery({
    queryKey: ['tablePreview', datasourceId, tableId, page, pageSize, orderBy, filters] as const,
    queryFn: () => api.post<QueryResult>(
      `/datasources/${datasourceId}/tables/${tableId}/preview`,
      { page, page_size: pageSize, order_by: orderBy, filters }
    ),
    enabled: !!datasourceId && !!tableId,
  })
}

// Search Tables
export function useSearchTables(datasourceId: string, query: string) {
  return useQuery({
    queryKey: ['searchTables', datasourceId, query] as const,
    queryFn: () => api.get<Table[]>(`/datasources/${datasourceId}/search-tables?q=${encodeURIComponent(query)}`),
    enabled: !!datasourceId && !!query,
  })
}

// Scan Single Column
export function useScanColumn() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { datasourceId: string; tableId: string; columnId: string; limit?: number }) =>
      api.post<ColumnScanResult>(
        `/datasources/${params.datasourceId}/tables/${params.tableId}/columns/${params.columnId}/scan`,
        { limit: params.limit ?? 1000 }
      ),
    onSuccess: (_, params) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.columns(params.tableId) })
    },
  })
}

// Questions
export function useQuestions() {
  return useQuery({
    queryKey: queryKeys.questions,
    queryFn: () => api.get<Question[]>('/questions'),
  })
}

export function useQuestion(id: string) {
  return useQuery({
    queryKey: queryKeys.question(id),
    queryFn: () => api.get<Question>(`/questions/${id}`),
    enabled: !!id,
  })
}

export function useCreateQuestion() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Question>) => api.post<Question>('/questions', data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.questions }),
  })
}

export function useUpdateQuestion() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<Question> & { id: string }) =>
      api.put<Question>(`/questions/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.question(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.questions })
    },
  })
}

export function useDeleteQuestion() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete(`/questions/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.questions }),
  })
}

export function useExecuteQuestion() {
  return useMutation({
    mutationFn: (id: string) => api.post<QueryResult>(`/questions/${id}/query`),
  })
}

// Query execution
export function useExecuteQuery() {
  return useMutation({
    mutationFn: (data: { datasource_id: string; query: Record<string, any> }) =>
      api.post<QueryResult>('/query', data),
  })
}

export interface VariableValue {
  type: 'text' | 'number' | 'date'
  value: any
}

export function useExecuteNativeQuery() {
  return useMutation({
    mutationFn: (data: {
      datasource_id: string
      query: string
      params?: any[]
      variables?: Record<string, VariableValue>
    }) => api.post<QueryResult>('/query/native', data),
  })
}

// Dashboards
export function useDashboards() {
  return useQuery({
    queryKey: queryKeys.dashboards,
    queryFn: () => api.get<Dashboard[]>('/dashboards'),
  })
}

export function useDashboard(id: string) {
  return useQuery({
    queryKey: queryKeys.dashboard(id),
    queryFn: () => api.get<Dashboard>(`/dashboards/${id}`),
    enabled: !!id,
  })
}

export function useCreateDashboard() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Dashboard>) => api.post<Dashboard>('/dashboards', data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.dashboards }),
  })
}

export function useUpdateDashboard() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<Dashboard> & { id: string }) =>
      api.put<Dashboard>(`/dashboards/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboard(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboards })
    },
  })
}

export function useDeleteDashboard() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete(`/dashboards/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.dashboards }),
  })
}

// Dashboard Cards
export function useDashboardCards(dashboardId: string) {
  return useQuery({
    queryKey: queryKeys.dashboardCards(dashboardId),
    queryFn: () => api.get<DashboardCard[]>(`/dashboards/${dashboardId}/cards`),
    enabled: !!dashboardId,
  })
}

export function useAddDashboardCard() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ dashboardId, ...card }: Partial<DashboardCard> & { dashboardId: string }) =>
      api.post<DashboardCard>(`/dashboards/${dashboardId}/cards`, card),
    onSuccess: (_, { dashboardId }) =>
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboardCards(dashboardId) }),
  })
}

export function useUpdateDashboardCard() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ dashboardId, id, ...card }: Partial<DashboardCard> & { dashboardId: string; id: string }) =>
      api.put<DashboardCard>(`/dashboards/${dashboardId}/cards/${id}`, card),
    onSuccess: (_, { dashboardId }) =>
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboardCards(dashboardId) }),
  })
}

export function useRemoveDashboardCard() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ dashboardId, cardId }: { dashboardId: string; cardId: string }) =>
      api.delete(`/dashboards/${dashboardId}/cards/${cardId}`),
    onSuccess: (_, { dashboardId }) =>
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboardCards(dashboardId) }),
  })
}

// Collections
export function useCollections() {
  return useQuery({
    queryKey: queryKeys.collections,
    queryFn: () => api.get<Collection[]>('/collections'),
  })
}

export function useCollection(id: string) {
  return useQuery({
    queryKey: queryKeys.collection(id),
    queryFn: () => api.get<Collection>(`/collections/${id}`),
    enabled: !!id,
  })
}

export function useCollectionItems(id: string) {
  return useQuery({
    queryKey: queryKeys.collectionItems(id),
    queryFn: () => api.get<{
      questions: Question[]
      dashboards: Dashboard[]
      subcollections: Collection[]
    }>(`/collections/${id}/items`),
    enabled: !!id,
  })
}

export function useCreateCollection() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Collection>) => api.post<Collection>('/collections', data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.collections }),
  })
}

export function useUpdateCollection() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<Collection> & { id: string }) =>
      api.put<Collection>(`/collections/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.collection(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.collections })
    },
  })
}

export function useDeleteCollection() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete(`/collections/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.collections }),
  })
}

// Special Collections (root/personal)
export function useRootCollection() {
  return useQuery({
    queryKey: ['collections', 'root'] as const,
    queryFn: () => api.get<Collection>('/collections/root'),
  })
}

export function useRootCollectionItems() {
  return useQuery({
    queryKey: ['collections', 'root', 'items'] as const,
    queryFn: () => api.get<{
      questions: Question[]
      dashboards: Dashboard[]
      subcollections: Collection[]
    }>('/collections/root/items'),
  })
}

export function usePersonalCollection() {
  return useQuery({
    queryKey: ['collections', 'personal'] as const,
    queryFn: () => api.get<Collection>('/collections/personal'),
  })
}

export function usePersonalCollectionItems() {
  return useQuery({
    queryKey: ['collections', 'personal', 'items'] as const,
    queryFn: () => api.get<{
      questions: Question[]
      dashboards: Dashboard[]
      subcollections: Collection[]
    }>('/collections/personal/items'),
  })
}

// Models
export function useModels() {
  return useQuery({
    queryKey: queryKeys.models,
    queryFn: () => api.get<Model[]>('/models'),
  })
}

export function useModel(id: string) {
  return useQuery({
    queryKey: queryKeys.model(id),
    queryFn: () => api.get<Model>(`/models/${id}`),
    enabled: !!id,
  })
}

export function useCreateModel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Model>) => api.post<Model>('/models', data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.models }),
  })
}

export function useUpdateModel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<Model> & { id: string }) =>
      api.put<Model>(`/models/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.model(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.models })
    },
  })
}

export function useDeleteModel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete(`/models/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.models }),
  })
}

// Metrics
export function useMetrics() {
  return useQuery({
    queryKey: queryKeys.metrics,
    queryFn: () => api.get<Metric[]>('/metrics'),
  })
}

export function useMetric(id: string) {
  return useQuery({
    queryKey: queryKeys.metric(id),
    queryFn: () => api.get<Metric>(`/metrics/${id}`),
    enabled: !!id,
  })
}

export function useCreateMetric() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Metric>) => api.post<Metric>('/metrics', data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.metrics }),
  })
}

export function useUpdateMetric() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<Metric> & { id: string }) =>
      api.put<Metric>(`/metrics/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.metric(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.metrics })
    },
  })
}

export function useDeleteMetric() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete(`/metrics/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.metrics }),
  })
}

// Alerts
export function useAlerts() {
  return useQuery({
    queryKey: queryKeys.alerts,
    queryFn: () => api.get<Alert[]>('/alerts'),
  })
}

export function useAlert(id: string) {
  return useQuery({
    queryKey: queryKeys.alert(id),
    queryFn: () => api.get<Alert>(`/alerts/${id}`),
    enabled: !!id,
  })
}

export function useCreateAlert() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Alert>) => api.post<Alert>('/alerts', data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.alerts }),
  })
}

export function useUpdateAlert() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<Alert> & { id: string }) =>
      api.put<Alert>(`/alerts/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.alert(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.alerts })
    },
  })
}

export function useDeleteAlert() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete(`/alerts/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.alerts }),
  })
}

// Subscriptions
export function useSubscriptions() {
  return useQuery({
    queryKey: queryKeys.subscriptions,
    queryFn: () => api.get<Subscription[]>('/subscriptions'),
  })
}

export function useSubscription(id: string) {
  return useQuery({
    queryKey: queryKeys.subscription(id),
    queryFn: () => api.get<Subscription>(`/subscriptions/${id}`),
    enabled: !!id,
  })
}

export function useCreateSubscription() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Subscription>) => api.post<Subscription>('/subscriptions', data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.subscriptions }),
  })
}

export function useUpdateSubscription() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<Subscription> & { id: string }) =>
      api.put<Subscription>(`/subscriptions/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.subscription(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.subscriptions })
    },
  })
}

export function useDeleteSubscription() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete(`/subscriptions/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.subscriptions }),
  })
}

// Users
export function useUsers() {
  return useQuery({
    queryKey: queryKeys.users,
    queryFn: () => api.get<User[]>('/users'),
  })
}

export function useCurrentUser() {
  return useQuery({
    queryKey: queryKeys.currentUser,
    queryFn: () => api.get<User>('/auth/me'),
    retry: false,
  })
}

export function useLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (credentials: { email: string; password: string }) =>
      api.post<{ token: string; user: User }>('/auth/login', credentials),
    onSuccess: (data) => {
      api.setToken(data.token)
      queryClient.setQueryData(queryKeys.currentUser, data.user)
    },
  })
}

export function useLogout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post('/auth/logout'),
    onSuccess: () => {
      api.setToken(null)
      queryClient.clear()
    },
  })
}

// Settings
export function useSettings() {
  return useQuery({
    queryKey: queryKeys.settings,
    queryFn: () => api.get<Settings>('/settings'),
  })
}

export function useUpdateSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Settings>) => api.put('/settings', data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.settings }),
  })
}

// User Profile
export function useUpdateProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { name?: string; email?: string }) => api.put<User>('/auth/me', data),
    onSuccess: (data) => queryClient.setQueryData(queryKeys.currentUser, data),
  })
}

export function useChangePassword() {
  return useMutation({
    mutationFn: (data: { current_password: string; new_password: string }) =>
      api.post('/auth/me/password', data),
  })
}

// User Management (Admin)
export function useUser(id: string) {
  return useQuery({
    queryKey: queryKeys.user(id),
    queryFn: () => api.get<User>(`/users/${id}`),
    enabled: !!id,
  })
}

export function useCreateUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { name: string; email: string; password: string; role: string }) =>
      api.post<User>('/users', data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.users }),
  })
}

export function useUpdateUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: { id: string; name?: string; email?: string; role?: string }) =>
      api.put<User>(`/users/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.user(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.users })
    },
  })
}

export function useDeleteUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete(`/users/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.users }),
  })
}

export function useDeactivateUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.post(`/users/${id}/deactivate`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.users }),
  })
}

export function useResetUserPassword() {
  return useMutation({
    mutationFn: (id: string) => api.post<{ temporary_password: string }>(`/users/${id}/reset-password`),
  })
}

// Test data source connection (before creating)
export function useTestDataSourceConnection() {
  return useMutation({
    mutationFn: (data: { engine: string; host?: string; port?: number; database: string; username?: string; password?: string }) =>
      api.post<{ success: boolean; error?: string }>('/datasources/test-connection', data),
  })
}

// Update column metadata
export function useUpdateColumn() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ tableId, columnId, ...data }: { tableId: string; columnId: string; display_name?: string; description?: string; semantic?: string }) =>
      api.put<Column>(`/datasources/tables/${tableId}/columns/${columnId}`, data),
    onSuccess: (_, { tableId }) => queryClient.invalidateQueries({ queryKey: queryKeys.columns(tableId) }),
  })
}

// Activity/Audit log
export function useActivityLog(filters?: { user_id?: string; type?: string; limit?: number }) {
  const params = new URLSearchParams()
  if (filters?.user_id) params.append('user_id', filters.user_id)
  if (filters?.type) params.append('type', filters.type)
  if (filters?.limit) params.append('limit', String(filters.limit))
  const queryString = params.toString()

  return useQuery({
    queryKey: ['activityLog', filters],
    queryFn: () => api.get<{
      activities: {
        id: string
        user_id: string
        user_name: string
        type: string
        description: string
        created_at: string
      }[]
    }>(`/admin/activity${queryString ? `?${queryString}` : ''}`),
  })
}
