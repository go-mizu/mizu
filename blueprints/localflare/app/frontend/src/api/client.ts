import type { ApiResponse } from '../types'

class ApiClient {
  private baseUrl = '/api'

  private async request<T>(
    method: string,
    path: string,
    data?: unknown,
    options?: RequestInit
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`
    const config: RequestInit = {
      method,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
      ...options,
    }

    if (data && method !== 'GET') {
      config.body = JSON.stringify(data)
    }

    const response = await fetch(url, config)

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Request failed' }))
      throw new Error(error.message || `HTTP ${response.status}`)
    }

    // Handle empty responses
    const text = await response.text()
    if (!text) {
      return {} as T
    }

    return JSON.parse(text)
  }

  async get<T>(path: string): Promise<T> {
    return this.request<T>('GET', path)
  }

  async post<T>(path: string, data?: unknown): Promise<T> {
    return this.request<T>('POST', path, data)
  }

  async put<T>(path: string, data?: unknown): Promise<T> {
    return this.request<T>('PUT', path, data)
  }

  async delete<T>(path: string): Promise<T> {
    return this.request<T>('DELETE', path)
  }

  // Durable Objects
  durableObjects = {
    listNamespaces: () =>
      this.get<ApiResponse<{ namespaces: import('../types').DurableObjectNamespace[] }>>('/durable-objects/namespaces'),
    getNamespace: (id: string) =>
      this.get<ApiResponse<import('../types').DurableObjectNamespace>>(`/durable-objects/namespaces/${id}`),
    createNamespace: (data: { name: string; class_name: string; script_name?: string }) =>
      this.post<ApiResponse<import('../types').DurableObjectNamespace>>('/durable-objects/namespaces', data),
    deleteNamespace: (id: string) =>
      this.delete<ApiResponse<void>>(`/durable-objects/namespaces/${id}`),
    listObjects: (namespaceId: string) =>
      this.get<ApiResponse<{ objects: import('../types').DurableObjectInstance[] }>>(`/durable-objects/namespaces/${namespaceId}/objects`),
    queryStorage: (namespaceId: string, objectId: string, query: string) =>
      this.post<ApiResponse<{ results: unknown[] }>>(`/durable-objects/namespaces/${namespaceId}/objects/${objectId}/query`, { query }),
  }

  // Queues
  queues = {
    list: () =>
      this.get<ApiResponse<{ queues: import('../types').Queue[] }>>('/queues'),
    get: (id: string) =>
      this.get<ApiResponse<import('../types').Queue>>(`/queues/${id}`),
    create: (data: { name: string; settings?: Partial<import('../types').QueueSettings> }) =>
      this.post<ApiResponse<import('../types').Queue>>('/queues', data),
    delete: (id: string) =>
      this.delete<ApiResponse<void>>(`/queues/${id}`),
    updateSettings: (id: string, settings: Partial<import('../types').QueueSettings>) =>
      this.put<ApiResponse<import('../types').Queue>>(`/queues/${id}/settings`, settings),
    sendMessage: (id: string, message: import('../types').QueueMessage) =>
      this.post<ApiResponse<{ message_id: string }>>(`/queues/${id}/messages`, message),
    addConsumer: (id: string, consumer: Omit<import('../types').QueueConsumer, 'id' | 'queue_id'>) =>
      this.post<ApiResponse<import('../types').QueueConsumer>>(`/queues/${id}/consumers`, consumer),
    removeConsumer: (queueId: string, consumerId: string) =>
      this.delete<ApiResponse<void>>(`/queues/${queueId}/consumers/${consumerId}`),
  }

  // Vectorize
  vectorize = {
    listIndexes: () =>
      this.get<ApiResponse<{ indexes: import('../types').VectorIndex[] }>>('/vectorize/indexes'),
    getIndex: (name: string) =>
      this.get<ApiResponse<import('../types').VectorIndex>>(`/vectorize/indexes/${name}`),
    createIndex: (data: { name: string; dimensions: number; metric: string; description?: string }) =>
      this.post<ApiResponse<import('../types').VectorIndex>>('/vectorize/indexes', data),
    deleteIndex: (name: string) =>
      this.delete<ApiResponse<void>>(`/vectorize/indexes/${name}`),
    query: (name: string, request: import('../types').VectorQueryRequest) =>
      this.post<ApiResponse<{ matches: import('../types').VectorMatch[] }>>(`/vectorize/indexes/${name}/query`, request),
    insert: (name: string, vectors: Array<{ id: string; values: number[]; metadata?: Record<string, unknown>; namespace?: string }>) =>
      this.post<ApiResponse<{ inserted: number }>>(`/vectorize/indexes/${name}/insert`, { vectors }),
  }

  // Analytics Engine
  analytics = {
    listDatasets: () =>
      this.get<ApiResponse<{ datasets: import('../types').AnalyticsDataset[] }>>('/analytics/datasets'),
    getDataset: (name: string) =>
      this.get<ApiResponse<import('../types').AnalyticsDataset>>(`/analytics/datasets/${name}`),
    createDataset: (data: { name: string }) =>
      this.post<ApiResponse<import('../types').AnalyticsDataset>>('/analytics/datasets', data),
    deleteDataset: (name: string) =>
      this.delete<ApiResponse<void>>(`/analytics/datasets/${name}`),
    query: (name: string, sql: string) =>
      this.post<ApiResponse<import('../types').AnalyticsQueryResult>>(`/analytics/datasets/${name}/query`, { query: sql }),
    write: (name: string, data: Record<string, unknown>) =>
      this.post<ApiResponse<void>>(`/analytics/datasets/${name}/write`, data),
  }

  // Workers AI
  ai = {
    listModels: () =>
      this.get<ApiResponse<{ models: import('../types').AIModel[] }>>('/ai/models'),
    run: (request: import('../types').AIInferenceRequest) =>
      this.post<ApiResponse<import('../types').AIInferenceResponse>>('/ai/run', request),
    getStats: () =>
      this.get<ApiResponse<{ requests_today: number; tokens_today: number; cost_today: number }>>('/ai/stats'),
  }

  // AI Gateway
  aiGateway = {
    list: () =>
      this.get<ApiResponse<{ gateways: import('../types').AIGateway[] }>>('/ai-gateway'),
    get: (id: string) =>
      this.get<ApiResponse<import('../types').AIGateway>>(`/ai-gateway/${id}`),
    create: (data: { name: string; settings?: Partial<import('../types').AIGatewaySettings> }) =>
      this.post<ApiResponse<import('../types').AIGateway>>('/ai-gateway', data),
    delete: (id: string) =>
      this.delete<ApiResponse<void>>(`/ai-gateway/${id}`),
    updateSettings: (id: string, settings: Partial<import('../types').AIGatewaySettings>) =>
      this.put<ApiResponse<import('../types').AIGateway>>(`/ai-gateway/${id}/settings`, settings),
    getLogs: (id: string, params?: { provider?: string; status?: string; model?: string; limit?: number; offset?: number }) => {
      const query = new URLSearchParams()
      if (params?.provider) query.set('provider', params.provider)
      if (params?.status) query.set('status', params.status)
      if (params?.model) query.set('model', params.model)
      if (params?.limit) query.set('limit', params.limit.toString())
      if (params?.offset) query.set('offset', params.offset.toString())
      return this.get<ApiResponse<{ logs: import('../types').AIGatewayLog[]; total: number }>>(`/ai-gateway/${id}/logs?${query}`)
    },
  }

  // Hyperdrive
  hyperdrive = {
    list: () =>
      this.get<ApiResponse<{ configs: import('../types').HyperdriveConfig[] }>>('/hyperdrive'),
    get: (id: string) =>
      this.get<ApiResponse<import('../types').HyperdriveConfig>>(`/hyperdrive/${id}`),
    create: (data: { name: string; origin: import('../types').HyperdriveOrigin & { password: string }; caching?: Partial<import('../types').HyperdriveCaching> }) =>
      this.post<ApiResponse<import('../types').HyperdriveConfig>>('/hyperdrive', data),
    delete: (id: string) =>
      this.delete<ApiResponse<void>>(`/hyperdrive/${id}`),
    update: (id: string, data: { origin?: Partial<import('../types').HyperdriveOrigin & { password?: string }>; caching?: Partial<import('../types').HyperdriveCaching> }) =>
      this.put<ApiResponse<import('../types').HyperdriveConfig>>(`/hyperdrive/${id}`, data),
    getStats: (id: string) =>
      this.get<ApiResponse<import('../types').HyperdriveStats>>(`/hyperdrive/${id}/stats`),
  }

  // Cron Triggers
  cron = {
    list: () =>
      this.get<ApiResponse<{ triggers: import('../types').CronTrigger[] }>>('/cron'),
    get: (id: string) =>
      this.get<ApiResponse<import('../types').CronTrigger>>(`/cron/${id}`),
    create: (data: { cron: string; script_name: string; enabled?: boolean }) =>
      this.post<ApiResponse<import('../types').CronTrigger>>('/cron', data),
    delete: (id: string) =>
      this.delete<ApiResponse<void>>(`/cron/${id}`),
    update: (id: string, data: { cron?: string; script_name?: string; enabled?: boolean }) =>
      this.put<ApiResponse<import('../types').CronTrigger>>(`/cron/${id}`, data),
    getExecutions: (id: string, limit?: number) =>
      this.get<ApiResponse<{ executions: import('../types').CronExecution[] }>>(`/cron/${id}/executions${limit ? `?limit=${limit}` : ''}`),
    trigger: (id: string) =>
      this.post<ApiResponse<import('../types').CronExecution>>(`/cron/${id}/trigger`),
  }

  // Dashboard
  dashboard = {
    getStats: () =>
      this.get<ApiResponse<import('../types').DashboardStats>>('/dashboard/stats'),
    getTimeSeries: (metric: string, range: '1h' | '24h' | '7d' | '30d') =>
      this.get<ApiResponse<{ data: import('../types').TimeSeriesData[] }>>(`/dashboard/timeseries?metric=${metric}&range=${range}`),
    getActivity: (limit?: number) =>
      this.get<ApiResponse<{ events: import('../types').ActivityEvent[] }>>(`/dashboard/activity${limit ? `?limit=${limit}` : ''}`),
    getStatus: () =>
      this.get<ApiResponse<{ services: import('../types').SystemStatus[] }>>('/dashboard/status'),
  }
}

export const api = new ApiClient()
