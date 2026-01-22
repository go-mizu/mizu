import { useQuery, useMutation, useQueryClient, UseQueryOptions } from '@tanstack/react-query'
import { api } from '../client'
import type {
  DataSource, Table, Column, Question, Dashboard, DashboardCard,
  Collection, Model, Metric, Alert, Subscription, User, QueryResult, Settings
} from '../types'

// Query keys
export const queryKeys = {
  datasources: ['datasources'] as const,
  datasource: (id: string) => ['datasources', id] as const,
  tables: (datasourceId: string) => ['tables', datasourceId] as const,
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
    mutationFn: (id: string) => api.post<{ tables_synced: number }>(`/datasources/${id}/sync`),
    onSuccess: (_, id) => queryClient.invalidateQueries({ queryKey: queryKeys.tables(id) }),
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

export function useColumns(tableId: string) {
  return useQuery({
    queryKey: queryKeys.columns(tableId),
    queryFn: () => api.get<Column[]>(`/datasources/tables/${tableId}/columns`),
    enabled: !!tableId,
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

export function useExecuteNativeQuery() {
  return useMutation({
    mutationFn: (data: { datasource_id: string; query: string; params?: any[] }) =>
      api.post<QueryResult>('/query/native', data),
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
    }>('/admin/activity', { params: filters }),
  })
}
