import { render, type RenderOptions } from '@testing-library/react'
import { MantineProvider, createTheme } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { BrowserRouter, MemoryRouter } from 'react-router-dom'
import { type ReactElement, type ReactNode } from 'react'
import type {
  DashboardStats,
  DurableObjectNamespace,
  Queue,
  VectorIndex,
  AnalyticsDataset,
  AIGateway,
  HyperdriveConfig,
  CronTrigger,
  SystemStatus,
  ActivityEvent,
  TimeSeriesData,
  Worker,
  KVNamespace,
  R2Bucket,
  D1Database,
  PagesProject,
  CloudflareImage,
  ImageVariant,
  StreamVideo,
  LiveInput,
  ApiResponse,
} from '../types'

// Test theme matching main.tsx
const testTheme = createTheme({
  primaryColor: 'orange',
  defaultRadius: 'md',
})

interface WrapperProps {
  children: ReactNode
  initialEntries?: string[]
}

// Default wrapper with all providers
function AllProviders({ children, initialEntries }: WrapperProps) {
  const Router = initialEntries ? MemoryRouter : BrowserRouter
  const routerProps = initialEntries ? { initialEntries } : {}

  return (
    <MantineProvider theme={testTheme} defaultColorScheme="light">
      <Notifications position="top-right" />
      <Router {...routerProps}>{children}</Router>
    </MantineProvider>
  )
}

// Custom render function
interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  initialEntries?: string[]
}

export function renderWithProviders(
  ui: ReactElement,
  options?: CustomRenderOptions
) {
  const { initialEntries, ...renderOptions } = options || {}

  return render(ui, {
    wrapper: ({ children }) => (
      <AllProviders initialEntries={initialEntries}>{children}</AllProviders>
    ),
    ...renderOptions,
  })
}

// API base URL for integration tests - connects to REAL backend server
// The backend server must be running at this URL for tests to pass
// Use VITE_TEST_API_URL env var to override (e.g., for CI/CD)
// Default: http://localhost:8787/api (matches the Makefile backend port)
const API_BASE_URL = process.env.VITE_TEST_API_URL || 'http://localhost:8787/api'

