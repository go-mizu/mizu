import type {
  ApiResponse,
  DurableObjectStorage,
  AnalyticsDataPoint,
  VectorInsertItem,
  WorkerCreateRequest,
  WorkerBindingRequest,
  QueueSettings,
  AIGatewaySettings,
  HyperdriveOrigin,
  HyperdriveCaching,
} from '../types'

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
      this.post<ApiResponse<{ results: DurableObjectStorage[] }>>(`/durable-objects/namespaces/${namespaceId}/objects/${objectId}/query`, { query }),
  }

  // Queues
  queues = {
    list: () =>
      this.get<ApiResponse<{ queues: import('../types').Queue[] }>>('/queues'),
    get: (id: string) =>
      this.get<ApiResponse<import('../types').Queue>>(`/queues/${id}`),
    create: (data: { name: string; settings?: Partial<QueueSettings> }) =>
      this.post<ApiResponse<import('../types').Queue>>('/queues', data),
    delete: (id: string) =>
      this.delete<ApiResponse<void>>(`/queues/${id}`),
    updateSettings: (id: string, settings: Partial<QueueSettings>) =>
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
    insert: (name: string, vectors: VectorInsertItem[]) =>
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
    write: (name: string, data: AnalyticsDataPoint) =>
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
    create: (data: { name: string; settings?: Partial<AIGatewaySettings> }) =>
      this.post<ApiResponse<import('../types').AIGateway>>('/ai-gateway', data),
    delete: (id: string) =>
      this.delete<ApiResponse<void>>(`/ai-gateway/${id}`),
    updateSettings: (id: string, settings: Partial<AIGatewaySettings>) =>
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
    create: (data: { name: string; origin: HyperdriveOrigin & { password: string }; caching?: Partial<HyperdriveCaching> }) =>
      this.post<ApiResponse<import('../types').HyperdriveConfig>>('/hyperdrive', data),
    delete: (id: string) =>
      this.delete<ApiResponse<void>>(`/hyperdrive/${id}`),
    update: (id: string, data: { origin?: Partial<HyperdriveOrigin & { password?: string }>; caching?: Partial<HyperdriveCaching> }) =>
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

  // Workers
  workers = {
    list: () =>
      this.get<ApiResponse<{ workers: import('../types').Worker[] }>>('/workers'),
    get: (name: string) =>
      this.get<ApiResponse<import('../types').Worker>>(`/workers/${name}`),
    create: (data: WorkerCreateRequest) =>
      this.post<ApiResponse<import('../types').Worker>>('/workers', data),
    delete: (name: string) =>
      this.delete<ApiResponse<void>>(`/workers/${name}`),
    deploy: (name: string) =>
      this.post<ApiResponse<void>>(`/workers/${name}/deploy`),
    getVersions: (name: string) =>
      this.get<ApiResponse<{ versions: import('../types').WorkerVersion[] }>>(`/workers/${name}/versions`),
    rollback: (name: string, versionId: string) =>
      this.post<ApiResponse<void>>(`/workers/${name}/rollback`, { version_id: versionId }),
    addBinding: (name: string, binding: WorkerBindingRequest) =>
      this.post<ApiResponse<void>>(`/workers/${name}/bindings`, binding),
    removeBinding: (name: string, bindingName: string) =>
      this.delete<ApiResponse<void>>(`/workers/${name}/bindings/${bindingName}`),
  }

  // KV
  kv = {
    listNamespaces: () =>
      this.get<ApiResponse<{ namespaces: import('../types').KVNamespace[] }>>('/kv/namespaces'),
    getNamespace: (id: string) =>
      this.get<ApiResponse<import('../types').KVNamespace>>(`/kv/namespaces/${id}`),
    createNamespace: (data: { title: string }) =>
      this.post<ApiResponse<import('../types').KVNamespace>>('/kv/namespaces', data),
    deleteNamespace: (id: string) =>
      this.delete<ApiResponse<void>>(`/kv/namespaces/${id}`),
    listKeys: (namespaceId: string, params?: { prefix?: string; limit?: number; cursor?: string }) => {
      const query = new URLSearchParams()
      if (params?.prefix) query.set('prefix', params.prefix)
      if (params?.limit) query.set('limit', params.limit.toString())
      if (params?.cursor) query.set('cursor', params.cursor)
      return this.get<ApiResponse<{ keys: import('../types').KVKey[]; cursor?: string }>>(`/kv/namespaces/${namespaceId}/keys?${query}`)
    },
    getKey: (namespaceId: string, key: string) =>
      this.get<ApiResponse<{ value: string }>>(`/kv/namespaces/${namespaceId}/values/${encodeURIComponent(key)}`),
    putKey: (namespaceId: string, key: string, data: { value: string; expiration_ttl?: number }) =>
      this.put<ApiResponse<void>>(`/kv/namespaces/${namespaceId}/values/${encodeURIComponent(key)}`, data),
    deleteKey: (namespaceId: string, key: string) =>
      this.delete<ApiResponse<void>>(`/kv/namespaces/${namespaceId}/values/${encodeURIComponent(key)}`),
  }

  // R2
  r2 = {
    listBuckets: () =>
      this.get<ApiResponse<{ buckets: import('../types').R2Bucket[] }>>('/r2/buckets'),
    getBucket: (name: string) =>
      this.get<ApiResponse<import('../types').R2Bucket>>(`/r2/buckets/${name}`),
    createBucket: (data: { name: string; location_hint?: string }) =>
      this.post<ApiResponse<import('../types').R2Bucket>>('/r2/buckets', data),
    deleteBucket: (name: string) =>
      this.delete<ApiResponse<void>>(`/r2/buckets/${name}`),
    listObjects: (bucket: string, params?: { prefix?: string; delimiter?: string; cursor?: string }) => {
      const query = new URLSearchParams()
      if (params?.prefix) query.set('prefix', params.prefix)
      if (params?.delimiter) query.set('delimiter', params.delimiter)
      if (params?.cursor) query.set('cursor', params.cursor)
      return this.get<ApiResponse<{ objects: import('../types').R2Object[]; common_prefixes?: string[] }>>(`/r2/buckets/${bucket}/objects?${query}`)
    },
    putObject: async (bucket: string, key: string, file: File) => {
      const formData = new FormData()
      formData.append('file', file)
      return fetch(`/api/r2/buckets/${bucket}/objects/${encodeURIComponent(key)}`, {
        method: 'PUT',
        body: formData,
      })
    },
    deleteObject: (bucket: string, key: string) =>
      this.delete<ApiResponse<void>>(`/r2/buckets/${bucket}/objects/${encodeURIComponent(key)}`),
  }

  // D1
  d1 = {
    listDatabases: () =>
      this.get<ApiResponse<{ databases: import('../types').D1Database[] }>>('/d1/databases'),
    getDatabase: (id: string) =>
      this.get<ApiResponse<import('../types').D1Database>>(`/d1/databases/${id}`),
    createDatabase: (data: { name: string }) =>
      this.post<ApiResponse<import('../types').D1Database>>('/d1/databases', data),
    deleteDatabase: (id: string) =>
      this.delete<ApiResponse<void>>(`/d1/databases/${id}`),
    getTables: (id: string) =>
      this.get<ApiResponse<{ tables: import('../types').D1Table[] }>>(`/d1/databases/${id}/tables`),
    query: (id: string, sql: string) =>
      this.post<ApiResponse<import('../types').D1QueryResult>>(`/d1/databases/${id}/query`, { sql }),
  }

  // Pages
  pages = {
    listProjects: () =>
      this.get<ApiResponse<{ projects: import('../types').PagesProject[] }>>('/pages/projects'),
    getProject: (name: string) =>
      this.get<ApiResponse<import('../types').PagesProject>>(`/pages/projects/${name}`),
    createProject: (data: { name: string; production_branch?: string }) =>
      this.post<ApiResponse<import('../types').PagesProject>>('/pages/projects', data),
    deleteProject: (name: string) =>
      this.delete<ApiResponse<void>>(`/pages/projects/${name}`),
    getDeployments: (name: string) =>
      this.get<ApiResponse<{ deployments: import('../types').PagesDeployment[] }>>(`/pages/projects/${name}/deployments`),
  }

  // Images
  images = {
    list: () =>
      this.get<ApiResponse<{ images: import('../types').CloudflareImage[] }>>('/images'),
    upload: async (file: File) => {
      const formData = new FormData()
      formData.append('file', file)
      return fetch('/api/images/upload', {
        method: 'POST',
        body: formData,
      })
    },
    delete: (id: string) =>
      this.delete<ApiResponse<void>>(`/images/${id}`),
    listVariants: () =>
      this.get<ApiResponse<{ variants: import('../types').ImageVariant[] }>>('/images/variants'),
  }

  // Stream
  stream = {
    listVideos: () =>
      this.get<ApiResponse<{ videos: import('../types').StreamVideo[] }>>('/stream/videos'),
    getVideo: (id: string) =>
      this.get<ApiResponse<import('../types').StreamVideo>>(`/stream/videos/${id}`),
    upload: async (file: File) => {
      const formData = new FormData()
      formData.append('file', file)
      return fetch('/api/stream/upload', {
        method: 'POST',
        body: formData,
      })
    },
    deleteVideo: (id: string) =>
      this.delete<ApiResponse<void>>(`/stream/videos/${id}`),
    listLiveInputs: () =>
      this.get<ApiResponse<{ live_inputs: import('../types').LiveInput[] }>>('/stream/live'),
    createLiveInput: (data: { name: string }) =>
      this.post<ApiResponse<import('../types').LiveInput>>('/stream/live', data),
  }

  // Observability
  observability = {
    getLogs: (params?: { level?: string; worker?: string; limit?: number }) => {
      const query = new URLSearchParams()
      if (params?.level) query.set('level', params.level)
      if (params?.worker) query.set('worker', params.worker)
      if (params?.limit) query.set('limit', params.limit.toString())
      return this.get<ApiResponse<{ logs: import('../types').LogEntry[] }>>(`/observability/logs?${query}`)
    },
    getTraces: () =>
      this.get<ApiResponse<{ traces: import('../types').Trace[] }>>('/observability/traces'),
    getMetrics: (range: string) =>
      this.get<ApiResponse<{ data: import('../types').TimeSeriesData[] }>>(`/observability/metrics?range=${range}`),
  }

  // Settings
  settings = {
    listTokens: () =>
      this.get<ApiResponse<{ tokens: import('../types').APIToken[] }>>('/settings/tokens'),
    createToken: (data: { name: string; permissions: string; expiration: string }) =>
      this.post<ApiResponse<{ token: string }>>('/settings/tokens', data),
    revokeToken: (id: string) =>
      this.delete<ApiResponse<void>>(`/settings/tokens/${id}`),
    listMembers: () =>
      this.get<ApiResponse<{ members: import('../types').AccountMember[] }>>('/settings/members'),
    inviteMember: (data: { email: string; role: string }) =>
      this.post<ApiResponse<void>>('/settings/members', data),
    removeMember: (id: string) =>
      this.delete<ApiResponse<void>>(`/settings/members/${id}`),
  }
}

export const api = new ApiClient()