// Real API client for integration tests
export const testApi = {
  async get<T>(path: string): Promise<ApiResponse<T>> {
    const response = await fetch(`${API_BASE_URL}${path}`)
    if (!response.ok) {
      throw new Error(`API request failed: ${response.status}`)
    }
    return response.json()
  },

  async post<T>(path: string, data?: unknown): Promise<ApiResponse<T>> {
    const response = await fetch(`${API_BASE_URL}${path}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: data ? JSON.stringify(data) : undefined,
    })
    if (!response.ok) {
      throw new Error(`API request failed: ${response.status}`)
    }
    return response.json()
  },

  async delete<T>(path: string): Promise<ApiResponse<T>> {
    const response = await fetch(`${API_BASE_URL}${path}`, {
      method: 'DELETE',
    })
    if (!response.ok) {
      throw new Error(`API request failed: ${response.status}`)
    }
    return response.json()
  },

  // Service-specific API methods
  dashboard: {
    getStats: () => testApi.get<DashboardStats>('/dashboard/stats'),
    getStatus: () => testApi.get<{ services: SystemStatus[] }>('/dashboard/status'),
    getActivity: (limit?: number) => testApi.get<{ events: ActivityEvent[] }>(`/dashboard/activity${limit ? `?limit=${limit}` : ''}`),
    getTimeSeries: (range = '24h') => testApi.get<{ data: TimeSeriesData[] }>(`/dashboard/timeseries?range=${range}`),
  },

  workers: {
    list: () => testApi.get<{ workers: Worker[] }>('/workers'),
    get: (id: string) => testApi.get<Worker>(`/workers/${id}`),
    create: (data: { name: string; script?: string }) => testApi.post<Worker>('/workers', data),
    delete: (id: string) => testApi.delete<void>(`/workers/${id}`),
  },

  queues: {
    list: () => testApi.get<{ queues: Queue[] }>('/queues'),
    get: (id: string) => testApi.get<Queue>(`/queues/${id}`),
    create: (data: { queue_name: string }) => testApi.post<Queue>('/queues', data),
    delete: (id: string) => testApi.delete<void>(`/queues/${id}`),
  },

  durableObjects: {
    listNamespaces: () => testApi.get<{ namespaces: DurableObjectNamespace[] }>('/durable-objects/namespaces'),
    createNamespace: (data: { name: string; class_name: string; script_name?: string }) =>
      testApi.post<DurableObjectNamespace>('/durable-objects/namespaces', data),
    deleteNamespace: (id: string) => testApi.delete<void>(`/durable-objects/namespaces/${id}`),
  },

  vectorize: {
    listIndexes: () => testApi.get<{ indexes: VectorIndex[] }>('/vectorize/indexes'),
    createIndex: (data: { name: string; dimensions: number; metric: string }) =>
      testApi.post<VectorIndex>('/vectorize/indexes', data),
    deleteIndex: (name: string) => testApi.delete<void>(`/vectorize/indexes/${name}`),
  },

  analytics: {
    listDatasets: () => testApi.get<{ datasets: AnalyticsDataset[] }>('/analytics-engine/datasets'),
    createDataset: (data: { name: string }) => testApi.post<AnalyticsDataset>('/analytics-engine/datasets', data),
    deleteDataset: (name: string) => testApi.delete<void>(`/analytics-engine/datasets/${name}`),
  },

  aiGateway: {
    list: () => testApi.get<{ gateways: AIGateway[] }>('/ai-gateway'),
    create: (data: { name: string }) => testApi.post<AIGateway>('/ai-gateway', data),
    delete: (id: string) => testApi.delete<void>(`/ai-gateway/${id}`),
  },

  hyperdrive: {
    list: () => testApi.get<{ configs: HyperdriveConfig[] }>('/hyperdrive/configs'),
    create: (data: { name: string; origin: { scheme: string; host: string; port: number; database: string; user: string; password: string } }) =>
      testApi.post<HyperdriveConfig>('/hyperdrive/configs', data),
    delete: (id: string) => testApi.delete<void>(`/hyperdrive/configs/${id}`),
  },

  cron: {
    list: () => testApi.get<{ triggers: CronTrigger[] }>('/cron/triggers'),
    create: (data: { cron: string; script_name: string }) => testApi.post<CronTrigger>('/cron/triggers', data),
    delete: (id: string) => testApi.delete<void>(`/cron/triggers/${id}`),
  },

  kv: {
    listNamespaces: () => testApi.get<{ namespaces: KVNamespace[] }>('/kv/namespaces'),
    createNamespace: (data: { title: string }) => testApi.post<KVNamespace>('/kv/namespaces', data),
    deleteNamespace: (id: string) => testApi.delete<void>(`/kv/namespaces/${id}`),
  },

  r2: {
    listBuckets: () => testApi.get<{ buckets: R2Bucket[] }>('/r2/buckets'),
    createBucket: (data: { name: string }) => testApi.post<R2Bucket>('/r2/buckets', data),
    deleteBucket: (name: string) => testApi.delete<void>(`/r2/buckets/${name}`),
  },

  d1: {
    listDatabases: () => testApi.get<{ databases: D1Database[] }>('/d1/databases'),
    createDatabase: (data: { name: string }) => testApi.post<D1Database>('/d1/databases', data),
    deleteDatabase: (id: string) => testApi.delete<void>(`/d1/databases/${id}`),
  },

  pages: {
    listProjects: () => testApi.get<{ projects: PagesProject[] }>('/pages/projects'),
    createProject: (data: { name: string }) => testApi.post<PagesProject>('/pages/projects', data),
    deleteProject: (name: string) => testApi.delete<void>(`/pages/projects/${name}`),
  },

  images: {
    list: () => testApi.get<{ images: CloudflareImage[] }>('/images'),
    listVariants: () => testApi.get<{ variants: ImageVariant[] }>('/images/variants'),
  },

  stream: {
    listVideos: () => testApi.get<{ videos: StreamVideo[] }>('/stream/videos'),
    listLiveInputs: () => testApi.get<{ live_inputs: LiveInput[] }>('/stream/live'),
    createLiveInput: (data: { name: string }) => testApi.post<LiveInput>('/stream/live', data),
  },

  ai: {
    listModels: () => testApi.get<unknown[]>('/ai/models'),
  },
}

// Type guard functions for type-safe assertions
export function isDashboardStats(data: unknown): data is DashboardStats {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return (
    'durable_objects' in d &&
    'queues' in d &&
    'vectorize' in d &&
    'analytics' in d &&
    'ai' in d &&
    'ai_gateway' in d &&
    'hyperdrive' in d &&
    'cron' in d
  )
}

export function isWorker(data: unknown): data is Worker {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  // API returns 'enabled' instead of 'status'
  return 'id' in d && 'name' in d && 'created_at' in d && 'enabled' in d
}

export function isQueue(data: unknown): data is Queue {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'id' in d && 'name' in d && 'created_at' in d && 'settings' in d
}

export function isDurableObjectNamespace(data: unknown): data is DurableObjectNamespace {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'id' in d && 'name' in d && 'class_name' in d && 'created_at' in d
}

export function isVectorIndex(data: unknown): data is VectorIndex {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'id' in d && 'name' in d && 'created_at' in d
}

export function isAnalyticsDataset(data: unknown): data is AnalyticsDataset {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'id' in d && 'name' in d && 'created_at' in d
}

export function isAIGateway(data: unknown): data is AIGateway {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'id' in d && 'name' in d && 'created_at' in d
}

export function isHyperdriveConfig(data: unknown): data is HyperdriveConfig {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'id' in d && 'name' in d && 'created_at' in d
}

export function isCronTrigger(data: unknown): data is CronTrigger {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'id' in d && 'cron' in d && 'script_name' in d
}

export function isKVNamespace(data: unknown): data is KVNamespace {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'id' in d && 'title' in d && 'created_at' in d
}

export function isR2Bucket(data: unknown): data is R2Bucket {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'name' in d && 'created_at' in d
}

export function isD1Database(data: unknown): data is D1Database {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return ('uuid' in d || 'id' in d) && 'name' in d && 'created_at' in d
}

export function isPagesProject(data: unknown): data is PagesProject {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'name' in d && 'subdomain' in d && 'created_at' in d
}

export function isCloudflareImage(data: unknown): data is CloudflareImage {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'id' in d && 'filename' in d && 'uploaded' in d
}

export function isStreamVideo(data: unknown): data is StreamVideo {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'uid' in d && 'name' in d && 'created' in d
}

export function isLiveInput(data: unknown): data is LiveInput {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'uid' in d && 'name' in d && 'rtmps' in d
}

export function isSystemStatus(data: unknown): data is SystemStatus {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'service' in d && 'status' in d
}

export function isActivityEvent(data: unknown): data is ActivityEvent {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'id' in d && 'type' in d && 'message' in d && 'timestamp' in d
}

export function isTimeSeriesData(data: unknown): data is TimeSeriesData {
  if (!data || typeof data !== 'object') return false
  const d = data as Record<string, unknown>
  return 'timestamp' in d && 'value' in d
}

// Assertion helpers that throw descriptive errors
export function assertValidWorkers(workers: unknown[]): asserts workers is Worker[] {
  if (!Array.isArray(workers)) {
    throw new Error('Expected workers to be an array')
  }
  workers.forEach((worker, index) => {
    if (!isWorker(worker)) {
      throw new Error(`Worker at index ${index} is missing required fields: ${JSON.stringify(worker)}`)
    }
  })
}

export function assertValidQueues(queues: unknown[]): asserts queues is Queue[] {
  if (!Array.isArray(queues)) {
    throw new Error('Expected queues to be an array')
  }
  queues.forEach((queue, index) => {
    if (!isQueue(queue)) {
      throw new Error(`Queue at index ${index} is missing required fields: ${JSON.stringify(queue)}`)
    }
  })
}

export function assertValidDurableObjectNamespaces(namespaces: unknown[]): asserts namespaces is DurableObjectNamespace[] {
  if (!Array.isArray(namespaces)) {
    throw new Error('Expected namespaces to be an array')
  }
  namespaces.forEach((ns, index) => {
    if (!isDurableObjectNamespace(ns)) {
      throw new Error(`Namespace at index ${index} is missing required fields: ${JSON.stringify(ns)}`)
    }
  })
}

export function assertValidVectorIndexes(indexes: unknown[]): asserts indexes is VectorIndex[] {
  if (!Array.isArray(indexes)) {
    throw new Error('Expected indexes to be an array')
  }
  indexes.forEach((idx, index) => {
    if (!isVectorIndex(idx)) {
      throw new Error(`Index at index ${index} is missing required fields: ${JSON.stringify(idx)}`)
    }
  })
}

export function assertValidCronTriggers(triggers: unknown[]): asserts triggers is CronTrigger[] {
  if (!Array.isArray(triggers)) {
    throw new Error('Expected triggers to be an array')
  }
  triggers.forEach((trigger, index) => {
    if (!isCronTrigger(trigger)) {
      throw new Error(`Trigger at index ${index} is missing required fields: ${JSON.stringify(trigger)}`)
    }
  })
}

export function assertValidSystemStatuses(statuses: unknown[]): asserts statuses is SystemStatus[] {
  if (!Array.isArray(statuses)) {
    throw new Error('Expected statuses to be an array')
  }
  statuses.forEach((status, index) => {
    if (!isSystemStatus(status)) {
      throw new Error(`Status at index ${index} is missing required fields: ${JSON.stringify(status)}`)
    }
  })
}

// Wait for loading to complete
export async function waitForLoadingToComplete() {
  const { waitFor } = await import('@testing-library/react')
  await waitFor(
    () => {
      const loadingElements = document.querySelectorAll('[data-loading="true"]')
      if (loadingElements.length > 0) {
        throw new Error('Still loading')
      }
    },
    { timeout: 5000 }
  )
}

// Helper to generate unique test names
let testCounter = 0
export function generateTestName(prefix: string): string {
  testCounter++
  return `${prefix}-test-${Date.now()}-${testCounter}`
}

// Re-export everything from testing-library
export * from '@testing-library/react'
export { default as userEvent } from '@testing-library/user-event'
